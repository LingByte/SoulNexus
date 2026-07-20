package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	llmcache "github.com/LingByte/lingllm/cache"
	"gorm.io/gorm"
)

// Level is check severity: ok, warn, error.
type Level string

const (
	LevelOK    Level = "ok"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Check is one startup or runtime readiness item.
type Check struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	Level     Level  `json:"level"`
	Message   string `json:"message"`
	Detail    string `json:"detail,omitempty"`
	LatencyMs int64  `json:"latencyMs,omitempty"`
}

// Snapshot is the latest preflight result.
type Snapshot struct {
	CheckedAt time.Time `json:"checkedAt"`
	Checks    []Check   `json:"checks"`
}

// Options controls which probes run.
type Options struct {
	DB         *gorm.DB
	HTTPAddr   string
	CheckPorts bool // bind-test ports (startup only)
}

var (
	mu       sync.RWMutex
	snapshot Snapshot
)

// StoreSnapshot saves the latest result for API reads.
func StoreSnapshot(s Snapshot) {
	mu.Lock()
	snapshot = s
	mu.Unlock()
}

// GetSnapshot returns the cached preflight result.
func GetSnapshot() Snapshot {
	mu.RLock()
	defer mu.RUnlock()
	return snapshot
}

// HasErrors reports whether any check is error level.
func (s Snapshot) HasErrors() bool {
	for _, c := range s.Checks {
		if c.Level == LevelError {
			return true
		}
	}
	return false
}

// Run executes filesystem, credential, port, and dependency probes.
func Run(ctx context.Context, opts Options) Snapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	var checks []Check
	checks = append(checks, checkFilesystem()...)
	checks = append(checks, checkJWTKeys()...)
	checks = append(checks, checkTLS()...)
	checks = append(checks, checkDatabase(ctx, opts.DB)...)
	checks = append(checks, checkCache(ctx)...)
	if opts.CheckPorts {
		checks = append(checks, checkHTTPPort(opts.HTTPAddr)...)
	} else {
		checks = append(checks, Check{
			ID: "port.http", Category: "ports", Level: LevelOK,
			Message: "HTTP 监听中", Detail: strings.TrimSpace(opts.HTTPAddr),
		})
	}
	checks = append(checks, checkSessionSecret()...)
	checks = append(checks, checkPlatformAdminPassword()...)

	out := Snapshot{CheckedAt: time.Now(), Checks: checks}
	StoreSnapshot(out)
	return out
}

func checkFilesystem() []Check {
	var out []Check
	paths := []struct {
		id, path, purpose string
		needWrite         bool
	}{
		{"fs.jwt_dir", filepath.Dir(config.GlobalConfig.JWT.KeyFile), "JWT 密钥目录", true},
		{"fs.log_dir", filepath.Dir(config.GlobalConfig.Log.Filename), "日志目录", true},
	}
	uploadDir := common.GetEnv(pkgconst.ENV_UPLOAD_DIR)
	if uploadDir == "" {
		uploadDir = pkgconst.DefaultUploadDir
	}
	paths = append(paths, struct {
		id, path, purpose string
		needWrite         bool
	}{"fs.uploads", uploadDir, "上传目录", true})

	if strings.EqualFold(config.GlobalConfig.Database.Driver, pkgconst.DBDriverSQLite) {
		dsn := strings.TrimSpace(config.GlobalConfig.Database.DSN)
		if dsn != "" && !strings.Contains(dsn, "://") {
			paths = append(paths, struct {
				id, path, purpose string
				needWrite         bool
			}{"fs.sqlite", filepath.Dir(dsn), "SQLite 数据目录", true})
		}
	}

	for _, p := range paths {
		dir := strings.TrimSpace(p.path)
		if dir == "" || dir == "." {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil {
			lvl := LevelWarn
			if os.IsNotExist(err) {
				if mkErr := os.MkdirAll(dir, pkgconst.KeyDirPerm); mkErr != nil {
					out = append(out, Check{
						ID: p.id, Category: "filesystem", Level: LevelError,
						Message: fmt.Sprintf("%s不可用", p.purpose),
						Detail:  fmt.Sprintf("无法创建 %s: %v", dir, mkErr),
					})
					continue
				}
				info, err = os.Stat(dir)
			}
			if err != nil {
				out = append(out, Check{
					ID: p.id, Category: "filesystem", Level: lvl,
					Message: fmt.Sprintf("%s不可用", p.purpose), Detail: err.Error(),
				})
				continue
			}
		}
		if !info.IsDir() {
			out = append(out, Check{
				ID: p.id, Category: "filesystem", Level: LevelError,
				Message: fmt.Sprintf("%s不是目录", p.purpose), Detail: dir,
			})
			continue
		}
		if p.needWrite {
			probe := filepath.Join(dir, ".preflight_write")
			if err := os.WriteFile(probe, []byte("ok"), 0600); err != nil {
				out = append(out, Check{
					ID: p.id, Category: "filesystem", Level: LevelError,
					Message: fmt.Sprintf("%s不可写", p.purpose), Detail: err.Error(),
				})
				continue
			}
			_ = os.Remove(probe)
		}
		out = append(out, Check{
			ID: p.id, Category: "filesystem", Level: LevelOK,
			Message: fmt.Sprintf("%s就绪", p.purpose), Detail: dir,
		})
	}
	return out
}

func checkJWTKeys() []Check {
	keyFile := strings.TrimSpace(config.GlobalConfig.JWT.KeyFile)
	if keyFile == "" {
		return []Check{{
			ID: "jwt.key_file", Category: "credentials", Level: LevelWarn,
			Message: "未配置 JWT 密钥文件", Detail: "JWT_KEY_FILE",
		}}
	}
	if _, err := os.ReadFile(keyFile); err != nil {
		return []Check{{
			ID: "jwt.key_file", Category: "credentials", Level: LevelWarn,
			Message: "JWT 密钥文件尚不存在（首次启动将自动生成）", Detail: keyFile,
		}}
	}
	return []Check{{
		ID: "jwt.key_file", Category: "credentials", Level: LevelOK,
		Message: "JWT 密钥文件可读", Detail: keyFile,
	}}
}

func checkTLS() []Check {
	if !config.GlobalConfig.Server.SSLEnabled {
		return []Check{{
			ID: "tls.enabled", Category: "credentials", Level: LevelOK,
			Message: "HTTPS 未启用", Detail: "SSL_ENABLED=false",
		}}
	}
	certFile := strings.TrimSpace(config.GlobalConfig.Server.SSLCertFile)
	keyFile := strings.TrimSpace(config.GlobalConfig.Server.SSLKeyFile)
	if certFile == "" || keyFile == "" {
		return []Check{{
			ID: "tls.files", Category: "credentials", Level: LevelError,
			Message: "已启用 SSL 但未配置证书路径",
			Detail:  "SSL_CERT_FILE / SSL_KEY_FILE",
		}}
	}
	if _, err := os.ReadFile(certFile); err != nil {
		return []Check{{
			ID: "tls.cert", Category: "credentials", Level: LevelError,
			Message: "SSL 证书不可读", Detail: err.Error(),
		}}
	}
	if _, err := os.ReadFile(keyFile); err != nil {
		return []Check{{
			ID: "tls.key", Category: "credentials", Level: LevelError,
			Message: "SSL 私钥不可读", Detail: err.Error(),
		}}
	}
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		return []Check{{
			ID: "tls.pair", Category: "credentials", Level: LevelError,
			Message: "SSL 证书与私钥不匹配", Detail: err.Error(),
		}}
	}
	return []Check{{
		ID: "tls.pair", Category: "credentials", Level: LevelOK,
		Message: "SSL/TLS 证书加载正常", Detail: certFile,
	}}
}

func checkDatabase(ctx context.Context, db *gorm.DB) []Check {
	if db == nil {
		return []Check{{
			ID: "db.conn", Category: "database", Level: LevelError,
			Message: "数据库未连接",
		}}
	}
	sqlDB, err := db.DB()
	if err != nil {
		return []Check{{
			ID: "db.conn", Category: "database", Level: LevelError,
			Message: "数据库连接池不可用", Detail: err.Error(),
		}}
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	start := time.Now()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return []Check{{
			ID: "db.ping", Category: "database", Level: LevelError,
			Message: "数据库账号/连接不可用", Detail: err.Error(),
			LatencyMs: time.Since(start).Milliseconds(),
		}}
	}
	return []Check{{
		ID: "db.ping", Category: "database", Level: LevelOK,
		Message:   "数据库连通正常",
		Detail:    fmt.Sprintf("%s", config.GlobalConfig.Database.Driver),
		LatencyMs: time.Since(start).Milliseconds(),
	}}
}

func checkCache(ctx context.Context) []Check {
	cacheType := strings.ToLower(strings.TrimSpace(common.GetEnv("CACHE_TYPE")))
	if cacheType == "" {
		cacheType = "local"
	}
	if cacheType != "redis" {
		return []Check{{
			ID: "cache.backend", Category: "dependencies", Level: LevelOK,
			Message: "缓存使用进程内 local", Detail: cacheType,
		}}
	}
	addr := strings.TrimSpace(common.GetEnv("REDIS_ADDR"))
	if addr == "" {
		return []Check{{
			ID: "cache.redis", Category: "dependencies", Level: LevelError,
			Message: "CACHE_TYPE=redis 但未配置 REDIS_ADDR",
		}}
	}
	c := llmcache.GetGlobalCache()
	if c == nil {
		return []Check{{
			ID: "cache.redis", Category: "dependencies", Level: LevelError,
			Message: "Redis 缓存未初始化", Detail: addr,
		}}
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	if err := c.Set(pingCtx, "__preflight_ping__", "1", time.Second); err != nil {
		return []Check{{
			ID: "cache.redis", Category: "dependencies", Level: LevelError,
			Message: "Redis 不可达", Detail: err.Error(),
			LatencyMs: time.Since(start).Milliseconds(),
		}}
	}
	_ = c.Delete(pingCtx, "__preflight_ping__")
	return []Check{{
		ID: "cache.redis", Category: "dependencies", Level: LevelOK,
		Message: "Redis 连通正常", Detail: addr,
		LatencyMs: time.Since(start).Milliseconds(),
	}}
}

func checkHTTPPort(addr string) []Check {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = pkgconst.DefaultServerAddr
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return []Check{{
			ID: "port.http", Category: "ports", Level: LevelError,
			Message: "HTTP 端口被占用或不可绑定", Detail: fmt.Sprintf("%s: %v", addr, err),
		}}
	}
	_ = ln.Close()
	return []Check{{
		ID: "port.http", Category: "ports", Level: LevelOK,
		Message: "HTTP 端口可用", Detail: addr,
	}}
}

var weakSessionSecrets = map[string]struct{}{
	"change-me-soulnexus-dev-secret-32b": {},
	"change-me":                          {},
	"changeme":                           {},
	"secret":                             {},
	"session_secret":                     {},
	"soulnexus":                          {},
	"admin123":                           {},
}

func isProductionMode() bool {
	if config.GlobalConfig == nil {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(config.GlobalConfig.Server.Mode))
	return mode == pkgconst.ENV_PROD || mode == "production"
}

func checkSessionSecret() []Check {
	secret := strings.TrimSpace(common.GetEnv(constants.ENV_SESSION_SECRET))
	prod := isProductionMode()
	if secret == "" {
		if prod {
			return []Check{{
				ID: "auth.session", Category: "credentials", Level: LevelError,
				Message: "生产环境必须设置 SESSION_SECRET（32+ 字节随机字符串）",
			}}
		}
		return []Check{{
			ID: "auth.session", Category: "credentials", Level: LevelWarn,
			Message: "未设置 SESSION_SECRET（开发模式使用临时会话）",
		}}
	}
	if _, weak := weakSessionSecrets[strings.ToLower(secret)]; weak {
		if prod {
			return []Check{{
				ID: "auth.session", Category: "credentials", Level: LevelError,
				Message: "生产环境禁止使用弱 SESSION_SECRET 默认值",
			}}
		}
		return []Check{{
			ID: "auth.session", Category: "credentials", Level: LevelWarn,
			Message: "SESSION_SECRET 为已知弱默认值，请更换后再用于生产",
		}}
	}
	if prod && len(secret) < 32 {
		return []Check{{
			ID: "auth.session", Category: "credentials", Level: LevelError,
			Message: "生产环境 SESSION_SECRET 长度须至少 32 字节",
			Detail:  fmt.Sprintf("当前长度=%d", len(secret)),
		}}
	}
	return []Check{{
		ID: "auth.session", Category: "credentials", Level: LevelOK,
		Message: "SESSION_SECRET 已配置",
	}}
}

func checkPlatformAdminPassword() []Check {
	if !isProductionMode() {
		return nil
	}
	pw := strings.TrimSpace(common.GetEnv("PLATFORM_ADMIN_PASSWORD"))
	if pw == "" {
		// Seed may be skipped; empty is OK when not seeding.
		return nil
	}
	if pw == "admin123" || len(pw) < 10 {
		return []Check{{
			ID: "auth.platform_admin_password", Category: "credentials", Level: LevelError,
			Message: "生产环境 PLATFORM_ADMIN_PASSWORD 过弱（禁止 admin123，长度须≥10）",
		}}
	}
	return []Check{{
		ID: "auth.platform_admin_password", Category: "credentials", Level: LevelOK,
		Message: "PLATFORM_ADMIN_PASSWORD 强度可接受",
	}}
}

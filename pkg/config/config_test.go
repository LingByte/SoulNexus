package config

import (
	"os"
	"testing"
)

// 为了避免不同用例间互相污染，统一用 t.Setenv 设置环境变量
func setAllEnvs(t *testing.T) {
	t.Setenv("MACHINE_ID", "7")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DSN", "host=127.0.0.1 user=u dbname=d sslmode=disable")
	t.Setenv("ADDR", ":8080")
	t.Setenv("MODE", "release")

	t.Setenv("DOCS_PREFIX", "/docs")
	t.Setenv("API_PREFIX", "/api")
	t.Setenv("ADMIN_PREFIX", "/admin")
	t.Setenv("AUTH_PREFIX", "/auth")

	t.Setenv("SESSION_EXPIRE_DAYS", "14")
	t.Setenv("SESSION_SECRET", "secret-xyz")

	// 日志
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_FILENAME", "app.log")
	t.Setenv("LOG_MAX_SIZE", "128")
	t.Setenv("LOG_MAX_AGE", "14")
	t.Setenv("LOG_MAX_BACKUPS", "7")

	// 邮件
	t.Setenv("SENDCLOUD_API_USER", "LingEcho")
	t.Setenv("SENDCLOUD_API_KEY", "14b6e48501c452407421917c943be0c3")
	t.Setenv("SENDCLOUD_FROM_EMAIL", "19511899044@163.com")

	// LLM
	t.Setenv("LLM_API_KEY", "ak")
	t.Setenv("LLM_BASE_URL", "https://llm.example.com")
	t.Setenv("LLM_MODEL", "gpt-x")

	// Search
	t.Setenv("SEARCH_ENABLED", "1")
	t.Setenv("SEARCH_PATH", "/var/search")
	t.Setenv("SEARCH_BATCH_SIZE", "500")

	t.Setenv("MONITOR_PREFIX", "/monitor")
	t.Setenv("LANGUAGE_ENABLED", "true")
	t.Setenv("API_SECRET_KEY", "api-secret")

	// 备份
	t.Setenv("BACKUP_ENABLED", "true")
	t.Setenv("BACKUP_PATH", "/var/backup")
	t.Setenv("BACKUP_SCHEDULE", "0 2 * * *")

	// 七牛 ASR/TTS
	t.Setenv("QINIU_ASR_API_KEY", "q-asr-ak")
	t.Setenv("QINIU_ASR_BASE_URL", "https://asr.qiniu.example.com")
	t.Setenv("QINIU_TTS_API_KEY", "q-tts-ak")
	t.Setenv("QINIU_TTS_BASE_URL", "https://tts.qiniu.example.com")
}

func TestLoad_WithExplicitAppEnv(t *testing.T) {
	// 显式设置 APP_ENV，触发 util.LoadEnv(env) 的非默认分支
	t.Setenv("APP_ENV", "production")
	setAllEnvs(t)

	// 清空全局，避免前序测试污染
	GlobalConfig = nil

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if GlobalConfig == nil {
		t.Fatalf("GlobalConfig is nil after Load")
	}

	// 基本字段
	if GlobalConfig.MachineID != 7 {
		t.Fatalf("MachineID=%d, want 7", GlobalConfig.MachineID)
	}
	if GlobalConfig.Database.Driver != "postgres" {
		t.Fatalf("DBDriver=%q", GlobalConfig.Database.Driver)
	}
	if GlobalConfig.Database.DSN != "host=127.0.0.1 user=u dbname=d sslmode=disable" {
		t.Fatalf("DSN=%q", GlobalConfig.Database.DSN)
	}
	if GlobalConfig.Server.Addr != ":8080" || GlobalConfig.Server.Mode != "release" {
		t.Fatalf("Addr=%q Mode=%q", GlobalConfig.Server.Addr, GlobalConfig.Server.Mode)
	}

	// 路由前缀
	if GlobalConfig.Server.DocsPrefix != "/docs" ||
		GlobalConfig.Server.APIPrefix != "/api" ||
		GlobalConfig.Server.AdminPrefix != "/admin" ||
		GlobalConfig.Server.AuthPrefix != "/auth" {
		t.Fatalf("prefix mismatch: %+v", *GlobalConfig)
	}

	// Session
	if GlobalConfig.Auth.SecretExpireDays != "14" || GlobalConfig.Auth.SessionSecret != "secret-xyz" {
		t.Fatalf("session mismatch: %+v", *GlobalConfig)
	}

	// 日志
	if GlobalConfig.Log.Level != "info" ||
		GlobalConfig.Log.Filename != "app.log" ||
		GlobalConfig.Log.MaxSize != 128 ||
		GlobalConfig.Log.MaxAge != 14 ||
		GlobalConfig.Log.MaxBackups != 7 {
		t.Fatalf("log config mismatch: %+v", GlobalConfig.Log)
	}

	// 邮件
	if GlobalConfig.Services.Mail.APIUser != "LingEcho" ||
		GlobalConfig.Services.Mail.APIKey != "14b6e48501c452407421917c943be0c3" ||
		GlobalConfig.Services.Mail.From != "19511899044@163.com" {
		t.Fatalf("mail config mismatch: %+v", GlobalConfig.Services.Mail)
	}

	// LLM
	if GlobalConfig.Services.LLM.APIKey != "ak" ||
		GlobalConfig.Services.LLM.BaseURL != "https://llm.example.com" ||
		GlobalConfig.Services.LLM.Model != "gpt-x" {
		t.Fatalf("llm mismatch: %+v", *GlobalConfig)
	}

	// 其他
	if GlobalConfig.Server.MonitorPrefix != "/monitor" ||
		GlobalConfig.Auth.APISecretKey != "api-secret" {
		t.Fatalf("monitor/lang/api mismatch: %+v", *GlobalConfig)
	}
}

func TestLoad_DefaultsWhenAppEnvEmpty(t *testing.T) {
	// APP_ENV 为空，走默认 development 分支
	_ = os.Unsetenv("APP_ENV")
	setAllEnvs(t)

	GlobalConfig = nil
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if GlobalConfig == nil {
		t.Fatalf("GlobalConfig is nil after Load")
	}

	// 抽查几个关键字段，确认仍能正确从环境取值
	if GlobalConfig.MachineID != 7 {
		t.Fatalf("MachineID=%d, want 7", GlobalConfig.MachineID)
	}
}

package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/apidocs"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/handlers"
	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/tasks"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	_ "github.com/LingByte/SoulNexus/pkg/dialog/bootstrap"
	"github.com/LingByte/SoulNexus/pkg/health"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/otelx"
	acmeutil "github.com/LingByte/SoulNexus/pkg/tlsutil/acme"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/backup"
	"github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	voiceMetrics "github.com/LingByte/SoulNexus/pkg/voice/metrics"
	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LingEchoApp struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func NewLingEchoApp(db *gorm.DB) *LingEchoApp {
	captcha.InitGlobalManager(captcha.DefaultConfig())
	return &LingEchoApp{
		db:       db,
		handlers: handlers.NewHandlers(db),
	}
}

func isProductionServerMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == pkgconst.ENV_PROD || m == "production"
}

func (app *LingEchoApp) RegisterRoutes(r *gin.Engine, api huma.API) {
	// Register system routes (with /api prefix) and document them via Huma.
	app.handlers.Register(r, api)
}

func main() {
	// 1. Parse command-line parameters
	init := flag.Bool("init", false, "run schema setup at startup (GORM AutoMigrate + optional on-disk goose SQL)")
	autoMigrate := flag.Bool("automigrate", false, "force GORM AutoMigrate without goose (same schema path as -init when no SQL files exist)")
	seed := flag.Bool("seed", false, "seed database")
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "optional tenant seed .sql (e.g. scripts/sql/init.sql); only runs when this flag is set")
	flag.Parse()

	// 2. Set environment variables
	if *mode != "" {
		os.Setenv(pkgconst.ENV_MODE, *mode)
	}

	// 3. Load global configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}
	if err := config.GlobalConfig.Validate(); err != nil {
		panic("config validate failed: " + err.Error())
	}

	// 4. Print banner
	if err := bootstrap.PrintBannerFromFile(pkgconst.DefaultBannerFile, config.GlobalConfig.Server.Name); err != nil {
		log.Fatalf("unload banner: %v", err)
	}

	// 5. Initialize logger
	err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Server.Mode)
	if err != nil {
		panic(err)
	}

	// 6. Log configuration
	bootstrap.LogConfigInfo()

	// 7. Setup database (-init: AutoMigrate + optional on-disk goose; -automigrate: AutoMigrate only)
	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath: *initSQL,
		MigrateSQL:  *init,
		AutoMigrate: *autoMigrate,
		SeedNonProd: *seed,
	})

	if err != nil {
		logger.Error("database setup failed", zap.Error(err))
		return
	}

	if err = bootstrap.InitializeKeyManager(); err != nil {
		logger.Error("key manager initialization failed", zap.Error(err))
		return
	}

	var addr = config.GlobalConfig.Server.Addr
	if addr == "" {
		addr = pkgconst.DefaultServerAddr
	}
	pfSnap := utils.Run(context.Background(), utils.Options{
		DB:         db,
		HTTPAddr:   addr,
		CheckPorts: true,
	})
	for _, chk := range pfSnap.Checks {
		switch chk.Level {
		case utils.LevelError:
			logger.Error("preflight", zap.String("id", chk.ID), zap.String("message", chk.Message), zap.String("detail", chk.Detail))
		case utils.LevelWarn:
			logger.Warn("preflight", zap.String("id", chk.ID), zap.String("message", chk.Message), zap.String("detail", chk.Detail))
		}
	}
	if pfSnap.HasErrors() && isProductionServerMode(config.GlobalConfig.Server.Mode) {
		logger.Fatal("startup preflight failed — fix errors above before running in production")
	}

	if utils.GetBoolEnv(pkgconst.ENV_BACKUP_ENABLED) {
		backup.StartBackupScheduler(db, backup.DatabaseConfig{
			Driver: config.GlobalConfig.Database.Driver,
			DSN:    config.GlobalConfig.Database.DSN,
		})
	}

	// 8. Resolve listen address (also used by preflight above)

	// 9. Create application
	app := NewLingEchoApp(db)
	models.InitBillingReservations(db)
	aiReportScheduler := tasks.NewAIReportScheduler(db)
	aiReportScheduler.Start()
	statsScheduler := tasks.NewStatsScheduler(db)
	statsScheduler.Start()
	logRetention := tasks.NewLogRetentionCleaner(pkgconst.DefaultLogRetentionInterval)
	logRetention.Start()
	tasks.StartNotificationCleaner(db)
	tasks.StartAccountDeletionFinalizer(db)
	tasks.StartWebhookRetryWorker(db)
	audioPrefetch := tasks.NewAudioPrefetchWarmup(db, 10*time.Minute)
	audioPrefetch.Start()
	defer audioPrefetch.Stop()

	// System monitoring hooks (pkg/utils/system):
	//
	//   1) StartSystemMonitor   — Samples CPU/Mem/Disk every 5s in the
	//      background, writes into atomic.Value for /system/status and
	//      listeners to read. It starts its own `go func()` but lacks
	//      panic recovery, so we wrap it with SafeGo as a safety net to
	//      prevent a single gopsutil panic on unusual platforms from
	//      tearing down the entire monitoring pipeline.
	//
	//   2) Monitor              — Automatically dumps pprof to ./pprof/
	//      when CPU stays high. Only enabled when PROFILE_AUTO_PPROF=true
	//      to avoid flooding dev machines with profile files.
	//
	//   3) StartPyroScope       — Only mounts when PYROSCOPE_URL is set;
	//      no-op otherwise. Errors are only warned — missing observability
	//      should never crash the process.
	logger.SafeGo("system-monitor", system.StartSystemMonitor)
	system.SyncPrometheusRuntimeGauges()
	if utils.GetEnv(pkgconst.ENV_PROFILE_AUTO_PPROF) == "true" {
		logger.SafeGo("auto-pprof-watcher", system.Monitor)
	}
	if err := system.StartPyroScope(); err != nil {
		logger.Warn("pyroscope start failed (continuing without continuous profiling)", zap.Error(err))
	}

	// 10. Initialize Gin engine
	gin.SetMode(gin.ReleaseMode)
	r := gin.New() // Use gin.New() instead of gin.Default() to avoid automatic redirects
	r.Use(middleware.PanicRecovery())
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.MaxMultipartMemory = pkgconst.DefaultMaxMultipartMemory // 32 MB

	// Cookie Register
	//
	// SESSION_SECRET present → persistent cookie store (sessions survive
	// process restarts; users stay logged in).
	//
	// SESSION_SECRET absent → in-memory store with a random secret
	// regenerated each boot. Every restart wipes all sessions and
	// invalidates all cookies. Acceptable for dev; loud warning so
	// operators don't ship to production this way.
	secret := utils.GetEnv(constants.ENV_SESSION_SECRET)
	if secret != "" {
		expireDays := utils.GetIntEnv(constants.ENV_SESSION_EXPIRE_DAYS)
		if expireDays <= 0 {
			expireDays = pkgconst.DefaultSessionExpireDays
		}
		r.Use(middleware.WithCookieSession(secret, int(expireDays)*24*3600))
	} else {
		logger.Warn("SESSION_SECRET UNSET using ephemeral memstore — sessions reset on every restart (DEV ONLY)")
		r.Use(middleware.WithMemSession(utils.RandText(pkgconst.DefaultSessionRandomKeyLen)))
	}

	bootstrap.ValidateProductionSecurityEnv()

	// OpenTelemetry (optional): set OTEL_EXPORTER_OTLP_ENDPOINT to enable.
	otelShutdown, err := otelx.Init(config.GlobalConfig.Server.Name)
	if err != nil {
		logger.Warn("OpenTelemetry init failed (continuing without tracing)", zap.Error(err))
		otelShutdown = func(context.Context) error { return nil }
	}

	// Traffic control + IP ACL (env-driven; no-op when disabled).
	middleware.InitTrafficControl(config.GlobalConfig.Middleware, config.GlobalConfig.Server.APIPrefix)
	middleware.InitIPACL(config.GlobalConfig.Middleware.IPACL)

	toCfg := middleware.DefaultTimeoutConfig()
	toCfg.DefaultTimeout = config.GlobalConfig.Middleware.Timeout.DefaultTimeout
	if config.GlobalConfig.Middleware.Timeout.FallbackResponse != nil {
		toCfg.FallbackResponse = config.GlobalConfig.Middleware.Timeout.FallbackResponse
	}
	cbCfg := middleware.CircuitBreakerConfig{
		FailureThreshold:      config.GlobalConfig.Middleware.CircuitBreaker.FailureThreshold,
		SuccessThreshold:      config.GlobalConfig.Middleware.CircuitBreaker.SuccessThreshold,
		Timeout:               config.GlobalConfig.Middleware.CircuitBreaker.Timeout,
		OpenTimeout:           config.GlobalConfig.Middleware.CircuitBreaker.OpenTimeout,
		MaxConcurrentRequests: config.GlobalConfig.Middleware.CircuitBreaker.MaxConcurrentRequests,
	}
	middleware.InitTimeoutCircuitManager(
		toCfg,
		cbCfg,
		config.GlobalConfig.Middleware.EnableTimeout,
		config.GlobalConfig.Middleware.EnableCircuitBreaker,
	)

	serviceName := config.GlobalConfig.Server.Name
	if serviceName == "" {
		serviceName = "soulnexus"
	}

	// Security headers, XSS/input sanitization.
	// CSRF: omitted for JWT/API-key APIs (Authorization header is not cookie-auth).
	// Cookie session routes (if any) rely on SameSite + CORS allowlist; re-audit
	// before exposing cookie-only mutating endpoints to third-party origins.
	for _, mw := range middleware.SecurityMiddlewareChain() {
		r.Use(mw)
	}

	// Cors Handle Middleware
	r.Use(middleware.CorsMiddleware())

	// Global IP ACL → rate limit → timeout/circuit (before auth).
	r.Use(middleware.GlobalIPACL())
	r.Use(middleware.BaseRateLimit())
	r.Use(middleware.CombinedTimeoutCircuitMiddleware())

	// Request id (X-ReqId) for all HTTP requests — must run before access log + handlers.
	r.Use(middleware.RequestIDMiddleware())
	r.Use(otelx.GinMiddleware(serviceName))

	// HTTP access log (includes X-ReqId)
	r.Use(middleware.LoggerMiddleware(logger.Lg))
	r.Use(middleware.HTTPMetricsMiddleware())
	r.Use(middleware.LocaleMiddleware())
	uploadDir := utils.GetEnv(pkgconst.ENV_UPLOAD_DIR)
	if uploadDir == "" {
		uploadDir = pkgconst.DefaultUploadDir
	}
	r.Use(middleware.UploadsACL())
	r.Static(pkgconst.UploadRoute, uploadDir)

	// Liveness / readiness (outside /api — no auth).
	health.RegisterGin(r, db)

	// Huma OpenAPI docs first — handlers.Register documents Gin routes via humax.
	api := apidocs.Mount(r, apidocs.Options{
		Title:     "SoulNexus", // topbar overrides via GET /api/system/init (SITE_NAME)
		Version:   "1.0.0",
		DocsPath:  config.GlobalConfig.Server.DocsPrefix,
		APIPrefix: config.GlobalConfig.Server.APIPrefix,
	})

	// 11. Register routes
	app.RegisterRoutes(r, api)

	// Expose the in-process metrics registry over Prometheus text
	// exposition. Default-deny via METRICS_ALLOWED_IPS — this used to
	// be wide-open and leaked deployment topology / live traffic
	// patterns to anyone who could reach the listener. Configure the
	// env to a comma list of IPs or CIDRs (e.g. "127.0.0.1,10.0.0.0/8")
	// or "*" if you front /metrics with mTLS / k8s NetworkPolicy.
	r.GET("/metrics", middleware.MetricsACL(), gin.WrapH(voiceMetrics.Handler()))

	// 12. Initialize system listeners
	listeners.InitAuthMailListeners(db)
	listeners.InitNotifyListeners(db)

	// 12.5 Workflow event listener + cron scheduler (SoulNexus port)
	workflowEventListener := workflowdef.NewWorkflowEventListener(db)
	if err := workflowEventListener.Start(); err != nil {
		logger.Error("Failed to start workflow event listener", zap.Error(err))
	} else {
		logger.Info("Workflow event listener started")
	}
	workflowScheduler := workflowdef.GetWorkflowScheduler(db)
	if err := workflowScheduler.Start(); err != nil {
		logger.Error("Failed to start workflow scheduler", zap.Error(err))
	} else {
		logger.Info("Workflow scheduler started")
	}
	listeners.InitSystemListeners()

	if strings.EqualFold(strings.TrimSpace(utils.GetEnv("CACHE_TYPE")), "redis") {
		redisAddr := strings.TrimSpace(utils.GetEnv("REDIS_ADDR"))
		if redisAddr != "" {
			health.RegisterChecker("redis", func(ctx context.Context) error {
				d := net.Dialer{Timeout: 2 * time.Second}
				conn, err := d.DialContext(ctx, "tcp", redisAddr)
				if err != nil {
					return err
				}
				_ = conn.Close()
				return nil
			})
		}
	}
	health.MarkReady()

	// 15. Start HTTP/HTTPS server
	//
	// Timeout rationale:
	//   - ReadHeaderTimeout: 30s — slowloris defence. Attackers that
	//     dribble the request line / headers byte-by-byte get cut off
	//     before they can hold a goroutine indefinitely.
	//   - ReadTimeout: 5min — generous enough for the largest legitimate
	//     uploads (32MB WAV / avatars) on slow links, but not unbounded.
	//   - WriteTimeout: 0 (disabled) — long-poll-ish endpoints (WS upgrades
	//     hijack the conn so this doesn't apply, but JSON handlers also
	//     occasionally take >30s when the LLM/ASR providers are slow).
	//     gin.Recovery + handler-level ctx timeouts manage misbehaviour;
	//     relying on WriteTimeout alone risks killing slow-but-correct
	//     streaming responses mid-flight.
	//   - IdleTimeout: 120s — close idle keep-alive sockets; prevents
	//     resource leaks from clients that disappear silently.
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: pkgconst.DefaultReadHeaderTimeout,
		ReadTimeout:       pkgconst.DefaultReadTimeout,
		WriteTimeout:      0,
		IdleTimeout:       pkgconst.DefaultIdleTimeout,
		MaxHeaderBytes:    pkgconst.DefaultMaxHeaderBytes,
	}

	shutdownAll := func() {
		health.MarkNotReady()
		ctx, cancel := context.WithTimeout(context.Background(), pkgconst.DefaultShutdownTimeout)
		defer cancel()
		_ = otelShutdown(ctx)
		if workflowEventListener != nil {
			workflowEventListener.Stop()
		}
		if workflowScheduler != nil {
			workflowScheduler.Stop()
		}
		app.handlers.StopKnowledgeWorker()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown", zap.Error(err))
		}
	}

	if listeners.IsSSLEnabled() {
		tlsConfig, err := listeners.GetTLSConfig()
		if err != nil {
			logger.Error("failed to get TLS config", zap.Error(err))
			return
		}
		if tlsConfig != nil {
			httpServer.TLSConfig = tlsConfig
		} else {
			logger.Warn("SSL enabled but TLS config is nil, falling back to HTTP")
		}
	}

	if m := acmeutil.HTTPHandler(); m != nil {
		if http01 := strings.TrimSpace(os.Getenv("SSL_ACME_HTTP_ADDR")); http01 != "" {
			go func() {
				logger.Info("ACME HTTP-01 challenge listener", zap.String("addr", http01))
				if err := http.ListenAndServe(http01, m.HTTPHandler(nil)); err != nil {
					logger.Warn("ACME HTTP-01 listener stopped", zap.Error(err))
				}
			}()
		}
	}

	go func() {
		var err error
		if httpServer.TLSConfig != nil {
			logger.Info("Starting HTTPS server", zap.String("addr", addr))
			err = httpServer.ListenAndServeTLS("", "")
		} else {
			logger.Info("Starting HTTP server", zap.String("addr", addr))
			err = httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server run failed", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, pkgconst.SignalChannelBufSize)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutdown signal received")
	shutdownAll()
}

package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// User service: standalone HTTP process exposing auth/user routes (register, login, profile, etc.).
// Uses the same database and config as the main API server until the schema is split.

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	handlers "github.com/LingByte/SoulNexus/internal/handler"
	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/task"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/metrics"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	utilscaptcha "github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type userServiceApp struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func newUserServiceApp(db *gorm.DB) *userServiceApp {
	return &userServiceApp{
		db:       db,
		handlers: handlers.NewUserServiceHandlers(db),
	}
}

func (app *userServiceApp) registerRoutes(r *gin.Engine) {
	app.handlers.RegisterUserServiceRoutes(r)
}

func main() {
	seed := flag.Bool("seed", false, "seed database")
	init := flag.Bool("init", false, "run GORM AutoMigrate on startup")
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")
	addrFlag := flag.String("addr", "", "HTTP listen address (overrides USER_SERVICE_HTTP_ADDR; default :7074)")
	flag.Parse()

	if *mode != "" {
		os.Setenv("APP_ENV", *mode)
	}

	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}

	if err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Server.Mode); err != nil {
		panic(err)
	}

	if err := bootstrap.PrintBannerFromFile("user-banner.txt", "SOULNEXUS USER"); err != nil {
		log.Fatalf("unload banner: %v", err)
	}

	bootstrap.LogConfigInfo()

	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath: *initSQL,
		AutoMigrate: *init,
		SeedNonProd: *seed,
	})
	if err != nil {
		logger.Error("database setup failed", zap.Error(err))
		return
	}

	addr := *addrFlag
	if addr == "" {
		addr = os.Getenv("USER_SERVICE_HTTP_ADDR")
	}
	if addr == "" {
		addr = ":7074"
	}

	logger.Info("user-service listen", zap.String("addr", addr), zap.Bool("init", *init))

	if err := cache.InitGlobalCache(config.GlobalConfig.Cache); err != nil {
		logger.Error("failed to initialize cache", zap.Error(err))
		logger.Info("falling back to default local cache")
	}
	utils.InitGlobalCache(1024, 5*time.Minute)
	utils.InitGlobalRegistrationGuard(logger.Lg)
	utils.InitGlobalDistributedLock()
	utilscaptcha.InitGlobalManager(nil)
	utils.InitGlobalLoginSecurityManager(logger.Lg)
	utils.InitGlobalIntelligentRiskControl(logger.Lg)

	app := newUserServiceApp(db)

	maxSpans := int(utils.GetIntEnv("METRICS_MAX_SPANS"))
	if maxSpans == 0 {
		maxSpans = 500
	}
	maxQueries := int(utils.GetIntEnv("METRICS_MAX_QUERIES"))
	if maxQueries == 0 {
		maxQueries = 500
	}
	maxStats := int(utils.GetIntEnv("METRICS_MAX_STATS"))
	if maxStats == 0 {
		maxStats = 100
	}
	enableTracing := utils.GetBoolEnv("METRICS_ENABLE_TRACING")
	enableSQLAnalysis := utils.GetBoolEnv("METRICS_ENABLE_SQL_ANALYSIS")
	if !enableSQLAnalysis && utils.GetEnv("METRICS_ENABLE_SQL_ANALYSIS") == "" {
		enableSQLAnalysis = true
	}
	enableSystemMonitor := utils.GetBoolEnv("METRICS_ENABLE_SYSTEM_MONITOR")
	if !enableSystemMonitor && utils.GetEnv("METRICS_ENABLE_SYSTEM_MONITOR") == "" {
		enableSystemMonitor = true
	}

	monitor := metrics.NewMonitor(&metrics.MonitorConfig{
		EnableMetrics:       true,
		EnableTracing:       enableTracing,
		MaxSpans:            maxSpans,
		EnableSQLAnalysis:   enableSQLAnalysis,
		MaxQueries:          maxQueries,
		SlowThreshold:       100 * time.Millisecond,
		EnableSystemMonitor: enableSystemMonitor,
		MaxStats:            maxStats,
		MonitorInterval:     30 * time.Second,
	})
	metrics.SetGlobalMonitor(monitor)
	monitor.Start()
	defer monitor.Stop()

	middleware.InitGlobalMiddlewareManager(config.GlobalConfig.Middleware)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecureResponseHeaders())
	templatesFS := LingEcho.NewCombineEmbedFS(
		LingEcho.HintAssetsRoot("templates"),
		LingEcho.EmbedFS{"templates", LingEcho.EmbedTemplates},
	)
	r.HTMLRender = LingEcho.NewCombineTemplates(templatesFS)
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.MaxMultipartMemory = 32 << 20

	r.Use(metrics.MonitorMiddleware(monitor))

	secret := utils.GetEnv(constants.ENV_SESSION_SECRET)
	if secret == "" && config.GlobalConfig != nil {
		secret = config.GlobalConfig.Auth.SessionSecret
	}
	if secret != "" {
		expireDays := utils.GetIntEnv(constants.ENV_SESSION_EXPIRE_DAYS)
		if expireDays <= 0 {
			expireDays = 7
		}
		r.Use(middleware.WithCookieSession(secret, int(expireDays)*24*3600))
	} else {
		r.Use(middleware.WithMemSession(utils.RandText(32)))
	}

	r.Use(middleware.CorsMiddleware())
	r.Use(middleware.LoggerMiddleware(zap.L()))

	r.Use(LingEcho.WithStaticAssets(r, utils.GetEnv(constants.ENV_STATIC_PREFIX), utils.GetEnv(constants.ENV_STATIC_ROOT)))
	staticRootDir := utils.GetEnv(constants.ENV_STATIC_ROOT)
	if staticRootDir == "" {
		staticRootDir = "static"
	}
	staticAssets := LingEcho.NewCombineEmbedFS(LingEcho.HintAssetsRoot(staticRootDir), LingEcho.EmbedFS{"static", LingEcho.EmbedStaticAssets})
	apiPrefix := config.GlobalConfig.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r.StaticFS(apiPrefix+"/static", http.FS(staticAssets))
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	app.registerRoutes(r)

	monitorPrefix := config.GlobalConfig.Server.MonitorPrefix
	if monitorPrefix == "" {
		monitorPrefix = "/metrics"
	}
	fullMonitorPrefix := apiPrefix + monitorPrefix
	monitorGroup := r.Group(fullMonitorPrefix)
	metrics.NewMonitorAPI(monitor).RegisterRoutes(monitorGroup)
	logger.Info("user-service metrics", zap.String("prefix", fullMonitorPrefix))

	listeners.InitSystemListeners()
	utils.Sig().Emit(models.SigInitSystemConfig, nil)

	task.StartAccountDeletionScheduler(db)

	httpServer := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    120 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if config.GlobalConfig.Server.SSLEnabled && listeners.IsSSLEnabled() {
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

	if httpServer.TLSConfig != nil {
		logger.Info("user-service HTTPS", zap.String("addr", addr))
		if err := httpServer.ListenAndServeTLS("", ""); err != nil {
			logger.Error("user-service failed", zap.Error(err))
		}
		return
	}

	logger.Info("user-service HTTP", zap.String("addr", addr))
	if err := httpServer.ListenAndServe(); err != nil {
		logger.Error("user-service failed", zap.Error(err))
	}
}

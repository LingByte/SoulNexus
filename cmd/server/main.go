package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/captcha"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/metrics"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/backup"
	"github.com/LingByte/SoulNexus/pkg/utils/search"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LingEchoApp struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func NewLingEchoApp(db *gorm.DB) *LingEchoApp {
	return &LingEchoApp{
		db:       db,
		handlers: handlers.NewHandlers(db),
	}
}

func (app *LingEchoApp) RegisterRoutes(r *gin.Engine) {
	// Register system routes (with /api prefix)
	app.handlers.Register(r)
}

func main() {
	// 1. Print Banner
	if err := bootstrap.PrintBannerFromFile("banner.txt"); err != nil {
		log.Fatalf("unload banner: %v", err)
	}

	// 2. Parse Command Line Parameters
	// Deprecated: parsed for backward compatibility; bootstrap always runs GORM AutoMigrate when connecting.
	init := flag.Bool("init", false, "deprecated: ignored; schema migration always runs at startup")
	seed := flag.Bool("seed", false, "seed database")
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")
	flag.Parse()

	// 3. Set Environment Variables
	if *mode != "" {
		os.Setenv("APP_ENV", *mode)
	}

	// 4. Load Global Configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}

	// 5. Load Log Configuration
	err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Server.Mode)
	if err != nil {
		panic(err)
	}

	// 6. Print Configuration
	bootstrap.LogConfigInfo()

	// 7. Load Data Source
	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath: *initSQL,
		AutoMigrate: *init,
		SeedNonProd: *seed,
	})
	if err != nil {
		logger.Error("database setup failed", zap.Error(err))
		return
	}

	// 8. Load Base Configs
	var addr = config.GlobalConfig.Server.Addr
	if addr == "" {
		addr = ":7072"
	}

	var DBDriver = config.GlobalConfig.Database.Driver
	if DBDriver == "" {
		DBDriver = "sqlite"
	}

	var DSN = config.GlobalConfig.Database.DSN
	if DSN == "" {
		DSN = "file::memory:?cache=shared"
	}
	flag.StringVar(&addr, "addr", addr, "HTTP Serve address")
	flag.StringVar(&DBDriver, "db-driver", DBDriver, "database driver")
	flag.StringVar(&DSN, "dsn", DSN, "database source name")

	logger.Info("checked config -- addr: ", zap.String("addr", addr))
	logger.Info("checked config -- db-driver: ", zap.String("db-driver", DBDriver), zap.String("dsn", DSN))
	logger.Info("checked config -- mode: ", zap.String("mode", config.GlobalConfig.Server.Mode))

	// 9. Load Global Cache (new cache system)
	if err := cache.InitGlobalCache(config.GlobalConfig.Cache); err != nil {
		logger.Error("failed to initialize cache", zap.Error(err))
		logger.Info("falling back to default local cache")
	}
	utils.InitGlobalCache(1024, 5*time.Minute)

	// Initialize global registration guard
	utils.InitGlobalRegistrationGuard(logger.Lg)

	// Initialize global distributed lock
	utils.InitGlobalDistributedLock()

	// Initialize global captcha manager
	captcha.InitGlobalCaptchaManager(nil) // Use memory storage, can be replaced with Redis storage

	// Initialize global login security manager
	utils.InitGlobalLoginSecurityManager(logger.Lg)

	// Initialize global intelligent risk control manager
	utils.InitGlobalIntelligentRiskControl(logger.Lg)

	//// 11. New App
	app := NewLingEchoApp(db)

	// 12. Initialize Monitoring System
	// Can be overridden via environment variables, default values suitable for 2GB memory servers
	maxSpansEnv := utils.GetIntEnv("METRICS_MAX_SPANS")
	maxQueriesEnv := utils.GetIntEnv("METRICS_MAX_QUERIES")
	maxStatsEnv := utils.GetIntEnv("METRICS_MAX_STATS")

	maxSpans := int(maxSpansEnv)
	if maxSpans == 0 {
		maxSpans = 500 // Default 500 (originally 10000), reducing 95% memory usage
	}

	maxQueries := int(maxQueriesEnv)
	if maxQueries == 0 {
		maxQueries = 500 // Default 500 (originally 10000), reducing 95% memory usage
	}

	maxStats := int(maxStatsEnv)
	if maxStats == 0 {
		maxStats = 100 // Default 100 (originally 1000), reducing 90% memory usage
	}

	// Tracing feature consumes the most memory, disabled by default
	enableTracing := utils.GetBoolEnv("METRICS_ENABLE_TRACING")
	enableSQLAnalysis := utils.GetBoolEnv("METRICS_ENABLE_SQL_ANALYSIS")
	if !enableSQLAnalysis && utils.GetEnv("METRICS_ENABLE_SQL_ANALYSIS") == "" {
		enableSQLAnalysis = true // Enable SQL analysis by default
	}
	enableSystemMonitor := utils.GetBoolEnv("METRICS_ENABLE_SYSTEM_MONITOR")
	if !enableSystemMonitor && utils.GetEnv("METRICS_ENABLE_SYSTEM_MONITOR") == "" {
		enableSystemMonitor = true // Enable system monitoring by default
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

	// 13. Set Global Monitor
	metrics.SetGlobalMonitor(monitor)

	monitor.Start()
	defer monitor.Stop()

	// 13.5. Initialize Global Middleware Manager
	middleware.InitGlobalMiddlewareManager(config.GlobalConfig.Middleware)
	logger.Info("Global middleware manager initialized with config",
		zap.Bool("rateLimit", config.GlobalConfig.Middleware.EnableRateLimit),
		zap.Bool("timeout", config.GlobalConfig.Middleware.EnableTimeout),
		zap.Bool("circuitBreaker", config.GlobalConfig.Middleware.EnableCircuitBreaker),
		zap.Bool("operationLog", config.GlobalConfig.Middleware.EnableOperationLog))

	// 14. Initialize Neo4j Graph Database (if enabled)
	if config.GlobalConfig.Services.KnowledgeBase.Neo4j.Enabled {
		graphStore, err := graph.NewNeo4jStore(
			config.GlobalConfig.Services.KnowledgeBase.Neo4j.URI,
			config.GlobalConfig.Services.KnowledgeBase.Neo4j.Username,
			config.GlobalConfig.Services.KnowledgeBase.Neo4j.Password,
			config.GlobalConfig.Services.KnowledgeBase.Neo4j.Database,
		)
		if err != nil {
			logger.Error("Failed to initialize Neo4j", zap.Error(err))
			logger.Warn("Graph processing will be disabled")
			task.InitGraphProcessor(nil, false)
			graph.SetDefaultStore(nil)
		} else {
			logger.Info("Neo4j graph database initialized successfully")
			task.InitGraphProcessor(graphStore, true)
			// Set global default graph storage instance for real-time assistants and other components to read user profiles
			graph.SetDefaultStore(graphStore)
			defer func() {
				if err := graphStore.Close(); err != nil {
					logger.Error("Failed to close Neo4j connection", zap.Error(err))
				}
			}()
		}
	} else {
		logger.Info("Neo4j is disabled, graph processing will be skipped")
		task.InitGraphProcessor(nil, false)
		graph.SetDefaultStore(nil)
	}

	// 15. Start Timed task
	task.StartEmailCleaner(db)
	task.StartQuotaAlertChecker(db)
	if config.GlobalConfig.Features.BackupEnabled {
		backup.StartBackupScheduler(db)
	}

	// 15. Initialize Gin Routing
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()        // Use gin.New() instead of gin.Default() to avoid automatic redirects
	r.Use(gin.Recovery()) // Manually add Recovery middleware
	r.LoadHTMLGlob("templates/**/**")
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.MaxMultipartMemory = 32 << 20 // 32 MB

	// 16. use middleware
	r.Use(metrics.MonitorMiddleware(monitor))

	// Cookie Register
	secret := utils.GetEnv(constants.ENV_SESSION_SECRET)
	if secret != "" {
		expireDays := utils.GetIntEnv(constants.ENV_SESSION_EXPIRE_DAYS)
		if expireDays <= 0 {
			expireDays = 7
		}
		r.Use(middleware.WithCookieSession(secret, int(expireDays)*24*3600))
	} else {
		r.Use(middleware.WithMemSession(utils.RandText(32)))
	}

	// Cors Handle Middleware
	r.Use(middleware.CorsMiddleware())

	// Logger Handle Middleware
	r.Use(middleware.LoggerMiddleware(zap.L()))

	// Assets Middleware
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

	// 18. Register Routes
	app.RegisterRoutes(r)
	// Get monitor prefix from config (default: /metrics)
	monitorPrefix := config.GlobalConfig.Server.MonitorPrefix
	if monitorPrefix == "" {
		monitorPrefix = "/metrics"
	}
	// Combine API prefix with monitor prefix: /api/metrics
	fullMonitorPrefix := apiPrefix + monitorPrefix
	monitorGroup := r.Group(fullMonitorPrefix)
	monitorAPI := metrics.NewMonitorAPI(monitor)
	monitorAPI.RegisterRoutes(monitorGroup)
	logger.Info("Metrics monitor routes registered", zap.String("prefix", fullMonitorPrefix))

	// 19. Initialize System Listener
	listeners.InitLLMListenerWithDB(db)
	listeners.InitBillingListenerWithDB(db)
	listeners.InitSystemListeners()

	// 20. Start Search Indexer (if enabled)
	searchEnabled := utils.GetBoolValue(db, constants.KEY_SEARCH_ENABLED)
	if !searchEnabled && config.GlobalConfig != nil {
		searchEnabled = config.GlobalConfig.Features.SearchEnabled
	}

	if searchEnabled {
		// Get search engine instance
		var searchEngine search.Engine
		if app.handlers.GetSearchHandler() != nil {
			searchEngine = app.handlers.GetSearchHandler().GetEngine()
		}
		if searchEngine != nil {
			// Start scheduled task
			task.StartSearchIndexer(db, searchEngine)
			// Asynchronously execute initial indexing (delayed execution to avoid memory spikes at startup)
			// For small memory servers, you can set environment variable SEARCH_DELAY_INDEX=true to delay indexing
			delayIndex := utils.GetBoolEnv("SEARCH_DELAY_INDEX")
			if delayIndex {
				// Delay 30 seconds before executing indexing, giving time for system startup
				go func() {
					time.Sleep(30 * time.Second)
					task.IndexUserDataAsync(db, searchEngine)
				}()
			} else {
				// Execute immediately by default (maintain original behavior)
				task.IndexUserDataAsync(db, searchEngine)
			}
		}
	}

	// 21. Emit system initialization signal
	utils.Sig().Emit(models.SigInitSystemConfig, nil)

	// 21.5. Start Workflow Event Listener and Scheduler
	eventListener := workflowdef.NewWorkflowEventListener(db)
	if err := eventListener.Start(); err != nil {
		logger.Error("Failed to start workflow event listener", zap.Error(err))
	} else {
		logger.Info("Workflow event listener started")
	}

	// Start workflow scheduler
	scheduler := workflowdef.GetWorkflowScheduler(db)
	if err := scheduler.Start(); err != nil {
		logger.Error("Failed to start workflow scheduler", zap.Error(err))
	}

	// 22. Start HTTP/HTTPS Server
	httpServer := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    300 * time.Second, // 5分钟，适合语音会话的长静音期
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Check if SSL is enabled
	if config.GlobalConfig.Server.SSLEnabled && listeners.IsSSLEnabled() {
		tlsConfig, err := listeners.GetTLSConfig()
		if err != nil {
			logger.Error("failed to get TLS config", zap.Error(err))
			return
		}

		if tlsConfig != nil {
			httpServer.TLSConfig = tlsConfig
			logger.Info("Starting HTTPS server", zap.String("addr", addr))
			if err := httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTPS server run failed", zap.Error(err))
			}
		} else {
			logger.Warn("SSL enabled but TLS config is nil, falling back to HTTP")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTP server run failed", zap.Error(err))
			}
		}
	} else {
		logger.Info("Starting HTTP server", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server run failed", zap.Error(err))
		}
	}
}

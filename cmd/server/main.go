package main

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	handlers "github.com/LingByte/SoulNexus/internal/handler"
	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/schema"
	"github.com/LingByte/SoulNexus/internal/task"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/health"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/backup"
	utilscaptcha "github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/LingByte/SoulNexus/pkg/utils/search"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LingEchoService struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func NewLingEchoService(db *gorm.DB) *LingEchoService {
	return &LingEchoService{
		db:       db,
		handlers: handlers.NewHandlers(db),
	}
}

func (app *LingEchoService) RegisterRoutes(r *gin.Engine) {
	app.handlers.Register(r)
}

func main() {
	// 1. Parse Command Line Parameters
	// Deprecated: parsed for backward compatibility; bootstrap always runs GORM AutoMigrate when connecting.
	init := flag.Bool("init", false, "deprecated: ignored; schema migration always runs at startup")
	seed := flag.Bool("seed", false, "seed database")
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")

	// 2. Set Environment Variables
	if *mode != "" {
		os.Setenv("APP_ENV", *mode)
	}

	if s := strings.TrimSpace(os.Getenv("SOULNEXUS_SERVICE")); s != "" {
		health.SetServiceName(s)
	} else {
		health.SetServiceName("api")
	}

	// 3. Load Global Configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}

	// 4. Load Log Configuration
	err := logger.Init(&config.GlobalConfig.Log, config.GlobalConfig.Server.Mode)
	if err != nil {
		panic(err)
	}

	// 5. Print Banner
	if err := bootstrap.PrintBannerFromFile("system-banner.txt", config.GlobalConfig.Server.Name); err != nil {
		logger.Error(fmt.Sprintf("unload banner: %v", err))
	}

	// 6. Print Configuration
	bootstrap.LogConfigInfo()

	// 7. Load Data Source
	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath:   *initSQL,
		AutoMigrate:   *init,
		SeedNonProd:   *seed,
		MigrateModels: schema.ServerEntities,
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
	flag.Parse()

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
	utilscaptcha.InitGlobalManager(nil) // Use memory storage, can be replaced with Redis storage

	// Initialize global login security manager
	utils.InitGlobalLoginSecurityManager(logger.Lg)

	// Initialize global intelligent risk control manager
	utils.InitGlobalIntelligentRiskControl(logger.Lg)

	if err := bootstrap.InitializeKeyManager(); err != nil {
		logger.Error("JWT key manager initialization failed", zap.Error(err))
		return
	}

	//// 11. New App
	app := NewLingEchoService(db)

	// 12. Initialize Global Middleware Manager
	middleware.InitGlobalMiddlewareManager(config.GlobalConfig.Middleware)
	logger.Info("Global middleware manager initialized with config",
		zap.Bool("rateLimit", config.GlobalConfig.Middleware.EnableRateLimit),
		zap.Bool("timeout", config.GlobalConfig.Middleware.EnableTimeout),
		zap.Bool("circuitBreaker", config.GlobalConfig.Middleware.EnableCircuitBreaker),
		zap.Bool("operationLog", config.GlobalConfig.Middleware.EnableOperationLog))

	// 14. Initialize Neo4j Graph Database (if enabled)
	if config.GlobalConfig.Services.GraphMemory.Neo4j.Enabled {
		graphStore, err := graph.NewNeo4jStore(
			config.GlobalConfig.Services.GraphMemory.Neo4j.URI,
			config.GlobalConfig.Services.GraphMemory.Neo4j.Username,
			config.GlobalConfig.Services.GraphMemory.Neo4j.Password,
			config.GlobalConfig.Services.GraphMemory.Neo4j.Database,
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
	r.Use(middleware.SecureResponseHeaders())
	templatesFS := LingEcho.NewCombineEmbedFS(
		LingEcho.HintAssetsRoot("templates"),
		LingEcho.EmbedFS{"templates", LingEcho.EmbedTemplates},
	)
	r.HTMLRender = LingEcho.NewCombineTemplates(templatesFS)
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.MaxMultipartMemory = 32 << 20 // 32 MB

	// Cookie Register
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

	task.StartAccountDeletionScheduler(db)

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

	shutdownAll := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown", zap.Error(err))
		}
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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	logger.Info("shutdown signal received")
	shutdownAll()
}

package main

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

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

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	handlers "github.com/LingByte/SoulNexus/internal/handler"
	"github.com/LingByte/SoulNexus/internal/schema"
	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/sipserver"
	"github.com/LingByte/SoulNexus/internal/task"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/health"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/backup"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LingEchoSIPService struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func NewLingEchoSIPService(db *gorm.DB) *LingEchoSIPService {
	return &LingEchoSIPService{
		db:       db,
		handlers: handlers.NewSIPServiceHandlers(db),
	}
}

func (app *LingEchoSIPService) RegisterRoutes(r *gin.Engine) {
	app.handlers.RegisterSIPServiceRoutes(r)
}

func main() {
	// 1. Parse Command Line Parameters
	init := flag.Bool("init", false, "deprecated: ignored; schema migration always runs at startup")
	seed := flag.Bool("seed", false, "seed database")
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")
	sipHost := flag.String("sip-host", "0.0.0.0", "embedded SIP UDP listen host")
	sipPort := flag.Int("sip-port", 5060, "embedded SIP UDP listen port")
	sipLocalIP := flag.String("sip-local-ip", "127.0.0.1", "SDP c= line IP (RTP reachable from SIP peers)")
	flag.Parse()

	// 2. Set Environment Variables
	if *mode != "" {
		os.Setenv("MODE", *mode)
	}

	if s := strings.TrimSpace(os.Getenv("SOULNEXUS_SERVICE")); s != "" {
		health.SetServiceName(s)
	} else {
		health.SetServiceName("sip")
	}

	// 3. Load Global Configuration
	if err := config.Load(); err != nil {
		panic("config load failed: " + err.Error())
	}

	// 4. Print Banner
	if err := bootstrap.PrintBannerFromFile("system-banner.txt", config.GlobalConfig.Server.Name); err != nil {
		logger.Error(fmt.Sprintf("unload banner: %v", err))
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
		addr = ":8082"
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
	//// 11. New App
	app := NewLingEchoSIPService(db)
	sipUserCleaner := task.NewSIPUserOnlineCleaner(db, time.Duration(utils.GetIntEnv("SIP_USER_ONLINE_SWEEP_SECONDS"))*time.Second)
	sipUserCleaner.Start()
	if config.GlobalConfig.Features.BackupEnabled {
		backup.StartBackupScheduler(db)
	}

	// 15. Initialize Gin Routing
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()        // Use gin.New() instead of gin.Default() to avoid automatic redirects
	r.Use(gin.Recovery()) // Manually add Recovery middleware
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.MaxMultipartMemory = 32 << 20 // 32 MB

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

	// 18. Register Routes
	app.RegisterRoutes(r)
	// 19. Initialize System Listener
	listeners.InitSystemListeners()
	var sipEmbedded *sipserver.Embedded
	se, err := sipserver.Start(sipserver.Config{
		Host:    *sipHost,
		Port:    *sipPort,
		LocalIP: *sipLocalIP,
		DB:      db,
	})
	if err != nil {
		logger.Fatal("embedded SIP stack failed to start", zap.Error(err))
	}
	sipEmbedded = se
	app.handlers.SetCampaignService(sipEmbedded.CampaignService())
	logger.Info("embedded SIP stack started",
		zap.String("sip_host", *sipHost),
		zap.Int("sip_port", *sipPort),
		zap.String("sip_local_ip", *sipLocalIP))
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
		sipUserCleaner.Stop()
		if sipEmbedded != nil {
			sipEmbedded.Shutdown(ctx)
		}
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

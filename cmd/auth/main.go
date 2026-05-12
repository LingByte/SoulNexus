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

	LingEcho "github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	handlers "github.com/LingByte/SoulNexus/internal/handler"
	"github.com/LingByte/SoulNexus/internal/listeners"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/schema"
	"github.com/LingByte/SoulNexus/internal/task"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
	utilscaptcha "github.com/LingByte/SoulNexus/pkg/utils/captcha"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LingEchoAuthService struct {
	db       *gorm.DB
	handlers *handlers.Handlers
}

func NewLingEchoAuthService(db *gorm.DB) *LingEchoAuthService {
	return &LingEchoAuthService{
		db:       db,
		handlers: handlers.NewUserServiceHandlers(db),
	}
}

func (app *LingEchoAuthService) registerRoutes(r *gin.Engine) {
	app.handlers.RegisterUserServiceRoutes(r)
}

func main() {
	seed := flag.Bool("seed", false, "seed database")
	// 历史 -init 参数兼容保留：当前总是触发 GORM AutoMigrate。
	_ = flag.Bool("init", false, "deprecated: ignored; schema migration always runs at startup")
	mode := flag.String("mode", "", "running environment (development, test, production)")
	initSQL := flag.String("init-sql", "", "path to database init .sql script (optional)")
	addrFlag := flag.String("addr", "", "HTTP listen address (overrides USER_SERVICE_HTTP_ADDR; default :7074)")
	flag.Parse()

	if *mode != "" {
		os.Setenv("APP_ENV", *mode)
	}

	if err := config.LoadAuth(); err != nil {
		panic("config load failed: " + err.Error())
	}
	if config.AuthGlobalConfig == nil {
		panic("config: AuthGlobalConfig is nil after LoadAuth")
	}
	a := config.AuthGlobalConfig

	if err := logger.Init(&a.Log, a.Server.Mode); err != nil {
		panic(err)
	}

	if err := bootstrap.PrintBannerFromFile("user-banner.txt", "SOULNEXUS USER"); err != nil {
		log.Fatalf("unload banner: %v", err)
	}

	bootstrap.LogConfigInfo()

	db, err := bootstrap.SetupDatabase(os.Stdout, &bootstrap.Options{
		InitSQLPath:   *initSQL,
		AutoMigrate:   true,
		SeedNonProd:   *seed,
		MigrateModels: schema.AuthEntities,
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

	logger.Info("user-service listen", zap.String("addr", addr), zap.Bool("auto_migrate", true))

	if err := cache.InitGlobalCache(a.Cache); err != nil {
		logger.Error("failed to initialize cache", zap.Error(err))
		logger.Info("falling back to default local cache")
	}
	utils.InitGlobalCache(1024, 5*time.Minute)
	utils.InitGlobalRegistrationGuard(logger.Lg)
	utils.InitGlobalDistributedLock()
	utilscaptcha.InitGlobalManager(nil)
	utils.InitGlobalLoginSecurityManager(logger.Lg)
	utils.InitGlobalIntelligentRiskControl(logger.Lg)

	// Initialize KeyManager
	if err := bootstrap.InitializeKeyManager(); err != nil {
		logger.Error("key manager initialization failed", zap.Error(err))
		return
	}

	app := NewLingEchoAuthService(db)

	middleware.InitGlobalMiddlewareManager(a.Middleware)

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

	secret := utils.GetEnv(constants.ENV_SESSION_SECRET)
	if secret == "" {
		secret = a.Auth.SessionSecret
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
	apiPrefix := a.Server.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api"
	}
	r.StaticFS(apiPrefix+"/static", http.FS(staticAssets))
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	app.registerRoutes(r)

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

	if a.Server.SSLEnabled && listeners.IsSSLEnabled() {
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

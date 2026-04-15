package main

import (
	"log"

	"github.com/LingByte/SoulNexus/sso-service/internal/bootstrap"
	"github.com/LingByte/SoulNexus/sso-service/internal/config"
	"github.com/LingByte/SoulNexus/sso-service/internal/handler"
	"github.com/LingByte/SoulNexus/sso-service/internal/security"
	"github.com/LingByte/SoulNexus/sso-service/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := storage.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to open sso database: %v", err)
	}

	if err := bootstrap.SeedDefaults(db); err != nil {
		log.Fatalf("failed to seed defaults: %v", err)
	}

	signingKey, err := security.EnsureSigningKey(db)
	if err != nil {
		log.Fatalf("failed to ensure signing key: %v", err)
	}

	oidcHandler, err := handler.NewOIDCHandler(cfg, db, signingKey)
	if err != nil {
		log.Fatalf("failed to initialize oidc handler: %v", err)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	handler.RegisterRoutes(r, oidcHandler)

	log.Printf("sso-service listening on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start sso-service: %v", err)
	}
}

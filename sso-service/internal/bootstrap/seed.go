package bootstrap

import (
	"errors"

	"github.com/LingByte/SoulNexus/sso-service/internal/models"
	"github.com/LingByte/SoulNexus/sso-service/internal/security"
	"gorm.io/gorm"
)

func SeedDefaults(db *gorm.DB) error {
	if err := seedDefaultClient(db); err != nil {
		return err
	}
	if err := seedDemoUser(db); err != nil {
		return err
	}
	return nil
}

func seedDefaultClient(db *gorm.DB) error {
	var client models.OAuthClient
	err := db.First(&client, "id = ?", "portal-web").Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(&models.OAuthClient{
		ID:             "portal-web",
		Name:           "Portal Web",
		Secret:         "portal-web-secret",
		RedirectURI:    "http://localhost:5173/callback",
		IsConfidential: true,
	}).Error
}

func seedDemoUser(db *gorm.DB) error {
	var user models.User
	err := db.First(&user, "id = ?", "user-demo-1").Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	passwordHash, err := security.HashPassword("demo123456")
	if err != nil {
		return err
	}
	return db.Create(&models.User{
		ID:           "user-demo-1",
		Email:        "demo@soulnexus.local",
		Name:         "SoulNexus Demo",
		PasswordHash: passwordHash,
		Status:       "active",
	}).Error
}

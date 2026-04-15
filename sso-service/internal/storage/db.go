package storage

import (
	"github.com/LingByte/SoulNexus/sso-service/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(databasePath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.OAuthClient{},
		&models.AuthorizationCode{},
		&models.RefreshToken{},
		&models.RevokedToken{},
		&models.SigningKey{},
		&models.UserSession{},
		&models.AuditLog{},
	); err != nil {
		return nil, err
	}

	return db, nil
}

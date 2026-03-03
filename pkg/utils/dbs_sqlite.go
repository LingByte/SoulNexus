//go:build !mysql && !pg

package utils

import (
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func createDatabaseInstance(cfg *gorm.Config, driver, dsn string) (*gorm.DB, error) {
	switch driver {
	case "mysql":
		db, err := gorm.Open(mysql.Open(dsn), cfg)
		if err != nil {
			return nil, err
		}

		// Set MySQL charset and collation to resolve utf8mb4_0900_ai_ci and utf8mb3_general_ci mismatch issue
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}

		// Execute SET NAMES to ensure utf8mb4 charset is used
		_, err = sqlDB.Exec("SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci")
		if err != nil {
			// If execution fails, log error but don't block connection
			// Because some MySQL versions may not support this syntax
			// Try using more compatible approach
			_, _ = sqlDB.Exec("SET NAMES utf8mb4")
		}

		return db, nil
	case "pg":
		return gorm.Open(postgres.Open(dsn), cfg)
	}
	if dsn == "" {
		dsn = "file::memory:"
	}
	return gorm.Open(sqlite.Open(dsn), cfg)
}

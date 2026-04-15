package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDatabase(logWrite io.Writer, driver, dsn string) (*gorm.DB, error) {
	if driver == "" {
		driver = GetEnv(constants.ENV_DB_DRIVER)
	}
	if dsn == "" {
		dsn = GetEnv(constants.ENV_DSN)
	}

	var newLogger logger.Interface
	if logWrite == nil {
		logWrite = os.Stdout
	}

	newLogger = logger.New(
		log.New(logWrite, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)

	cfg := &gorm.Config{
		Logger:                                   newLogger,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: true, // Disable foreign key constraint checking during migration
	}

	// Create database connection
	db, err := createDatabaseInstance(cfg, driver, dsn)
	if err != nil {
		return nil, err
	}

	// Configure database connection pool
	ConfigureConnectionPool(db)

	return db, nil
}

// ConfigureConnectionPool configure database connection pool
func ConfigureConnectionPool(db *gorm.DB) {
	// Get the underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get database instance: %v", err)
		return
	}

	// Set maximum idle connections
	sqlDB.SetMaxIdleConns(10)

	// Set maximum open connections
	sqlDB.SetMaxOpenConns(100)

	// Set connection maximum lifetime
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Set connection maximum idle time
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)
}

func MakeMigrates(db *gorm.DB, insts []any) error {
	for _, v := range insts {
		if err := db.AutoMigrate(v); err != nil {
			return err
		}
	}
	return nil
}

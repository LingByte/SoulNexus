//go:build pg

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package utils

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func createDatabaseInstance(cfg *gorm.Config, driver, dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), cfg)
}

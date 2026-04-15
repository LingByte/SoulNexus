package usersvc

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package usersvc provides a narrow read-side boundary for "other domains" that need user rows.
// Current implementation is intentionally local DB only.

import (
	"context"

	"github.com/LingByte/SoulNexus/internal/models"
	"gorm.io/gorm"
)

// Reader is read-only user lookup for cross-domain use (billing, groups, assistants, etc.).
type Reader interface {
	UserByUID(ctx context.Context, uid uint) (*models.User, error)
}

// DBReader implements Reader using the application database (monolith or shared-schema microservices).
type DBReader struct {
	DB *gorm.DB
}

// UserByUID loads a user by primary key.
func (d *DBReader) UserByUID(ctx context.Context, uid uint) (*models.User, error) {
	_ = ctx
	return models.GetUserByUID(d.DB, uid)
}

// NewDBReader returns a Reader backed by GORM.
func NewDBReader(db *gorm.DB) Reader {
	return &DBReader{DB: db}
}

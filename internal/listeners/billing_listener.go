package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InitBillingListenerWithDB Initialize billing listener (with database connection)
// Note: LLM usage recording is handled in llm_listener.go
// Call usage recording is directly called in models.RecordCallUsage when call ends
func InitBillingListenerWithDB(db *gorm.DB) {
	// Currently billing records are integrated into other listeners and business logic
	// Retain initialization function here for future expansion of other listening events
	logger.Info("Billing listener initialized", zap.Bool("db_available", db != nil))
}

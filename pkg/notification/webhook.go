// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package notification provides outbound mail/SMS/webhook and in-app inbox delivery.
//
// Subpackages:
//   - mail / sms / webhook — outbound channel delivery
//   - inbox                — inner notification
package notification

import (
	"github.com/LingByte/SoulNexus/pkg/notification/webhook"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DispatchWebhook emits one call lifecycle event to all matching tenant webhooks (async).
func DispatchWebhook(db *gorm.DB, lg *zap.Logger, tenantID uint, event, callID, from, to, direction string, extra map[string]any) {
	webhook.Dispatch(db, lg, tenantID, event, callID, from, to, direction, extra)
}

// ValidWebhookEvent reports whether event is in the supported webhook catalog.
func ValidWebhookEvent(event string) bool {
	return webhook.ValidEvent(event)
}

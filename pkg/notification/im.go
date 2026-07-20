// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package notification

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/notification/im"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DispatchIM sends a markdown/text notice to all enabled tenant IM channels
// (WeCom / Feishu). Best-effort; returns how many channels accepted the message.
func DispatchIM(ctx context.Context, db *gorm.DB, lg *zap.Logger, tenantID uint, title, content string) (int, error) {
	return im.Dispatch(ctx, db, lg, tenantID, im.Message{Title: title, Content: content})
}

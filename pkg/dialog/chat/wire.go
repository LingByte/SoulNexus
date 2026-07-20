// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package chat

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WireSessionBridge installs session.TextChatBridge so internal callers
// (e.g. internal text-chat bridge) persist turns via this Service.
// Public text chat uses /lingecho/dialog/v1 directly; voice-session no longer
// exposes transport=text.
func WireSessionBridge(db *gorm.DB) {
	if db == nil {
		return
	}
	lg := logger.Lg
	if lg == nil {
		lg = zap.NewNop()
	}
	svc := New(db, lg.Named("dialog-chat"))
	session.SetTextChatBridge(session.TextChatBridge{
		EnsureConversation: func(ctx context.Context, tenantID, assistantID uint, channel, externalUserID string) (uint, string, error) {
			conv, err := svc.EnsureConversation(ctx, EnsureParams{
				TenantID:       tenantID,
				AssistantID:    assistantID,
				Channel:        channel,
				ExternalUserID: externalUserID,
			})
			if err != nil {
				return 0, "", err
			}
			welcome := ""
			if env, ok, rErr := tenantcfg.Resolve(ctx, tenantID, conv.CallKey()); rErr == nil && ok {
				welcome = tenantcfg.ResolvedAssistantWelcome(env)
			}
			return conv.ID, welcome, nil
		},
		HandleUserText: func(ctx context.Context, tenantID, conversationID uint, text string) (string, int64, error) {
			res, err := svc.HandleUserText(ctx, tenantID, conversationID, text)
			if err != nil {
				return "", 0, err
			}
			return res.Reply, res.LatencyMs, nil
		},
		EndConversation: func(ctx context.Context, tenantID, conversationID uint) error {
			return svc.EndConversation(ctx, tenantID, conversationID)
		},
	})
}

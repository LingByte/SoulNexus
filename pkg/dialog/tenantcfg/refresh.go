// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import (
	"context"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
)

// RefreshVoiceEnvForCall reloads VoiceEnv when call-scoped assistant was set
// after an earlier tenant-only resolve (attach fast-path prep race).
func RefreshVoiceEnvForCall(ctx context.Context, tenantID uint, callID string, env VoiceEnv) VoiceEnv {
	callID = strings.TrimSpace(callID)
	if tenantID == 0 || callID == "" {
		return env
	}
	aid := callbinding.GetAssistantID(callID)
	if aid == 0 {
		return env
	}
	// Prep already resolved for this assistant — avoid a second Resolve on ACK
	// (can cost seconds and delays Attach / early-welcome handover).
	if env.AssistantID == aid {
		return env
	}
	fresh, ok, err := Resolve(ctx, tenantID, callID)
	if err != nil || !ok {
		return env
	}
	return fresh
}

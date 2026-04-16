package conversation

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"go.uber.org/zap"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

func registerSIPTransferTool(provider llm.LLMProvider, callID string, lg *zap.Logger) {
	if strings.TrimSpace(callID) == "" {
		return
	}
	// Current pkg/llm handlers no longer expose provider-specific function tool registration.
	// Keep this hook as a no-op so call sites remain stable and fallback transfer logic still works.
	if lg != nil {
		lg.Debug("sip voice: transfer tool registration skipped (unsupported by current llm handler)",
			zap.String("call_id", callID),
		)
	}
}

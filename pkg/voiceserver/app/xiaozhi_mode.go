// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

const (
	// XiaozhiModePipeline is the classic ASR → dialog WS (LLM) → TTS path.
	XiaozhiModePipeline = "pipeline"
	// XiaozhiModeRealtime uses pkg/realtime (e.g. Qwen-Omni) end-to-end.
	XiaozhiModeRealtime = "realtime"
)

// XiaozhiMode reads XIAOZHIMODE from the environment (via .env-voice).
func XiaozhiMode() string {
	return ModeFromConfig(utils.GetEnv("XIAOZHIMODE"))
}

// ModeFromConfig maps config/env aliases to a canonical mode slug.
func ModeFromConfig(m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	switch m {
	case XiaozhiModeRealtime, "omni", "multimodal":
		return XiaozhiModeRealtime
	case "", XiaozhiModePipeline, "classic", "legacy":
		return XiaozhiModePipeline
	default:
		return XiaozhiModePipeline
	}
}

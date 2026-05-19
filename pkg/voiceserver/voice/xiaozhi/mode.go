// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

import "strings"

const (
	// ModePipeline: ASR + dialog-plane WebSocket (LLM) + TTS.
	ModePipeline = "pipeline"
	// ModeRealtime: pkg/realtime multimodal agent (e.g. Qwen-Omni).
	ModeRealtime = "realtime"
)

func normalizeMode(m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	switch m {
	case ModeRealtime, "omni", "multimodal":
		return ModeRealtime
	default:
		return ModePipeline
	}
}

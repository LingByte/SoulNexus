// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package meter

// Metering helpers will gradually move here from internal/models.
// For now this package exists so CI and imports resolve.
type Kind string

const (
	CallMinutes Kind = "call_minutes"
	LLMTokens   Kind = "llm_tokens"
	StorageGB   Kind = "storage_gb"
)

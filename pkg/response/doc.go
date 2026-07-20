// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package response provides unified HTTP JSON envelopes and application errors for SoulNexus APIs.
//
//	{"code": 200, "msg": "...", "data": ...}
//
// Error responses use the appropriate HTTP status with:
//
//	{"code": <business code>, "msg": "...", "error": "<CODE>", "data": ...}
//
// Service layers return *AppError; HTTP handlers call Render or WriteError once.
package response

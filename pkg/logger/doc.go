// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package logger provides structured logging for SoulNexus.
//
// Initialization:
//
//	logger.Init(&cfg.Log, cfg.Server.Mode) must run after config.Load so
//	SYSTEM_TIMEZONE (via pkg/utils/timeutil) is applied before timestamps
//	and daily log rotation boundaries are computed.
//
// Features:
//   - JSON file output with lumberjack rotation (optional daily filenames)
//   - Colored console output in local/dev/prod modes (stdout info, stderr error)
//   - Sensitive field redaction (LOG_SENSITIVE_FIELDS)
//   - Request ID prefixing and Gin helpers (FromGin, GinZapFields)
//   - Context-aware logging (InfoCtx, ErrorCtx, …)
//   - SafeGo for panic-safe background goroutines
//   - Log retention purge (PurgeExpiredLogFiles)
package logger

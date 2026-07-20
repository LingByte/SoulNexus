// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package cascaded implements the native engine.Engine for cascaded
// (ASR + LLM + TTS) dialog mode via pipeline.Stage composition.
//
// Production attach uses StreamingPort + session.AttachEngine(ModeCascaded)
// via the voice session wiring. Stages wire real ASR/LLM/TTS providers
// from tenant configuration.
//
// Boundary discipline:
//
//   - Imports only pkg/dialog/{engine,pipeline,tenantcfg}. Never
//     imports telephony packages (would re-introduce the cycle this whole
//     refactor was designed to break).
//   - Logging is via engine.Logger. No zap dependency in this
//     package.
package cascaded

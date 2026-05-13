// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package app holds the cmd/voice runtime business logic shared between
// the standalone voice server (cmd/voice) and the HTTP transport handlers
// (internal/handler/voice). It owns:
//
//   - ASR/TTS factories for the WebRTC and xiaozhi (WS-Hardware) adapters
//   - SIPCall lifecycle persistence (callPersister + persisterAdapter)
//   - Recording factory wrapping pkg/voiceserver/voice/recorder
//   - Dialog-plane auth merging and reconnect configuration
//   - Process-wide runtime flags (barge-in / denoise / TTS prewarm / hold prompts)
//   - SFU manager lifecycle, voiceserver.db helpers, and small env helpers
//
// Everything here is process-singleton in spirit: cmd/voice calls the
// SetX setters once during startup (after config.LoadVoice) and the
// transports read the same values via the constructors NewBargeInDetector,
// NewDenoiserOrNil, ApplyDialogReconnect, MakePersisterFactory,
// MakeRecorderFactory, NewXiaozhiFactory, NewWebRTCFactory.
package app

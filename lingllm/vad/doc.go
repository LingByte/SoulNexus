// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package vad provides in-process voice activity helpers used by voice barge-in.
//
// Production paths:
//   - EnergyDetector (this package) — RMS barge-in while TTS plays
//   - vad/local — WebRTC / Silero / Levad (TinySilero, TinyTen) via CGO tags
//
// The former Python HTTP VAD microservice (repo-root vad/) and the HTTP/WebSocket
// remote clients were removed after in-process VAD covered the same needs.
package vad

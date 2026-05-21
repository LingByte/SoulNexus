// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package voice exposes HTTP/WebSocket route handlers for the standalone
// voice server (cmd/voice). It mirrors the urls.go pattern used by
// internal/handler (the main API server) and internal/handler for the auth
// service: a single Handlers struct is registered against a *gin.Engine
// (or any gin.IRoutes), and each transport — WebSocket-Hardware (xiaozhi
// + soulnexus-hw), WebSocket-Web (SFU), WebRTC, and the media file
// server — owns its own file in this package.
//
// The handlers themselves are thin: business logic (factories, persister,
// recorder, hold prompts, dialog reconnect, SFU manager lifecycle) lives
// in pkg/voiceserver/app and is consumed via that package's exported
// API. cmd/voice owns the SIP UAS/UAC plus this package's Handlers
// instance.
package voice

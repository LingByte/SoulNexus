package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// voiceUpgrader is used for OpenAPI streaming ASR/TTS WebSocket endpoints.
var voiceUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

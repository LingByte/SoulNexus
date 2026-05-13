// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// MergeDialogAuthQuery merges DIALOG_API_KEY, DIALOG_API_SECRET, and
// DIALOG_AGENT_ID into the VOICE_DIALOG_WS URL query string so operators
// can keep secrets out of argv and load them from the environment (or
// .env) instead. Non-empty env values overwrite existing query keys
// with the same name so a single base URL can stay stable while
// credentials rotate.
func MergeDialogAuthQuery(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse dialog-ws: %w", err)
	}
	q := u.Query()
	mergeQueryFromEnv(q, "apiKey", "DIALOG_API_KEY")
	mergeQueryFromEnv(q, "apiSecret", "DIALOG_API_SECRET")
	mergeQueryFromEnv(q, "agentId", "DIALOG_AGENT_ID")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func mergeQueryFromEnv(q url.Values, key, envName string) {
	if v := strings.TrimSpace(utils.GetEnv(envName)); v != "" {
		q.Set(key, v)
	}
}

// WarnDialogAuthIfNeeded logs when the merged dialog URL still lacks
// apiKey, apiSecret, or agentId. SoulNexus GET /ws/call rejects the
// WebSocket upgrade with HTTP 400 in that case (clients often report
// "bad handshake").
func WarnDialogAuthIfNeeded(dialogURL string) {
	dialogURL = strings.TrimSpace(dialogURL)
	if dialogURL == "" {
		return
	}
	u, err := url.Parse(dialogURL)
	if err != nil {
		return
	}
	q := u.Query()
	if q.Get("apiKey") != "" && q.Get("apiSecret") != "" && q.Get("agentId") != "" {
		return
	}
	log.Printf("[warn] dialog-plane URL still missing apiKey, apiSecret, or agentId after merging DIALOG_* env. " +
		"If the peer is SoulNexus /ws/call, the upgrade returns HTTP 400 (websocket: bad handshake). " +
		"WebRTC: POST these fields in /webrtc/v1/offer JSON from the browser; SIP/xiaozhi: set DIALOG_* env or add query params to VOICE_DIALOG_WS.")
}

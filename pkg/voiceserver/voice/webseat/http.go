// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webseat

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// HTTP layout:
//
//   GET  /webseat/awaiting           — JSON list of awaiting Call-IDs
//   POST /webseat/join               — body {call_id, sdp} → {sdp}
//   POST /webseat/hangup             — body {call_id, reason}
//
// All endpoints honour Hub.Token via ?token=... (or X-Webseat-Token
// header). Empty configured token disables auth — only acceptable for
// dev. The handler set is intentionally tiny so cmd code can mount
// them on any net/http mux without pulling in a router framework.

// Handlers wires the hub's HTTP endpoints onto the supplied mux. The
// pathPrefix is applied to all three routes — pass "/webseat/v1"
// (or empty for "/awaiting" etc) depending on your URL convention.
func (h *Hub) Handlers(pathPrefix string) http.Handler {
	mux := http.NewServeMux()
	prefix := strings.TrimRight(pathPrefix, "/")

	mux.HandleFunc(prefix+"/awaiting", h.handleAwaiting)
	mux.HandleFunc(prefix+"/join", h.handleJoin)
	mux.HandleFunc(prefix+"/hangup", h.handleHangup)
	return mux
}

func (h *Hub) authOK(r *http.Request) bool {
	if h.cfg.Token == "" {
		return true
	}
	got := strings.TrimSpace(r.URL.Query().Get("token"))
	if got == "" {
		got = strings.TrimSpace(r.Header.Get("X-Webseat-Token"))
	}
	return h.tokenOK(got)
}

// handleAwaiting returns the current awaiting Call-IDs as JSON.
// Cheap snapshot; clients should poll or (better) subscribe to the
// websocket presence feed.
func (h *Hub) handleAwaiting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	body, _ := json.Marshal(map[string]any{
		"awaiting": h.Awaiting(),
	})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

// joinRequest is the browser-side input to POST /join. The browser
// has already done getUserMedia + RTCPeerConnection.setLocalDescription;
// it forwards the offer SDP here, gets an answer SDP back, and feeds
// that into setRemoteDescription. The hub then begins audio splicing.
type joinRequest struct {
	CallID string `json:"call_id"`
	SDP    string `json:"sdp"`
}

type joinResponse struct {
	SDP string `json:"sdp"`
}

func (h *Hub) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req joinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CallID) == "" || strings.TrimSpace(req.SDP) == "" {
		http.Error(w, "missing call_id or sdp", http.StatusBadRequest)
		return
	}
	answer, err := h.Pickup(r.Context(), req.CallID, req.SDP)
	if err != nil {
		// Map well-known errors to HTTP status codes the browser can
		// react to without parsing free-form text.
		switch {
		case errors.Is(err, ErrNotAwaiting):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	body, _ := json.Marshal(joinResponse{SDP: answer})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

type hangupRequest struct {
	CallID string `json:"call_id"`
	Reason string `json:"reason"`
}

func (h *Hub) handleHangup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.authOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var req hangupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.CallID) == "" {
		http.Error(w, "missing call_id", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "agent-hangup"
	}
	if err := h.Hangup(req.CallID, reason); err != nil {
		// Try ReleaseAwaiting — browser might have hit hangup on a
		// call still in awaiting state.
		if errors.Is(err, ErrNotBridged) {
			if err2 := h.ReleaseAwaiting(req.CallID, reason); err2 == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

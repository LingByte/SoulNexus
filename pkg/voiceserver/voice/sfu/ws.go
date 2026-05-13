// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsConn wraps *websocket.Conn with:
//
//   - a write mutex (gorilla requires serialised writes; pion callbacks,
//     signaling handler, and heartbeat all send concurrently)
//   - sendJSON / readMessage helpers specialised to our Envelope shape
//   - a close signal that is idempotent and non-blocking
//
// The read pump lives in handler.go — only one goroutine reads; writes
// may come from anywhere.
type wsConn struct {
	conn   *websocket.Conn
	wmu    sync.Mutex
	closed chan struct{}
	once   sync.Once
}

// newWSUpgrader returns a websocket.Upgrader with an Origin-check
// closure built from Config.AllowedOrigins. An empty list OR "*" allows
// any origin (legacy behaviour). Otherwise the request's Origin header
// must exactly match one entry (case-insensitive on scheme+host+port).
// Browsers always set Origin on WS upgrades; missing-Origin requests
// (non-browser clients) are allowed because they're not subject to
// the same-origin policy that this header exists to enforce.
func newWSUpgrader(allowed []string) *websocket.Upgrader {
	whitelist := normaliseOrigins(allowed)
	return &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			if len(whitelist) == 0 {
				return true
			}
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				// No Origin header: non-browser client (curl, native
				// app, server-to-server). Allow.
				return true
			}
			norm := canonicalOrigin(origin)
			for _, allow := range whitelist {
				if allow == "*" || allow == norm {
					return true
				}
			}
			return false
		},
	}
}

// normaliseOrigins lowercases and trims whitespace so AllowedOrigins
// match the canonicalised Origin header at request time.
func normaliseOrigins(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if s == "*" {
			out = append(out, "*")
			continue
		}
		out = append(out, canonicalOrigin(s))
	}
	return out
}

// canonicalOrigin returns scheme://host[:port] lowercased. Strips
// trailing slashes and any path/query so "https://x/" and "https://x"
// compare equal.
func canonicalOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.ToLower(strings.TrimRight(raw, "/"))
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)
	return scheme + "://" + host
}

// newWSConn upgrades an HTTP request and returns a wrapped wsConn. The
// ping/pong handlers are pre-wired so every pong refreshes the read
// deadline, which is how we detect silently-dead clients.
func newWSConn(w http.ResponseWriter, r *http.Request, heartbeat time.Duration, up *websocket.Upgrader) (*wsConn, error) {
	c, err := up.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	// Pong extends the read deadline. Two missed heartbeats → the read
	// pump's next ReadMessage returns an error and the connection tears
	// down. The initial SetReadDeadline seeds the window for the first
	// message (join); we refresh on every pong below.
	readTimeout := heartbeat*2 + 5*time.Second
	_ = c.SetReadDeadline(time.Now().Add(readTimeout))
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(readTimeout))
	})
	// Soft cap on inbound frame size — SDP offers from chromium top out
	// around 20 KB; 128 KB gives plenty of headroom without letting a
	// malicious client exhaust memory.
	c.SetReadLimit(128 * 1024)
	return &wsConn{conn: c, closed: make(chan struct{})}, nil
}

// sendJSON writes one Envelope{Type, Data=marshal(payload)} on the
// socket. Returns an error when the socket is already closed so the
// caller can give up early.
func (w *wsConn) sendJSON(env Envelope, payload any) error {
	if w.isClosed() {
		return errors.New("sfu: ws closed")
	}
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		env.Data = raw
	}
	buf, err := json.Marshal(env)
	if err != nil {
		return err
	}
	w.wmu.Lock()
	defer w.wmu.Unlock()
	// Write deadline shields the server from a half-open TCP peer that
	// has filled its receive window but never ACKs; 10 s is generous
	// for SDP-sized messages on any real link.
	_ = w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.conn.WriteMessage(websocket.TextMessage, buf)
}

// sendError is syntactic sugar for emitting an Envelope{Type: error}
// with the code/message pair. Never returns an error — emission best-
// effort by design (the peer is usually about to be closed anyway).
func (w *wsConn) sendError(code, message string) {
	_ = w.sendJSON(Envelope{Type: MsgError}, ErrorData{Code: code, Message: message})
}

// sendPing writes a gorilla-level ping frame. Heartbeat loop calls
// this; the pong handler extends the read deadline.
func (w *wsConn) sendPing() error {
	if w.isClosed() {
		return errors.New("sfu: ws closed")
	}
	w.wmu.Lock()
	defer w.wmu.Unlock()
	_ = w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.conn.WriteMessage(websocket.PingMessage, nil)
}

// readEnvelope pulls one text message, decodes the envelope, and
// returns the parsed form. Blocks until a message arrives, an error
// occurs, or the read deadline expires.
func (w *wsConn) readEnvelope() (Envelope, error) {
	var env Envelope
	_, buf, err := w.conn.ReadMessage()
	if err != nil {
		return env, err
	}
	if err := json.Unmarshal(buf, &env); err != nil {
		return env, err
	}
	return env, nil
}

// close writes a normal-closure frame (best-effort) and closes the
// underlying TCP connection. Idempotent.
func (w *wsConn) close() {
	w.once.Do(func() {
		close(w.closed)
		if w.conn != nil {
			w.wmu.Lock()
			_ = w.conn.SetWriteDeadline(time.Now().Add(time.Second))
			_ = w.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			w.wmu.Unlock()
			_ = w.conn.Close()
		}
	})
}

// isClosed is non-blocking.
func (w *wsConn) isClosed() bool {
	select {
	case <-w.closed:
		return true
	default:
		return false
	}
}

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// TestOriginAllowlistAllowsConfigured: configured Origin accepted.
func TestOriginAllowlistAllowsConfigured(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		AllowedOrigins:       []string{"https://app.example.com"},
	}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	h := http.Header{"Origin": []string{"https://app.example.com"}}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	// Confirm we can talk.
	_ = writeEnvelope(c, MsgJoin, JoinData{Room: "r", Name: "n"})
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if env, _ := readEnvelope(c); env.Type != MsgJoined {
		t.Fatalf("expected joined, got %s", env.Type)
	}
}

// TestOriginAllowlistRejectsOthers: foreign Origin rejected with HTTP 403.
func TestOriginAllowlistRejectsOthers(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		AllowedOrigins:       []string{"https://app.example.com"},
	}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	h := http.Header{"Origin": []string{"https://evil.example.com"}}
	_, resp, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err == nil {
		t.Fatal("expected upgrade rejection")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got resp=%v err=%v", resp, err)
	}
}

// TestOriginWildcardAllowsAny: "*" is the legacy permissive setting.
func TestOriginWildcardAllowsAny(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		AllowedOrigins:       []string{"*"},
	}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(),
		http.Header{"Origin": []string{"https://anything.example"}})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
}

// TestOriginEmptyAllowsAll: empty list keeps legacy permissive behaviour.
func TestOriginEmptyAllowsAll(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(),
		http.Header{"Origin": []string{"https://anything.example"}})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
}

// TestOriginCaseInsensitive: scheme/host casing differences are ignored.
func TestOriginCaseInsensitive(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		AllowedOrigins:       []string{"HTTPS://App.Example.com/"},
	}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(),
		http.Header{"Origin": []string{"https://app.example.com"}})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
}

// TestCanonicalOrigin is a pure-function sanity for the helper.
func TestCanonicalOrigin(t *testing.T) {
	cases := map[string]string{
		"https://A.B.com":              "https://a.b.com",
		"HTTPS://a.B.com/":             "https://a.b.com",
		"http://localhost:7080/x?y=1":  "http://localhost:7080",
		"weird-input":                  "weird-input",
	}
	for in, want := range cases {
		if got := canonicalOrigin(in); got != want {
			t.Errorf("canonicalOrigin(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMaxRoomsCap: third room rejected with too_many_rooms.
func TestMaxRoomsCap(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		MaxRooms:             2,
	}, zap.NewNop())
	if r, err := mgr.getOrCreateRoom("a"); err != nil || r == nil {
		t.Fatalf("first: %v", err)
	}
	if r, err := mgr.getOrCreateRoom("b"); err != nil || r == nil {
		t.Fatalf("second: %v", err)
	}
	if _, err := mgr.getOrCreateRoom("c"); err == nil {
		t.Fatal("expected too_many_rooms")
	}
	// Existing room lookup must still succeed even at the cap.
	if r, err := mgr.getOrCreateRoom("a"); err != nil || r == nil {
		t.Fatalf("existing lookup at cap: %v", err)
	}
}

// TestMaxRoomsUnlimited: -1 disables the cap.
func TestMaxRoomsUnlimited(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		MaxRooms:             -1,
	}, zap.NewNop())
	for i := 0; i < 100; i++ {
		if _, err := mgr.getOrCreateRoom(string(rune('a' + i%26))); err != nil {
			t.Fatalf("unlimited cap rejected: %v", err)
		}
	}
}

// TestRoomFullProtocolError: cap-hit join surfaces as MsgError with
// code "too_many_rooms".
func TestRoomFullProtocolError(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		MaxRooms:             1,
	}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"

	c1, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c1.Close()
	_ = writeEnvelope(c1, MsgJoin, JoinData{Room: "r1", Name: "a"})
	_ = c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_ = mustRead(t, c1, MsgJoined, 2*time.Second)

	c2, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c2.Close()
	_ = writeEnvelope(c2, MsgJoin, JoinData{Room: "r2", Name: "b"})
	_ = c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	env, err := readEnvelope(c2)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != MsgError {
		t.Fatalf("expected error, got %s", env.Type)
	}
	var ed ErrorData
	_ = json.Unmarshal(env.Data, &ed)
	if !strings.Contains(ed.Code, "too_many_rooms") {
		t.Errorf("expected too_many_rooms, got %s", ed.Code)
	}
}

// TestManagerCloseIsIdempotent + TestManagerCloseShutsServeWS verify
// the lifecycle additions.
func TestManagerCloseIsIdempotent(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, zap.NewNop())
	mgr.Close()
	mgr.Close() // must not panic
	if !mgr.closed.Load() {
		t.Error("closed flag not set")
	}
}

func TestManagerCloseShutsServeWS(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	mgr.Close()
	// Plain HTTP GET to the WS endpoint: should now be 503.
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
}

// TestWebhookEmitterShutdownExitsLoop launches a webhook server, sends
// some events, closes the emitter, and confirms the loop goroutine
// exits. Uses the emitter's loopDone channel rather than a global
// goroutine count so the test isn't racy against other concurrent
// tests in the package.
func TestWebhookEmitterShutdownExitsLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	cfg := (&Config{WebhookURL: srv.URL, WebhookTimeout: time.Second}).Normalise()

	em := newWebhookEmitter(cfg, zap.NewNop())
	em.emit(Event{Type: EventRoomStarted})
	em.emit(Event{Type: EventRoomEnded})
	time.Sleep(100 * time.Millisecond)
	em.shutdown()
	em.shutdown() // idempotent

	select {
	case <-em.loopDone:
		// loop exited cleanly
	case <-time.After(3 * time.Second):
		t.Fatal("webhook loop did not exit within 3s of shutdown()")
	}
	// Emits after shutdown must be no-ops, not panics.
	em.emit(Event{Type: EventRoomStarted})
}

// TestWebhookEmitterShutdownWithoutURL verifies shutdown() is safe on
// a disabled emitter (no loop, no loopDone channel).
func TestWebhookEmitterShutdownWithoutURL(t *testing.T) {
	em := newWebhookEmitter(&Config{}, zap.NewNop())
	em.shutdown()
	em.shutdown()
}

// TestICEServersForClientCredentialString / NonString cover the new
// safe credential extraction.
func TestICEServersForClientCredentialString(t *testing.T) {
	out := iceServersForClient([]pionwebrtc.ICEServer{
		{URLs: []string{"turn:t"}, Username: "u", Credential: "pw"},
	})
	if len(out) != 1 || out[0].Credential != "pw" {
		t.Errorf("string credential lost: %+v", out)
	}
}

func TestICEServersForClientCredentialNonString(t *testing.T) {
	out := iceServersForClient([]pionwebrtc.ICEServer{
		{URLs: []string{"turn:t"}, Credential: struct{ X int }{X: 1}},
	})
	if out[0].Credential != "" {
		t.Errorf("non-string credential leaked: %q", out[0].Credential)
	}
}

// TestICERestartSlotReleaseWatchdog: drives onICEConnectionStateChange
// with Failed; the watchdog must drain the negotiating slot after 10s
// even if the client never answers. We shrink the constant by patching
// at runtime is not possible, so we instead verify the immediate
// release-on-error path (HandleAnswer-free).
func TestICERestartSlotReleaseOnError(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	p := h.join(t, "icer", "x")

	// Close the PC so CreateOffer with ICERestart will fail.
	_ = p.pc.Close()
	atomic.StoreInt32(new(int32), 0) // silence linter on unused atomic
	p.onICEConnectionStateChange(pionwebrtc.ICEConnectionStateFailed)
	// Give the goroutine a moment.
	time.Sleep(200 * time.Millisecond)
	// Slot must be drained — try to acquire it.
	select {
	case p.negotiating <- struct{}{}:
		// Good, slot was empty (released).
		<-p.negotiating
	default:
		t.Error("negotiating slot not released after restart error")
	}
}

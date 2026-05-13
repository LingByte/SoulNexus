// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webseat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
)

func TestHandlers_AwaitingListing(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())
	_ = hub.RegisterAwaiting("c2", fakeLeg())

	req := httptest.NewRequest(http.MethodGet, "/webseat/v1/awaiting", nil)
	rec := httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Awaiting []string `json:"awaiting"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Awaiting) != 2 {
		t.Errorf("awaiting=%v", got.Awaiting)
	}
}

func TestHandlers_JoinHappyPath(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())

	body := strings.NewReader(`{"call_id":"c1","sdp":"v=0\noffer\n"}`)
	req := httptest.NewRequest(http.MethodPost, "/webseat/v1/join", body)
	rec := httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp joinResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.SDP == "" {
		t.Fatal("empty answer")
	}
}

func TestHandlers_JoinNotAwaitingIs404(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	body := strings.NewReader(`{"call_id":"missing","sdp":"v=0\n"}`)
	req := httptest.NewRequest(http.MethodPost, "/webseat/v1/join", body)
	rec := httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status=%d, want 404", rec.Code)
	}
}

func TestHandlers_HangupOnAwaitingFallsBackToRelease(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())

	body := strings.NewReader(`{"call_id":"c1","reason":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/webseat/v1/hangup", body)
	rec := httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if got := hub.Awaiting(); len(got) != 0 {
		t.Errorf("awaiting after hangup-on-awaiting = %v", got)
	}
}

func TestHandlers_AuthRequired(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, Token: "shh"})
	_ = hub.RegisterAwaiting("c1", fakeLeg())

	// No token → 401
	req := httptest.NewRequest(http.MethodGet, "/webseat/v1/awaiting", nil)
	rec := httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no-token status=%d, want 401", rec.Code)
	}

	// Wrong token via header → 401
	req = httptest.NewRequest(http.MethodGet, "/webseat/v1/awaiting", nil)
	req.Header.Set("X-Webseat-Token", "wrong")
	rec = httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong-token status=%d, want 401", rec.Code)
	}

	// Right token → 200
	req = httptest.NewRequest(http.MethodGet, "/webseat/v1/awaiting?token=shh", nil)
	rec = httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("good-token status=%d, want 200", rec.Code)
	}
}

func TestHandlers_MethodMismatch(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}})
	cases := []struct {
		method, path string
	}{
		{http.MethodPost, "/webseat/v1/awaiting"},
		{http.MethodGet, "/webseat/v1/join"},
		{http.MethodGet, "/webseat/v1/hangup"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		rec := httptest.NewRecorder()
		hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s status=%d, want 405", c.method, c.path, rec.Code)
		}
	}
}

func TestHandlers_BadJSONIs400(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}})
	req := httptest.NewRequest(http.MethodPost, "/webseat/v1/join", strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", rec.Code)
	}
}

// Probe that ctx cancellation propagates into Bridge.Connect.
func TestHandlers_Join_ContextCancels(t *testing.T) {
	br := &slowBridge{}
	hub := New(Config{Bridge: br, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())

	body := strings.NewReader(`{"call_id":"c1","sdp":"x"}`)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/webseat/v1/join", body).WithContext(ctx)
	rec := httptest.NewRecorder()
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	hub.Handlers("/webseat/v1").ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Errorf("expected non-200 on canceled join, got %d", rec.Code)
	}
}

type slowBridge struct{}

func (slowBridge) Connect(ctx context.Context, _ string, _ *session.MediaLeg, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}
func (slowBridge) Disconnect(string) error { return nil }

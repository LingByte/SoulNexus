// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webseat

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
)

// fakeBridge records calls and lets tests inject behaviours.
type fakeBridge struct {
	mu          sync.Mutex
	connectErr  error
	connectCallIDs []string
	disconnectCallIDs []string
}

func (f *fakeBridge) Connect(_ context.Context, callID string, _ *session.MediaLeg, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connectCallIDs = append(f.connectCallIDs, callID)
	if f.connectErr != nil {
		return "", f.connectErr
	}
	return "v=0\r\nfake-answer-sdp\r\n", nil
}

func (f *fakeBridge) Disconnect(callID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disconnectCallIDs = append(f.disconnectCallIDs, callID)
	return nil
}

// fakeLeg is a non-nil pointer; the hub never dereferences it in
// tests, only stores/forwards it to the bridge.
func fakeLeg() *session.MediaLeg { return &session.MediaLeg{} }

func TestRegisterAwaiting_BasicAndDuplicate(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	if err := hub.RegisterAwaiting("c1", fakeLeg()); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := hub.RegisterAwaiting("c1", fakeLeg()); !errors.Is(err, ErrAlreadyAwaiting) {
		t.Errorf("dup register err = %v, want ErrAlreadyAwaiting", err)
	}
	if got := hub.Awaiting(); len(got) != 1 || got[0] != "c1" {
		t.Errorf("Awaiting()=%v", got)
	}
}

func TestRegisterAwaiting_RejectsEmpty(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	if err := hub.RegisterAwaiting("", fakeLeg()); err == nil {
		t.Fatal("empty callID should error")
	}
	if err := hub.RegisterAwaiting("x", nil); err == nil {
		t.Fatal("nil leg should error")
	}
}

func TestPickup_HappyPath(t *testing.T) {
	br := &fakeBridge{}
	hub := New(Config{Bridge: br, JoinTimeout: time.Hour})
	if err := hub.RegisterAwaiting("c1", fakeLeg()); err != nil {
		t.Fatal(err)
	}
	answer, err := hub.Pickup(context.Background(), "c1", "v=0\r\noffer\r\n")
	if err != nil {
		t.Fatalf("pickup: %v", err)
	}
	if answer == "" {
		t.Fatal("empty answer")
	}
	if got := hub.Awaiting(); len(got) != 0 {
		t.Errorf("after pickup Awaiting()=%v", got)
	}
	br.mu.Lock()
	defer br.mu.Unlock()
	if len(br.connectCallIDs) != 1 || br.connectCallIDs[0] != "c1" {
		t.Errorf("bridge.Connect calls=%v", br.connectCallIDs)
	}
}

func TestPickup_NotAwaiting(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	_, err := hub.Pickup(context.Background(), "missing", "x")
	if !errors.Is(err, ErrNotAwaiting) {
		t.Errorf("err=%v, want ErrNotAwaiting", err)
	}
}

func TestPickup_BridgeFailureRestoresAwaiting(t *testing.T) {
	br := &fakeBridge{connectErr: errors.New("dial")}
	hub := New(Config{Bridge: br, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())
	if _, err := hub.Pickup(context.Background(), "c1", "x"); err == nil {
		t.Fatal("expected error from failing bridge")
	}
	// Awaiting must be restored so a different browser can retry.
	if got := hub.Awaiting(); len(got) != 1 || got[0] != "c1" {
		t.Errorf("after failed pickup Awaiting()=%v", got)
	}
}

func TestHangup_BridgedCall(t *testing.T) {
	br := &fakeBridge{}
	hub := New(Config{Bridge: br, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())
	_, _ = hub.Pickup(context.Background(), "c1", "x")
	if err := hub.Hangup("c1", "agent-bye"); err != nil {
		t.Fatalf("hangup: %v", err)
	}
	br.mu.Lock()
	defer br.mu.Unlock()
	if len(br.disconnectCallIDs) != 1 || br.disconnectCallIDs[0] != "c1" {
		t.Errorf("bridge.Disconnect calls=%v", br.disconnectCallIDs)
	}
}

func TestHangup_NotBridgedReturnsErrNotBridged(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	if err := hub.Hangup("nope", "x"); !errors.Is(err, ErrNotBridged) {
		t.Errorf("err=%v, want ErrNotBridged", err)
	}
}

func TestReleaseAwaiting(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, JoinTimeout: time.Hour})
	_ = hub.RegisterAwaiting("c1", fakeLeg())
	if err := hub.ReleaseAwaiting("c1", "sip-bye"); err != nil {
		t.Fatalf("release: %v", err)
	}
	if got := hub.Awaiting(); len(got) != 0 {
		t.Errorf("after release Awaiting()=%v", got)
	}
}

func TestWatchdog_TimeoutFiresOnEnded(t *testing.T) {
	var (
		mu      sync.Mutex
		ended   []string
	)
	hub := New(Config{
		Bridge:      &fakeBridge{},
		JoinTimeout: 30 * time.Millisecond,
		OnEnded: func(callID, reason string) {
			mu.Lock()
			defer mu.Unlock()
			ended = append(ended, callID+":"+reason)
		},
	})
	_ = hub.RegisterAwaiting("slow", fakeLeg())
	time.Sleep(80 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if len(ended) != 1 || ended[0] != "slow:timeout" {
		t.Errorf("ended=%v", ended)
	}
}

func TestNopBridge_ConnectErrors(t *testing.T) {
	hub := New(Config{}) // default = NopBridge
	_ = hub.RegisterAwaiting("c1", fakeLeg())
	if _, err := hub.Pickup(context.Background(), "c1", "x"); err == nil {
		t.Fatal("nop bridge should error on Connect")
	}
}

func TestTokenOK(t *testing.T) {
	hub := New(Config{Bridge: &fakeBridge{}, Token: "secret"})
	if !hub.tokenOK("secret") {
		t.Error("matching token should pass")
	}
	if hub.tokenOK("wrong") {
		t.Error("wrong token must not pass")
	}
	if hub.tokenOK("") {
		t.Error("empty token against configured non-empty must not pass")
	}

	// Empty configured token = auth disabled.
	open := New(Config{Bridge: &fakeBridge{}})
	if !open.tokenOK("") {
		t.Error("empty configured token should accept anything")
	}
}

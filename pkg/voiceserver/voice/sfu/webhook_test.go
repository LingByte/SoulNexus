// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestWebhookEmitterDisabled(t *testing.T) {
	// No URL → emit should be a no-op and not start the loop goroutine.
	em := newWebhookEmitter(&Config{}, zap.NewNop())
	em.emit(Event{Type: EventRoomStarted}) // must not panic
}

func TestWebhookEmitterDelivery(t *testing.T) {
	var got int32
	var lastSig string
	var lastBody []byte
	mu := sync.Mutex{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		lastSig = r.Header.Get("X-SFU-Signature")
		lastBody = b
		mu.Unlock()
		atomic.AddInt32(&got, 1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := (&Config{
		AuthSecret:     "k",
		WebhookURL:     srv.URL,
		WebhookTimeout: 2 * time.Second,
	}).Normalise()
	em := newWebhookEmitter(cfg, zap.NewNop())
	em.emit(Event{Type: EventParticipantJoined, Room: "r", Identity: "alice", Timestamp: 1})

	// Wait up to 2s for delivery.
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&got) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&got) != 1 {
		t.Fatalf("webhook not delivered")
	}

	mu.Lock()
	sig := lastSig
	body := append([]byte(nil), lastBody...)
	mu.Unlock()

	// Verify signature.
	mac := hmac.New(sha256.New, []byte("k"))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	if sig != want {
		t.Errorf("bad signature: got %s want %s", sig, want)
	}

	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != EventParticipantJoined || ev.Identity != "alice" {
		t.Errorf("event payload wrong: %+v", ev)
	}
}

func TestWebhookEmitterRejected(t *testing.T) {
	// 4xx response → only logged, no panic, drains successfully.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	cfg := (&Config{AuthSecret: "k", WebhookURL: srv.URL, WebhookTimeout: time.Second}).Normalise()
	em := newWebhookEmitter(cfg, zap.NewNop())
	em.emit(Event{Type: EventRoomEnded, Room: "r"})
	time.Sleep(200 * time.Millisecond) // let loop drain
}

func TestWebhookEmitterBufferOverflow(t *testing.T) {
	// Block the server so the buffer fills, then queue >128 events;
	// extras must drop without panicking.
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
		w.WriteHeader(200)
	}))
	defer srv.Close()
	defer close(block)
	cfg := (&Config{WebhookURL: srv.URL, WebhookTimeout: 5 * time.Second}).Normalise()
	em := newWebhookEmitter(cfg, zap.NewNop())
	for i := 0; i < 300; i++ {
		em.emit(Event{Type: EventRoomStarted})
	}
}

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// EventType enumerates the lifecycle moments the SFU notifies the
// business backend about. Kept small on purpose: a caller that wants
// finer telemetry should scrape /metrics or tail the logger instead.
type EventType string

const (
	EventRoomStarted       EventType = "room.started"
	EventRoomEnded         EventType = "room.ended"
	EventParticipantJoined EventType = "participant.joined"
	EventParticipantLeft   EventType = "participant.left"
	EventTrackPublished    EventType = "track.published"
	EventRecordingFinished EventType = "recording.finished"
)

// Event is the JSON payload POSTed to Config.WebhookURL. Unused fields
// are omitted so subscribers can switch on Type without worrying about
// null fields polluting their typed structs.
type Event struct {
	Type          EventType `json:"type"`
	Room          string    `json:"room,omitempty"`
	ParticipantID string    `json:"participantId,omitempty"`
	Identity      string    `json:"identity,omitempty"`
	TrackID       string    `json:"trackId,omitempty"`
	TrackKind     string    `json:"trackKind,omitempty"`
	Reason        string    `json:"reason,omitempty"`
	// RecordingURL is populated for EventRecordingFinished; value is
	// the PublicURL reported by the store backing Config.RecordBucket.
	RecordingURL string `json:"recordingUrl,omitempty"`
	// DurationMs is the wall-clock duration for recording events.
	DurationMs int64 `json:"durationMs,omitempty"`
	Timestamp  int64 `json:"timestamp"`
}

// webhookEmitter asynchronously POSTs events to Config.WebhookURL. A
// small buffered channel absorbs bursts (e.g. a 16-peer room joining
// in parallel); when the channel fills, events are dropped with a log
// warning so the SFU never blocks on a slow subscriber.
type webhookEmitter struct {
	url    string
	secret string
	logger *zap.Logger
	client *http.Client

	ch       chan Event
	done     chan struct{} // closed by shutdown(); loop range-exits on ch close
	loopDone chan struct{} // closed by loop() when it returns (nil if no loop running)
	shutOnce sync.Once
}

func newWebhookEmitter(cfg *Config, logger *zap.Logger) *webhookEmitter {
	w := &webhookEmitter{
		url:    strings.TrimSpace(cfg.WebhookURL),
		secret: cfg.AuthSecret,
		logger: logger,
		client: &http.Client{Timeout: cfg.WebhookTimeout},
		ch:     make(chan Event, 128),
		done:   make(chan struct{}),
	}
	if w.url == "" {
		// No URL → no loop goroutine to spawn, but we still want
		// shutdown() to be a no-op-safe method. emit() short-circuits.
		return w
	}
	w.loopDone = make(chan struct{})
	// One background goroutine drains the channel. Serial delivery
	// preserves event ordering which receivers usually rely on
	// (room.started before participant.joined, etc.).
	go w.loop()
	return w
}

// emit enqueues an event. Non-blocking — drops with a log if the
// buffer is full so a stalled subscriber can never stall the SFU.
// Post-shutdown emits are silently dropped.
func (w *webhookEmitter) emit(ev Event) {
	if w == nil || w.url == "" {
		return
	}
	select {
	case <-w.done:
		// Shutdown already started; ignore.
		return
	default:
	}
	select {
	case w.ch <- ev:
	case <-w.done:
	default:
		w.logger.Warn("sfu: webhook buffer full, dropping event", zap.String("type", string(ev.Type)))
	}
}

// shutdown stops the delivery loop and waits for any in-flight POST to
// finish (bounded by client timeout). Idempotent.
func (w *webhookEmitter) shutdown() {
	if w == nil {
		return
	}
	w.shutOnce.Do(func() {
		close(w.done)
		close(w.ch)
	})
}

func (w *webhookEmitter) loop() {
	defer close(w.loopDone)
	for ev := range w.ch {
		w.post(ev)
	}
}

// post serialises the event, signs the body, and performs the HTTP
// POST. Errors are logged; there is no retry by design (the caller
// should treat webhooks as best-effort and reconcile state another
// way when it matters).
func (w *webhookEmitter) post(ev Event) {
	body, err := json.Marshal(ev)
	if err != nil {
		w.logger.Warn("sfu: marshal webhook", zap.Error(err))
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		w.logger.Warn("sfu: build webhook req", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if w.secret != "" {
		mac := hmac.New(sha256.New, []byte(w.secret))
		_, _ = mac.Write(body)
		req.Header.Set("X-SFU-Signature", hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Warn("sfu: webhook post", zap.Error(err), zap.String("type", string(ev.Type)))
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 400 {
		w.logger.Warn("sfu: webhook rejected",
			zap.Int("status", resp.StatusCode),
			zap.String("type", string(ev.Type)))
	}
}

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/gorilla/websocket"
)

// TestClient_ASRSentenceFilter_SuppressesPartials wires a real
// asr.SentenceFilter into the Client and verifies that:
//
//	1. mid-sentence partials are suppressed (no asr.partial event)
//	2. once a terminator arrives, an asr.partial event fires with
//	   ONLY the freshly-completed sentence
//	3. final transcripts always pass through, even when the filter
//	   would otherwise return "" (case: final equals last emit)
func TestClient_ASRSentenceFilter_SuppressesPartials(t *testing.T) {
	// Mini WS dialog server that captures events.
	var (
		mu        sync.Mutex
		gotEvents []Event
		ready     = make(chan struct{})
	)
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		close(ready)
		for {
			_, raw, err := c.ReadMessage()
			if err != nil {
				return
			}
			var ev Event
			if err := json.Unmarshal(raw, &ev); err == nil {
				mu.Lock()
				gotEvents = append(gotEvents, ev)
				mu.Unlock()
			}
		}
	}))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	// Reuse the existing fakeASR + minimal voice.Attached scaffolding
	// from client_test.go (same package, so we can build directly).
	fa := &fakeASR{}
	att := &voice.Attached{}
	asrPipe, err := asr.New(asr.Options{ASR: fa, SampleRate: 16000, MinFeedBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	att.ASR = asrPipe

	cli, err := NewClient(ClientConfig{
		URL:               wsURL,
		Attached:          att,
		CallID:            "call-sf",
		ASRSentenceFilter: asr.NewSentenceFilter(0.85),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.Start(context.Background(), StartMeta{From: "a", To: "b", Codec: "pcma", PCMHz: 8000}); err != nil {
		t.Fatal(err)
	}
	defer cli.Close("test")
	<-ready

	// Connect the recogniser delivery hook.
	_ = asrPipe.ProcessPCM(context.Background(), make([]byte, 4))
	if fa.tr == nil {
		t.Fatal("asr pipeline did not connect")
	}

	// Mid-sentence partials — must NOT produce events.
	fa.tr("今天", false, 0, "")
	fa.tr("今天天气", false, 0, "")
	// Sentence terminator — should produce ONE asr.partial.
	fa.tr("今天天气不错。", false, 0, "")
	// Final tail without terminator — should produce ONE asr.final.
	fa.tr("今天天气不错。我也这么觉得", true, 0, "")

	// Allow goroutines to drain.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(gotEvents)
		mu.Unlock()
		if n >= 3 { // call.started + asr.partial + asr.final
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	var partials, finals []string
	for _, ev := range gotEvents {
		switch ev.Type {
		case EvASRPartial:
			partials = append(partials, ev.Text)
		case EvASRFinal:
			finals = append(finals, ev.Text)
		}
	}
	if len(partials) != 1 {
		t.Fatalf("partials=%v, want exactly 1 (the sentence-terminated one)", partials)
	}
	if partials[0] != "今天天气不错。" {
		t.Errorf("partial[0]=%q, want %q", partials[0], "今天天气不错。")
	}
	if len(finals) != 1 {
		t.Fatalf("finals=%v, want exactly 1", finals)
	}
	if finals[0] != "我也这么觉得" {
		t.Errorf("final[0]=%q, want %q", finals[0], "我也这么觉得")
	}
}

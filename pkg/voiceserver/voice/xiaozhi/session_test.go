// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

// End-to-end smoke test for the xiaozhi adapter. We simulate three pieces:
//
//  1. A fake "device" — a gorilla/websocket client that opens the xiaozhi
//     endpoint, sends hello (PCM mode so we don't need libopus in tests),
//     listen.start, a couple of binary PCM frames, then listen.stop.
//  2. A fake "dialog plane" — another gorilla/websocket server that
//     accepts the gateway.Client connection, observes call.started, and
//     pushes a tts.speak command back so the device should receive a
//     tts:start … <binary> … tts:stop envelope.
//  3. A stub ASR that immediately fires a final transcript on its first
//     audio chunk, and a stub TTS that emits a tiny PCM stream.
//
// The test asserts that the wire-protocol contract holds end-to-end:
// stt push to device, tts:start envelope, binary frames, tts:stop.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"

	"github.com/gorilla/websocket"
)

// ----- Stub recognizer -----

type stubASR struct {
	mu     sync.Mutex
	tr     recognizer.TranscribeResult
	fired  bool
	active bool
}

func (a *stubASR) Init(tr recognizer.TranscribeResult, _ recognizer.ProcessError) {
	a.mu.Lock()
	a.tr = tr
	a.mu.Unlock()
}
func (a *stubASR) Vendor() string                  { return "stub" }
func (a *stubASR) ConnAndReceive(_ string) error    { a.mu.Lock(); a.active = true; a.mu.Unlock(); return nil }
func (a *stubASR) Activity() bool                   { a.mu.Lock(); defer a.mu.Unlock(); return a.active }
func (a *stubASR) RestartClient()                   {}
func (a *stubASR) SendAudioBytes(b []byte) error {
	a.mu.Lock()
	tr := a.tr
	already := a.fired
	if !already && tr != nil && len(b) > 0 {
		a.fired = true
	}
	a.mu.Unlock()
	if !already && tr != nil && len(b) > 0 {
		go tr("hello world", true, 100*time.Millisecond, "u1")
	}
	return nil
}
func (a *stubASR) SendEnd() error  { return nil }
func (a *stubASR) StopConn() error { a.mu.Lock(); a.active = false; a.mu.Unlock(); return nil }

// ----- Stub TTS -----

type stubTTS struct{}

func (stubTTS) SynthesizeStream(_ context.Context, text string,
	onChunk func(pcm []byte) error) error {
	// One 320-byte frame (10ms @ 16kHz PCM16 mono) is enough to trigger
	// the OnTTSStart / OnTurn lifecycle without a real TTS vendor.
	if text == "" {
		return nil
	}
	return onChunk(make([]byte, 320))
}
func (stubTTS) SampleRate() int { return 16000 }

// ----- Stub factory -----

type stubFactory struct{}

func (stubFactory) NewASR(_ context.Context, _ string) (recognizer.TranscribeService, int, error) {
	return &stubASR{}, 16000, nil
}
func (stubFactory) TTS(_ context.Context, _ string) (voicetts.Service, int, error) {
	return stubTTS{}, 16000, nil
}

// ----- Stub dialog plane -----

// dialogServer is a minimal gateway-protocol counterpart used by the test
// to drive the xiaozhi session. It accepts the WS, reads call.started,
// then pushes one tts.speak so the device side receives the full audio
// envelope.
func newDialogServer(t *testing.T, gotEvent chan<- string) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("dialog upgrade: %v", err)
			return
		}
		defer conn.Close()

		// Push a tts.speak as soon as we see the asr.final event, so
		// the test exercises the inbound audio → ASR → dialog → TTS
		// → outbound audio loop end-to-end.
		go func() {
			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var ev map[string]any
				if err := json.Unmarshal(raw, &ev); err == nil {
					if t, _ := ev["type"].(string); t != "" {
						select {
						case gotEvent <- t:
						default:
						}
						if t == "asr.final" {
							_ = conn.WriteJSON(map[string]any{
								"type":         "tts.speak",
								"call_id":      ev["call_id"],
								"text":         "hi",
								"utterance_id": "u1",
							})
						}
					}
				}
			}
		}()
		// Block until the client closes.
		<-r.Context().Done()
	}))
}

// runFakeDevice connects to the xiaozhi WS and drives a hello + listen
// cycle. It collects every text and binary frame the server pushes back
// for assertion. Returns once it has observed at least one tts:stop or
// the timeout fires.
func runFakeDevice(t *testing.T, wsURL string) (texts []string, binaryCount int) {
	t.Helper()
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("device dial: %v", err)
	}
	defer conn.Close()

	// PCM mode so we don't need libopus on test machines without cgo.
	hello := map[string]any{
		"type":      "hello",
		"version":   1,
		"transport": "websocket",
		"audio_params": map[string]any{
			"format":         "pcm",
			"sample_rate":    16000,
			"channels":       1,
			"frame_duration": 60,
		},
	}
	if err := conn.WriteJSON(hello); err != nil {
		t.Fatal(err)
	}

	// Background reader: stash everything we see.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			mt, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			switch mt {
			case websocket.TextMessage:
				texts = append(texts, string(raw))
				if strings.Contains(string(raw), `"state":"stop"`) {
					_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
				}
			case websocket.BinaryMessage:
				binaryCount++
			}
		}
	}()

	// Wait for the welcome reply, then listen.start + a PCM frame big
	// enough to clear the asr.Pipeline 20-ms feed threshold (640 bytes
	// at 16 kHz PCM16 mono). 3200 bytes = 100 ms keeps us comfortably
	// above the buffer floor.
	time.Sleep(200 * time.Millisecond)
	_ = conn.WriteJSON(map[string]any{"type": "listen", "state": "start", "mode": "auto"})
	_ = conn.WriteMessage(websocket.BinaryMessage, make([]byte, 3200))
	// listen.stop triggers asr.Pipeline.Flush so the buffered tail is
	// pushed even when the vendor never declares end-of-utterance on
	// its own.
	_ = conn.WriteJSON(map[string]any{"type": "listen", "state": "stop"})

	<-done
	return
}

func TestXiaozhi_EndToEnd_PCMRoundTrip(t *testing.T) {
	dialogEvents := make(chan string, 16)
	dialogSrv := newDialogServer(t, dialogEvents)
	defer dialogSrv.Close()
	dialogWS := "ws" + strings.TrimPrefix(dialogSrv.URL, "http")

	srv, err := NewServer(ServerConfig{
		SessionFactory: stubFactory{},
		DialogWSURL:    dialogWS,
		CallIDPrefix:   "xz-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	httpSrv := httptest.NewServer(http.HandlerFunc(srv.Handle))
	defer httpSrv.Close()
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http") + "/"

	texts, bins := runFakeDevice(t, wsURL)

	// Check the dialog plane saw the SIP-style envelope: at minimum a
	// call.started and an asr.final.
	seen := map[string]bool{}
	for done := false; !done; {
		select {
		case e := <-dialogEvents:
			seen[e] = true
		default:
			done = true
		}
	}
	if !seen["call.started"] {
		t.Errorf("dialog never saw call.started; got %v", seen)
	}
	if !seen["asr.final"] {
		t.Errorf("dialog never saw asr.final; got %v", seen)
	}

	// Device-side: must have hello, stt, tts:start, binary frames, tts:stop.
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, `"type":"hello"`) {
		t.Errorf("device missing welcome hello:\n%s", joined)
	}
	if !strings.Contains(joined, `"type":"stt"`) {
		t.Errorf("device missing stt push:\n%s", joined)
	}
	if !strings.Contains(joined, `"state":"start"`) {
		t.Errorf("device missing tts:start envelope:\n%s", joined)
	}
	if !strings.Contains(joined, `"state":"stop"`) {
		t.Errorf("device missing tts:stop envelope:\n%s", joined)
	}
	if bins == 0 {
		t.Errorf("device received no binary audio frames")
	}
}

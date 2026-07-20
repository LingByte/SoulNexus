package aliyunomni

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/lingllm/realtime"
	"github.com/gorilla/websocket"
)

// TestRegistration: the provider self-registers under its canonical slug
// and accepted aliases.
func TestRegistration(t *testing.T) {
	for _, slug := range []string{"aliyun_omni", "qwen_omni", "dashscope_omni"} {
		if realtime.Lookup(slug) == nil {
			t.Fatalf("provider %q should be registered", slug)
		}
	}
}

// TestNew_RequiresAPIKey: missing apiKey is rejected at construction so
// voice attach surfaces a typed error before opening any connection.
func TestNew_RequiresAPIKey(t *testing.T) {
	_, err := New(map[string]any{}, realtime.Options{OnEvent: func(realtime.Event) {}})
	if err == nil || !strings.Contains(err.Error(), "apiKey") {
		t.Fatalf("expected apiKey error, got %v", err)
	}
}

// TestNew_AppliesDefaults verifies the credential parser applies vendor
// defaults and honours both camelCase and snake_case keys.
func TestNew_AppliesDefaults(t *testing.T) {
	a, err := New(
		map[string]any{
			"apiKey":          "sk-test",
			"baseUrl":         "ws://example/foo",
			"voice":           "Ethan",
			"dial_timeout_ms": 1234,
		},
		realtime.Options{OnEvent: func(realtime.Event) {}},
	)
	if err != nil {
		t.Fatal(err)
	}
	ag := a.(*agent)
	if ag.cfg.Model != defaultModel {
		t.Fatalf("expected default model, got %q", ag.cfg.Model)
	}
	if ag.cfg.Voice != "Ethan" {
		t.Fatalf("voice override lost: %q", ag.cfg.Voice)
	}
	if ag.cfg.DialTimeoutMs != 1234 {
		t.Fatalf("snake_case dial_timeout_ms not applied: %d", ag.cfg.DialTimeoutMs)
	}
}

func TestBuildURL(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		model   string
		want    string
		wantErr bool
	}{
		{"default base", "", defaultModel, defaultBaseURL + "?model=" + defaultModel, false},
		{"custom base no model", "ws://x/y", "m1", "ws://x/y?model=m1", false},
		{"custom preserves caller model", "ws://x/y?model=already", "m1", "ws://x/y?model=already", false},
		{"https rejected", "https://x", "m", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildURL(tc.base, tc.model)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

// TestEndToEnd_FakeServer is the integration spine: spin up a WS server
// that follows the real Qwen-Omni protocol shape, drive a full Start →
// PushAudio → receive transcript+audio+turn-end loop, and assert every
// event reached our callbacks in the right shape.
func TestEndToEnd_FakeServer(t *testing.T) {
	var (
		gotSessionUpdate atomic.Bool
		gotAudioFrame    atomic.Bool
		gotCancel        atomic.Bool
	)

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth check: realtime layer MUST send the bearer token.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, "missing bearer", http.StatusUnauthorized)
			return
		}
		// Model query param: factory MUST inject it from cfg["model"].
		if r.URL.Query().Get("model") == "" {
			http.Error(w, "missing model", http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send session.created right after the handshake — the real
		// vendor does this before any client message.
		mustWriteJSON(t, conn, map[string]any{"type": "session.created"})

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					return
				}
				var head map[string]any
				_ = json.Unmarshal(raw, &head)
				switch head["type"] {
				case "session.update":
					gotSessionUpdate.Store(true)
					mustWriteJSON(t, conn, map[string]any{"type": "session.updated"})
				case "input_audio_buffer.append":
					if !gotAudioFrame.Load() {
						gotAudioFrame.Store(true)
						// Server "VAD" fires speech_started so the
						// agent surfaces a barge-in trigger.
						mustWriteJSON(t, conn, map[string]any{"type": "input_audio_buffer.speech_started"})
						mustWriteJSON(t, conn, map[string]any{
							"type":       "conversation.item.input_audio_transcription.completed",
							"transcript": "你好",
						})
						// Stream a delta + audio + done turn.
						mustWriteJSON(t, conn, map[string]any{
							"type":  "response.audio_transcript.delta",
							"delta": "你好",
						})
						mustWriteJSON(t, conn, map[string]any{
							"type":  "response.audio.delta",
							"delta": base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03, 0x04}),
						})
						mustWriteJSON(t, conn, map[string]any{
							"type":       "response.audio_transcript.done",
							"transcript": "你好，请问有什么可以帮你？",
						})
						mustWriteJSON(t, conn, map[string]any{"type": "response.done"})
					}
				case "response.cancel":
					gotCancel.Store(true)
				}
			}
		}()
		wg.Wait()
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	var (
		mu          sync.Mutex
		sessionOpen int
		userTrans   []string
		assistant   []string
		audioBytes  []byte
		turnEnded   int
		speechBegan int
	)
	collected := make(chan struct{})
	a, err := New(
		map[string]any{"apiKey": "sk-test", "baseUrl": wsURL, "model": "qwen3-omni-flash-realtime"},
		realtime.Options{
			SystemPrompt: "你是个人助理小云",
			Voice:        "Ethan",
			OnEvent: func(ev realtime.Event) {
				mu.Lock()
				defer mu.Unlock()
				switch ev.Type {
				case realtime.EventSessionOpen:
					sessionOpen++
				case realtime.EventUserSpeechStarted:
					speechBegan++
				case realtime.EventUserTranscript:
					userTrans = append(userTrans, ev.Text)
				case realtime.EventAssistantText:
					if ev.Final {
						assistant = append(assistant, "FINAL:"+ev.Text)
					} else {
						assistant = append(assistant, "DELTA:"+ev.Text)
					}
				case realtime.EventAssistantAudio:
					audioBytes = append(audioBytes, ev.AudioPC...)
				case realtime.EventAssistantTurnEnd:
					turnEnded++
					select {
					case <-collected:
					default:
						close(collected)
					}
				case realtime.EventError:
					t.Errorf("unexpected error event: %v", ev.Err)
				}
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// One 20ms@16k frame = 640 bytes of zeros. The fake server only needs
	// to see *any* append to advance.
	if err := a.PushAudio(make([]byte, 640)); err != nil {
		t.Fatalf("PushAudio: %v", err)
	}
	// Hit Cancel so we also exercise response.cancel path.
	if err := a.Cancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	select {
	case <-collected:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for turn end: openCount=%d transcripts=%v assistant=%v audioLen=%d",
			sessionOpen, userTrans, assistant, len(audioBytes))
	}
	_ = a.Close()

	mu.Lock()
	defer mu.Unlock()
	if sessionOpen != 1 {
		t.Fatalf("EventSessionOpen should fire exactly once, got %d", sessionOpen)
	}
	if !gotSessionUpdate.Load() {
		t.Fatal("server did not receive session.update")
	}
	if !gotAudioFrame.Load() {
		t.Fatal("server did not receive input_audio_buffer.append")
	}
	if !gotCancel.Load() {
		t.Fatal("server did not receive response.cancel")
	}
	if speechBegan != 1 {
		t.Fatalf("speech_started not surfaced (count=%d)", speechBegan)
	}
	if len(userTrans) != 1 || userTrans[0] != "你好" {
		t.Fatalf("user transcript wrong: %v", userTrans)
	}
	if len(assistant) != 2 || assistant[0] != "DELTA:你好" || !strings.HasPrefix(assistant[1], "FINAL:") {
		t.Fatalf("assistant text events wrong: %v", assistant)
	}
	if string(audioBytes) != string([]byte{0x01, 0x02, 0x03, 0x04}) {
		t.Fatalf("decoded audio mismatch: % x", audioBytes)
	}
	if turnEnded != 1 {
		t.Fatalf("turn end count: %d", turnEnded)
	}
}

func TestFunctionCalling_FakeServer(t *testing.T) {
	var (
		gotToolOutput atomic.Bool
		gotRespCreate atomic.Bool
	)
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		mustWriteJSON(t, conn, map[string]any{"type": "session.created"})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var head struct {
				Type string          `json:"type"`
				Item json.RawMessage `json:"item"`
			}
			_ = json.Unmarshal(raw, &head)
			switch head.Type {
			case "session.update":
				mustWriteJSON(t, conn, map[string]any{"type": "session.updated"})
			case "input_audio_buffer.append":
				if !gotToolOutput.Load() {
					mustWriteJSON(t, conn, map[string]any{
						"type":      "response.function_call_arguments.done",
						"call_id":   "call_test_1",
						"name":      "get_current_time",
						"arguments": `{"timezone":"Asia/Shanghai"}`,
					})
					mustWriteJSON(t, conn, map[string]any{"type": "response.done"})
				}
			case "conversation.item.create":
				var item struct {
					Type string `json:"type"`
				}
				_ = json.Unmarshal(head.Item, &item)
				if item.Type == "function_call_output" {
					gotToolOutput.Store(true)
				}
			case "response.create":
				gotRespCreate.Store(true)
			}
			if gotToolOutput.Load() && gotRespCreate.Load() {
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	var toolCalled atomic.Bool
	done := make(chan struct{})
	a, err := New(
		map[string]any{"apiKey": "sk-test", "baseUrl": wsURL},
		realtime.Options{
			Tools: []realtime.Tool{{
				Name:        "get_current_time",
				Description: "获取当前时间",
				Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
			}},
			ToolHandler: func(name string, args map[string]any) string {
				if name == "get_current_time" {
					toolCalled.Store(true)
				}
				return `{"ok":true}`
			},
			OnEvent: func(ev realtime.Event) {
				if ev.Type == realtime.EventAssistantTurnEnd {
					select {
					case <-done:
					default:
						close(done)
					}
				}
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := a.PushAudio(make([]byte, 640)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	time.Sleep(50 * time.Millisecond)
	_ = a.Close()
	if !toolCalled.Load() {
		t.Fatal("ToolHandler not invoked")
	}
	if !gotToolOutput.Load() {
		t.Fatal("server did not receive function_call_output")
	}
	if !gotRespCreate.Load() {
		t.Fatal("server did not receive response.create after tool output")
	}
}

// TestPushAudio_AfterClose is a regression guard for the session teardown
// race: callers MUST NOT panic when pushing audio after Close.
func TestPushAudio_AfterClose(t *testing.T) {
	a, err := New(
		map[string]any{"apiKey": "sk-test"},
		realtime.Options{OnEvent: func(realtime.Event) {}},
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = a.Close()
	if err := a.PushAudio([]byte{0x01}); err != realtime.ErrAgentClosed {
		t.Fatalf("expected ErrAgentClosed, got %v", err)
	}
	if err := a.Cancel(); err != realtime.ErrAgentClosed {
		t.Fatalf("expected ErrAgentClosed, got %v", err)
	}
	if err := a.UpdateInstructions("test"); err != realtime.ErrAgentClosed {
		t.Fatalf("expected ErrAgentClosed on UpdateInstructions, got %v", err)
	}
}

func mustWriteJSON(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	buf, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, buf); err != nil {
		t.Fatal(err)
	}
}

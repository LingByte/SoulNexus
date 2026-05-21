package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBuildAliyunWSURL(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		model     string
		wantQuery string
		wantErr   bool
	}{
		{"default base", "", "qwen3-tts-flash-realtime", "model=qwen3-tts-flash-realtime", false},
		{"base with no model", "wss://example.com/api-ws/v1/realtime", "m1", "model=m1", false},
		{"base preserves caller model", "wss://example.com/realtime?model=preset", "ignored", "model=preset", false},
		{"https rejected", "https://example.com", "x", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildAliyunWSURL(tt.base, tt.model)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !strings.Contains(got, tt.wantQuery) {
				t.Fatalf("url=%q does not contain %q", got, tt.wantQuery)
			}
		})
	}
}

func TestAliyunServiceDefaults(t *testing.T) {
	svc := NewAliyunService(AliyunTTSConfig{APIKey: "sk-test"})
	if svc.Provider() != ProviderAliyun {
		t.Fatalf("provider=%v", svc.Provider())
	}
	f := svc.Format()
	if f.SampleRate != 24000 || f.Channels != 1 || f.BitDepth != 16 {
		t.Fatalf("unexpected format: %+v", f)
	}
	if svc.opt.Model != "qwen3-tts-flash-realtime" {
		t.Fatalf("model=%s", svc.opt.Model)
	}
	if svc.opt.Voice != "Cherry" || svc.opt.Mode != "server_commit" {
		t.Fatalf("voice/mode=%s/%s", svc.opt.Voice, svc.opt.Mode)
	}
}

func TestAliyunSynthesizeStream_EmptyText(t *testing.T) {
	svc := NewAliyunService(AliyunTTSConfig{APIKey: "sk-test"})
	called := false
	err := svc.SynthesizeStream(context.Background(), "  \n", func([]byte) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if called {
		t.Fatalf("callback should not fire on empty text")
	}
}

func TestAliyunSynthesizeStream_MissingAPIKey(t *testing.T) {
	svc := NewAliyunService(AliyunTTSConfig{}) // env unset in CI
	if svc.opt.APIKey != "" {
		t.Skip("DASHSCOPE_API_KEY is set in this environment")
	}
	err := svc.SynthesizeStream(context.Background(), "hello", func([]byte) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "DASHSCOPE_API_KEY") {
		t.Fatalf("expected api-key error, got %v", err)
	}
}

// TestAliyunSynthesizeStream_FakeServer exercises the WS handshake, session
// update, append, finish, audio.delta drain and clean shutdown without
// touching real DashScope endpoints.
func TestAliyunSynthesizeStream_FakeServer(t *testing.T) {
	var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("model") != "qwen3-tts-flash-realtime" {
			t.Errorf("missing model query: %s", r.URL.RawQuery)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("missing bearer auth: %q", r.Header.Get("Authorization"))
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		gotSessionUpdate, gotAppend, gotFinish := false, false, false
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ev map[string]any
			if err := json.Unmarshal(raw, &ev); err != nil {
				t.Errorf("bad json: %v", err)
				return
			}
			switch ev["type"] {
			case "session.update":
				gotSessionUpdate = true
				sess, _ := ev["session"].(map[string]any)
				if sess["voice"] != "Cherry" {
					t.Errorf("voice=%v", sess["voice"])
				}
				if sess["response_format"] != "pcm" {
					t.Errorf("response_format=%v", sess["response_format"])
				}
			case "input_text_buffer.append":
				gotAppend = true
				if ev["text"] != "你好世界" {
					t.Errorf("text=%v", ev["text"])
				}
			case "session.finish":
				gotFinish = true
				if !gotSessionUpdate || !gotAppend {
					t.Errorf("finish before update/append: u=%v a=%v", gotSessionUpdate, gotAppend)
				}
				// Send two audio chunks + response.done + session.finished.
				for _, chunk := range [][]byte{{0x10, 0x00, 0x20, 0x00}, {0x30, 0x00, 0x40, 0x00}} {
					_ = conn.WriteJSON(map[string]any{
						"type":  "response.audio.delta",
						"delta": base64.StdEncoding.EncodeToString(chunk),
					})
				}
				_ = conn.WriteJSON(map[string]any{"type": "response.done"})
				_ = conn.WriteJSON(map[string]any{"type": "session.finished"})
				return
			}
			_ = gotFinish
		}
	}))
	defer srv.Close()

	wsBase := "ws" + strings.TrimPrefix(srv.URL, "http")
	svc := NewAliyunService(AliyunTTSConfig{
		APIKey:  "sk-test",
		BaseURL: wsBase,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var got []byte
	err := svc.SynthesizeStream(ctx, "你好世界", func(p []byte) error {
		got = append(got, p...)
		return nil
	})
	if err != nil {
		t.Fatalf("SynthesizeStream err=%v", err)
	}
	if len(got) != 8 {
		t.Fatalf("expected 8 bytes pcm, got %d (%v)", len(got), got)
	}
}

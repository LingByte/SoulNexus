package aliyunomni

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestPreviewSpeech_FakeServer(t *testing.T) {
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
				Type string `json:"type"`
			}
			_ = json.Unmarshal(raw, &head)
			switch head.Type {
			case "session.update":
				mustWriteJSON(t, conn, map[string]any{"type": "session.updated"})
			case "response.create":
				mustWriteJSON(t, conn, map[string]any{
					"type":  "response.audio.delta",
					"delta": "AQID", // base64 [1,2,3]
				})
				mustWriteJSON(t, conn, map[string]any{"type": "response.done"})
				return
			}
		}
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pcm, sr, err := PreviewSpeech(ctx, map[string]any{
		"apiKey":  "sk-test",
		"baseUrl": wsURL,
		"model":   "qwen3.5-omni-flash-realtime-2026-03-15",
	}, "Tina", "你好")
	if err != nil {
		t.Fatal(err)
	}
	if sr != previewOutputSampleRate {
		t.Fatalf("sample rate = %d, want %d", sr, previewOutputSampleRate)
	}
	if string(pcm) != string([]byte{1, 2, 3}) {
		t.Fatalf("pcm = %v", pcm)
	}
}

package aliyunomni

// PreviewSpeech opens a short-lived Qwen-Omni-Realtime session, disables VAD,
// and asks the model to read `text` aloud with `voice`. Used by the voice console
// voice preview API for tenant realtime (aliyun_omni) mode.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const previewOutputSampleRate = 24000

// PreviewSpeech returns mono PCM16LE @ 24 kHz.
func PreviewSpeech(ctx context.Context, cfg map[string]any, voice, text string) ([]byte, int, error) {
	apiKey := firstString(cfg, "apiKey", "api_key")
	if apiKey == "" {
		return nil, 0, fmt.Errorf("aliyunomni preview: apiKey is required")
	}
	model := firstString(cfg, "model")
	if model == "" {
		model = defaultModel
	}
	baseURL := firstString(cfg, "baseUrl", "base_url")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	voice = strings.TrimSpace(voice)
	if voice == "" {
		voice = defaultVoice
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, 0, fmt.Errorf("aliyunomni preview: empty text")
	}

	wsURL, err := buildURL(baseURL, model)
	if err != nil {
		return nil, 0, err
	}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+apiKey)
	headers.Set("X-DashScope-OmniRealtime", "true")

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		status := -1
		if resp != nil {
			status = resp.StatusCode
			resp.Body.Close()
		}
		return nil, 0, fmt.Errorf("aliyunomni preview: dial (status=%d): %w", status, err)
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	if err := writePreviewJSON(conn, map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"voice":               voice,
			"modalities":          []string{"text", "audio"},
			"input_audio_format":  "pcm",
			"output_audio_format": "pcm",
			"turn_detection":      nil,
			"instructions":        "你是语音试听助手。请只用当前音色直接朗读用户给出的文字，不要添加任何解释、问候或其他内容。",
		},
	}); err != nil {
		return nil, 0, fmt.Errorf("aliyunomni preview: session.update: %w", err)
	}

	readPrompt := fmt.Sprintf("请朗读：%s", text)
	if err := writePreviewJSON(conn, map[string]any{
		"type": "response.create",
		"response": map[string]any{
			"modalities":   []string{"text", "audio"},
			"instructions": readPrompt,
		},
	}); err != nil {
		return nil, 0, fmt.Errorf("aliyunomni preview: response.create: %w", err)
	}

	var buf []byte
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil, 0, ctx.Err()
			}
			if len(buf) > 0 && websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			return nil, 0, fmt.Errorf("aliyunomni preview: read: %w", err)
		}
		var head wireHead
		if err := json.Unmarshal(raw, &head); err != nil {
			continue
		}
		switch head.Type {
		case "response.audio.delta", "response.output_audio.delta":
			var msg wireDelta
			_ = json.Unmarshal(raw, &msg)
			if msg.Delta == "" {
				continue
			}
			pcm, decErr := base64.StdEncoding.DecodeString(msg.Delta)
			if decErr != nil {
				return nil, 0, fmt.Errorf("aliyunomni preview: bad audio base64: %w", decErr)
			}
			buf = append(buf, pcm...)
		case "response.done", "session.finished":
			return buf, previewOutputSampleRate, nil
		case "error":
			var msg wireError
			_ = json.Unmarshal(raw, &msg)
			text := "unknown error"
			if msg.Error != nil {
				if msg.Error.Message != "" {
					text = msg.Error.Message
				} else if msg.Error.Code != "" {
					text = msg.Error.Code
				}
			}
			return nil, 0, fmt.Errorf("aliyunomni preview: server error: %s", text)
		}
	}
	if len(buf) == 0 {
		return nil, 0, fmt.Errorf("aliyunomni preview: no audio received")
	}
	return buf, previewOutputSampleRate, nil
}

func writePreviewJSON(conn *websocket.Conn, event map[string]any) error {
	if event["event_id"] == nil {
		event["event_id"] = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}
	buf, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, buf)
}

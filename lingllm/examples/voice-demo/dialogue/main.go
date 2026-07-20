// Dialog WebSocket server: receives voice-plane events and streams LLM replies.
//
// Usage:
//
//	export OPENAI_API_KEY=sk-...
//	go run ./examples/voice-demo/dialogue -addr :8082 -provider openai -model gpt-4o-mini
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/anthropic"
	_ "github.com/LingByte/lingllm/protocol/ollama"
	_ "github.com/LingByte/lingllm/protocol/openai"
	_ "github.com/LingByte/lingllm/protocol/response"
	"github.com/LingByte/lingllm/protocol/voice/dialog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	addr := flag.String("addr", ":8082", "listen address")
	provider := flag.String("provider", "openai", "LLM provider (openai, anthropic, ollama, openai-response)")
	model := flag.String("model", "gpt-4o-mini", "LLM model name")
	apiKey := flag.String("apikey", "", "API key (or OPENAI_API_KEY / ANTHROPIC_API_KEY env)")
	baseURL := flag.String("base_url", "", "optional provider base URL")
	systemPrompt := flag.String("system", "你是语音助手。每次只用1-2句口语化中文回答，总长不超过50字。不要列表、不要 markdown。", "system prompt")
	maxTokens := flag.Int("max_tokens", 80, "max LLM completion tokens per turn (lower = faster voice replies)")

	// 降噪器配置
	denoiserConfig := RegisterDenoiserFlags()
	flag.Parse()

	// 记录降噪器配置
	denoiserConfig.LogConfig()

	key := *apiKey
	if key == "" {
		key = osGetenv("OPENAI_API_KEY", "ANTHROPIC_API_KEY")
	}

	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderType(*provider),
		APIKey:   key,
		BaseURL:  *baseURL,
	})
	if err != nil {
		log.Fatalf("llm client: %v", err)
	}

	// 创建降噪器
	denoiser, err := denoiserConfig.CreateDenoiser()
	if err != nil {
		log.Fatalf("create denoiser: %v", err)
	}

	hub := &dialogHub{
		client:    client,
		model:     *model,
		system:    *systemPrompt,
		maxTokens: *maxTokens,
		denoiser:  denoiser,
		calls:     make(map[string]*callState),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/dialog", hub.handleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v\n\nPort busy? Try another: -addr :8082\nOr find the holder: lsof -nP -iTCP%s -sTCP:LISTEN",
			*addr, err, portFromAddr(*addr))
	}
	log.Printf("dialog server listening on %s (ws://localhost%s/ws/dialog)", *addr, *addr)
	log.Fatal((&http.Server{Handler: mux}).Serve(ln))
}

func portFromAddr(addr string) string {
	if i := strings.LastIndex(addr, ":"); i >= 0 && i < len(addr)-1 {
		return addr[i+1:]
	}
	return addr
}

type dialogHub struct {
	client    protocol.ChatModel
	model     string
	system    string
	maxTokens int
	denoiser  interface{} // ASR 降噪器组件

	mu    sync.Mutex
	calls map[string]*callState
}

type callState struct {
	history []protocol.Message
	cancel  context.CancelFunc
}

func (h *dialogHub) handleWS(w http.ResponseWriter, r *http.Request) {
	callID := strings.TrimSpace(r.URL.Query().Get("call_id"))
	if callID == "" {
		http.Error(w, "missing call_id", http.StatusBadRequest)
		return
	}

	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	log.Printf("[dialog] ws connected call_id=%s remote=%s", callID, r.RemoteAddr)

	h.mu.Lock()
	h.calls[callID] = &callState{
		history: []protocol.Message{protocol.SystemMessage(h.system)},
	}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		if st := h.calls[callID]; st != nil && st.cancel != nil {
			st.cancel()
		}
		delete(h.calls, callID)
		h.mu.Unlock()
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var ev dialog.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			continue
		}
		if ev.CallID == "" {
			ev.CallID = callID
		}
		switch ev.Type {
		case dialog.EvCallStarted:
			log.Printf("[dialog] call.started call_id=%s from=%s codec=%s", ev.CallID, ev.From, ev.Codec)
		case dialog.EvASRFinal:
			log.Printf("[dialog] asr.final call_id=%s text=%q", ev.CallID, ev.Text)
			text := strings.TrimSpace(ev.Text)
			if text == "" {
				continue
			}
			go h.handleTurn(conn, callID, text)
		case dialog.EvTTSInterrupt:
			log.Printf("[dialog] tts.interrupt call_id=%s", ev.CallID)
			h.cancelTurn(callID)
		case dialog.EvCallEnded:
			log.Printf("[dialog] call.ended call_id=%s reason=%s", ev.CallID, ev.Reason)
			return
		}
	}
}

func (h *dialogHub) cancelTurn(callID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if st := h.calls[callID]; st != nil && st.cancel != nil {
		st.cancel()
		st.cancel = nil
	}
}

func (h *dialogHub) handleTurn(conn *websocket.Conn, callID, userText string) {
	h.cancelTurn(callID)
	// Stop in-flight TTS from the previous turn before streaming a new reply.
	_ = writeCommand(conn, dialog.Command{
		Type:   dialog.CmdTTSInterrupt,
		CallID: callID,
	})

	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	st := h.calls[callID]
	if st == nil {
		h.mu.Unlock()
		cancel()
		return
	}
	st.cancel = cancel
	st.history = append(st.history, protocol.UserMessage(userText))
	messages := append([]protocol.Message(nil), st.history...)
	h.mu.Unlock()
	defer cancel()

	utterID := uuid.NewString()
	start := time.Now()
	firstToken := time.Duration(-1)

	stream, err := h.client.StreamChat(ctx, protocol.ChatRequest{
		Model:     h.model,
		Messages:  messages,
		MaxTokens: h.maxTokens,
	})
	if err != nil {
		log.Printf("call=%s stream: %v", callID, err)
		return
	}
	defer stream.Close()

	var full strings.Builder
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("call=%s recv: %v", callID, err)
			return
		}
		if chunk.Delta == "" {
			continue
		}
		if firstToken < 0 {
			firstToken = time.Since(start)
			log.Printf("[dialog] call=%s first_llm_chunk=%s utterance=%s", callID, firstToken.Round(time.Millisecond), utterID)
		}
		full.WriteString(chunk.Delta)
		if err := writeCommand(conn, dialog.Command{
			Type:        dialog.CmdTTSStream,
			CallID:      callID,
			UtteranceID: utterID,
			Text:        chunk.Delta,
			Meta: &dialog.CommandMeta{
				LLMModel:   h.model,
				LLMFirstMs: int(firstToken.Milliseconds()),
				UserText:   userText,
			},
		}); err != nil {
			return
		}
	}

	wall := time.Since(start)
	exutil.LogTurn("dialog", callID, stream, start, firstToken)
	assistant := strings.TrimSpace(full.String())
	if assistant != "" {
		h.mu.Lock()
		if st := h.calls[callID]; st != nil {
			st.history = append(st.history, protocol.AssistantMessage(assistant))
		}
		h.mu.Unlock()
	}

	_ = writeCommand(conn, dialog.Command{
		Type:        dialog.CmdTTSStream,
		CallID:      callID,
		UtteranceID: utterID,
		StreamEnd:   true,
		Meta: &dialog.CommandMeta{
			LLMModel:   h.model,
			LLMFirstMs: int(firstToken.Milliseconds()),
			LLMWallMs:  int(wall.Milliseconds()),
			UserText:   userText,
		},
	})
	log.Printf("[dialog] call=%s turn_done user=%q llm_ttft=%s llm_wall=%s chars=%d",
		callID, userText, firstToken.Round(time.Millisecond), wall.Round(time.Millisecond), full.Len())
}

func writeCommand(conn *websocket.Conn, cmd dialog.Command) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, data)
}

func osGetenv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

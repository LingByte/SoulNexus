// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// xiaozhi-client is a self-contained AI-conversation demo for a running
// VoiceServer instance. It simulates one full round-trip:
//
//	YOU (text)                                          ┐
//	  └──[ tencent TTS via TTS_* env vars ]──> PCM16 ───┤
//	                                                    │
//	YOUR voice (PCM16 16k mono) ──[ xiaozhi WS ]────────┼──> VoiceServer
//	                                                    │       │
//	                                       VoiceServer ─┴───────┘
//	                                            │
//	                                            └─[ -dialog-ws ]─> dialog plane
//	                                            │
//	                                            ◄─ dialog reply (LLM text)
//	                                            │
//	                                            └─[ tencent TTS ]─> PCM16
//	                                            │
//	  AI voice ◄──[ xiaozhi WS, binary frames ]─┘
//	  AI text  ◄──[ xiaozhi WS, "stt"/"tts" json ]
//
// What this CLI does end-to-end:
//
//  1. Synthesizes the user-side prompt (-text, default "你好，请简单介绍一下你自己")
//     into 16 kHz mono PCM16 using the same Tencent TTS env vars the server
//     uses. No microphone, no .wav file, no ffmpeg required — just env vars.
//  2. Dials the xiaozhi WS endpoint (-url).
//  3. hello{format=pcm, sr=16000} → waits for server hello reply.
//  4. listen{state:start, mode:manual} → streams the synthesized PCM at
//     real-time pace (60 ms frames) → listen{state:stop}.
//  5. Reads back the server's text frames (stt/tts state) and the binary
//     PCM frames carrying the AI's spoken reply.
//  6. Prints the full transcript: "YOU said …", "AI replied …", and writes
//     the AI voice to a WAV file you can play.
//
// If you'd rather drive the call with your microphone, use the WebRTC
// browser demo at http://<server>:<http-port>/webrtc/v1/demo — same
// dialog plane, real audio in/out, no setup beyond clicking start.
//
// Required env vars (same as voiceserver itself):
//
//	TTS_APPID, TTS_SECRET_ID, TTS_SECRET_KEY    (Tencent TTS credentials)
//
// Usage:
//
//	go run ./examples/voice/xiaozhi-client \
//	    -url ws://127.0.0.1:7080/xiaozhi/v1/ \
//	    -text "你好，请介绍一下你自己" \
//	    -out ai-reply.wav
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

func main() {
	url := flag.String("url", "ws://127.0.0.1:7080/xiaozhi/v1/", "cmd/voice xiaozhi WebSocket endpoint")
	text := flag.String("text", "你好，请简单介绍一下你自己", "What you 'say' to the AI. Synthesized into your voice via TTS env vars.")
	in := flag.String("in", "", "(optional) Skip TTS and use this file as the user audio. Must be 16 kHz mono PCM16-LE WAV.")
	out := flag.String("out", "ai-reply.wav", "Where to write the AI's spoken reply as a WAV file")
	deviceID := flag.String("device", "demo-device-001", "X-Device-Id header for the WS upgrade")
	frameMs := flag.Int("frame-ms", 60, "send frame size in milliseconds (xiaozhi default is 60)")
	listenAfter := flag.Duration("listen-after", 12*time.Second, "how long to wait for the AI's spoken reply after sending the audio")
	flag.Parse()

	// Best-effort .env load so `go run` from the repo root picks up the
	// same TTS_* / TTS_APPID values the server itself uses.
	_ = godotenv.Overload(".env")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pcm, err := loadOrSynthesizeUserVoice(ctx, *in, *text)
	if err != nil {
		log.Fatalf("user voice: %v", err)
	}
	durMs := len(pcm) * 1000 / (16000 * 2)
	if *in != "" {
		log.Printf("YOU (from %s, %d ms of PCM)", *in, durMs)
	} else {
		log.Printf("YOU said: %q (synthesized to %d ms of PCM)", *text, durMs)
	}

	// --- WS dial ----------------------------------------------------------
	hdr := http.Header{}
	hdr.Set("X-Device-Id", *deviceID)
	hdr.Set("X-Client-Id", "xiaozhi-cli/1.0")
	dial := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dial.DialContext(ctx, *url, hdr)
	if err != nil {
		log.Fatalf("dial %s: %v", *url, err)
	}
	defer conn.Close()
	log.Printf("connected to %s", *url)

	// --- read loop --------------------------------------------------------
	var (
		mu       sync.Mutex
		ttsBuf   bytes.Buffer
		ttsCodec = "pcm"
		aiText   string
	)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			t, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			switch t {
			case websocket.TextMessage:
				handleServerText(raw, &ttsCodec, &aiText, &mu)
			case websocket.BinaryMessage:
				mu.Lock()
				ttsBuf.Write(raw)
				mu.Unlock()
			}
		}
	}()

	// --- protocol: hello → listen:start → audio → listen:stop -------------
	hello := map[string]any{
		"type":      "hello",
		"version":   1,
		"transport": "websocket",
		"audio_params": map[string]any{
			"format":         "pcm",
			"sample_rate":    16000,
			"channels":       1,
			"frame_duration": *frameMs,
			"bit_depth":      16,
		},
	}
	if err := writeJSON(conn, hello); err != nil {
		log.Fatalf("send hello: %v", err)
	}
	if err := writeJSON(conn, map[string]any{"type": "listen", "state": "start", "mode": "manual"}); err != nil {
		log.Fatalf("send listen:start: %v", err)
	}

	// Stream PCM at real-time pace — what an ESP32 device would do.
	bytesPerFrame := 16000 * 2 * (*frameMs) / 1000 // 16 kHz × 2 bytes × ms/1000
	frameDur := time.Duration(*frameMs) * time.Millisecond
	tick := time.NewTicker(frameDur)
	defer tick.Stop()
	for off := 0; off < len(pcm); off += bytesPerFrame {
		end := off + bytesPerFrame
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, pcm[off:end]); err != nil {
			log.Fatalf("send pcm: %v", err)
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
			break
		}
	}
	if err := writeJSON(conn, map[string]any{"type": "listen", "state": "stop"}); err != nil {
		log.Fatalf("send listen:stop: %v", err)
	}

	// --- wait for AI reply, then close ------------------------------------
	log.Printf("waiting up to %v for AI reply…", *listenAfter)
	select {
	case <-time.After(*listenAfter):
	case <-ctx.Done():
	}
	_ = conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "demo done"),
		time.Now().Add(time.Second))
	conn.Close()
	<-readDone

	// --- summary ----------------------------------------------------------
	mu.Lock()
	tts := ttsBuf.Bytes()
	finalAI := aiText
	mu.Unlock()
	fmt.Println()
	fmt.Println("─── conversation summary ──────────────────────────────────")
	if *in != "" {
		fmt.Printf("YOU  (audio): %s\n", *in)
	} else {
		fmt.Printf("YOU  said:    %s\n", *text)
	}
	if finalAI != "" {
		fmt.Printf("AI   replied: %s\n", finalAI)
	} else {
		fmt.Printf("AI   replied: (no text frame received — check -dialog-ws backend)\n")
	}
	if len(tts) > 0 {
		if ttsCodec != "pcm" {
			log.Printf("WARN: server replied with %s frames; this CLI only decodes PCM. Saving raw bytes.", ttsCodec)
		}
		if err := writeWAV(*out, tts, 16000, 1); err != nil {
			log.Fatalf("write %s: %v", *out, err)
		}
		fmt.Printf("AI voice:    saved %d ms to %s — open in any player\n", len(tts)*1000/(16000*2), *out)
	} else {
		fmt.Printf("AI voice:    (no audio frames received)\n")
	}
	fmt.Println("───────────────────────────────────────────────────────────")
}

// loadOrSynthesizeUserVoice produces the PCM16 LE @ 16 kHz mono "user
// voice" payload. When `in` is non-empty we treat it as a WAV file and
// load it directly; otherwise we synthesize `text` via the same
// Tencent TTS the server uses, so the demo is self-contained as long
// as TTS_* env vars are set.
func loadOrSynthesizeUserVoice(ctx context.Context, in, text string) ([]byte, error) {
	if strings.TrimSpace(in) != "" {
		pcm, sr, err := loadPCM(in)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", in, err)
		}
		if sr != 16000 {
			return nil, fmt.Errorf("input %s is %d Hz; need 16000 Hz mono PCM16", in, sr)
		}
		return pcm, nil
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("nothing to send: pass -text or -in")
	}
	appID := strings.TrimSpace(os.Getenv("TTS_APPID"))
	sid := strings.TrimSpace(os.Getenv("TTS_SECRET_ID"))
	skey := strings.TrimSpace(os.Getenv("TTS_SECRET_KEY"))
	if appID == "" || sid == "" || skey == "" {
		return nil, fmt.Errorf("TTS_APPID / TTS_SECRET_ID / TTS_SECRET_KEY must be set to synthesize the user voice (or pass -in)")
	}
	const sampleRate = 16000
	var voice int64 = 101001 // a different voice than the server's reply (101007)
	cfg := synthesizer.NewQcloudTTSConfig(appID, sid, skey, voice, "pcm", sampleRate)
	svc := synthesizer.NewQCloudService(cfg)
	h := &collectHandler{}
	if err := svc.Synthesize(ctx, h, text); err != nil {
		return nil, fmt.Errorf("tencent TTS: %w", err)
	}
	if len(h.buf) == 0 {
		return nil, fmt.Errorf("tencent TTS returned empty PCM")
	}
	return h.buf, nil
}

// collectHandler accumulates streaming TTS PCM into a single buffer.
// Implements synthesizer.SynthesisHandler — the upstream interface
// is callback-based to support streaming, but for the demo we just
// want the full clip in memory.
type collectHandler struct{ buf []byte }

func (h *collectHandler) OnMessage(b []byte)                          { h.buf = append(h.buf, b...) }
func (h *collectHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}

// handleServerText processes one xiaozhi text frame. Captures the
// negotiated audio format, the latest stt text the server pushes, and
// logs important state transitions.
func handleServerText(raw []byte, ttsCodec, aiText *string, mu *sync.Mutex) {
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		log.Printf("← text (unparseable): %s", raw)
		return
	}
	t, _ := generic["type"].(string)
	switch t {
	case "hello":
		if ap, ok := generic["audio_params"].(map[string]any); ok {
			if f, ok := ap["format"].(string); ok && f != "" {
				mu.Lock()
				*ttsCodec = f
				mu.Unlock()
			}
		}
		log.Printf("← server hello: %s", raw)
	case "stt":
		if s, ok := generic["text"].(string); ok && strings.TrimSpace(s) != "" {
			log.Printf("← VoiceServer ASR (what the server heard you say): %q", s)
		}
	case "tts":
		state, _ := generic["state"].(string)
		if state == "start" {
			if s, ok := generic["text"].(string); ok && s != "" {
				mu.Lock()
				*aiText = s
				mu.Unlock()
				log.Printf("← AI text (from dialog plane): %q", s)
			} else {
				log.Printf("← tts:start (binary PCM frames will follow)")
			}
		} else {
			log.Printf("← tts:%s", state)
		}
	default:
		log.Printf("← %s", raw)
	}
}

func writeJSON(conn *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, b)
}

// loadPCM reads a WAV file (PCM16 LE) and returns its data section + sample rate.
// Falls back to "raw PCM16 LE @ 16 kHz" when the file doesn't start with RIFF.
func loadPCM(path string) ([]byte, int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	if len(b) < 44 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return b, 16000, nil
	}
	channels := binary.LittleEndian.Uint16(b[22:24])
	sr := binary.LittleEndian.Uint32(b[24:28])
	bits := binary.LittleEndian.Uint16(b[34:36])
	if channels != 1 || bits != 16 {
		return nil, 0, fmt.Errorf("WAV must be PCM16 LE mono; got channels=%d bits=%d", channels, bits)
	}
	off := 12
	for off+8 <= len(b) {
		id := string(b[off : off+4])
		sz := int(binary.LittleEndian.Uint32(b[off+4 : off+8]))
		if id == "data" {
			end := off + 8 + sz
			if end > len(b) {
				end = len(b)
			}
			return b[off+8 : end], int(sr), nil
		}
		off += 8 + sz
	}
	return nil, 0, fmt.Errorf("no data chunk in WAV")
}

// writeWAV writes a PCM16 LE buffer as a canonical RIFF/WAVE file.
func writeWAV(path string, pcm []byte, sampleRate, channels int) error {
	const (
		bitsPerSample = 16
		fmtChunkSize  = 16
		audioFormat   = 1
	)
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	totalSize := 4 + (8 + fmtChunkSize) + (8 + len(pcm))
	w := &bytes.Buffer{}
	w.WriteString("RIFF")
	_ = binary.Write(w, binary.LittleEndian, uint32(totalSize))
	w.WriteString("WAVE")
	w.WriteString("fmt ")
	_ = binary.Write(w, binary.LittleEndian, uint32(fmtChunkSize))
	_ = binary.Write(w, binary.LittleEndian, uint16(audioFormat))
	_ = binary.Write(w, binary.LittleEndian, uint16(channels))
	_ = binary.Write(w, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(w, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(w, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(w, binary.LittleEndian, uint16(bitsPerSample))
	w.WriteString("data")
	_ = binary.Write(w, binary.LittleEndian, uint32(len(pcm)))
	w.Write(pcm)
	return os.WriteFile(path, w.Bytes(), 0o644)
}

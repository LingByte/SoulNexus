// Voice media server: WebRTC browser client + xiaozhi WebSocket.
//
// Xiaozhi mode (pipeline or realtime) is selected by the client in the hello message.
// If not specified, defaults to pipeline.
//
// Pipeline mode (default):
//
//	go run ./examples/voice-demo/dialogue   # terminal 1
//	go run -tags opus ./examples/voice-demo/voice      # terminal 2 (Opus required for WebRTC)
//	open http://localhost:8080/
//
// Realtime mode (browser web1 or xiaozhi device):
//
//	export REALTIME_CONFIG_JSON='{"provider":"aliyun_omni","api_key":"..."}'
//	go run ./examples/voice-demo/realtime
//	open http://localhost:8080/
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/LingByte/lingllm/examples/voice-demo/voiceutil"
	"github.com/LingByte/lingllm/media/encoder"
	"github.com/LingByte/lingllm/protocol/voice/webrtc"
	"github.com/LingByte/lingllm/protocol/voice/xiaozhi"
	_ "github.com/LingByte/lingllm/realtime/aliyunomni"
	_ "github.com/LingByte/lingllm/realtime/volcdialogue"
)

type voiceServer struct {
	wrtc       *webrtc.Server
	xz         *xiaozhi.Server
	staticRoot string
}

func (s *voiceServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/health" && r.Method == http.MethodGet:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	case path == "/webrtc/v1/offer" && (r.Method == http.MethodPost || r.Method == http.MethodOptions):
		s.wrtc.HandleOffer(w, r)
	case path == "/webrtc/v1/hangup" && (r.Method == http.MethodPost || r.Method == http.MethodOptions):
		s.wrtc.HandleHangup(w, r)
	case strings.HasPrefix(path, "/xiaozhi/v1/"):
		s.xz.Handle(w, r)
	case r.Method == http.MethodGet:
		s.serveStatic(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *voiceServer) serveStatic(w http.ResponseWriter, r *http.Request) {
	index := filepath.Join(s.staticRoot, "index.html")
	if r.URL.Path == "/" {
		if _, err := os.Stat(index); err == nil {
			http.ServeFile(w, r, index)
			return
		}
	}
	http.FileServer(http.Dir(s.staticRoot)).ServeHTTP(w, r)
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dialogURL := flag.String("dialog", "ws://127.0.0.1:8082/ws/dialog", "dialog WebSocket URL")
	webDir := flag.String("web", "", "static web root (default: examples/voice-demo/web)")
	flag.Parse()

	if !encoder.HasCodec("opus") {
		log.Fatalf("Opus codec not compiled in. WebRTC requires libopus — run with:\n  go run -tags opus ./examples/voice-demo/voice ...")
	}

	factory, err := voiceutil.NewFactory()
	if err != nil {
		log.Fatalf("factory: %v", err)
	}

	wrtc, err := webrtc.NewServer(webrtc.ServerConfig{
		SessionFactory: factory,
		DialogWSURL:    *dialogURL,
		OnSessionStart: func(_ context.Context, callID, peer string) {
			log.Printf("[voice] webrtc call=%s started peer=%s", callID, peer)
		},
		OnSessionEnd: func(_ context.Context, callID, reason string) {
			log.Printf("[voice] webrtc call=%s ended: %s", callID, reason)
		},
	})
	if err != nil {
		log.Fatalf("webrtc: %v", err)
	}

	xzCfg := xiaozhi.ServerConfig{
		DialogWSURL:     *dialogURL,
		SessionFactory:  factory,
		RealtimeFactory: factory.RealtimeAgentFactory(),
	}
	xz, err := xiaozhi.NewServer(xzCfg)
	if err != nil {
		log.Fatalf("xiaozhi: %v", err)
	}

	staticRoot := *webDir
	if staticRoot == "" {
		staticRoot = defaultWebDir()
	}

	srv := &voiceServer{wrtc: wrtc, xz: xz, staticRoot: staticRoot}

	log.Printf("voice server on http://localhost%s", *addr)
	log.Printf("  web:    http://localhost%s/", *addr)
	log.Printf("  webrtc: POST http://localhost%s/webrtc/v1/offer (needs pipeline + ASR/TTS + dialogue)", *addr)
	log.Printf("  xiaozhi ws: ws://localhost%s/xiaozhi/v1/ (mode selected by client in hello message)", *addr)
	log.Printf("  dialog: %s", *dialogURL)
	logStartupWarnings()
	log.Fatal(http.ListenAndServe(*addr, srv))
}

func logStartupWarnings() {
	if strings.TrimSpace(os.Getenv("ASR_CONFIG_JSON")) == "" &&
		strings.TrimSpace(os.Getenv("ASR_APP_ID")) == "" {
		log.Printf("WARN: ASR not configured — WebRTC calls will fail after connect. Set ASR_CONFIG_JSON (or ASR_APP_ID/ASR_TOKEN).")
	}
	if strings.TrimSpace(os.Getenv("TTS_CONFIG_JSON")) == "" &&
		strings.TrimSpace(os.Getenv("TTS_API_KEY")) == "" &&
		strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
		log.Printf("WARN: TTS not configured — WebRTC calls will fail after connect. Set TTS_CONFIG_JSON (or TTS_API_KEY).")
	}
}

func defaultWebDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join(".", "examples", "voice-demo", "web")
	}
	return filepath.Join(filepath.Dir(file), "..", "web")
}

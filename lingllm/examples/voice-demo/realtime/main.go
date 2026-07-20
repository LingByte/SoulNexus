// Realtime voice demo: browser (web1) or xiaozhi device → xiaozhi WS → multimodal agent.
//
// Clients must specify "mode": "realtime" in the hello message.
//
// Aliyun Qwen-Omni:
//
//	export REALTIME_CONFIG_JSON='{"provider":"aliyun_omni","api_key":"sk-...","model":"qwen3.5-omni-flash-realtime-2026-03-15"}'
//	export REALTIME_VOICE=Cherry
//	go run ./examples/voice-demo/realtime
//	open http://localhost:8080/
//
// Volcengine realtime dialogue:
//
//	export REALTIME_CONFIG_JSON='{"provider":"volcengine_dialogue","appId":"...","accessKey":"..."}'
//	go run ./examples/voice-demo/realtime
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
	"github.com/LingByte/lingllm/protocol/voice/xiaozhi"
	"github.com/LingByte/lingllm/realtime"
	_ "github.com/LingByte/lingllm/realtime/aliyunomni"
	_ "github.com/LingByte/lingllm/realtime/volcdialogue"
)

type realtimeServer struct {
	xz         *xiaozhi.Server
	staticRoot string
}

func (s *realtimeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/health" && r.Method == http.MethodGet:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	case strings.HasPrefix(path, "/xiaozhi/v1/"):
		s.xz.Handle(w, r)
	case r.Method == http.MethodGet:
		s.serveStatic(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *realtimeServer) serveStatic(w http.ResponseWriter, r *http.Request) {
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
	webDir := flag.String("web", "", "static web root (default: examples/voice-demo/web1)")
	flag.Parse()

	factory, err := voiceutil.NewFactory()
	if err != nil {
		log.Fatalf("factory: %v", err)
	}

	xz, err := xiaozhi.NewServer(xiaozhi.ServerConfig{
		SessionFactory:  factory,
		RealtimeFactory: loggingRealtimeFactory{inner: factory.RealtimeAgentFactory()},
		DialogWSURL:     "ws://127.0.0.1:8082/ws/dialog",
		OnSessionStart: func(_ context.Context, callID, deviceID string) {
			log.Printf("[realtime] session start call=%s device=%s", callID, deviceID)
		},
		OnSessionEnd: func(_ context.Context, callID, reason string) {
			log.Printf("[realtime] session end call=%s reason=%s", callID, reason)
		},
	})
	if err != nil {
		log.Fatalf("xiaozhi: %v", err)
	}

	staticRoot := *webDir
	if staticRoot == "" {
		staticRoot = defaultWebDir()
	}

	srv := &realtimeServer{xz: xz, staticRoot: staticRoot}

	log.Printf("realtime demo on http://localhost%s", *addr)
	log.Printf("  web:    http://localhost%s/  (xiaozhi WS browser client)", *addr)
	log.Printf("  ws:     ws://localhost%s/xiaozhi/v1/", *addr)
	logRealtimeWarnings()
	log.Fatal(http.ListenAndServe(*addr, srv))
}

func defaultWebDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join(".", "examples", "voice-demo", "web1")
	}
	return filepath.Join(filepath.Dir(file), "..", "web1")
}

func logRealtimeWarnings() {
	if strings.TrimSpace(os.Getenv("REALTIME_CONFIG_JSON")) == "" {
		log.Printf("WARN: REALTIME_CONFIG_JSON is empty — sessions will fail at hello.")
		log.Printf("      Example (Aliyun): {\"provider\":\"aliyun_omni\",\"api_key\":\"sk-...\"}")
		log.Printf("      Example (Volc):   {\"provider\":\"volcengine_dialogue\",\"appId\":\"...\",\"accessKey\":\"...\"}")
	}
}

type loggingRealtimeFactory struct {
	inner xiaozhi.RealtimeAgentFactory
}

func (l loggingRealtimeFactory) NewAgent(ctx context.Context, callID string, onEvent func(realtime.Event)) (realtime.Agent, int, int, error) {
	return l.inner.NewAgent(ctx, callID, func(ev realtime.Event) {
		switch ev.Type {
		case realtime.EventUserTranscript, realtime.EventAssistantText, realtime.EventError, realtime.EventSessionClose:
			log.Printf("[realtime] call=%s event=%s text=%q err=%v fatal=%v", callID, ev.Type, ev.Text, ev.Err, ev.Fatal)
		case realtime.EventUserSpeechStarted, realtime.EventUserSpeechEnded, realtime.EventAssistantTurnEnd, realtime.EventSessionOpen:
			log.Printf("[realtime] call=%s event=%s", callID, ev.Type)
		case realtime.EventAssistantAudio:
			log.Printf("[realtime] call=%s event=%s bytes=%d", callID, ev.Type, len(ev.AudioPC))
		}
		if onEvent != nil {
			onEvent(ev)
		}
	})
}

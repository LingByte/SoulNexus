package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/gorilla/websocket"
)

// ---- minimal fakes ----

type fakeASR struct{ tr recognizer.TranscribeResult }

func (f *fakeASR) Init(tr recognizer.TranscribeResult, _ recognizer.ProcessError) {
	f.tr = tr
}
func (f *fakeASR) Vendor() string              { return "fake" }
func (f *fakeASR) ConnAndReceive(string) error { return nil }
func (f *fakeASR) Activity() bool              { return true }
func (f *fakeASR) RestartClient()              {}
func (f *fakeASR) SendAudioBytes([]byte) error { return nil }
func (f *fakeASR) SendEnd() error              { return nil }
func (f *fakeASR) StopConn() error             { return nil }

func TestClient_EventFlowAndCommandDispatch(t *testing.T) {
	// Dialog WS server: capture events, send back commands.
	var (
		mu          sync.Mutex
		gotEvents   []Event
		serverConn  *websocket.Conn
		serverReady = make(chan struct{})
	)
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		mu.Lock()
		serverConn = c
		mu.Unlock()
		close(serverReady)
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
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/?x=1"

	// Build a real voice.Attached using a fake ASR + a tts.ServiceFunc whose
	// Sink target is a no-op, so Speak() actually runs to completion.
	fa := &fakeASR{}
	att := &voice.Attached{}
	// Manually wire: mimic voice.Attach but without a MediaLeg.
	asrPipe, err := asr.New(asr.Options{ASR: fa, SampleRate: 16000, MinFeedBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	att.ASR = asrPipe

	var spoken atomic.Int32
	ttsPipe, err := tts.New(tts.Config{
		Service: tts.ServiceFunc(func(ctx context.Context, text string, on func([]byte) error) error {
			// Emit one frame so the pipeline reframes and calls Sink.
			return on(make([]byte, 1280))
		}),
		InputSampleRate: 16000,
		Sink:            func([]byte) error { spoken.Add(1); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	ttsPipe.Start(context.Background())
	att.TTS = ttsPipe

	hangupCh := make(chan string, 1)
	cli, err := NewClient(ClientConfig{
		URL:      wsURL,
		Attached: att,
		CallID:   "call-xyz",
		OnHangup: func(reason string) { hangupCh <- reason },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.Start(context.Background(), StartMeta{From: "alice", To: "bob", Codec: "pcma", PCMHz: 8000}); err != nil {
		t.Fatal(err)
	}
	defer cli.Close("test")

	<-serverReady

	// Force lazy-connect of the asr pipeline so the recognizer's Init runs
	// (which captures fa.tr → our delivery hook into Pipeline → gateway events).
	_ = asrPipe.ProcessPCM(context.Background(), make([]byte, 4))
	if fa.tr == nil {
		t.Fatal("asr pipeline did not connect")
	}
	fa.tr("hi", false, 0, "")
	fa.tr("hi there", true, 0, "")

	// Server sends a tts.speak command back.
	mu.Lock()
	sc := serverConn
	mu.Unlock()
	if sc == nil {
		t.Fatal("no server conn")
	}
	cmd := Command{Type: CmdTTSSpeak, CallID: "call-xyz", Text: "你好"}
	data, _ := json.Marshal(cmd)
	_ = sc.WriteMessage(websocket.TextMessage, data)

	// Wait until TTS frames arrive.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && spoken.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if spoken.Load() == 0 {
		t.Fatal("TTS speak was not dispatched")
	}

	// Server sends hangup.
	hcmd, _ := json.Marshal(Command{Type: CmdHangup, CallID: "call-xyz", Reason: "user"})
	_ = sc.WriteMessage(websocket.TextMessage, hcmd)
	select {
	case r := <-hangupCh:
		if r != "user" {
			t.Errorf("hangup reason=%q", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hangup not delivered")
	}

	// PushDTMF / Close should send events without panicking.
	cli.PushDTMF("5", true)

	// Allow the events to be observed by the server.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	hasStart := false
	hasFinal := false
	hasDTMF := false
	for _, ev := range gotEvents {
		switch ev.Type {
		case EvCallStarted:
			hasStart = true
			if ev.From != "alice" || ev.Codec != "pcma" || ev.PCMHz != 8000 {
				t.Errorf("call.started fields: %+v", ev)
			}
		case EvASRFinal:
			if ev.Text == "hi there" {
				hasFinal = true
			}
		case EvDTMF:
			if ev.Digit == "5" && ev.End {
				hasDTMF = true
			}
		}
	}
	if !hasStart || !hasFinal || !hasDTMF {
		t.Errorf("missing events: start=%v final=%v dtmf=%v", hasStart, hasFinal, hasDTMF)
	}
}

func TestClient_RejectsBadConfig(t *testing.T) {
	if _, err := NewClient(ClientConfig{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := NewClient(ClientConfig{URL: "ws://x"}); err == nil {
		t.Fatal("expected error nil-attached")
	}
}

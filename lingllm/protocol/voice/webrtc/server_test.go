package webrtc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/tts"

	pionwebrtc "github.com/pion/webrtc/v4"
)

type stubFactory struct{}

func (stubFactory) NewASR(_ context.Context, _ string) (asr.Engine, int, error) {
	return nil, 16000, errSkipASR
}
func (stubFactory) NewTTS(_ context.Context, _ string) (tts.TTSService, int, error) {
	return nil, 16000, errSkipTTS
}

var (
	errSkipASR = stubErr("asr disabled in test")
	errSkipTTS = stubErr("tts disabled in test")
)

type stubErr string

func (e stubErr) Error() string { return string(e) }

func TestServer_OfferAnswer_ProducesNegotiableSDP(t *testing.T) {
	srv, err := NewServer(ServerConfig{
		SessionFactory: stubFactory{},
		DialogWSURL:    "ws://127.0.0.1:1/never",
	})
	if err != nil {
		t.Fatal(err)
	}

	api, err := BuildAPI(EngineConfig{})
	if err != nil {
		t.Fatal(err)
	}
	offerer, err := api.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer offerer.Close()

	if _, err := offerer.AddTransceiverFromKind(
		pionwebrtc.RTPCodecTypeAudio,
		pionwebrtc.RTPTransceiverInit{Direction: pionwebrtc.RTPTransceiverDirectionSendrecv},
	); err != nil {
		t.Fatal(err)
	}
	offer, err := offerer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	gather := pionwebrtc.GatheringCompletePromise(offerer)
	if err := offerer.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	<-gather
	finalOffer := offerer.LocalDescription()

	httpSrv := httptest.NewServer(http.HandlerFunc(srv.HandleOffer))
	defer httpSrv.Close()

	body, _ := json.Marshal(OfferRequest{SDP: finalOffer.SDP, Type: "offer"})
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var ans SDPMessage
	if err := json.NewDecoder(resp.Body).Decode(&ans); err != nil {
		t.Fatal(err)
	}
	if ans.Type != "answer" || strings.TrimSpace(ans.SDP) == "" {
		t.Fatalf("bad answer: %+v", ans)
	}
	if !strings.Contains(strings.ToLower(ans.SDP), "useinbandfec=1") {
		t.Errorf("answer missing useinbandfec=1:\n%s", ans.SDP)
	}
	if !strings.Contains(strings.ToLower(ans.SDP), "transport-cc") {
		t.Errorf("answer missing transport-cc:\n%s", ans.SDP)
	}
	if ans.CallID == "" {
		t.Error("answer missing call_id")
	}

	if err := offerer.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer, SDP: ans.SDP,
	}); err != nil {
		t.Fatal(err)
	}
	connected := make(chan struct{}, 1)
	var connectedOnce atomic.Bool
	offerer.OnICEConnectionStateChange(func(s pionwebrtc.ICEConnectionState) {
		if (s == pionwebrtc.ICEConnectionStateConnected ||
			s == pionwebrtc.ICEConnectionStateCompleted) &&
			connectedOnce.CompareAndSwap(false, true) {
			connected <- struct{}{}
		}
	})
	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatalf("ICE never connected; offerer state=%s", offerer.ICEConnectionState())
	}
}

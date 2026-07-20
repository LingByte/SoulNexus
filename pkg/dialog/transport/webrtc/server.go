package webrtc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/transport/pcm"
	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/media/encoder"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

const webrtcWireRate = 48000 // Opus wire rate (browser standard)

// OfferRequest is the browser SDP offer.
type OfferRequest struct {
	SessionID string `json:"sessionId"`
	SDP       string `json:"sdp"`
	Type      string `json:"type"`
}

// AnswerResponse is returned after successful negotiation.
type AnswerResponse struct {
	SessionID string `json:"sessionId"`
	SDP       string `json:"sdp"`
	Type      string `json:"type"`
}

type mediaBinding struct {
	sess       *session.Session
	port       *pcm.Port
	lg         engine.Logger
	bridgeRate int
	transcript *transcriptSink

	pumpOnce  sync.Once
	startOnce sync.Once
}

func (b *mediaBinding) startPump(ctx context.Context, tr *webrtc.TrackRemote) {
	if b == nil || tr == nil {
		return
	}
	b.pumpOnce.Do(func() {
		go pumpInbound(ctx, tr, b.port, b.bridgeRate)
	})
}

func (b *mediaBinding) startDialog(ctx context.Context) {
	if b == nil {
		return
	}
	b.startOnce.Do(func() {
		go b.runDialog(ctx)
	})
}

func (b *mediaBinding) runDialog(ctx context.Context) {
	mode := "pipeline"
	env, envOK, _ := tenantcfg.Resolve(context.Background(), b.sess.TenantID, b.sess.CallID)
	if envOK {
		env = session.EffectiveVoiceEnv(b.sess, env)
		mode = string(session.ResolveMode(env))
	}
	if b.transcript != nil {
		b.sess.BindTurnNotify(b.transcript.write, mode, "webrtc")
	}
	if err := b.sess.StartEngine(ctx, b.lg); err != nil {
		if b.lg != nil {
			fields := []engine.Field{engine.F("err", err.Error())}
			if envOK {
				fields = append(fields,
					engine.F("voice_mode", env.VoiceMode),
					engine.F("assistant_id", b.sess.AssistantID),
					engine.F("tenant_id", b.sess.TenantID),
				)
				if hint := tenantcfg.VoiceReadinessReason(env); hint != "" {
					fields = append(fields, engine.F("hint", hint))
				}
			}
			b.lg.Warn("webrtc: dialog engine failed to start", fields...)
		}
		return
	}
	if envOK && mode != string(engine.ModeRealtime) {
		welcome, _ := session.PlayAssistantWelcome(ctx, b.sess.CallID, env, b.bridgeRate, func(pcmData []byte) error {
			return b.port.SendOutputPCM(engine.PCMFrame{Data: pcmData, SampleRate: b.bridgeRate})
		}, false)
		if welcome != "" && b.transcript != nil {
			b.transcript.sendAssistantText(welcome)
		}
	}
}

// NegotiateOffer completes WebRTC negotiation and returns the SDP answer immediately.
// Media pump and dialog engine start after the peer connection is live.
func NegotiateOffer(ctx context.Context, req OfferRequest, lg engine.Logger) (AnswerResponse, error) {
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.SessionID == "" || strings.TrimSpace(req.SDP) == "" || req.Type != "offer" {
		return AnswerResponse{}, errors.New("expected sessionId, sdp, type=offer")
	}
	mgr := session.Default()
	if mgr == nil {
		return AnswerResponse{}, errors.New("voice session manager not initialized")
	}
	sess, ok := mgr.Get(req.SessionID)
	if !ok || sess == nil {
		return AnswerResponse{}, errors.New("unknown session")
	}

	runCtx, runCancel := context.WithCancel(context.Background())

	bridgeRate := sess.SampleRate
	if bridgeRate <= 0 {
		bridgeRate = 16000
	}
	binding := &mediaBinding{
		sess:       sess,
		lg:         lg,
		bridgeRate: bridgeRate,
		transcript: newTranscriptSink(sess.ID),
	}

	// Backup kick if Create-time resolve missed credentials; overlaps ICE.
	if env, ok, _ := tenantcfg.Resolve(context.Background(), sess.TenantID, sess.CallID); ok {
		env = session.EffectiveVoiceEnv(sess, env)
		session.StartAssistantWelcomePrewarm(sess.CallID, env, bridgeRate)
	}

	pc, port, cleanup, err := bindPeer(runCtx, runCancel, binding)
	if err != nil {
		runCancel()
		return AnswerResponse{}, err
	}
	binding.port = port

	// Bind port before SDP negotiation — OnTrack / Connected can fire
	// while ICE is gathering; a late BindPort leaves StartEngine with
	// no port and startOnce prevents retry.
	if err := sess.BindPort(port); err != nil {
		cleanup()
		return AnswerResponse{}, err
	}

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: req.SDP}); err != nil {
		cleanup()
		return AnswerResponse{}, fmt.Errorf("set remote: %w", err)
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		cleanup()
		return AnswerResponse{}, fmt.Errorf("create answer: %w", err)
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		cleanup()
		return AnswerResponse{}, fmt.Errorf("set local: %w", err)
	}
	<-webrtc.GatheringCompletePromise(pc)

	ld := pc.LocalDescription()
	return AnswerResponse{
		SessionID: req.SessionID,
		SDP:       ld.SDP,
		Type:      ld.Type.String(),
	}, nil
}

// HandleOffer completes WebRTC negotiation and starts the dialog engine.
func HandleOffer(w http.ResponseWriter, r *http.Request, lg engine.Logger) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	var req OfferRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	answer, err := NegotiateOffer(r.Context(), req, lg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}

func bindPeer(
	ctx context.Context,
	cancel context.CancelFunc,
	binding *mediaBinding,
) (*webrtc.PeerConnection, *pcm.Port, func(), error) {
	bridgeRate := binding.bridgeRate
	if bridgeRate <= 0 {
		bridgeRate = 16000
		binding.bridgeRate = bridgeRate
	}

	m := &webrtc.MediaEngine{}
	if err := registerOpusCodec(m); err != nil {
		return nil, nil, nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if _, err := pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		_ = pc.Close()
		return nil, nil, nil, fmt.Errorf("add recv transceiver: %w", err)
	}

	// Browser is the SDP offerer and must create the "dialog" data
	// channel; we bind it here when the offer arrives. (Answerer-created
	// channels are not reliably surfaced in all browsers.)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc == nil || binding.transcript == nil {
			return
		}
		if strings.EqualFold(dc.Label(), "dialog") {
			binding.transcript.bindDC(dc)
		}
	})

	txTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: webrtcWireRate, Channels: 2},
		"audio", "dialog-out",
	)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	txSender, err := pc.AddTrack(txTrack)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}
	go drainRTCP(txSender)

	wireCodec := media.CodecConfig{Codec: "opus", SampleRate: webrtcWireRate, Channels: 2, BitDepth: 16, FrameDuration: "20ms"}
	bridgePCM := media.CodecConfig{Codec: "pcm", SampleRate: bridgeRate, BitDepth: 16, Channels: 1, FrameDuration: "20ms"}
	encodeFn, err := encoder.CreateEncode(wireCodec, bridgePCM)
	if err != nil {
		_ = pc.Close()
		return nil, nil, nil, err
	}

	var writeMu sync.Mutex
	// PaceRealtime is false on cascaded TTS so Speak can return before
	// wall-clock playout finishes. voice MediaSession paces RTP; WebRTC
	// WriteSample does not — burst writes flood the browser JB and later
	// phrases go silent. Server-side queue (~6s) absorbs synthesis; the
	// loop WriteSample-paces to the browser. Kick-burst after idle so
	// first audible audio is not delayed waiting for JB fill.
	pacer := newDownlinkPacer(ctx, txTrack, &writeMu, 20*time.Millisecond, defaultPacerDepth)
	port := pcm.NewPort(pcm.Config{
		SessionID:  binding.sess.ID,
		TenantID:   strconv.FormatUint(uint64(binding.sess.TenantID), 10),
		SampleRate: bridgeRate,
	})
	port.OnBargeIn(pacer.Flush)
	port.OutputFn = func(f engine.PCMFrame) error {
		if encodeFn == nil || txTrack == nil {
			return nil
		}
		pcmBridge := f.Data
		inRate := f.SampleRate
		if inRate <= 0 {
			inRate = bridgeRate
		}
		if inRate != bridgeRate {
			if out, err := media.ResamplePCM(pcmBridge, inRate, bridgeRate); err == nil && len(out) > 0 {
				pcmBridge = out
			}
		}
		outPkts, err := encodeFn(&media.AudioPacket{Payload: pcmBridge})
		if err != nil || len(outPkts) == 0 {
			return err
		}
		for _, pkt := range outPkts {
			ap, ok := pkt.(*media.AudioPacket)
			if !ok || ap == nil || len(ap.Payload) == 0 {
				continue
			}
			if err := pacer.Enqueue(ap.Payload); err != nil {
				return err
			}
		}
		return nil
	}

	pc.OnTrack(func(tr *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if tr == nil || tr.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		binding.startPump(ctx, tr)
		binding.startDialog(ctx)
	})

	cleanup := func() {
		cancel()
		pacer.Close()
		_ = pc.Close()
		_ = port.Close()
		binding.sess.Close()
	}
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			binding.startDialog(ctx)
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			cleanup()
		}
	})
	return pc, port, cleanup, nil
}

func drainRTCP(sender *webrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buf); err != nil {
			return
		}
	}
}

func pumpInbound(ctx context.Context, tr *webrtc.TrackRemote, port *pcm.Port, bridgeRate int) {
	decodeFn := inboundDecoder(tr, bridgeRate)
	if decodeFn == nil {
		return
	}
	c := tr.Codec()
	srcSR := int(c.ClockRate)
	if srcSR <= 0 {
		srcSR = webrtcWireRate
	}
	depack := &codecs.OpusPacket{}
	sb := samplebuilder.New(50, depack, uint32(srcSR),
		samplebuilder.WithMaxTimeDelay(200*time.Millisecond))

	for {
		if ctx.Err() != nil {
			return
		}
		_ = tr.SetReadDeadline(time.Now().Add(60 * time.Second))
		pkt, _, err := tr.ReadRTP()
		if err != nil {
			return
		}
		sb.Push(pkt)
		for {
			sample := sb.Pop()
			if sample == nil {
				break
			}
			if len(sample.Data) == 0 {
				continue
			}
			outPkts, err := decodeFn(&media.AudioPacket{Payload: sample.Data})
			if err != nil || len(outPkts) == 0 {
				continue
			}
			ap, ok := outPkts[0].(*media.AudioPacket)
			if !ok || len(ap.Payload) == 0 {
				continue
			}
			_ = port.PushInput(ap.Payload)
		}
	}
}

func inboundDecoder(tr *webrtc.TrackRemote, bridgeRate int) media.EncoderFunc {
	if tr == nil {
		return nil
	}
	bridgePCM := media.CodecConfig{Codec: "pcm", SampleRate: bridgeRate, BitDepth: 16, Channels: 1, FrameDuration: "20ms"}
	c := tr.Codec()
	mime := strings.ToLower(c.MimeType)
	switch {
	case strings.Contains(mime, "opus"):
		srcCh := int(c.Channels)
		if srcCh < 1 {
			srcCh = 2
		}
		wire := media.CodecConfig{
			Codec:                     "opus",
			SampleRate:                int(c.ClockRate),
			Channels:                  srcCh,
			BitDepth:                  16,
			FrameDuration:             "20ms",
			OpusDecodeChannels:        srcCh,
			OpusPCMBridgeDecodeStereo: srcCh >= 2,
		}
		if wire.SampleRate <= 0 {
			wire.SampleRate = webrtcWireRate
		}
		fn, err := encoder.CreateDecode(wire, bridgePCM)
		if err == nil {
			return fn
		}
	case strings.Contains(mime, "pcmu"):
		wire := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, BitDepth: 16, Channels: 1}
		fn, err := encoder.CreateDecode(wire, bridgePCM)
		if err == nil {
			return fn
		}
	case strings.Contains(mime, "pcma"):
		wire := media.CodecConfig{Codec: "pcma", SampleRate: 8000, BitDepth: 16, Channels: 1}
		fn, err := encoder.CreateDecode(wire, bridgePCM)
		if err == nil {
			return fn
		}
	}
	return nil
}

func registerOpusCodec(m *webrtc.MediaEngine) error {
	return m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   webrtcWireRate,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio)
}

// BindPeerForTest exposes bindPeer for unit tests.
func BindPeerForTest(ctx context.Context, sess *session.Session, lg engine.Logger) (*webrtc.PeerConnection, *pcm.Port, func(), error) {
	runCtx, cancel := context.WithCancel(ctx)
	bridgeRate := sess.SampleRate
	if bridgeRate <= 0 {
		bridgeRate = 16000
	}
	binding := &mediaBinding{sess: sess, lg: lg, bridgeRate: bridgeRate, transcript: newTranscriptSink(sess.ID)}
	return bindPeer(runCtx, cancel, binding)
}

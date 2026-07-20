package webrtc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/media/encoder"
	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/dialog"
	"github.com/LingByte/lingllm/protocol/voice/gateway"
	"github.com/LingByte/lingllm/protocol/voice/transport"
	"github.com/pion/rtp/codecs"
	pionwebrtc "github.com/pion/webrtc/v4"
	pionmedia "github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

type session struct {
	cfg           ServerConfig
	pc            *pionwebrtc.PeerConnection
	callID        string
	clientMeta    string
	dialogDialURL string

	localTrack *pionwebrtc.TrackLocalStaticSample
	voiceSess  *dialog.Session
	gw         *gateway.Client
	denoiser   asr.Denoiser

	opusDec  media.EncoderFunc
	opusEnc  media.EncoderFunc
	bridgeSR int

	mu        sync.Mutex
	closeOnce sync.Once
	closed    atomic.Bool
	done      chan struct{}

	sessCtx    context.Context
	sessCancel context.CancelFunc
}

func newSession(ctx context.Context, cfg ServerConfig, api *pionwebrtc.API, offer OfferRequest) (*session, SDPMessage, error) {
	callID := fmt.Sprintf("%s-%d", cfg.CallIDPrefix, time.Now().UnixNano())

	ps := strings.TrimSpace(string(offer.Payload))
	var merged string
	var err error
	if len(offer.Payload) > 0 && ps != "" && ps != "null" {
		merged, err = gateway.MergeDialogPayloadQuery(cfg.DialogWSURL, offer.Payload)
	} else {
		merged, err = gateway.MergeDialogQueryParams(cfg.DialogWSURL, offer.ApiKey, offer.ApiSecret, offer.AgentId)
	}
	if err != nil {
		return nil, SDPMessage{}, fmt.Errorf("webrtc: merge dialog URL: %w", err)
	}

	pc, err := api.NewPeerConnection(pionwebrtc.Configuration{
		ICEServers:   cfg.Engine.ICEServers,
		BundlePolicy: pionwebrtc.BundlePolicyMaxBundle,
	})
	if err != nil {
		return nil, SDPMessage{}, fmt.Errorf("webrtc: new peer: %w", err)
	}

	if _, err := pc.AddTransceiverFromKind(
		pionwebrtc.RTPCodecTypeAudio,
		pionwebrtc.RTPTransceiverInit{Direction: pionwebrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: recv transceiver: %w", err)
	}

	localTrack, err := pionwebrtc.NewTrackLocalStaticSample(
		pionwebrtc.RTPCodecCapability{
			MimeType:  pionwebrtc.MimeTypeOpus,
			ClockRate: 48000,
			Channels:  2,
		},
		"audio-tts", callID,
	)
	if err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: tts track: %w", err)
	}
	rtpSender, err := pc.AddTrack(localTrack)
	if err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: add track: %w", err)
	}
	go drainRTCP(rtpSender)

	sessCtx, sessCancel := context.WithCancel(context.Background())
	s := &session{
		cfg:           cfg,
		pc:            pc,
		callID:        callID,
		dialogDialURL: merged,
		localTrack:    localTrack,
		bridgeSR:      16000,
		done:          make(chan struct{}),
		sessCtx:       sessCtx,
		sessCancel:    sessCancel,
	}

	pc.OnConnectionStateChange(s.onConnectionStateChange)
	pc.OnTrack(func(track *pionwebrtc.TrackRemote, recv *pionwebrtc.RTPReceiver) {
		go s.handleRemoteTrack(s.sessCtx, track, recv)
	})

	if err := pc.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer, SDP: offer.SDP,
	}); err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: set remote: %w", err)
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: create answer: %w", err)
	}

	gatherDone := pionwebrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: set local: %w", err)
	}
	select {
	case <-gatherDone:
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		_ = pc.Close()
		return nil, SDPMessage{}, ctx.Err()
	}

	final := pc.LocalDescription()
	if final == nil {
		_ = pc.Close()
		return nil, SDPMessage{}, errors.New("webrtc: nil local description after gather")
	}
	return s, SDPMessage{SDP: final.SDP, Type: "answer", CallID: callID}, nil
}

func drainRTCP(sender *pionwebrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buf); err != nil {
			return
		}
	}
}

func (s *session) onConnectionStateChange(state pionwebrtc.PeerConnectionState) {
	log.Printf("[webrtc] call=%s pc state: %s", s.callID, state.String())
	switch state {
	case pionwebrtc.PeerConnectionStateFailed, pionwebrtc.PeerConnectionStateClosed:
		s.teardown("pc-state:" + state.String())
	}
}

func (s *session) handleRemoteTrack(ctx context.Context, track *pionwebrtc.TrackRemote, _ *pionwebrtc.RTPReceiver) {
	codecParams := track.Codec()
	srcSR := int(codecParams.ClockRate)
	srcCh := int(codecParams.Channels)
	if srcCh < 1 {
		srcCh = 1
	}

	dec, err := encoder.CreateDecode(
		media.CodecConfig{
			Codec:                     "opus",
			SampleRate:                srcSR,
			Channels:                  srcCh,
			FrameDuration:             "20ms",
			OpusDecodeChannels:        srcCh,
			OpusPCMBridgeDecodeStereo: srcCh >= 2,
		},
		media.CodecConfig{Codec: "pcm", SampleRate: s.bridgeSR, Channels: 1},
	)
	if err != nil {
		log.Printf("[webrtc] call=%s opus decoder init failed: %v (rebuild with: go run -tags opus ...)", s.callID, err)
		s.teardown("decoder-error")
		return
	}
	enc, err := encoder.CreateEncode(
		media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 2, FrameDuration: "20ms"},
		media.CodecConfig{Codec: "pcm", SampleRate: s.bridgeSR, Channels: 1},
	)
	if err != nil {
		log.Printf("[webrtc] call=%s opus encoder init failed: %v (rebuild with: go run -tags opus ...)", s.callID, err)
		s.teardown("encoder-error")
		return
	}
	s.opusDec = dec
	s.opusEnc = enc

	if err := s.startVoice(ctx); err != nil {
		log.Printf("[webrtc] call=%s pipeline start failed: %v", s.callID, err)
		s.teardown("pipeline-error")
		return
	}

	depack := &codecs.OpusPacket{}
	sb := samplebuilder.New(50, depack, uint32(srcSR),
		samplebuilder.WithMaxTimeDelay(200*time.Millisecond))

	for {
		if s.closed.Load() {
			return
		}
		_ = track.SetReadDeadline(time.Now().Add(60 * time.Second))
		pkt, _, err := track.ReadRTP()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.teardown("rtp-read-end")
			}
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
			out, err := dec(&media.AudioPacket{Payload: sample.Data})
			if err != nil || len(out) == 0 {
				continue
			}
			ap, _ := out[0].(*media.AudioPacket)
			if ap == nil || len(ap.Payload) == 0 {
				continue
			}
			_ = s.voiceSess.ProcessAudio(ctx, ap.Payload)
		}
	}
}

func (s *session) startVoice(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.voiceSess != nil {
		return nil
	}

	var dn asr.Denoiser
	if s.cfg.DenoiserFactory != nil {
		dn = s.cfg.DenoiserFactory()
		s.denoiser = dn
	}

	meta := dialog.StartMeta{
		From:  s.clientMeta,
		To:    "webrtc",
		Codec: "opus",
		PCMHz: s.bridgeSR,
	}

	gwCfg := gateway.ClientConfig{}
	if s.cfg.ConfigureClient != nil {
		s.cfg.ConfigureClient(&gwCfg)
	}

	paceRealtime := true
	voiceSess, gw, err := transport.NewCall(ctx, transport.CallConfig{
		CallID:           s.callID,
		DialogURL:        s.dialogDialURL,
		Meta:             meta,
		Factory:          s.cfg.SessionFactory,
		OnAudioOut:       s.ttsSink,
		InputCodec:       "pcm",
		OutputCodec:      "pcm",
		PCMSampleRate:    s.bridgeSR,
		PaceRealtime:     &paceRealtime,
		TTSFrameDuration: 20 * time.Millisecond,
		Denoiser:         dn,
		Gateway:          gwCfg,
		OnHangup:         func(reason string) { s.teardown("dialog-hangup:" + reason) },
		OnFirstAudio: func(ev dialog.FirstAudioEvent) {
			log.Printf("[voice] call=%s first_audio utter=%s tts_first=%dms e2e_first=%dms",
				s.callID, ev.UtteranceID, ev.TTSFirstByteMs, ev.E2EFirstByteMs)
		},
		OnTurn: func(ev dialog.TurnEvent) {
			log.Printf("[voice] call=%s utter=%s tts_first=%dms e2e_first=%dms play=%dms ok=%v text_len=%d",
				s.callID, ev.UtteranceID, ev.TTSFirstByteMs, ev.E2EFirstByteMs, ev.DurationMs, ev.OK, len(ev.LLMText))
		},
	})
	if err != nil {
		return err
	}
	if err := gw.Start(ctx); err != nil {
		return err
	}
	if err := voiceSess.Start(ctx); err != nil {
		return err
	}

	s.voiceSess = voiceSess
	s.gw = gw
	if s.cfg.OnSessionStart != nil {
		s.cfg.OnSessionStart(ctx, s.callID, s.clientMeta)
	}
	return nil
}

func (s *session) ttsSink(pcm []byte) error {
	if s.closed.Load() {
		return errors.New("webrtc: closed")
	}
	out, err := s.opusEnc(&media.AudioPacket{Payload: pcm})
	if err != nil {
		return err
	}
	for _, p := range out {
		ap, _ := p.(*media.AudioPacket)
		if ap == nil || len(ap.Payload) == 0 {
			continue
		}
		if err := s.localTrack.WriteSample(pionmedia.Sample{
			Data:     ap.Payload,
			Duration: 20 * time.Millisecond,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) teardown(reason string) {
	s.closeOnce.Do(func() {
		log.Printf("[webrtc] call=%s teardown: %s", s.callID, reason)
		s.closed.Store(true)
		if s.gw != nil {
			s.gw.Close(reason)
		}
		if s.voiceSess != nil {
			s.voiceSess.Close(reason)
		}
		if s.pc != nil {
			_ = s.pc.Close()
		}
		if closer, ok := s.denoiser.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		if s.cfg.OnSessionEnd != nil {
			s.cfg.OnSessionEnd(context.Background(), s.callID, reason)
		}
		if s.sessCancel != nil {
			s.sessCancel()
		}
		close(s.done)
	})
}

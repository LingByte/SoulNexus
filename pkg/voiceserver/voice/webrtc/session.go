// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webrtc

// session.go is the per-call WebRTC bridge: PeerConnection ↔ ASR/TTS ↔
// dialog gateway. The lifecycle:
//
//  1. handleOffer: parse SDP, build pc, attach an inbound audio transceiver
//     (recvonly from peer's mic) and one outbound TrackLocalStaticSample
//     for our TTS reply.
//  2. setRemoteDescription(offer) → CreateAnswer → setLocalDescription →
//     wait for ICE gathering complete (no trickle) → return SDP answer.
//  3. OnTrack: drain inbound RTP through a samplebuilder (jitter buffer)
//     into Opus samples → opus.Decode → PCM16 → ASR pipeline.
//  4. TTS pipeline Sink: PCM16 → opus encode → WriteSample on the local
//     track. pion handles RTP packetization, SRTP, NACK responses.
//  5. ICE state Failed/Closed → tear everything down.
//
// All audio handling is goroutine-safe; pion guarantees one OnTrack
// callback per remote track.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"

	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	pionwebrtc "github.com/pion/webrtc/v4"
	pionmedia "github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
	"go.uber.org/zap"
)

// SessionFactory is the same shape as in pkg/voice/xiaozhi — one
// recognizer per call, optionally-shared TTS service.
type SessionFactory interface {
	NewASR(ctx context.Context, callID string) (svc recognizer.TranscribeService, sampleRate int, err error)
	TTS(ctx context.Context, callID string) (svc tts.Service, sampleRate int, err error)
}

// session is one live PeerConnection plus its dialog bridge.
type session struct {
	cfg        ServerConfig
	pc         *pionwebrtc.PeerConnection
	callID     string
	clientMeta string
	log        *zap.Logger

	// dialogDialURL is cfg.DialogWSURL with per-offer apiKey/apiSecret/agentId
	// merged into the query (SoulNexus /ws/call). gateway.Start still appends call_id.
	dialogDialURL string

	// Local track we publish TTS audio onto. pion handles RTP timestamps,
	// SRTP, NACK responses; we just call WriteSample at realtime pace.
	localTrack *pionwebrtc.TrackLocalStaticSample

	// Inbound audio decoded into PCM16 @ 16 kHz mono goes through this
	// pipeline. Built lazily in OnTrack so we know the negotiated sample
	// rate / channels.
	asrPipe *asr.Pipeline
	ttsPipe *tts.Pipeline
	att     *voice.Attached
	gw      *gateway.Client

	// Codec workers. Opus is the only audio codec we negotiate.
	opusDec media.EncoderFunc
	opusEnc media.EncoderFunc

	// Internal PCM bridge rate. We collapse Opus 48 kHz stereo down to
	// 16 kHz mono so all downstream pipelines (ASR, TTS, persistence)
	// share one sample rate. The encoder upsamples mono PCM16 back to
	// 48 kHz stereo Opus on the way out.
	bridgeSR int

	// Shared concurrency primitives.
	mu        sync.Mutex
	closeOnce sync.Once
	closed    atomic.Bool
	done      chan struct{}

	// sessCtx is the long-lived context for everything that runs AFTER
	// /offer has returned the SDP answer: OnTrack callbacks, the
	// dialog-plane WS dial, ASR/TTS pipelines, the stats poller. It is
	// rooted in context.Background() rather than the inbound HTTP
	// request so that ws.Close()/r.Body close don't propagate cancel
	// to a still-running call. teardown() fires sessCancel.
	//
	// Earlier we (incorrectly) reused the request ctx for OnTrack — the
	// browser would ICE-connect, but by the time pion fired OnTrack
	// (a few ms after we'd written the answer) the request ctx was
	// already done, so gateway.Start saw "operation was canceled" and
	// the whole call was torn down with reason=pipeline-error.
	sessCtx    context.Context
	sessCancel context.CancelFunc

	// ttsActive is set while a Speak is in flight; on Close we use it to
	// avoid racing the encoder against teardown.
	ttsActive atomic.Bool

	// persister is built from cfg.PersisterFactory at startPipelines
	// and used through the call lifecycle. nil = persistence disabled.
	persister gateway.SessionPersister

	// statsOnce guards the periodic stats poller so an ICE flap
	// (Disconnected → Connected → Disconnected → Connected) doesn't
	// spawn multiple poll loops on the same session.
	statsOnce sync.Once

	// rec is built from cfg.RecorderFactory at startPipelines. nil =
	// recording disabled. Captures decoded inbound PCM (browser → us,
	// after Opus decode + jitter buffer) and outbound PCM (TTS, before
	// Opus encode), produces a stereo WAV at teardown.
	rec *recorder.Recorder

	// denoiser is built from cfg.DenoiserFactory at startPipelines.
	// nil = denoising disabled. Closed at teardown — necessary for
	// CGO-backed implementations (rnnoise) to release native state.
	denoiser Denoiser

	// Dialog-side ASR-final hook fires the persistence callback (if any).
	onASRFinal func(text string)
	onTurn     func(t gateway.TurnEvent)
}

// newSession creates the PeerConnection, registers transceivers, and
// completes SDP negotiation against the supplied offer. The returned
// session is fully connected (or returning a non-nil error). Audio bridging
// starts asynchronously — OnTrack fires on the first inbound RTP packet.
func newSession(ctx context.Context, cfg ServerConfig, api *pionwebrtc.API, offer OfferRequest) (*session, SDPMessage, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
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
		ICEServers: cfg.ICEServers,
		// Bundle policy max-bundle keeps audio + (future) data on one
		// transport — fewer DTLS handshakes, fewer ICE candidates.
		BundlePolicy: pionwebrtc.BundlePolicyMaxBundle,
		// Trickle ICE off: we wait for full gathering before answering.
		// Half-trickle adds complexity without a measurable benefit
		// here (audio-only, low candidate count).
	})
	if err != nil {
		return nil, SDPMessage{}, fmt.Errorf("webrtc: new peer: %w", err)
	}

	// Inbound: we expect exactly one Opus track from the browser.
	// Direction is `recvonly` from our side.
	if _, err := pc.AddTransceiverFromKind(
		pionwebrtc.RTPCodecTypeAudio,
		pionwebrtc.RTPTransceiverInit{Direction: pionwebrtc.RTPTransceiverDirectionRecvonly},
	); err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: recv transceiver: %w", err)
	}

	// Outbound TTS track. 48 kHz / 2ch matches the Opus SDP we publish;
	// pion negotiates the same payload type the offer used.
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
	// Drain RTCP coming back on this sender so pion can process NACKs /
	// receiver reports / TWCC. Without this read loop the interceptor
	// stack stalls on backpressure.
	go drainRTCP(rtpSender)

	sessCtx, sessCancel := context.WithCancel(context.Background())
	s := &session{
		cfg:           cfg,
		pc:            pc,
		callID:        callID,
		dialogDialURL: merged,
		log:           logger.With(zap.String("call_id", callID)),
		localTrack:    localTrack,
		bridgeSR:      16000, // ASR-friendly, downstream-shared rate
		done:          make(chan struct{}),
		sessCtx:       sessCtx,
		sessCancel:    sessCancel,
	}

	// Wire ICE / connection-state observers BEFORE we set the remote
	// description so an early failure is observed.
	pc.OnConnectionStateChange(s.onConnectionStateChange)
	pc.OnICEConnectionStateChange(s.onICEStateChange)
	// IMPORTANT: pass sessCtx (not the inbound request ctx) — OnTrack
	// fires after /offer has already returned, by which point a request
	// ctx is canceled.
	pc.OnTrack(func(track *pionwebrtc.TrackRemote, recv *pionwebrtc.RTPReceiver) {
		go s.handleRemoteTrack(s.sessCtx, track, recv)
	})

	// Negotiate.
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

	// Block until ICE gathering finishes so the answer carries every
	// candidate. Without this the browser would see an answer with no
	// candidates and fall back to its own gathering — which works but
	// wastes a few hundred ms on every connect.
	gatherDone := pionwebrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return nil, SDPMessage{}, fmt.Errorf("webrtc: set local: %w", err)
	}
	select {
	case <-gatherDone:
	case <-time.After(5 * time.Second):
		// Keep going with whatever candidates we already have. Most
		// browsers handle a partial answer fine.
		s.log.Warn("webrtc: ICE gathering timeout; sending partial answer")
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

// drainRTCP keeps the RTCP read loop hot for the outbound track. Most of
// the packets are receiver reports / NACK requests handled internally by
// the interceptor chain; we just need to keep the buffer drained.
func drainRTCP(s *pionwebrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		_, _, err := s.Read(buf)
		if err != nil {
			return
		}
		// We could parse pkt with rtcp.Unmarshal here for telemetry
		// (twcc.PacketReport, etc.) but pion's interceptors already
		// handle the protocol; this loop exists just to drain.
		_ = rtcp.Packet(nil)
	}
}

// onConnectionStateChange surfaces state on the log and triggers teardown
// on Failed/Closed. Disconnected is ignored — pion auto-reconnects within
// the ICE timeouts we set.
func (s *session) onConnectionStateChange(state pionwebrtc.PeerConnectionState) {
	s.log.Info("webrtc: pc state", zap.Stringer("state", state))
	if s.persister != nil {
		detail, _ := json.Marshal(map[string]any{"state": state.String()})
		switch state {
		case pionwebrtc.PeerConnectionStateConnected:
			s.persister.OnEvent(context.Background(), "dtls.connected", "info", detail)
		case pionwebrtc.PeerConnectionStateFailed,
			pionwebrtc.PeerConnectionStateClosed:
			s.persister.OnEvent(context.Background(), "call.terminated", "info", detail)
		}
	}
	switch state {
	case pionwebrtc.PeerConnectionStateFailed,
		pionwebrtc.PeerConnectionStateClosed:
		s.teardown("pc-state:" + state.String())
	}
}

// onICEStateChange logs ICE transitions. Pion exposes both PC and ICE
// state machines; we only act on the higher-level PC state but logging
// both helps diagnose flaky NAT topologies.
func (s *session) onICEStateChange(state pionwebrtc.ICEConnectionState) {
	s.log.Info("webrtc: ice state", zap.Stringer("state", state))
	if s.persister == nil {
		return
	}
	detail, _ := json.Marshal(map[string]any{"state": state.String()})
	switch state {
	case pionwebrtc.ICEConnectionStateConnected,
		pionwebrtc.ICEConnectionStateCompleted:
		s.persister.OnEvent(context.Background(), "ice.connected", "info", detail)
		// Once ICE is up, kick off the periodic stats poller so the
		// per-call media-quality timeline starts filling in. Idempotent.
		s.startStatsPoller()
	case pionwebrtc.ICEConnectionStateFailed:
		s.persister.OnEvent(context.Background(), "ice.failed", "error", detail)
	case pionwebrtc.ICEConnectionStateDisconnected:
		s.persister.OnEvent(context.Background(), "ice.disconnect", "warn", detail)
	}
}

// handleRemoteTrack runs once per inbound track. It builds the codec
// workers + ASR/TTS pipelines lazily (so we know the actual negotiated
// sample rate) and then drains RTP through a jitter buffer.
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
		s.log.Error("webrtc: build opus decoder", zap.Error(err))
		s.teardown("decoder-error")
		return
	}
	enc, err := encoder.CreateEncode(
		media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 2, FrameDuration: "20ms"},
		media.CodecConfig{Codec: "pcm", SampleRate: s.bridgeSR, Channels: 1},
	)
	if err != nil {
		s.log.Error("webrtc: build opus encoder", zap.Error(err))
		s.teardown("encoder-error")
		return
	}
	s.opusDec = dec
	s.opusEnc = enc

	if err := s.startPipelines(ctx); err != nil {
		// Surface to stderr too so a Nop zap logger doesn't hide the
		// root cause — operators just see "pipeline-error" otherwise.
		s.log.Error("webrtc: pipelines", zap.Error(err))
		log.Printf("[webrtc] call=%s startPipelines failed: %v", s.callID, err)
		s.teardown("pipeline-error")
		return
	}

	// Jitter buffer: keep up to 200 ms of late packets, ordered by
	// sequence number, before declaring them lost. The Opus
	// depacketizer pulls one frame at a time; samplebuilder gives us
	// PrevDroppedPackets so we can feed Opus's PLC if needed.
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
				s.log.Debug("webrtc: rtp read end", zap.Error(err))
			}
			s.teardown("rtp-read-end")
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
			// Decode Opus → PCM16 mono @ bridgeSR.
			out, err := dec(&media.AudioPacket{Payload: sample.Data})
			if err != nil || len(out) == 0 {
				if sample.PrevDroppedPackets > 0 {
					// One lost packet inside an Opus stream: the next
					// frame's FEC payload reconstructs it on the peer
					// side. NACK additionally retransmits the missed
					// RTP. Nothing else to do here.
					s.log.Debug("webrtc: opus decode after loss",
						zap.Uint16("dropped", sample.PrevDroppedPackets),
						zap.Error(err))
				}
				continue
			}
			ap, _ := out[0].(*media.AudioPacket)
			if ap == nil || len(ap.Payload) == 0 {
				continue
			}
			if err := s.asrPipe.ProcessPCM(ctx, ap.Payload); err != nil {
				s.log.Debug("webrtc: asr feed", zap.Error(err))
			}
			if s.rec != nil {
				s.rec.WriteCaller(ap.Payload)
			}
		}
	}
}

// startPipelines mints ASR + TTS pipelines and dials the dialog plane.
// Idempotent — only the first call does work.
func (s *session) startPipelines(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.asrPipe != nil {
		return nil
	}

	asrSvc, asrSR, err := s.cfg.SessionFactory.NewASR(ctx, s.callID)
	if err != nil || asrSvc == nil {
		return fmt.Errorf("asr: %w", err)
	}
	asrPipe, err := asr.New(asr.Options{
		ASR:             asrSvc,
		InputSampleRate: s.bridgeSR,
		SampleRate:      asrSR,
		Channels:        1,
		DialogID:        s.callID,
		Logger:          s.log,
	})
	if err != nil {
		return fmt.Errorf("asr pipeline: %w", err)
	}
	s.asrPipe = asrPipe

	// Optional per-call denoiser (typically rnnoise). Wired before the
	// session starts feeding audio so the very first frame benefits.
	if s.cfg.DenoiserFactory != nil {
		if dn := s.cfg.DenoiserFactory(); dn != nil {
			s.denoiser = dn
			asrPipe.SetDenoiser(dn)
		}
	}

	ttsSvc, ttsSR, err := s.cfg.SessionFactory.TTS(ctx, s.callID)
	if err != nil || ttsSvc == nil {
		return fmt.Errorf("tts: %w", err)
	}
	ttsPipe, err := tts.New(tts.Config{
		Service:          ttsSvc,
		InputSampleRate:  ttsSR,
		OutputSampleRate: s.bridgeSR,
		Channels:         1,
		FrameDuration:    20 * time.Millisecond,
		PaceRealtime:     true, // realtime pace = WebRTC needs steady RTP timestamps
		Sink:             s.ttsSink,
		Logger:           s.log,
	})
	if err != nil {
		return fmt.Errorf("tts pipeline: %w", err)
	}
	ttsPipe.Start(ctx)
	s.ttsPipe = ttsPipe

	s.att = &voice.Attached{ASR: asrPipe, TTS: ttsPipe}

	gwCfg := gateway.ClientConfig{
		URL:      s.dialogDialURL,
		Attached: s.att,
		CallID:   s.callID,
		BargeIn:  bargeInFromFactory(s.cfg.BargeInFactory),
		OnHangup: func(reason string) {
			// Record the dialog-side hangup so the timeline shows the
			// LLM/dialog plane initiated the drop, separately from the
			// terminating call.terminated event the teardown emits.
			if s.persister != nil {
				detail, _ := json.Marshal(map[string]any{"reason": reason})
				s.persister.OnEvent(context.Background(), "dialog.hangup", "info", detail)
			}
			s.teardown("dialog-hangup:" + reason)
		},
		OnTTSStart: func(_, _ string) { s.ttsActive.Store(true) },
		OnTurn: func(t gateway.TurnEvent) {
			s.ttsActive.Store(false)
			if s.persister != nil {
				s.persister.OnTurn(context.Background(), t)
			}
			if s.onTurn != nil {
				s.onTurn(t)
			}
		},
		OnASRFinal: func(text string) {
			if s.persister != nil {
				s.persister.OnASRFinal(context.Background(), text)
			}
			if s.onASRFinal != nil {
				s.onASRFinal(text)
			}
		},
		Logger: s.log,
	}
	if s.cfg.ConfigureClient != nil {
		s.cfg.ConfigureClient(&gwCfg)
	}
	gw, err := gateway.NewClient(gwCfg)
	if err != nil {
		return fmt.Errorf("gateway client: %w", err)
	}
	if err := gw.Start(ctx, gateway.StartMeta{
		From:  s.clientMeta,
		To:    "webrtc",
		Codec: "opus",
		PCMHz: s.bridgeSR,
	}); err != nil {
		return fmt.Errorf("gateway dial: %w", err)
	}
	s.gw = gw

	if s.cfg.OnSessionStart != nil {
		s.cfg.OnSessionStart(ctx, s.callID, s.clientMeta)
	}
	// Build the recorder before the persister so OnRecording (in
	// teardown) sees the metadata it needs to write a row.
	if s.cfg.RecorderFactory != nil {
		s.rec = s.cfg.RecorderFactory(s.callID, "opus", s.bridgeSR)
	}
	if s.cfg.PersisterFactory != nil {
		s.persister = s.cfg.PersisterFactory(ctx, s.callID, s.clientMeta, "webrtc", s.clientMeta)
		if s.persister != nil {
			// WebRTC always negotiates Opus today.
			s.persister.OnAccept(ctx, "opus", s.bridgeSR, s.clientMeta)
		}
	}
	log.Printf("[webrtc] call=%s connected: bridge=%dHz asr=%dHz tts=%dHz dialog=%s",
		s.callID, s.bridgeSR, asrSR, ttsSR, gateway.RedactDialogDialURL(s.dialogDialURL))
	return nil
}

// ttsSink is what the TTS Pipeline calls for each paced PCM16 frame.
// We Opus-encode and push as a media.Sample — pion fills in the RTP
// header (sequence numbers, timestamps), packetises, and SRTP-encrypts.
//
// The Pipeline already paces frames at wall-clock rate, so we don't need
// extra rate limiting here; the Sample.Duration tells pion how to advance
// its internal RTP timestamp.
func (s *session) ttsSink(pcm []byte) error {
	if s.closed.Load() {
		return errors.New("webrtc: closed")
	}
	if s.rec != nil {
		s.rec.WriteAI(pcm)
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

// teardown closes the dialog bridge, the audio pipelines, and the
// PeerConnection in the right order. Idempotent — fires once even if
// called from multiple state-change goroutines.
//
// Order matters:
//
//  1. closed=true stops any new TTS frames being written into the
//     recorder or out to the local track.
//  2. captureStats(final=true) runs WHILE the PeerConnection is still
//     live so pion's GetStats() returns a complete snapshot. If we
//     called it after pc.Close() the inbound/outbound stream stats
//     would be empty and the per-call summary row in
//     `call_media_stats` would be useless.
//  3. Close gw → att → pc in that order: dialog first so the dialog
//     plane sees the call end before our pipelines tear down; ASR/TTS
//     next so they drain cleanly; PC last so any in-flight RTP /
//     RTCP from pion's interceptor stack has somewhere to land until
//     we explicitly stop it.
//  4. Flush recorder → OnRecording → OnTerminate. Recorder runs after
//     pc.Close so we can be sure no late ttsSink writes are racing
//     the Flush; persister sees OnRecording before OnTerminate so the
//     `call_recording` row exists before the SIPCall row flips to
//     ended (matches the SIP and xiaozhi flush ordering).
func (s *session) teardown(reason string) {
	s.closeOnce.Do(func() {
		s.closed.Store(true)

		// Snapshot stats BEFORE closing pc so we get a real summary.
		if s.persister != nil {
			s.captureStats(true)
		}
		if s.gw != nil {
			s.gw.Close(reason)
		}
		if s.att != nil {
			s.att.Close()
		}
		if s.pc != nil {
			_ = s.pc.Close()
		}
		// Release the denoiser BEFORE the recorder flush so any final
		// audio queued in the recorder is unaffected (the recorder
		// captured raw PCM, not denoised PCM — denoising lives on
		// the ASR feed only).
		if s.denoiser != nil {
			s.denoiser.Close()
			s.denoiser = nil
		}
		if s.rec != nil {
			if info, ok := s.rec.Flush(context.Background()); ok && s.persister != nil {
				s.persister.OnRecording(context.Background(), info)
			}
		}
		if s.persister != nil {
			s.persister.OnTerminate(context.Background(), reason)
		}
		if s.cfg.OnSessionEnd != nil {
			s.cfg.OnSessionEnd(context.Background(), s.callID, reason)
		}
		// Cancel the session-scoped context AFTER all the persistence
		// hooks above have run — they take ctx-free helpers, but any
		// long-lived goroutines still parked on sessCtx (RTP read loop,
		// stats poller, gateway client) get a clean exit signal here.
		if s.sessCancel != nil {
			s.sessCancel()
		}
		close(s.done)
		log.Printf("[webrtc] call=%s closed: %s", s.callID, reason)
	})
}

// captureStats and startStatsPoller live in stats.go.

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/server"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"gorm.io/gorm"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type gatewayRegistry struct {
	mu      sync.RWMutex
	clients map[string]*gateway.Client
}

func newGatewayRegistry() *gatewayRegistry {
	return &gatewayRegistry{clients: make(map[string]*gateway.Client)}
}

func (r *gatewayRegistry) put(callID string, c *gateway.Client) {
	if r == nil || strings.TrimSpace(callID) == "" || c == nil {
		return
	}
	r.mu.Lock()
	r.clients[callID] = c
	r.mu.Unlock()
}

func (r *gatewayRegistry) get(callID string) *gateway.Client {
	if r == nil || strings.TrimSpace(callID) == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[callID]
}

func (r *gatewayRegistry) drop(callID string) {
	if r == nil || strings.TrimSpace(callID) == "" {
		return
	}
	r.mu.Lock()
	delete(r.clients, callID)
	r.mu.Unlock()
}

type echoInviteHandler struct {
	serial    *int64
	record    bool
	asrEcho   bool
	replyText string
	dialogWS  string
	srv       *server.SIPServer
	db        *gorm.DB         // optional persistence; nil = disabled
	gateways  *gatewayRegistry // tracks per-call *gateway.Client for REFER lookup
}

// OnRefer implements server.TransferHandler. The SIP server invokes
// this when a REFER arrives for a live call (Refer-To URI in
// referTo). We don't bridge the second leg ourselves — instead we
// surface the request to the dialog plane via transfer.request, then
// acknowledge the REFER subscription with NOTIFY 200 OK so the peer
// is happy. The dialog app decides whether to honour the transfer
// (typical: prompt the LLM to confirm, then send a hangup command;
// the carrier's B2BUA / SBC executes the actual redirect).
func (h *echoInviteHandler) OnRefer(ctx context.Context, callID, referTo string, notify func(frag, subState string)) {
	target := strings.TrimSpace(referTo)
	log.Printf("[refer] call=%s target=%q", callID, target)
	if h.gateways != nil {
		if cli := h.gateways.get(callID); cli != nil {
			cli.ForwardTransferRequest(target)
		}
	}
	if notify != nil {
		// 200 OK + terminated subscription completes the implicit
		// REFER subscription per RFC 3515. We ack it as accepted
		// regardless of whether the dialog plane will actually
		// redirect — the alternative (waiting for the dialog plane)
		// would leave the SIP peer's transaction dangling.
		notify("SIP/2.0 200 OK", "terminated;reason=noresource")
	}
}

func (h *echoInviteHandler) OnIncomingCall(ctx context.Context, inv *server.IncomingCall) (server.Decision, error) {
	n := atomic.AddInt64(h.serial, 1)
	log.Printf("[call#%d] INVITE from=%s call-id=%s", n, inv.FromURI, inv.CallID)

	if inv.SDP == nil || len(inv.SDP.Codecs) == 0 {
		log.Printf("[call#%d] rejecting: no usable codecs", n)
		return server.Decision{StatusCode: 488, ReasonPhrase: "Not Acceptable Here"}, nil
	}

	// Optional recording sink: captures decoded PCM from both directions and
	// writes a stereo WAV (L=caller / R=AI-TTS) on teardown. Codec-agnostic.
	var recorder *pcmRecorder
	if h.record {
		recorder = newRTPRecorder(inv.CallID)
	}

	// Optional persistence row created right at INVITE so we have a row even
	// if codec negotiation later fails.
	pers := app.NewCallPersister(ctx, h.db, inv, persist.DirectionInbound)

	// Reuse the server-allocated RTP socket. Allocating a fresh siprtp.NewSession
	// here would cause the SDP answer (built from inv.RTPSession) to advertise
	// a port nobody is listening on, and inbound RTP would be dropped silently.
	rtpSess := inv.RTPSession
	if rtpSess == nil {
		log.Printf("[call#%d] missing inv.RTPSession", n)
		return server.Decision{StatusCode: 500, ReasonPhrase: "Server Misconfigured"}, nil
	}
	if err := session.ApplyRemoteSDP(rtpSess, inv.SDP); err != nil {
		log.Printf("[call#%d] apply remote sdp: %v", n, err)
		return server.Decision{StatusCode: 488, ReasonPhrase: "Not Acceptable Here"}, nil
	}

	legCfg := session.MediaLegConfig{}
	if recorder != nil {
		legCfg.InputFilters = append(legCfg.InputFilters, recorder.inFilter)
		legCfg.OutputFilters = append(legCfg.OutputFilters, recorder.outFilter)
	}

	leg, err := session.NewMediaLeg(ctx, inv.CallID, rtpSess, inv.SDP.Codecs, legCfg)
	if err != nil {
		log.Printf("[call#%d] media leg: %v", n, err)
		return server.Decision{StatusCode: 488, ReasonPhrase: "Not Acceptable Here"}, nil
	}

	if recorder != nil {
		neg := leg.NegotiatedSDP()
		recorder.setCodec(neg.Name, neg.ClockRate)
		recorder.setPCMRate(leg.PCMSampleRate())
	}

	// Voice control plane wiring. Three mutually-exclusive modes:
	//   1) VOICE_DIALOG_WS set: bridge ASR/DTMF events out to the dialog app via
	//      WebSocket, and let the dialog app drive TTS by sending tts.speak
	//      commands back. This is the "pure media service" mode.
	//   2) -asr-echo set: built-in echo (QCloud ASR → fixed reply via TTS).
	//      Useful for local smoke tests without any dialog backend.
	//   3) Neither: pure SIP UAS, no voice intelligence wired.
	var voiceAtt *voice.Attached
	var gw *gateway.Client
	switch {
	case strings.TrimSpace(h.dialogWS) != "":
		va, c, err := attachVoiceGateway(ctx, leg, inv, n, h.dialogWS, h.srv, pers)
		if err != nil {
			log.Printf("[call#%d] dialog-ws disabled: %v", n, err)
		} else {
			voiceAtt, gw = va, c
			// Register so the SIP REFER (TransferHandler.OnRefer)
			// can reach this client mid-call. Removed in
			// OnTerminate below to avoid leaks.
			h.gateways.put(inv.CallID, c)
		}
	case h.asrEcho:
		va, err := attachVoiceEcho(ctx, leg, inv.CallID, n, h.replyText)
		if err != nil {
			log.Printf("[call#%d] asr-echo disabled: %v", n, err)
		} else {
			voiceAtt = va
		}
	}

	log.Printf("[call#%d] accepted, codec=%s rtp=%s", n, leg.NegotiatedSDP().Name, rtpSess.LocalAddr.String())

	// Stamp negotiated codec/rtp on the persisted row.
	remoteRTP := ""
	if inv.SDP != nil && inv.SDP.IP != "" && inv.SDP.Port > 0 {
		remoteRTP = net.JoinHostPort(inv.SDP.IP, strconv.Itoa(inv.SDP.Port))
	}
	pers.OnAccept(ctx, leg, remoteRTP)
	if recorder != nil {
		recorder.persister = pers
	}

	return server.Decision{
		Accept:   true,
		MediaLeg: leg,
		OnTerminate: func(reason string) {
			log.Printf("[call#%d] terminated: %s", n, reason)
			if gw != nil {
				gw.Close(reason)
				h.gateways.drop(inv.CallID)
			}
			if voiceAtt != nil {
				voiceAtt.Close()
			}
			if recorder != nil {
				recorder.flush()
			}
			// Persistence flush comes last so recording_url is captured.
			pers.OnTerminate(context.Background(), reason)
		},
	}, nil
}

// attachVoiceEcho wires QCloud ASR → final transcript → QCloud TTS("您好，已收到").
func attachVoiceEcho(ctx context.Context, leg *session.MediaLeg, callID string, callN int64, replyText string) (*voice.Attached, error) {
	asrAppID := strings.TrimSpace(utils.GetEnv("ASR_APPID"))
	asrSID := strings.TrimSpace(utils.GetEnv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(utils.GetEnv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(utils.GetEnv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(utils.GetEnv("TTS_APPID"))
	ttsSID := strings.TrimSpace(utils.GetEnv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(utils.GetEnv("TTS_SECRET_KEY"))
	if asrAppID == "" || asrSID == "" || asrSKey == "" {
		return nil, fmt.Errorf("missing ASR_APPID/ASR_SECRET_ID/ASR_SECRET_KEY")
	}
	if ttsAppID == "" || ttsSID == "" || ttsSKey == "" {
		return nil, fmt.Errorf("missing TTS_APPID/TTS_SECRET_ID/TTS_SECRET_KEY")
	}

	// ---- ASR ----
	asrOpt := recognizer.NewQcloudASROption(asrAppID, asrSID, asrSKey)
	if asrModel != "" {
		asrOpt.ModelType = asrModel
	}
	asrSvc := recognizer.NewQcloudASR(asrOpt)

	// ASR output rate inferred from model name (8k_* → 8000, otherwise 16000).
	asrRate := 16000
	if strings.Contains(strings.ToLower(asrOpt.ModelType), "8k") {
		asrRate = 8000
	}

	// ---- TTS ----
	bridgeSR := leg.PCMSampleRate()
	ttsSR := bridgeSR
	if ttsSR < 8000 {
		ttsSR = 16000
	}
	ttsCfg := synthesizer.NewQcloudTTSConfig(ttsAppID, ttsSID, ttsSKey, 101007 /* 知性女声 */, "pcm", ttsSR)
	ttsSvc := synthesizer.NewQCloudService(ttsCfg)

	// Attach both pipelines to the MediaLeg.
	att, err := voice.Attach(ctx, leg, voice.AttachConfig{
		ASR:             asrSvc,
		ASRSampleRate:   asrRate,
		DialogID:        callID,
		TTSService:      voicetts.FromSynthesisService(ttsSvc),
		TTSInputRate:    ttsSR,
		TTSPaceRealtime: true,
	})
	if err != nil {
		return nil, err
	}

	// On every final transcript, speak the fixed reply text.
	var (
		busy   atomic.Bool
		lastMu sync.Mutex
		last   string
	)
	att.ASR.SetTextCallback(func(text string, isFinal bool) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		if !isFinal {
			log.Printf("[call#%d][asr-partial] %s", callN, text)
			return
		}
		// Dedupe: some vendors emit the same final twice back-to-back.
		lastMu.Lock()
		if text == last {
			lastMu.Unlock()
			return
		}
		last = text
		lastMu.Unlock()

		log.Printf("[call#%d][asr-final ] %s", callN, text)
		if !busy.CompareAndSwap(false, true) {
			return // a reply is already playing; skip to avoid overlap
		}
		go func() {
			defer busy.Store(false)
			if err := att.TTS.Speak(replyText); err != nil {
				log.Printf("[call#%d][tts] speak failed: %v", callN, err)
			}
		}()
	})
	att.ASR.SetErrorCallback(func(err error, fatal bool) {
		log.Printf("[call#%d][asr] error fatal=%v: %v", callN, fatal, err)
	})

	log.Printf("[call#%d][voice] attached: asr=qcloud(%s,%dHz) tts=qcloud(%dHz) reply=%q",
		callN, asrOpt.ModelType, asrRate, ttsSR, replyText)
	return att, nil
}

// attachVoiceGateway wires QCloud ASR + TTS to the MediaLeg and bridges them
// to an external dialog application over a per-call WebSocket. ASR / DTMF
// events flow up; tts.speak / tts.interrupt / hangup commands flow down.
func attachVoiceGateway(ctx context.Context, leg *session.MediaLeg, inv *server.IncomingCall, callN int64, dialogURL string, srv *server.SIPServer, pers *app.CallPersister) (*voice.Attached, *gateway.Client, error) {
	asrAppID := strings.TrimSpace(utils.GetEnv("ASR_APPID"))
	asrSID := strings.TrimSpace(utils.GetEnv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(utils.GetEnv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(utils.GetEnv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(utils.GetEnv("TTS_APPID"))
	ttsSID := strings.TrimSpace(utils.GetEnv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(utils.GetEnv("TTS_SECRET_KEY"))
	if asrAppID == "" || asrSID == "" || asrSKey == "" {
		return nil, nil, fmt.Errorf("missing ASR_APPID/ASR_SECRET_ID/ASR_SECRET_KEY")
	}
	if ttsAppID == "" || ttsSID == "" || ttsSKey == "" {
		return nil, nil, fmt.Errorf("missing TTS_APPID/TTS_SECRET_ID/TTS_SECRET_KEY")
	}

	asrOpt := recognizer.NewQcloudASROption(asrAppID, asrSID, asrSKey)
	if asrModel != "" {
		asrOpt.ModelType = asrModel
	}
	asrSvc := recognizer.NewQcloudASR(asrOpt)
	asrRate := 16000
	if strings.Contains(strings.ToLower(asrOpt.ModelType), "8k") {
		asrRate = 8000
	}

	bridgeSR := leg.PCMSampleRate()
	ttsSR := bridgeSR
	if ttsSR < 8000 {
		ttsSR = 16000
	}
	var ttsVoice int64 = 101007
	ttsCfg := synthesizer.NewQcloudTTSConfig(ttsAppID, ttsSID, ttsSKey, ttsVoice, "pcm", ttsSR)
	ttsSvc := synthesizer.NewQCloudService(ttsCfg)

	// Wrap the raw TTS in a process-level PCM cache: repeated phrases skip
	// the QCloud round-trip entirely. Voice key isolates by sample rate +
	// voice id so two calls with different profiles never collide.
	ttsService := voicetts.FromSynthesisService(ttsSvc)
	cacheVoiceKey := fmt.Sprintf("qcloud-%d-%d", ttsVoice, ttsSR)
	caching, err := voicetts.NewCachingService(ttsService, voicetts.CacheConfig{
		VoiceKey:   cacheVoiceKey,
		MaxRunes:   80,
		ChunkBytes: 0,
	})
	if err == nil {
		ttsService = caching
		// Best-effort prewarm of common short replies. Errors are logged
		// but never fail the call. Gated behind -tts-prewarm because
		// running it unconditionally on every boot burns N QCloud TTS
		// calls before any user has dialed in — surprising in dev.
		if app.TTSPrewarmEnabled() {
			go caching.Prewarm(ctx, app.PrewarmTexts(), func(t string, e error) {
				log.Printf("[tts-cache] prewarm %q failed: %v", t, e)
			})
		}
	}

	att, err := voice.Attach(ctx, leg, voice.AttachConfig{
		ASR:             asrSvc,
		ASRSampleRate:   asrRate,
		DialogID:        inv.CallID,
		TTSService:      ttsService,
		TTSInputRate:    ttsSR,
		TTSPaceRealtime: true,
	})
	if err != nil {
		return nil, nil, err
	}

	// adapter exposes the SIP-side callPersister as a SessionPersister
	// so the OnASRFinal / OnTurn / OnTerminate callbacks below take the
	// SAME write paths the xiaozhi and WebRTC sessions use. This is what
	// keeps `call_events` rows uniform across all three transports.
	var adapter gateway.SessionPersister = pers.AsSessionPersister()

	gwCfg := gateway.ClientConfig{
		URL:      dialogURL,
		Attached: att,
		CallID:   inv.CallID,
		BargeIn:  app.NewBargeInDetector(),
		OnHangup: func(reason string) {
			log.Printf("[call#%d][gw] hangup requested: %q", callN, reason)
			// Stamp a dialog.hangup event so the timeline carries the
			// reason the dialog plane asked us to drop the call. The
			// terminating call.terminated event comes later from the
			// SIP teardown path (OnTerminate).
			if pers != nil {
				pers.AppendEvent(context.Background(),
					persist.EventKindDialogHangup, persist.EventLevelInfo,
					app.JSONObject(map[string]any{"reason": reason}))
			}
			if srv != nil {
				srv.HangupInboundCall(inv.CallID)
			}
		},
		// Route ASR finals and TTS turns through the SessionPersister
		// adapter rather than calling pers.onASRFinal / pers.onTurn
		// directly. The adapter writes BOTH the SIPCall row updates
		// AND a row into `call_events` (asr.final / tts.end), so the
		// SIP timeline matches xiaozhi and WebRTC byte-for-byte.
		OnASRFinal: func(text string) {
			adapter.OnASRFinal(context.Background(), text)
		},
		OnTurn: func(t gateway.TurnEvent) {
			adapter.OnTurn(context.Background(), t)
		},
	}
	app.ApplyDialogReconnect(&gwCfg)
	cli, err := gateway.NewClient(gwCfg)
	if err != nil {
		att.Close()
		return nil, nil, err
	}
	// StartMeta.To is also the metrics transport label upstream — for
	// SIP we send the literal "sip" rather than the To URI so the
	// /metrics output stays low-cardinality (one series per transport,
	// not one per callee).
	if err := cli.Start(ctx, gateway.StartMeta{
		From:  inv.FromURI,
		To:    "sip",
		Codec: leg.NegotiatedSDP().Name,
		PCMHz: bridgeSR,
	}); err != nil {
		att.Close()
		return nil, nil, fmt.Errorf("dial dialog ws: %w", err)
	}
	log.Printf("[call#%d][gw] dialog ws connected: %s", callN, gateway.RedactDialogDialURL(dialogURL))
	return att, cli, nil
}

// ---------- Recording: SIP wraps pkg/voice/recorder --------------------------

// pcmRecorder is the SIP-side adapter to pkg/voice/recorder.Recorder.
//
// MediaLeg's filter API takes (bool, error) callbacks, so we expose
// inFilter/outFilter that bridge MediaPacket → WriteCaller / WriteAI.
// The underlying recorder is constructed lazily once the negotiated
// PCM bridge rate is known (setPCMRate); audio frames received before
// that point are dropped — under MediaLeg's lifecycle the filter chain
// only fires after MediaLeg.Start, which happens after we already
// called setPCMRate, so no real frames are lost in practice.
//
// All the actual WAV math (wall-clock alignment, stereo interleave,
// RIFF/WAVE wrapping, upload to stores.Default()) lives in
// pkg/voice/recorder so SIP, xiaozhi, and WebRTC produce identical
// recordings. This wrapper is just the SIP bridge.

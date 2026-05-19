// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/cmd/bootstrap"
	"github.com/LingByte/SoulNexus/internal/config"
	voicehttp "github.com/LingByte/SoulNexus/internal/handler/voice"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	siprtp "github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/server"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// gatewayRegistry tracks the live *gateway.Client for each accepted SIP
// call. The TransferHandler (REFER) and any other out-of-band SIP
// signal that needs to talk to the dialog plane mid-call looks up the
// client through this registry.
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
	db        *gorm.DB
	gateways  *gatewayRegistry
}

func (h *echoInviteHandler) OnRefer(ctx context.Context, callID, referTo string, notify func(frag, subState string)) {
	target := strings.TrimSpace(referTo)
	logger.Info(fmt.Sprintf("[refer] call=%s target=%q", callID, target))
	if h.gateways != nil {
		if cli := h.gateways.get(callID); cli != nil {
			cli.ForwardTransferRequest(target)
		}
	}
	if notify != nil {
		notify("SIP/2.0 200 OK", "terminated;reason=noresource")
	}
}

func (h *echoInviteHandler) OnIncomingCall(ctx context.Context, inv *server.IncomingCall) (server.Decision, error) {
	n := atomic.AddInt64(h.serial, 1)
	logger.Info(fmt.Sprintf("[call#%d] INVITE from=%s call-id=%s", n, inv.FromURI, inv.CallID))

	if inv.SDP == nil || len(inv.SDP.Codecs) == 0 {
		logger.Info(fmt.Sprintf("[call#%d] rejecting: no usable codecs", n))
		return server.Decision{StatusCode: 488, ReasonPhrase: "Not Acceptable Here"}, nil
	}

	var recorder *pcmRecorder
	if h.record {
		recorder = newRTPRecorder(inv.CallID)
	}

	pers := app.NewCallPersister(ctx, h.db, inv, persist.DirectionInbound)

	rtpSess := inv.RTPSession
	if rtpSess == nil {
		logger.Info(fmt.Sprintf("[call#%d] missing inv.RTPSession", n))
		return server.Decision{StatusCode: 500, ReasonPhrase: "Server Misconfigured"}, nil
	}
	if err := session.ApplyRemoteSDP(rtpSess, inv.SDP); err != nil {
		logger.Info(fmt.Sprintf("[call#%d] apply remote sdp: %v", n, err))
		return server.Decision{StatusCode: 488, ReasonPhrase: "Not Acceptable Here"}, nil
	}

	legCfg := session.MediaLegConfig{}
	if recorder != nil {
		legCfg.InputFilters = append(legCfg.InputFilters, recorder.inFilter)
		legCfg.OutputFilters = append(legCfg.OutputFilters, recorder.outFilter)
	}

	leg, err := session.NewMediaLeg(ctx, inv.CallID, rtpSess, inv.SDP.Codecs, legCfg)
	if err != nil {
		logger.Info(fmt.Sprintf("[call#%d] media leg: %v", n, err))
		return server.Decision{StatusCode: 488, ReasonPhrase: "Not Acceptable Here"}, nil
	}

	if recorder != nil {
		neg := leg.NegotiatedSDP()
		recorder.setCodec(neg.Name, neg.ClockRate)
		recorder.setPCMRate(leg.PCMSampleRate())
	}

	var voiceAtt *voice.Attached
	var gw *gateway.Client
	switch {
	case strings.TrimSpace(h.dialogWS) != "":
		va, c, err := attachVoiceGateway(ctx, leg, inv, n, h.dialogWS, h.srv, pers)
		if err != nil {
			logger.Info(fmt.Sprintf("[call#%d] dialog-ws disabled: %v", n, err))
		} else {
			voiceAtt, gw = va, c
			h.gateways.put(inv.CallID, c)
		}
	case h.asrEcho:
		va, err := attachVoiceEcho(ctx, leg, inv.CallID, n, h.replyText)
		if err != nil {
			logger.Info(fmt.Sprintf("[call#%d] asr-echo disabled: %v", n, err))
		} else {
			voiceAtt = va
		}
	}

	logger.Info(fmt.Sprintf("[call#%d] accepted, codec=%s rtp=%s", n, leg.NegotiatedSDP().Name, rtpSess.LocalAddr.String()))

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
			logger.Info(fmt.Sprintf("[call#%d] terminated: %s", n, reason))
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
			pers.OnTerminate(context.Background(), reason)
		},
	}, nil
}

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
	ttsCfg := synthesizer.NewQcloudTTSConfig(ttsAppID, ttsSID, ttsSKey, 101007, "pcm", ttsSR)
	ttsSvc := synthesizer.NewQCloudService(ttsCfg)

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
			logger.Info(fmt.Sprintf("[call#%d][asr-partial] %s", callN, text))
			return
		}
		lastMu.Lock()
		if text == last {
			lastMu.Unlock()
			return
		}
		last = text
		lastMu.Unlock()

		logger.Info(fmt.Sprintf("[call#%d][asr-final ] %s", callN, text))
		if !busy.CompareAndSwap(false, true) {
			return
		}
		go func() {
			defer busy.Store(false)
			if err := att.TTS.Speak(replyText); err != nil {
				logger.Info(fmt.Sprintf("[call#%d][tts] speak failed: %v", callN, err))
			}
		}()
	})
	att.ASR.SetErrorCallback(func(err error, fatal bool) {
		logger.Info(fmt.Sprintf("[call#%d][asr] error fatal=%v: %v", callN, fatal, err))
	})

	logger.Info(fmt.Sprintf("[call#%d][voice] attached: asr=qcloud(%s,%dHz) tts=qcloud(%dHz) reply=%q",
		callN, asrOpt.ModelType, asrRate, ttsSR, replyText))
	return att, nil
}

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

	ttsService := voicetts.FromSynthesisService(ttsSvc)
	cacheVoiceKey := fmt.Sprintf("qcloud-%d-%d", ttsVoice, ttsSR)
	caching, err := voicetts.NewCachingService(ttsService, voicetts.CacheConfig{
		VoiceKey:   cacheVoiceKey,
		MaxRunes:   80,
		ChunkBytes: 0,
	})
	if err == nil {
		ttsService = caching
		if app.TTSPrewarmEnabled() {
			go caching.Prewarm(ctx, app.PrewarmTexts(), func(t string, e error) {
				logger.Info(fmt.Sprintf("[tts-cache] prewarm %q failed: %v", t, e))
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

	var adapter gateway.SessionPersister = pers.AsSessionPersister()

	gwCfg := gateway.ClientConfig{
		URL:      dialogURL,
		Attached: att,
		CallID:   inv.CallID,
		BargeIn:  app.NewBargeInDetector(),
		OnHangup: func(reason string) {
			logger.Info(fmt.Sprintf("[call#%d][gw] hangup requested: %q", callN, reason))
			if pers != nil {
				pers.AppendEvent(context.Background(),
					persist.EventKindDialogHangup, persist.EventLevelInfo,
					app.JSONObject(map[string]any{"reason": reason}))
			}
			if srv != nil {
				srv.HangupInboundCall(inv.CallID)
			}
		},
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
	if err := cli.Start(ctx, gateway.StartMeta{
		From:  inv.FromURI,
		To:    "sip",
		Codec: leg.NegotiatedSDP().Name,
		PCMHz: bridgeSR,
	}); err != nil {
		att.Close()
		return nil, nil, fmt.Errorf("dial dialog ws: %w", err)
	}
	logger.Info(fmt.Sprintf("[call#%d][gw] dialog ws connected: %s", callN, gateway.RedactDialogDialURL(dialogURL)))
	return att, cli, nil
}

type loggingDTMFSink struct{}

func (loggingDTMFSink) OnDTMF(callID, digit string, end bool) {
	logger.Info(fmt.Sprintf("[dtmf] call=%s digit=%s end=%v", callID, digit, end))
}

type loggingObserver struct{}

func (loggingObserver) OnCallPreHangup(callID string) bool {
	logger.Info(fmt.Sprintf("[call=%s] pre-hangup", callID))
	return false
}

func (loggingObserver) OnCallCleanup(callID string) {
	logger.Info(fmt.Sprintf("[call=%s] cleanup", callID))
}

type outboundConfig struct {
	TargetURI  string
	LocalIP    string
	HoldFor    time.Duration
	Recorder   *pcmRecorder
	AsrEcho    bool
	ReplyText  string
	RecordOnly bool
}

func runOutbound(ctx context.Context, cfg outboundConfig) error {
	target, err := parseSIPTarget(cfg.TargetURI)
	if err != nil {
		return fmt.Errorf("parse target: %w", err)
	}

	mgr := transaction.NewManager()
	ep := stack.NewEndpoint(stack.EndpointConfig{
		Host: "0.0.0.0",
		Port: 0,
		OnSIPResponse: func(resp *stack.Message, src *net.UDPAddr) {
			mgr.HandleResponse(resp, src)
		},
	})
	if err := ep.Open(); err != nil {
		return fmt.Errorf("endpoint open: %w", err)
	}
	go func() { _ = ep.Serve(ctx) }()
	defer ep.Close()

	localSigAddr := ep.ListenAddr()
	localSigPort := 0
	if ua, ok := localSigAddr.(*net.UDPAddr); ok {
		localSigPort = ua.Port
	}
	logger.Info(fmt.Sprintf("[out] signalling bound: %s", localSigAddr))

	rtpSess, err := siprtp.NewSession(0)
	if err != nil {
		return fmt.Errorf("rtp alloc: %w", err)
	}
	defer rtpSess.Close()

	offerCodecs := []sdp.Codec{
		{PayloadType: 8, Name: "PCMA", ClockRate: 8000, Channels: 1},
		{PayloadType: 0, Name: "PCMU", ClockRate: 8000, Channels: 1},
	}
	offerBody := sdp.Generate(cfg.LocalIP, rtpSess.LocalAddr.Port, offerCodecs)

	callID := newCallID()
	fromTag := newTag()
	branch := "z9hG4bK-" + newTag()

	invite := buildOutboundInvite(buildInviteArgs{
		RequestURI:   target.uri,
		Branch:       branch,
		LocalIP:      cfg.LocalIP,
		LocalSigPort: localSigPort,
		FromURI:      fmt.Sprintf("sip:voiceserver@%s", cfg.LocalIP),
		FromTag:      fromTag,
		ToURI:        target.uri,
		CallID:       callID,
		CSeq:         1,
		Body:         offerBody,
	})

	logger.Info(fmt.Sprintf("[out] INVITE → %s (call-id=%s)", target.addr, callID))
	sendFn := func(m *stack.Message, a *net.UDPAddr) error { return ep.Send(m, a) }

	inviteCtx, cancelInvite := context.WithTimeout(ctx, 30*time.Second)
	defer cancelInvite()
	result, err := mgr.RunInviteClient(inviteCtx, invite, target.addr, sendFn, func(prov *stack.Message) {
		logger.Info(fmt.Sprintf("[out] %d %s", prov.StatusCode, prov.StatusText))
	})
	if err != nil {
		return fmt.Errorf("run invite: %w", err)
	}
	final := result.Final
	logger.Info(fmt.Sprintf("[out] final: %d %s", final.StatusCode, final.StatusText))

	if final.StatusCode < 200 || final.StatusCode >= 300 {
		ack, berr := transaction.BuildAckForInvite(invite, final, invite.RequestURI)
		if berr == nil {
			_ = ep.Send(ack, result.Remote)
		}
		return fmt.Errorf("call rejected: %d %s", final.StatusCode, final.StatusText)
	}

	ackURI := transaction.AckRequestURIFor2xx(final, invite.RequestURI)
	ack, err := transaction.BuildAckForInvite(invite, final, ackURI)
	if err != nil {
		return fmt.Errorf("build ack: %w", err)
	}
	if err := ep.Send(ack, result.Remote); err != nil {
		return fmt.Errorf("send ack: %w", err)
	}
	logger.Info(fmt.Sprintf("[out] ACK → %s", result.Remote))

	remoteSDP, err := sdp.Parse(final.Body)
	if err != nil {
		logger.Info(fmt.Sprintf("[out] WARN parse 200 sdp: %v", err))
	} else if err := session.ApplyRemoteSDP(rtpSess, remoteSDP); err != nil {
		logger.Info(fmt.Sprintf("[out] WARN apply remote sdp: %v", err))
	}

	var leg *session.MediaLeg
	if remoteSDP != nil && len(remoteSDP.Codecs) > 0 {
		legCfg := session.MediaLegConfig{}
		if cfg.Recorder != nil {
			legCfg.InputFilters = append(legCfg.InputFilters, cfg.Recorder.inFilter)
			legCfg.OutputFilters = append(legCfg.OutputFilters, cfg.Recorder.outFilter)
		}
		leg, err = session.NewMediaLeg(ctx, callID, rtpSess, remoteSDP.Codecs, legCfg)
		if err != nil {
			logger.Info(fmt.Sprintf("[out] WARN media leg: %v", err))
		} else {
			neg := leg.NegotiatedSDP()
			logger.Info(fmt.Sprintf("[out] leg up: codec=%s rtp_local=%s rtp_remote=%s:%d",
				neg.Name, rtpSess.LocalAddr.String(), remoteSDP.IP, remoteSDP.Port))
			if cfg.Recorder != nil {
				cfg.Recorder.setCodec(neg.Name, neg.ClockRate)
				cfg.Recorder.setPCMRate(leg.PCMSampleRate())
			}
			if cfg.AsrEcho {
				if va, err := attachVoiceEcho(ctx, leg, callID, 0, cfg.ReplyText); err != nil {
					logger.Info(fmt.Sprintf("[out] asr-echo disabled: %v", err))
				} else {
					defer va.Close()
				}
			}
			leg.Start()
		}
	}

	logger.Info(fmt.Sprintf("[out] holding for %s", cfg.HoldFor))
	select {
	case <-ctx.Done():
	case <-time.After(cfg.HoldFor):
	}

	if leg != nil {
		leg.Stop()
	}

	bye := buildOutboundBye(buildByeArgs{
		RequestURI:   ackURI,
		Branch:       "z9hG4bK-" + newTag(),
		LocalIP:      cfg.LocalIP,
		LocalSigPort: localSigPort,
		FromURI:      fmt.Sprintf("sip:voiceserver@%s", cfg.LocalIP),
		FromTag:      fromTag,
		ToRaw:        final.GetHeader("To"),
		CallID:       callID,
		CSeq:         2,
	})
	logger.Info(fmt.Sprintf("[out] BYE → %s", result.Remote))
	byeCtx, cancelBye := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelBye()
	byeRes, err := mgr.RunNonInviteClient(byeCtx, bye, result.Remote, sendFn)
	if err != nil {
		logger.Info(fmt.Sprintf("[out] BYE error: %v", err))
	} else {
		logger.Info(fmt.Sprintf("[out] BYE final: %d %s", byeRes.Final.StatusCode, byeRes.Final.StatusText))
	}

	if cfg.Recorder != nil {
		cfg.Recorder.flush()
	}
	return nil
}

type sipTarget struct {
	uri  string
	addr *net.UDPAddr
}

func parseSIPTarget(raw string) (sipTarget, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return sipTarget{}, fmt.Errorf("empty target")
	}
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	if !strings.HasPrefix(strings.ToLower(s), "sip:") {
		return sipTarget{}, fmt.Errorf("target must start with sip:")
	}
	rest := s[4:]
	at := strings.Index(rest, "@")
	if at < 0 {
		return sipTarget{}, fmt.Errorf("target missing user@host")
	}
	hostport := rest[at+1:]
	if semi := strings.Index(hostport, ";"); semi >= 0 {
		hostport = hostport[:semi]
	}
	host, portStr, err := net.SplitHostPort(hostport)
	port := 5060
	if err != nil {
		host = hostport
	} else if portStr != "" {
		p, perr := strconv.Atoi(portStr)
		if perr != nil || p <= 0 || p > 65535 {
			return sipTarget{}, fmt.Errorf("bad port in target: %q", portStr)
		}
		port = p
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return sipTarget{}, fmt.Errorf("resolve %q: %w", host, err)
		}
		for _, candidate := range ips {
			if v4 := candidate.To4(); v4 != nil {
				ip = v4
				break
			}
		}
		if ip == nil {
			ip = ips[0]
		}
	}
	canonical := fmt.Sprintf("sip:%s@%s:%d", rest[:at], host, port)
	return sipTarget{uri: canonical, addr: &net.UDPAddr{IP: ip, Port: port}}, nil
}

type buildInviteArgs struct {
	RequestURI   string
	Branch       string
	LocalIP      string
	LocalSigPort int
	FromURI      string
	FromTag      string
	ToURI        string
	CallID       string
	CSeq         int
	Body         string
}

func buildOutboundInvite(a buildInviteArgs) *stack.Message {
	m := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodInvite,
		RequestURI:   a.RequestURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	m.SetHeader("Via", fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", a.LocalIP, a.LocalSigPort, a.Branch))
	m.SetHeader("Max-Forwards", "70")
	m.SetHeader("From", fmt.Sprintf("<%s>;tag=%s", a.FromURI, a.FromTag))
	m.SetHeader("To", fmt.Sprintf("<%s>", a.ToURI))
	m.SetHeader("Call-ID", a.CallID)
	m.SetHeader("CSeq", fmt.Sprintf("%d INVITE", a.CSeq))
	m.SetHeader("Contact", fmt.Sprintf("<sip:voiceserver@%s:%d>", a.LocalIP, a.LocalSigPort))
	m.SetHeader("Allow", "INVITE, ACK, BYE, CANCEL, OPTIONS, INFO, PRACK, UPDATE")
	m.SetHeader("User-Agent", "VoiceServer-Dialer/1.0")
	if a.Body != "" {
		m.SetHeader("Content-Type", "application/sdp")
		m.Body = a.Body
	}
	m.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(a.Body)))
	return m
}

type buildByeArgs struct {
	RequestURI   string
	Branch       string
	LocalIP      string
	LocalSigPort int
	FromURI      string
	FromTag      string
	ToRaw        string
	CallID       string
	CSeq         int
}

func buildOutboundBye(a buildByeArgs) *stack.Message {
	m := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodBye,
		RequestURI:   a.RequestURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	m.SetHeader("Via", fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", a.LocalIP, a.LocalSigPort, a.Branch))
	m.SetHeader("Max-Forwards", "70")
	m.SetHeader("From", fmt.Sprintf("<%s>;tag=%s", a.FromURI, a.FromTag))
	m.SetHeader("To", a.ToRaw)
	m.SetHeader("Call-ID", a.CallID)
	m.SetHeader("CSeq", fmt.Sprintf("%d BYE", a.CSeq))
	m.SetHeader("User-Agent", "VoiceServer-Dialer/1.0")
	m.SetHeader("Content-Length", "0")
	return m
}

func newTag() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func newCallID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:]) + "@voiceserver"
}

type pcmRecorder struct {
	callID    string
	codec     string
	persister *app.CallPersister

	mu  sync.Mutex
	rec *recorder.Recorder
}

func newRTPRecorder(callID string) *pcmRecorder {
	return &pcmRecorder{callID: callID}
}

func (r *pcmRecorder) setCodec(name string, _ int) {
	r.mu.Lock()
	r.codec = name
	r.mu.Unlock()
}

func (r *pcmRecorder) setPCMRate(rate int) {
	if rate <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rec != nil {
		return
	}
	r.rec = recorder.New(recorder.Config{
		CallID:     r.callID,
		SampleRate: rate,
		Transport:  "sip",
		Codec:      r.codec,
	})
}

func (r *pcmRecorder) inFilter(p media.MediaPacket) (bool, error) {
	if rec := r.recorder(); rec != nil {
		if ap, ok := p.(*media.AudioPacket); ok && ap != nil && len(ap.Payload) > 0 && !ap.IsSynthesized {
			rec.WriteCaller(ap.Payload)
		}
	}
	return false, nil
}

func (r *pcmRecorder) outFilter(p media.MediaPacket) (bool, error) {
	if rec := r.recorder(); rec != nil {
		if ap, ok := p.(*media.AudioPacket); ok && ap != nil && len(ap.Payload) > 0 && ap.IsSynthesized {
			rec.WriteAI(ap.Payload)
		}
	}
	return false, nil
}

func (r *pcmRecorder) flush() {
	rec := r.recorder()
	if rec == nil {
		return
	}
	info, ok := rec.Flush(context.Background())
	if !ok {
		logger.Info(fmt.Sprintf("[record] call=%s empty (no inbound/outbound PCM) or upload failed, skip", r.callID))
		return
	}
	logger.Info(fmt.Sprintf("[record] call=%s wrote %d bytes stereo WAV → %s",
		r.callID, info.Bytes, info.URL))
	if r.persister == nil {
		return
	}
	r.persister.OnRecording(context.Background(), info.URL, info.Bytes)
	r.persister.AppendRecording(context.Background(), info)
}

func (r *pcmRecorder) recorder() *recorder.Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rec
}

func main() {
	if err := config.LoadVoice(); err != nil {
		logger.Fatal(fmt.Sprintf("config load failed: %v", err))
	}

	if err := bootstrap.PrintBannerFromFile("voice-banner.txt", config.VoiceGlobalConfig.Name); err != nil {
		logger.Error(fmt.Sprintf("unload banner: %v", err))
	}

	err := logger.Init(&config.VoiceGlobalConfig.Log, config.VoiceGlobalConfig.Mode)
	if err != nil {
		panic(err)
	}
	cfg := config.VoiceGlobalConfig
	app.SetTTSPrewarm(cfg.TTSPrewarm)
	app.SetBargeIn(cfg.BargeIn, cfg.BargeInThreshold, cfg.BargeInFrames)
	app.SetDenoise(cfg.Denoise)
	app.SetASRSentenceFilter(cfg.ASRSentenceFilter, cfg.ASRSentenceFilterSimilarity)
	app.SetDialogReconnect(cfg.DialogReconnect, cfg.DialogReconnectBackoff)
	app.SetRecordChunk(cfg.RecordChunk)
	app.LoadHoldMessages(cfg.HoldMessages)

	dialogWSEffective, err := app.MergeDialogAuthQuery(cfg.DialogWS)
	if err != nil {
		logger.Fatal(fmt.Sprintf("invalid VOICE_DIALOG_WS: %v", err))
	}
	app.WarnDialogAuthIfNeeded(dialogWSEffective)

	host, port, err := app.SplitHostPort(cfg.SIPAddr)
	if err != nil {
		logger.Fatal(fmt.Sprintf("invalid VOICE_SIP_ADDR: %v", err))
	}
	ip := strings.TrimSpace(cfg.LocalIP)
	if ip == "" {
		ip = host
		if ip == "" || ip == "0.0.0.0" {
			ip = "127.0.0.1"
		}
	}

	if strings.TrimSpace(cfg.OutboundURI) != "" {
		var rec *pcmRecorder
		if cfg.Record {
			rec = newRTPRecorder("outbound-" + newTag())
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := runOutbound(ctx, outboundConfig{
			TargetURI: cfg.OutboundURI,
			LocalIP:   ip,
			HoldFor:   cfg.OutboundHold,
			Recorder:  rec,
			AsrEcho:   cfg.ASREcho,
			ReplyText: cfg.ReplyText,
		}); err != nil {
			logger.Fatal(fmt.Sprintf("outbound failed: %v", err))
		}
		return
	}

	srv := server.New(server.Config{
		Host:             host,
		Port:             port,
		LocalIP:          ip,
		RTPPortStart:     cfg.RTPStart,
		RTPPortEnd:       cfg.RTPEnd,
		InviteRingbackMS: 200,
		InviteSend180:    true,
	})

	var callCount int64
	invite := &echoInviteHandler{
		serial:    &callCount,
		record:    cfg.Record,
		gateways:  newGatewayRegistry(),
		asrEcho:   cfg.ASREcho,
		replyText: cfg.ReplyText,
		dialogWS:  dialogWSEffective,
		srv:       srv,
	}

	db, err := app.OpenVoiceServerDB()
	if err != nil {
		logger.Info(fmt.Sprintf("[persist] disabled: %v", err))
	}
	if db != nil {
		logger.Info(fmt.Sprintf("[persist] voice db ready (driver=%s)", config.GlobalConfig.Database.Driver))
	}
	invite.db = db

	srv.SetInviteHandler(invite)
	srv.SetTransferHandler(invite)
	srv.SetDTMFSink(&loggingDTMFSink{})
	srv.SetCallLifecycleObserver(&loggingObserver{})
	srv.SetInboundAllowUnknownDID(true)

	if err := srv.Start(); err != nil {
		logger.Fatal(fmt.Sprintf("sip start: %v", err))
	}
	defer srv.Stop()

	h, p := srv.ListenAddr()
	logger.Info(fmt.Sprintf("voiceserver ready: sip=udp:%s:%d local_ip=%s rtp=%d..%d record=%v asr-echo=%v",
		h, p, ip, cfg.RTPStart, cfg.RTPEnd, cfg.Record, cfg.ASREcho))
	logger.Info(fmt.Sprintf("place a call to sip:anyone@%s:%d to test", h, p))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startVoiceHTTPListener(ctx, cfg.HTTPAddr, voicehttp.NewHandlers(voicehttp.Config{
		DialogWS:                       dialogWSEffective,
		DB:                             db,
		Record:                         cfg.Record,
		EnableXiaozhi:                  cfg.EnableXiaozhi,
		XiaozhiMode:                    cfg.XiaozhiMode,
		XiaozhiPath:                    cfg.XiaozhiPath,
		SoulnexusHardwarePath:          cfg.SoulnexusHardwarePath,
		SoulnexusHardwareBindingURL:    cfg.SoulnexusHardwareBindingURL,
		SoulnexusHardwareBindingSecret: cfg.SoulnexusHardwareBindingSecret,
		EnableWebRTC:                   cfg.EnableWebRTC,
		WebRTCOfferPath:                cfg.WebRTCOfferPath,
		EnableSFU:                      cfg.EnableSFU,
		SFUPath:                        cfg.SFUPath,
		SFUSecret:                      cfg.SFUSecret,
		SFUAllowAnon:                   cfg.SFUAllowAnon,
		SFUMaxParticipants:             cfg.SFUMaxParticipants,
		SFUMaxRooms:                    cfg.SFUMaxRooms,
		SFUAllowedOrigins:              cfg.SFUAllowedOrigins,
		SFUTokenAdminSecret:            cfg.SFUTokenAdminSecret,
		SFURecord:                      cfg.SFURecord,
		SFUWebhookURL:                  cfg.SFUWebhookURL,
	}))

	<-ctx.Done()
	logger.Info(fmt.Sprintf("shutting down (handled %d calls)", atomic.LoadInt64(&callCount)))
}

func startVoiceHTTPListener(ctx context.Context, addr string, h *voicehttp.Handlers) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		logger.Info(fmt.Sprintf("[http] disabled: VOICE_HTTP_ADDR is empty (xiaozhi/webrtc/sfu require an HTTP listener)"))
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	mounted := h.Register(r)
	if mounted == 0 {
		logger.Info(fmt.Sprintf("[http] %s reachable, no transports mounted; only /healthz, /metrics and /media will respond", addr))
	}

	httpSrv := &http.Server{Addr: addr, Handler: r}
	go func() {
		logger.Info(fmt.Sprintf("[http] listening on %s (xiaozhi=%v transports=%d)", addr, h.XiaozhiServer() != nil, mounted))
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Info(fmt.Sprintf("[http] server error: %v", err))
		}
	}()
	go func() {
		<-ctx.Done()
		app.ShutdownAllSFUManagers()
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()
}

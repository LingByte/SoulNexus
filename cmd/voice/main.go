// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// voice — minimal runnable example (SIP / HTTP xiaozhi+WebRTC voice gateway).
//
// Features
//   - Answers any inbound SIP INVITE on UDP (UAS mode).
//   - Negotiates the first supported codec (PCMA/PCMU/G.722/Opus).
//   - Optional call recording: raw codec bytes streamed to a pkg/stores Store.
//   - Optional ASR echo demo: decoded PCM frames tapped into a user-supplied
//     transcriber (the scaffolding is fully wired; the bundled implementation
//     is a byte-counter so the demo runs without external dependencies).
//
// Usage
//
//	go run ./cmd/voice
//	go run ./cmd/voice -sip 0.0.0.0:5060 -local-ip 192.168.1.10 -record
//	go run ./cmd/voice -record -asr-echo
//
// Place a call to sip:anything@<host>:<port>. Hang up to tear the leg down.
// Recordings are written to $UPLOAD_DIR (default ./uploads) when -record is
// set.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/audio/rnnoise"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/server"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/vad"
	voicertc "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/webrtc"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

func main() {
	// Best-effort .env load (repo root / cwd). Ignore if missing.
	_ = godotenv.Overload(".env")
	sipAddr := flag.String("sip", "127.0.0.1:5060", "SIP UDP listen address (host:port)")
	localIP := flag.String("local-ip", "", "IP advertised in SDP c= (defaults to host from -sip)")
	rtpStart := flag.Int("rtp-start", 30000, "RTP port range start (inclusive)")
	rtpEnd := flag.Int("rtp-end", 30100, "RTP port range end (inclusive)")
	record := flag.Bool("record", false, "Record inbound RTP payload to the default Store (UPLOAD_DIR) per call")
	recordBucket := flag.String("record-bucket", "voiceserver-recordings", "Bucket name passed to the Store")
	recordChunk := flag.Duration("record-chunk", 30*time.Second, "Roll a partial recording up to the Store every interval as a crash-safety net. Each chunk is a self-contained stereo WAV named <callid>-part-<seq>-<ts>.wav and contains only the frames captured since the previous chunk; the final <callid>-<ts>.wav at hangup still has the complete call. 0 = disable rolling chunks (single final WAV only).")
	asrEcho := flag.Bool("asr-echo", false, "Enable real ASR+TTS echo: on every final transcript reply with -reply-text via QCloud TTS. Requires ASR_*/TTS_* env vars.")
	replyText := flag.String("reply-text", "您好，已收到", "Fixed reply played via TTS on every final ASR transcript when -asr-echo is set")
	dialogWS := flag.String("dialog-ws", "", "Dialog-plane WebSocket base URL (ws:// or wss://). Per-call call_id is appended by the gateway. SoulNexus /ws/call also needs apiKey, apiSecret, agentId: WebRTC browsers send them in POST /webrtc/v1/offer JSON; SIP/xiaozhi can set DIALOG_API_KEY, DIALOG_API_SECRET, DIALOG_AGENT_ID or add query params here. Requires ASR_*/TTS_* env vars.")
	httpAddr := flag.String("http", "", "If set, run a single HTTP listener on this address (e.g. 0.0.0.0:7080) hosting BOTH xiaozhi WS (ESP32 + browser web client) AND WebRTC signaling+demo. Each protocol mounts on its own path so they share one listener / one TLS cert / one auth surface. Requires -dialog-ws and ASR_*/TTS_* env vars. Optional env for WebRTC: WEBRTC_ICE_SERVERS (CSV stun:/turn:), WEBRTC_PUBLIC_IPS (NAT 1:1), WEBRTC_UDP_PORT (single-port ICE), WEBRTC_ALLOWED_ORIGINS (CORS).")
	xiaozhiPath := flag.String("xiaozhi-ws-path", "/xiaozhi/v1/", "Path the xiaozhi WS handler mounts on (relative to -http)")
	webrtcOfferPath := flag.String("webrtc-http-path", "/webrtc/v1/offer", "Path the WebRTC offer endpoint mounts on (relative to -http; companion /hangup and /demo are mounted alongside)")
	enableXiaozhi := flag.Bool("xiaozhi", true, "Mount the xiaozhi WS handler on -http (set =false to disable while keeping WebRTC)")
	enableWebRTC := flag.Bool("webrtc", true, "Mount the WebRTC signaling + demo handlers on -http (set =false to disable while keeping xiaozhi)")
	ttsPrewarm := flag.Bool("tts-prewarm", false, "On startup, synthesize a small set of canned phrases (welcome / hold / fallback) and prime the TTS PCM cache. Reduces first-byte latency for the first call but burns a few QCloud TTS calls at boot. Off by default — switch on for production or scripted demos.")
	bargeIn := flag.Bool("barge-in", true, "Enable barge-in across all transports (SIP / xiaozhi / WebRTC): when the user starts talking during TTS playback, immediately interrupt the TTS and emit a tts.interrupt event to the dialog plane. Uses an RMS-based VAD on the ASR pipeline's PCM path. Set =false to keep the AI talking over the user.")
	bargeInThreshold := flag.Float64("barge-in-threshold", 1500, "VAD RMS ceiling used for barge-in detection. Lower = more sensitive (browser auto-gain often needs ~800-1200); higher = only trigger on clearly spoken input. Has no effect when -barge-in=false.")
	bargeInFrames := flag.Int("barge-in-frames", 1, "Consecutive over-threshold frames required to fire barge-in. 1 = most responsive (~20ms) but may false-trigger on short sounds; 2-3 = more robust at ~40-60ms extra latency.")
	denoise := flag.Bool("denoise", false, "Enable RNNoise (Xiph) noise suppression on the WebRTC ASR feed. Requires the binary to be built with -tags rnnoise + CGO_ENABLED=1 + librnnoise installed; otherwise this silently falls back to passthrough at startup. Off by default because it is link-time conditional.")
	asrSentenceFilter := flag.Bool("asr-sentence-filter", false, "Suppress mid-sentence ASR partials and dedupe near-duplicate hypotheses before forwarding to the dialog plane. Reduces LLM thrash from chatty recognisers (QCloud / Volcengine) at the cost of ~1 partial of latency per turn. Off = legacy per-delta forwarding.")
	asrSentenceFilterSim := flag.Float64("asr-sentence-filter-similarity", 0.85, "Similarity threshold (0-1) used by -asr-sentence-filter to drop near-duplicate hypotheses. 0.85 catches cosmetic jitter (extra spaces, swapped punctuation) without merging genuinely different utterances. Has no effect when -asr-sentence-filter=false.")
	holdMessages := flag.String("hold-messages", "scripts/hold/hold_messages.json", "Path to a JSON file with hold-prompt phrases the gateway speaks while reconnecting a dropped dialog-plane WebSocket. Falls back to hardcoded Chinese defaults if the file is missing or malformed. Empty = use defaults.")
	dialogReconnect := flag.Int("dialog-reconnect", 3, "How many times the gateway will redial the dialog-plane WS after it drops mid-call. 0 = disable reconnect (legacy fail-fast). Each attempt uses exponential backoff starting from -dialog-reconnect-backoff capped at 30s.")
	dialogReconnectBackoff := flag.Duration("dialog-reconnect-backoff", time.Second, "Initial wait before the first dialog-plane redial. Doubles per attempt up to 30s.")
	outbound := flag.String("outbound", "", "One-shot client mode: dial this SIP URI (e.g. sip:bob@192.168.1.20:5060) instead of running as UAS")
	outboundHold := flag.Duration("outbound-hold", 15*time.Second, "How long to keep an outbound call up before sending BYE")
	enableSFU := flag.Bool("sfu", false, "Mount the multi-party SFU WebSocket signaling endpoint on -http (default off; needs -sfu-secret unless -sfu-allow-anon).")
	sfuPath := flag.String("sfu-ws-path", "/sfu/v1/ws", "Path the SFU signaling WS handler mounts on (relative to -http).")
	sfuSecret := flag.String("sfu-secret", "", "HMAC-SHA256 secret used to sign and verify SFU access tokens. Required unless -sfu-allow-anon. Tokens are minted by the business backend with sfu.NewAccessToken using the same secret.")
	sfuAllowAnon := flag.Bool("sfu-allow-anon", false, "Skip SFU access-token verification entirely. Dev/demo only — allows any client to join any room. Never enable in production.")
	sfuMaxParticipants := flag.Int("sfu-max-participants", 16, "Per-room participant cap. New joins past the cap are rejected with room_full.")
	sfuMaxRooms := flag.Int("sfu-max-rooms", 1024, "Process-wide cap on concurrent rooms. -1 = unlimited.")
	sfuAllowedOrigins := flag.String("sfu-allowed-origins", "", "CSV of allowed WebSocket Origin headers (e.g. https://app.example.com,https://other.example.com). Empty or '*' allows any. Origin matching is case-insensitive on scheme+host+port.")
	sfuTokenAdminSecret := flag.String("sfu-token-admin-secret", "", "If set, the built-in /token endpoint requires X-SFU-Admin: <secret> header. Without this, /token is disabled outside -sfu-allow-anon mode (production-safe default).")
	sfuRecord := flag.Bool("sfu-record", false, "Record each published audio track to the default Store. Webhook fires recording.finished with the public URL on each upload.")
	sfuRecordBucket := flag.String("sfu-record-bucket", "sfu-recordings", "Bucket name passed to the Store for SFU recordings.")
	sfuWebhookURL := flag.String("sfu-webhook-url", "", "If set, POST JSON events (room/participant/track/recording lifecycle) to this URL. Each request is signed with X-SFU-Signature = hex(HMAC_SHA256(-sfu-secret, body)).")
	// `go run` sets argv[0] to a temp path under /var/folders/... — override usage so -h is readable.
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of voice:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	ttsPrewarmFlag = *ttsPrewarm
	bargeInFlag = *bargeIn
	bargeInThresholdFlag = *bargeInThreshold
	bargeInFramesFlag = *bargeInFrames
	denoiseFlag = *denoise
	asrSentenceFilterFlag = *asrSentenceFilter
	asrSentenceFilterSimilarity = *asrSentenceFilterSim
	dialogReconnectFlag = *dialogReconnect
	dialogReconnectBackoffFlag = *dialogReconnectBackoff
	recordChunkFlag = *recordChunk
	loadHoldMessages(*holdMessages)

	dialogWSEffective, err := mergeDialogAuthQuery(*dialogWS)
	if err != nil {
		log.Fatalf("invalid -dialog-ws: %v", err)
	}
	warnDialogAuthIfNeeded(dialogWSEffective)

	host, port, err := splitHostPort(*sipAddr)
	if err != nil {
		log.Fatalf("invalid -sip: %v", err)
	}
	ip := strings.TrimSpace(*localIP)
	if ip == "" {
		ip = host
		if ip == "" || ip == "0.0.0.0" {
			ip = "127.0.0.1"
		}
	}

	// --- Outbound (UAC) mode: no UAS, place one call, exit. ---
	if strings.TrimSpace(*outbound) != "" {
		var rec *pcmRecorder
		if *record {
			rec = newRTPRecorder("outbound-"+newTag(), *recordBucket)
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := runOutbound(ctx, outboundConfig{
			TargetURI: *outbound,
			LocalIP:   ip,
			HoldFor:   *outboundHold,
			Recorder:  rec,
			AsrEcho:   *asrEcho,
			ReplyText: *replyText,
		}); err != nil {
			log.Fatalf("outbound failed: %v", err)
		}
		return
	}

	srv := server.New(server.Config{
		Host:             host,
		Port:             port,
		LocalIP:          ip,
		RTPPortStart:     *rtpStart,
		RTPPortEnd:       *rtpEnd,
		InviteRingbackMS: 200,
		InviteSend180:    true,
	})

	var callCount int64
	invite := &echoInviteHandler{
		serial:       &callCount,
		record:       *record,
		recordBucket: *recordBucket,
		gateways:     newGatewayRegistry(),
		asrEcho:      *asrEcho,
		replyText:    *replyText,
		dialogWS:     dialogWSEffective,
		srv:          srv,
	}
	// --- Persistence (voiceserver.db SQLite by default; VOICESERVER_DB=off disables) ---
	db, err := openVoiceServerDB()
	if err != nil {
		log.Printf("[persist] disabled: %v", err)
	}
	if db != nil {
		log.Printf("[persist] voiceserver.db ready")
	}
	invite.db = db

	srv.SetInviteHandler(invite)
	// echoInviteHandler also implements server.TransferHandler, so a
	// REFER on a live call surfaces as a transfer.request event on
	// the dialog plane and gets a 200 NOTIFY. Without this, REFER
	// would fall back to "501 Not Implemented".
	srv.SetTransferHandler(invite)
	srv.SetDTMFSink(&loggingDTMFSink{})
	srv.SetCallLifecycleObserver(&loggingObserver{})
	// Demo mode: accept INVITEs even when no tenant/DID binding is configured.
	srv.SetInboundAllowUnknownDID(true)

	if err := srv.Start(); err != nil {
		log.Fatalf("sip start: %v", err)
	}
	defer srv.Stop()

	h, p := srv.ListenAddr()
	log.Printf("voiceserver ready: sip=udp:%s:%d local_ip=%s rtp=%d..%d record=%v asr-echo=%v",
		h, p, ip, *rtpStart, *rtpEnd, *record, *asrEcho)
	log.Printf("place a call to sip:anyone@%s:%d to test", h, p)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Single HTTP listener for both xiaozhi WS and WebRTC signaling/demo.
	// One TCP port, one TLS surface (when fronted by a reverse proxy),
	// one auth boundary. Each protocol mounts on its own path so they
	// can be enabled/disabled independently with -xiaozhi / -webrtc.
	// -record / -record-bucket apply uniformly across all three
	// transports (SIP / xiaozhi / WebRTC) so call_recording rows and
	// stereo WAVs look identical regardless of how the call came in.
	startHTTPListener(ctx, httpListenerConfig{
		Addr:                *httpAddr,
		DialogWS:            dialogWSEffective,
		EnableXiaozhi:       *enableXiaozhi,
		XiaozhiPath:         *xiaozhiPath,
		EnableWebRTC:        *enableWebRTC,
		WebRTCOfferPath:     *webrtcOfferPath,
		EnableSFU:           *enableSFU,
		SFUPath:             *sfuPath,
		SFUSecret:           *sfuSecret,
		SFUAllowAnon:        *sfuAllowAnon,
		SFUMaxParticipants:  *sfuMaxParticipants,
		SFUMaxRooms:         *sfuMaxRooms,
		SFUAllowedOrigins:   *sfuAllowedOrigins,
		SFUTokenAdminSecret: *sfuTokenAdminSecret,
		SFURecord:           *sfuRecord,
		SFURecordBucket:     *sfuRecordBucket,
		SFUWebhookURL:       *sfuWebhookURL,
		DB:                  db,
		Record:              *record,
		RecordBucket:        *recordBucket,
	})

	<-ctx.Done()
	log.Printf("shutting down (handled %d calls)", atomic.LoadInt64(&callCount))
}

// ---------- InviteHandler ---------------------------------------------------

// gatewayRegistry tracks the live *gateway.Client for each accepted SIP
// call. The TransferHandler (REFER) and any other out-of-band SIP
// signal that needs to talk to the dialog plane mid-call looks up the
// client through this registry. We use a plain map+RWMutex rather than
// sync.Map because (a) the membership only churns on call start/end,
// and (b) callers want a single Get-or-nil idiom rather than the more
// verbose LoadOrStore dance.
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
	serial       *int64
	record       bool
	recordBucket string
	asrEcho      bool
	replyText    string
	dialogWS     string
	srv          *server.SIPServer
	db           *gorm.DB         // optional persistence; nil = disabled
	gateways     *gatewayRegistry // tracks per-call *gateway.Client for REFER lookup
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
		recorder = newRTPRecorder(inv.CallID, h.recordBucket)
	}

	// Optional persistence row created right at INVITE so we have a row even
	// if codec negotiation later fails.
	pers := newCallPersister(ctx, h.db, inv, persist.DirectionInbound)

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
	//   1) -dialog-ws set: bridge ASR/DTMF events out to the dialog app via
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
	pers.onAccept(ctx, leg, remoteRTP)
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
			pers.onTerminate(context.Background(), reason)
		},
	}, nil
}

// attachVoiceEcho wires QCloud ASR → final transcript → QCloud TTS("您好，已收到").
func attachVoiceEcho(ctx context.Context, leg *session.MediaLeg, callID string, callN int64, replyText string) (*voice.Attached, error) {
	asrAppID := strings.TrimSpace(os.Getenv("ASR_APPID"))
	asrSID := strings.TrimSpace(os.Getenv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(os.Getenv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(os.Getenv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(os.Getenv("TTS_APPID"))
	ttsSID := strings.TrimSpace(os.Getenv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(os.Getenv("TTS_SECRET_KEY"))
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
func attachVoiceGateway(ctx context.Context, leg *session.MediaLeg, inv *server.IncomingCall, callN int64, dialogURL string, srv *server.SIPServer, pers *callPersister) (*voice.Attached, *gateway.Client, error) {
	asrAppID := strings.TrimSpace(os.Getenv("ASR_APPID"))
	asrSID := strings.TrimSpace(os.Getenv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(os.Getenv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(os.Getenv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(os.Getenv("TTS_APPID"))
	ttsSID := strings.TrimSpace(os.Getenv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(os.Getenv("TTS_SECRET_KEY"))
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
		if ttsPrewarmEnabled() {
			go caching.Prewarm(ctx, prewarmTexts(), func(t string, e error) {
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
	var adapter gateway.SessionPersister = persisterAdapter{p: pers}

	gwCfg := gateway.ClientConfig{
		URL:      dialogURL,
		Attached: att,
		CallID:   inv.CallID,
		BargeIn:  newBargeInDetector(),
		OnHangup: func(reason string) {
			log.Printf("[call#%d][gw] hangup requested: %q", callN, reason)
			// Stamp a dialog.hangup event so the timeline carries the
			// reason the dialog plane asked us to drop the call. The
			// terminating call.terminated event comes later from the
			// SIP teardown path (OnTerminate).
			if pers != nil {
				pers.appendEvent(context.Background(),
					persist.EventKindDialogHangup, persist.EventLevelInfo,
					jsonObject(map[string]any{"reason": reason}))
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
	applyDialogReconnect(&gwCfg)
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
type pcmRecorder struct {
	callID    string
	bucket    string
	codec     string
	persister *callPersister

	mu  sync.Mutex
	rec *recorder.Recorder // late-init in setPCMRate
}

func newRTPRecorder(callID, bucket string) *pcmRecorder {
	return &pcmRecorder{callID: callID, bucket: bucket}
}

// setCodec stamps the negotiated wire codec for the flush log line. The
// recording itself is always PCM16 LE at the bridge rate, so codec is
// purely informational here.
func (r *pcmRecorder) setCodec(name string, _ int) {
	r.mu.Lock()
	r.codec = name
	if r.rec != nil {
		// Recorder.Codec is read at flush time; safe to leave unset
		// when the recorder was already constructed (its log line
		// will simply not include the codec name).
	}
	r.mu.Unlock()
}

// setPCMRate latches the bridge rate and constructs the underlying
// recorder. Idempotent — called once per call from the InviteHandler
// after MediaLeg is built.
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
		Bucket:     r.bucket,
		SampleRate: rate,
		Transport:  "sip",
		Codec:      r.codec,
	})
}

// inFilter is a MediaLeg InputFilter: caller-side decoded PCM16 mono.
// Returns (false, nil) — the recorder never drops packets.
func (r *pcmRecorder) inFilter(p media.MediaPacket) (bool, error) {
	if rec := r.recorder(); rec != nil {
		if ap, ok := p.(*media.AudioPacket); ok && ap != nil && len(ap.Payload) > 0 && !ap.IsSynthesized {
			rec.WriteCaller(ap.Payload)
		}
	}
	return false, nil
}

// outFilter is a MediaLeg OutputFilter: AI PCM16 mono before encode.
// Only synthesized (TTS / playback) frames are recorded.
func (r *pcmRecorder) outFilter(p media.MediaPacket) (bool, error) {
	if rec := r.recorder(); rec != nil {
		if ap, ok := p.(*media.AudioPacket); ok && ap != nil && len(ap.Payload) > 0 && ap.IsSynthesized {
			rec.WriteAI(ap.Payload)
		}
	}
	return false, nil
}

// flush builds and writes the stereo WAV once per call. Stamps both the
// legacy SIPCall.recording_url field and a structured call_recording
// row. Idempotent — second call no-ops.
func (r *pcmRecorder) flush() {
	rec := r.recorder()
	if rec == nil {
		return
	}
	info, ok := rec.Flush(context.Background())
	if !ok {
		log.Printf("[record] call=%s empty (no inbound/outbound PCM) or upload failed, skip", r.callID)
		return
	}
	log.Printf("[record] call=%s wrote %d bytes stereo WAV → %s",
		r.callID, info.Bytes, info.URL)
	if r.persister == nil {
		return
	}
	r.persister.onRecording(context.Background(), info.URL, info.Bytes)
	r.persister.appendRecording(context.Background(), info)
}

func (r *pcmRecorder) recorder() *recorder.Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rec
}

// voiceLogger builds a zap logger that pipes Info+ entries to stderr
// with a `[<tag>]` prefix. Earlier we passed zap.NewNop() into the
// xiaozhi/WebRTC servers because a real prod logger needed structured
// JSON; the side-effect was that any error inside startPipelines was
// swallowed and operators only saw "pipeline-error" with no cause.
// A thin console logger keeps the noise low (Info+) while preserving
// the diagnostic value of Error logs.
func voiceLogger(tag string) *zap.Logger {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(os.Stderr),
		zapcore.InfoLevel,
	)
	return zap.New(core).Named(tag)
}

// bargeInFlag / bargeInThresholdFlag / bargeInFramesFlag are set from
// main() right after flag.Parse(). All three transports (SIP / xiaozhi
// / WebRTC) read them through newBargeInDetector so one CLI flag
// uniformly toggles barge-in semantics across every call, regardless
// of which listener accepted it.
var (
	bargeInFlag          bool
	bargeInThresholdFlag float64
	bargeInFramesFlag    int
)

// newBargeInDetector returns a configured *vad.Detector, or nil if
// barge-in is disabled globally. Callers pass the returned value into
// gateway.ClientConfig.BargeIn — a nil detector is the documented
// "disabled" signal there, so no extra branching is needed at the
// call site.
func newBargeInDetector() *vad.Detector {
	if !bargeInFlag {
		return nil
	}
	d := vad.NewDetector()
	if bargeInThresholdFlag > 0 {
		d.SetThreshold(bargeInThresholdFlag)
	}
	if bargeInFramesFlag > 0 {
		d.SetConsecutiveFrames(bargeInFramesFlag)
	}
	return d
}

// denoiseFlag toggles per-call RNNoise wiring on the WebRTC ASR feed
// path. Off by default because the real implementation needs librnnoise
// available at link-time (-tags rnnoise + CGO_ENABLED=1); the stub
// build returns ErrUnavailable from rnnoise.New() and we silently
// fall back to passthrough.
var denoiseFlag bool

// newDenoiserOrNil constructs one denoiser per call, ready to wire
// into pkg/voice/asr.Pipeline.SetDenoiser. Returns nil when:
//   - the flag is off (operator opt-out), or
//   - the rnnoise package is the stub build, so New() returned
//     ErrUnavailable. Logging the fallback once on the first call is
//     enough — repeated identical lines per call would just be noise.
//
// The returned value satisfies the voicertc.Denoiser interface
// (Process + Close) which pkg/audio/rnnoise.Denoiser implements
// directly. Returning the interface value rather than the concrete
// pointer keeps the factory signature aligned with what
// webrtc.ServerConfig.DenoiserFactory expects, regardless of whether
// the rnnoise build tag is on (real CGO type) or off (stub type).
func newDenoiserOrNil() voicertc.Denoiser {
	if !denoiseFlag {
		return nil
	}
	d, err := rnnoise.New()
	if err != nil {
		denoiseFallbackOnce.Do(func() {
			log.Printf("[denoise] disabled: %v (build with -tags rnnoise to enable)", err)
		})
		return nil
	}
	return d
}

// denoiseFallbackOnce gates the "library missing" log message — see
// newDenoiserOrNil.
var denoiseFallbackOnce sync.Once

// asrSentenceFilterFlag toggles the per-call asr.SentenceFilter
// installed by applyDialogReconnect. asrSentenceFilterSimilarity is
// the threshold passed to NewSentenceFilter (0 disables the
// similarity check, leaving only the sentence-boundary buffering).
// Both are read together by every transport — keeping the knob in
// one place is consistent with how barge-in / denoise behave.
var (
	asrSentenceFilterFlag       bool
	asrSentenceFilterSimilarity float64
)

// recordChunkFlag is the rolling-recording cadence set from -record-chunk.
// Read by makeRecorderFactory in persister.go via recordingChunkInterval().
var recordChunkFlag time.Duration

// dialogReconnectFlag / dialogReconnectBackoffFlag are set from main()
// right after flag.Parse(). Both webrtc/xiaozhi/sip transports read
// them through applyDialogReconnect when constructing their gateway
// clients, so reconnect semantics are uniform across all transports.
var (
	dialogReconnectFlag        int
	dialogReconnectBackoffFlag time.Duration
)

// holdMessages mirrors the JSON file under scripts/hold/. Loaded once
// at startup; falls back to hardcoded Chinese defaults if the file is
// missing or malformed (logged at warn).
type holdMessageSet struct {
	First  string `json:"first_attempt"`
	Retry  string `json:"retry"`
	GiveUp string `json:"give_up"`
}

var holdMsgs = holdMessageSet{
	First:  "请稍等。",
	Retry:  "正在重新连接，请稍候。",
	GiveUp: "抱歉，当前服务暂时不可用，请稍后再试。",
}

// loadHoldMessages reads scripts/hold/hold_messages.json (or the path
// passed via -hold-messages) and overlays its values onto holdMsgs.
// Missing keys keep the defaults; missing file = no-op + log line.
func loadHoldMessages(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[hold] using built-in defaults (read %s: %v)", path, err)
		return
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("[hold] using built-in defaults (parse %s: %v)", path, err)
		return
	}
	if v, ok := raw["first_attempt"]; ok && strings.TrimSpace(v) != "" {
		holdMsgs.First = v
	}
	if v, ok := raw["retry"]; ok && strings.TrimSpace(v) != "" {
		holdMsgs.Retry = v
	}
	if v, ok := raw["give_up"]; ok && strings.TrimSpace(v) != "" {
		holdMsgs.GiveUp = v
	}
	log.Printf("[hold] loaded reconnect prompts from %s", path)
}

// applyDialogReconnect copies the per-process reconnect knobs into a
// gateway.ClientConfig. Transports call this right before NewClient so
// every transport gets identical reconnect behaviour from one set of
// flags. Idempotent — safe to call on already-populated configs.
func applyDialogReconnect(cfg *gateway.ClientConfig) {
	if cfg == nil {
		return
	}
	cfg.ReconnectAttempts = dialogReconnectFlag
	cfg.ReconnectInitialBackoff = dialogReconnectBackoffFlag
	cfg.HoldTextFirst = holdMsgs.First
	cfg.HoldTextRetry = holdMsgs.Retry
	cfg.HoldTextGiveUp = holdMsgs.GiveUp
	// Per-call SentenceFilter when the operator opted in. Constructed
	// fresh per-call (no shared state across calls), or nil when the
	// flag is off — preserving legacy per-token forwarding semantics.
	if asrSentenceFilterFlag {
		cfg.ASRSentenceFilter = asr.NewSentenceFilter(asrSentenceFilterSimilarity)
	}
}

// ttsPrewarmFlag is set from main() right after flag.Parse(). Both the
// SIP path (cmd/voiceserver/main.go) and the xiaozhi path
// (cmd/voiceserver/xiaozhi.go) read this single source of truth so a
// user toggling -tts-prewarm gets uniform behaviour across all
// transports — there is exactly one knob to think about.
var ttsPrewarmFlag bool

// ttsPrewarmEnabled is the read accessor — keeps the variable
// unexported and avoids accidental mutation outside main().
func ttsPrewarmEnabled() bool { return ttsPrewarmFlag }

// prewarmTexts returns the small set of phrases whose first-byte latency
// matters most for a smoke-test demo. Real deployments should replace this
// with a tenant- or scenario-specific list (welcome line, hold tones, …).
func prewarmTexts() []string {
	return []string{
		"您好，已收到",
		"听得到，您请讲。",
		"请稍等。",
		"您好，我现在没法回答这个问题，请稍后再试。",
	}
}

// ---------- DTMFSink --------------------------------------------------------

type loggingDTMFSink struct{}

func (loggingDTMFSink) OnDTMF(callID, digit string, end bool) {
	log.Printf("[dtmf] call=%s digit=%s end=%v", callID, digit, end)
}

// ---------- CallLifecycleObserver ------------------------------------------

type loggingObserver struct{}

func (loggingObserver) OnCallPreHangup(callID string) bool {
	log.Printf("[call=%s] pre-hangup", callID)
	return false // let the server send BYE
}

func (loggingObserver) OnCallCleanup(callID string) {
	log.Printf("[call=%s] cleanup", callID)
}

// ---------- helpers ---------------------------------------------------------

func splitHostPort(addr string) (string, int, error) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return "", 0, fmt.Errorf("parse port %q: %w", p, err)
	}
	if port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("port out of range: %d", port)
	}
	return h, port, nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"

	"github.com/LingByte/SoulNexus/internal/config"
	voicehttp "github.com/LingByte/SoulNexus/internal/handler/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/server"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadVoice(); err != nil {
		logger.Fatal(fmt.Sprintf("config load failed: %v", err))
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

	// --- Outbound (UAC) mode: no UAS, place one call, exit. ---
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
	// --- Persistence (shared DB_DRIVER + DSN with main / auth; DSN=off disables) ---
	db, err := app.OpenVoiceServerDB()
	if err != nil {
		logger.Info(fmt.Sprintf("[persist] disabled: %v", err))
	}
	if db != nil {
		logger.Info(fmt.Sprintf("[persist] voice db ready (driver=%s)", config.GlobalConfig.Database.Driver))
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
	// Local mode: accept INVITEs even when no tenant/DID binding is configured.
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

	// Single Gin HTTP listener for xiaozhi WS, WebRTC and SFU
	// signaling. One TCP port, one TLS surface (when fronted by a
	// reverse proxy), one auth boundary. Each protocol mounts on its
	// own path so they can be enabled/disabled independently. Record
	// settings apply uniformly across SIP / xiaozhi / WebRTC so
	// call_recording rows and stereo WAVs look identical regardless of
	// how the call came in.
	startVoiceHTTPListener(ctx, cfg.HTTPAddr, voicehttp.NewHandlers(voicehttp.Config{
		DialogWS:                       dialogWSEffective,
		DB:                             db,
		Record:                         cfg.Record,
		EnableXiaozhi:                  cfg.EnableXiaozhi,
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

// ---------- InviteHandler ---------------------------------------------------

// gatewayRegistry tracks the live *gateway.Client for each accepted SIP
// call. The TransferHandler (REFER) and any other out-of-band SIP
// signal that needs to talk to the dialog plane mid-call looks up the
// client through this registry. We use a plain map+RWMutex rather than
// sync.Map because (a) the membership only churns on call start/end,
// and (b) callers want a single Get-or-nil idiom rather than the more
// verbose LoadOrStore dance.
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
		// Stop SFU first so active participants get clean
		// "manager_shutdown" disconnect frames, then drain HTTP.
		app.ShutdownAllSFUManagers()
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()
}

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/LingByte/SoulNexus/internal/sipcampaign"
	"github.com/LingByte/SoulNexus/internal/sipacd"
	"github.com/LingByte/SoulNexus/internal/sippersist"
	"github.com/LingByte/SoulNexus/internal/sipreg"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/sip/server"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"github.com/LingByte/SoulNexus/pkg/sip/webseat"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// resolveOutboundDialTarget prefers DB (registered SIP_TARGET_NUMBER) when a store is wired; otherwise .env.
func resolveOutboundDialTarget(store *sipreg.GormStore) (outbound.DialTarget, bool) {
	if store != nil {
		n := strings.TrimSpace(utils.GetEnv(outbound.EnvSIPTargetNumber))
		if n != "" {
			if dt, ok := store.DialTargetForUsername(context.Background(), n); ok {
				return dt, true
			}
		}
	}
	return outbound.DialTargetFromEnv()
}

// sipDSNForLog prints DSN for logs (truncate very long strings; same file path matters for sqlite).
func sipDSNForLog(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "(empty)"
	}
	const max = 220
	if len(dsn) > max {
		return fmt.Sprintf("%s…(len=%d)", dsn[:max], len(dsn))
	}
	return dsn
}

func main() {
	host := flag.String("host", "0.0.0.0", "SIP UDP listen host")
	port := flag.Int("port", 5060, "SIP UDP listen port")
	localIP := flag.String("local-ip", "127.0.0.1", "SDP c= line IP (RTP reachable from your phone)")
	flag.Parse()

	// Some modules rely on the global logger; init with defaults.
	logDir := filepath.Join(".", "logs")
	_ = os.MkdirAll(logDir, 0o755)
	_ = logger.Init(&logger.LogConfig{
		Level:      "info",
		Filename:   filepath.Join(logDir, "sip.log"),
		MaxSize:    50,
		MaxAge:     14,
		MaxBackups: 5,
		Daily:      true,
	}, "dev")

	// Align with cmd/server: config.Load loads .env / .env.<APP_ENV> and fills GlobalConfig (incl. default DSN ./ling.db).
	if err := config.Load(); err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("sip: config.Load failed", zap.Error(err))
		}
	}

	sipHost := *host
	if sipHost == "0.0.0.0" {
		sipHost = *localIP
	}

	var sipServer *server.SIPServer
	var outboundHTTPSrv *http.Server
	var campaignHTTPSrv *sipcampaign.HTTPServer
	var sipRegStore *sipreg.GormStore
	var sipCallPersist *sippersist.Store
	var campaignSvc *sipcampaign.Service
	var acdDB *gorm.DB

	callerUser, callerDisplay := outbound.CallerIdentityFromEnv()
	outMgr := outbound.NewManager(outbound.ManagerConfig{
		LocalIP:         *localIP,
		SIPHost:         sipHost,
		SIPPort:         *port,
		FromUser:        callerUser,
		FromDisplayName: callerDisplay,
		MediaAttach: func(ctx context.Context, cs *sipSession.CallSession) error {
			var voiceLog *zap.Logger
			if logger.Lg != nil {
				voiceLog = logger.Lg.Named("sip-voice")
			}
			return conversation.AttachVoicePipeline(ctx, cs, voiceLog)
		},
		OnRegisterSession: func(callID string, cs *sipSession.CallSession) {
			if sipServer != nil {
				sipServer.RegisterCallSession(callID, cs)
			}
		},
		OnTransferBridge: func(correlationID string, cs *sipSession.CallSession, outboundCallID string) {
			conversation.StartTransferBridge(correlationID, cs, outboundCallID, nil)
		},
		OnScript: func(ctx context.Context, leg outbound.EstablishedLeg, scriptID string) {
			if campaignSvc != nil {
				campaignSvc.RunScriptIfConfigured(ctx, leg, scriptID)
			}
		},
		OnEvent: func(evt outbound.DialEvent) {
			if campaignSvc != nil {
				campaignSvc.HandleDialEvent(context.Background(), evt)
			}
		},
		OnEstablished: func(leg outbound.EstablishedLeg) {
			if campaignSvc != nil {
				campaignSvc.PrepareCallPrompt(leg.CallID, leg.CorrelationID)
			}
			if sipCallPersist == nil || leg.Session == nil {
				return
			}
			neg := leg.Session.NegotiatedCodec()
			rs := leg.Session.RTPSession()
			localRTP, remoteRTP := "", ""
			if rs != nil {
				if la := rs.LocalAddr; la != nil {
					localRTP = la.String()
				}
				if ra := rs.RemoteAddr; ra != nil {
					remoteRTP = ra.String()
				}
			}
			ctx := context.Background()
			sipCallPersist.OnInvite(ctx, sippersist.InviteParams{
				CallID:      leg.CallID,
				From:        leg.FromHeader,
				To:          leg.ToHeader,
				RemoteSig:   leg.RemoteSignalingAddr,
				RemoteRTP:   remoteRTP,
				LocalRTP:    localRTP,
				Codec:       neg.Name,
				PayloadType: neg.PayloadType,
				ClockRate:   neg.ClockRate,
				CSeqInvite:  leg.CSeqInvite,
				Direction:   "outbound",
			})
			sipCallPersist.OnEstablished(ctx, leg.CallID)
		},
	})

	sipServer = server.New(server.Config{
		Host:          *host,
		Port:          *port,
		LocalIP:       *localIP,
		OnSIPResponse: outMgr.HandleSIPResponse,
	})

	if config.GlobalConfig != nil {
		driver := strings.TrimSpace(config.GlobalConfig.Database.Driver)
		dsn := strings.TrimSpace(config.GlobalConfig.Database.DSN)
		if dsn != "" {
			if logger.Lg != nil {
				logger.Lg.Info("sip: opening database for persistence",
					zap.String("driver", driver),
					zap.String("dsn", sipDSNForLog(dsn)),
				)
			}
			db, err := utils.InitDatabase(nil, driver, dsn)
			if err != nil {
				if logger.Lg != nil {
					logger.Lg.Warn("sip: database unavailable, REGISTER / dialog persistence disabled", zap.Error(err))
				}
			} else {
				acdDB = db
				campaignSvc = sipcampaign.NewService(db)
				_ = campaignSvc.AutoMigrate()
				campaignSvc.StartWorker(outMgr)
				sipRegStore = sipreg.NewGormStore(db)
				sipServer.SetRegisterStore(sipRegStore)
				sipCallPersist = sippersist.New(db, logger.Lg)
				sipServer.SetCallPersist(sipCallPersist)
				conversation.SetSIPTurnPersist(func(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string) {
					sipCallPersist.SaveConversationTurn(ctx, callID, userText, assistantText, asrProvider, llmModel, ttsProvider)
				})
				conversation.SetTransferDialTargetResolver(func(ctx context.Context) (outbound.DialTarget, bool) {
					return sipacd.PickTransferDialTarget(ctx, acdDB, sipRegStore)
				})
				if logger.Lg != nil {
					logger.Lg.Info("sip: database persistence enabled — AI dialog JSON on sip_calls.turns; use same DSN as web if UI should see rows",
						zap.String("dsn", sipDSNForLog(dsn)),
					)
				}
			}
		} else if logger.Lg != nil {
			logger.Lg.Warn("sip: database DSN is empty in config — set DSN / DB_DRIVER like cmd/server")
		}
	}

	outMgr.BindSender(sipServer)
	conversation.SetTransferDialer(outMgr)
	conversation.SetInboundSessionLookup(func(callID string) *sipSession.CallSession {
		if sipServer == nil {
			return nil
		}
		return sipServer.GetCallSession(callID)
	})
	conversation.SetCallStore(sipServer)
	conversation.SetTransferPeerCallbacks(outMgr.SendBYE, sipServer.SendUASBye)
	conversation.SetSIPHangup(sipServer.HangupInboundCall)

	webseat.InitDefault(webseat.Config{
		RemoveCallSession:     sipServer.RemoveCallSession,
		ForgetUASDialog:       sipServer.ForgetUASDialog,
		SendUASBye:            sipServer.SendUASBye,
		ReleaseTransferDedupe: conversation.ReleaseTransferStartDedupe,
	})
	if wsAddr := strings.TrimSpace(utils.GetEnv(webseat.EnvHTTPAddr)); wsAddr != "" {
		if err := webseat.StartHTTPServer(wsAddr); err != nil && logger.Lg != nil {
			logger.Lg.Warn("webseat: http server failed", zap.String("addr", wsAddr), zap.Error(err))
		}
	}
	if outHTTPAddr := strings.TrimSpace(utils.GetEnv(outbound.EnvSIPOutboundHTTPAddr)); outHTTPAddr != "" {
		srv, err := outbound.StartDialHTTPServer(
			outHTTPAddr,
			strings.TrimSpace(utils.GetEnv(outbound.EnvSIPOutboundHTTPToken)),
			outMgr,
		)
		if err != nil && logger.Lg != nil {
			logger.Lg.Warn("sip outbound http: start failed", zap.String("addr", outHTTPAddr), zap.Error(err))
		} else {
			outboundHTTPSrv = srv
		}
	}
	if campaignSvc != nil {
		if campaignAddr := strings.TrimSpace(utils.GetEnv(sipcampaign.EnvCampaignHTTPAddr)); campaignAddr != "" {
			srv, err := sipcampaign.StartHTTPServer(
				campaignAddr,
				strings.TrimSpace(utils.GetEnv(sipcampaign.EnvCampaignHTTPToken)),
				campaignSvc,
			)
			if err != nil && logger.Lg != nil {
				logger.Lg.Warn("sip campaign http: start failed", zap.String("addr", campaignAddr), zap.Error(err))
			} else {
				campaignHTTPSrv = srv
			}
		}
	}
	conversation.SetWebSeatTransfer(conversation.StartWebSeatHandoff)

	if err := sipServer.Start(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "sip: start failed: %v\n", err)
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "sip: listening on udp %s:%d (SDP local-ip=%s)\n", *host, *port, *localIP)

	// Outbound: DB-registered SIP_TARGET_NUMBER when DSN + store wired; else .env (SIP_OUTBOUND_* / SIP_OUTBOUND_REQUEST_URI).
	if dt, ok := resolveOutboundDialTarget(sipRegStore); ok {
		_, _ = fmt.Fprintf(os.Stdout, "sip: outbound target from env: uri=%s signaling=%s\n", dt.RequestURI, dt.SignalingAddr)
		if outbound.AutoDialFromEnv() {
			go func() {
				cid, err := outMgr.Dial(context.Background(), outbound.DialRequest{
					Scenario:     outbound.ScenarioCampaign,
					Target:       dt,
					MediaProfile: outbound.MediaProfileAI,
				})
				if err != nil {
					if logger.Lg != nil {
						logger.Lg.Warn("sip outbound auto-dial failed", zap.Error(err))
					} else {
						_, _ = fmt.Fprintf(os.Stderr, "sip: outbound auto-dial: %v\n", err)
					}
					return
				}
				if logger.Lg != nil {
					logger.Lg.Info("sip outbound auto-dial started", zap.String("call_id", cid))
				} else {
					_, _ = fmt.Fprintf(os.Stdout, "sip: outbound auto-dial call_id=%s\n", cid)
				}
			}()
		}
	} else {
		// Hint when only SIP_TARGET_NUMBER is set (common misconfiguration).
		if strings.TrimSpace(utils.GetEnv(outbound.EnvSIPTargetNumber)) != "" {
			_, _ = fmt.Fprintf(os.Stderr, "sip: SIP_TARGET_NUMBER is set but outbound target is incomplete; set SIP_OUTBOUND_HOST (or SIP_OUTBOUND_REQUEST_URI + SIP_SIGNALING_ADDR). See docs/SIP_OUTBOUND_MODULE.md\n")
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	_, _ = fmt.Fprintln(os.Stdout, "sip: shutting down...")
	if outboundHTTPSrv != nil {
		shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = outboundHTTPSrv.Shutdown(shCtx)
		cancel()
	}
	if campaignHTTPSrv != nil {
		shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = campaignHTTPSrv.Shutdown(shCtx)
		cancel()
	}
	if campaignSvc != nil {
		campaignSvc.StopWorker()
	}
	_ = sipServer.Stop()
}

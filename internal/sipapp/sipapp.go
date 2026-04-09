// Package sipapp starts the SIP UDP stack, WebSeat WebSocket/HTTP, outbound dial HTTP,
// and campaign HTTP — shared by cmd/server (embedded) and cmd/sip (standalone).
package sipapp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
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

// Config controls the embedded SIP sidecars.
type Config struct {
	Host    string
	Port    int
	LocalIP string
	// DB when non-nil is reused (e.g. same pool as the web app). When nil, a connection is opened from GlobalConfig if DSN is set.
	DB *gorm.DB
}

// Embedded holds started subsystems for graceful shutdown.
type Embedded struct {
	sipServer       *server.SIPServer
	outboundHTTPSrv *http.Server
	campaignHTTPSrv *sipcampaign.HTTPServer
	campaignSvc     *sipcampaign.Service
	outMgr          *outbound.Manager
}

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

// Start wires outbound manager, SIP server, persistence, WebSeat, optional HTTP APIs, and starts UDP.
func Start(cfg Config) (*Embedded, error) {
	sipHost := cfg.Host
	if sipHost == "0.0.0.0" {
		sipHost = cfg.LocalIP
	}

	var sipServerPtr *server.SIPServer
	var sipRegStore *sipreg.GormStore
	var sipCallPersist *sippersist.Store
	var campaignSvc *sipcampaign.Service
	var acdDB *gorm.DB

	callerUser, callerDisplay := outbound.CallerIdentityFromEnv()
	outMgr := outbound.NewManager(outbound.ManagerConfig{
		LocalIP:         cfg.LocalIP,
		SIPHost:         sipHost,
		SIPPort:         cfg.Port,
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
			if sipServerPtr != nil {
				sipServerPtr.RegisterCallSession(callID, cs)
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

	sipServerPtr = server.New(server.Config{
		Host:          cfg.Host,
		Port:          cfg.Port,
		LocalIP:       cfg.LocalIP,
		OnSIPResponse: outMgr.HandleSIPResponse,
	})

	em := &Embedded{
		sipServer: sipServerPtr,
		outMgr:    outMgr,
	}

	if cfg.DB != nil {
		acdDB = cfg.DB
		campaignSvc = sipcampaign.NewService(cfg.DB)
		_ = campaignSvc.AutoMigrate()
		sipRegStore = sipreg.NewGormStore(cfg.DB)
		campaignSvc.SetDialTargetResolver(func(ctx context.Context, phone string) (outbound.DialTarget, bool) {
			return sipRegStore.DialTargetForUsername(ctx, phone)
		})
		campaignSvc.StartWorker(outMgr)
		sipServerPtr.SetRegisterStore(sipRegStore)
		sipCallPersist = sippersist.New(cfg.DB, logger.Lg)
		sipServerPtr.SetCallPersist(sipCallPersist)
		conversation.SetSIPTurnPersist(func(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string) {
			sipCallPersist.SaveConversationTurn(ctx, callID, userText, assistantText, asrProvider, llmModel, ttsProvider)
		})
		conversation.SetTransferDialTargetResolver(func(ctx context.Context) (outbound.DialTarget, bool) {
			return sipacd.PickTransferDialTarget(ctx, acdDB, sipRegStore)
		})
		if logger.Lg != nil {
			logger.Lg.Info("sipapp: using shared database handle from web server")
		}
		em.campaignSvc = campaignSvc
	} else if config.GlobalConfig != nil {
		driver := strings.TrimSpace(config.GlobalConfig.Database.Driver)
		dsn := strings.TrimSpace(config.GlobalConfig.Database.DSN)
		if dsn != "" {
			if logger.Lg != nil {
				logger.Lg.Info("sipapp: opening database for persistence",
					zap.String("driver", driver),
					zap.String("dsn", sipDSNForLog(dsn)),
				)
			}
			db, err := utils.InitDatabase(nil, driver, dsn)
			if err != nil {
				if logger.Lg != nil {
					logger.Lg.Warn("sipapp: database unavailable, REGISTER / dialog persistence disabled", zap.Error(err))
				}
			} else {
				acdDB = db
				campaignSvc = sipcampaign.NewService(db)
				_ = campaignSvc.AutoMigrate()
				sipRegStore = sipreg.NewGormStore(db)
				campaignSvc.SetDialTargetResolver(func(ctx context.Context, phone string) (outbound.DialTarget, bool) {
					return sipRegStore.DialTargetForUsername(ctx, phone)
				})
				campaignSvc.StartWorker(outMgr)
				sipServerPtr.SetRegisterStore(sipRegStore)
				sipCallPersist = sippersist.New(db, logger.Lg)
				sipServerPtr.SetCallPersist(sipCallPersist)
				conversation.SetSIPTurnPersist(func(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string) {
					sipCallPersist.SaveConversationTurn(ctx, callID, userText, assistantText, asrProvider, llmModel, ttsProvider)
				})
				conversation.SetTransferDialTargetResolver(func(ctx context.Context) (outbound.DialTarget, bool) {
					return sipacd.PickTransferDialTarget(ctx, acdDB, sipRegStore)
				})
				if logger.Lg != nil {
					logger.Lg.Info("sipapp: database persistence enabled",
						zap.String("dsn", sipDSNForLog(dsn)),
					)
				}
				em.campaignSvc = campaignSvc
			}
		} else if logger.Lg != nil {
			logger.Lg.Warn("sipapp: database DSN is empty in config — set DSN / DB_DRIVER like cmd/server")
		}
	}

	outMgr.BindSender(sipServerPtr)
	conversation.SetTransferDialer(outMgr)
	conversation.SetInboundSessionLookup(func(callID string) *sipSession.CallSession {
		if sipServerPtr == nil {
			return nil
		}
		return sipServerPtr.GetCallSession(callID)
	})
	conversation.SetCallStore(sipServerPtr)
	conversation.SetTransferPeerCallbacks(outMgr.SendBYE, sipServerPtr.SendUASBye)
	conversation.SetSIPHangup(func(callID string) {
		callID = strings.TrimSpace(callID)
		if callID == "" {
			return
		}
		if err := outMgr.SendBYE(callID); err == nil {
			if logger.Lg != nil {
				logger.Lg.Info("sip: hangup outbound BYE sent", zap.String("call_id", callID))
			}
			// Finalize local leg persistence immediately for outbound calls:
			// OnBye/end_status/recording_url are otherwise skipped when we only get 200 to BYE.
			sipServerPtr.HangupInboundCall(callID)
			return
		}
		sipServerPtr.HangupInboundCall(callID)
	})

	webseat.InitDefault(webseat.Config{
		RemoveCallSession:     sipServerPtr.RemoveCallSession,
		ForgetUASDialog:       sipServerPtr.ForgetUASDialog,
		SendUASBye:            sipServerPtr.SendUASBye,
		ReleaseTransferDedupe: conversation.ReleaseTransferStartDedupe,
		FinalizeInboundPersist: func(ctx context.Context, callID, initiator string, raw []byte, codecName string, recordSampleRate, recordOpusChannels int) {
			if sipCallPersist == nil {
				return
			}
			sipCallPersist.OnBye(ctx, sippersist.ByeParams{
				CallID:             callID,
				RawPayload:         raw,
				CodecName:          codecName,
				Initiator:          initiator,
				RecordSampleRate:   recordSampleRate,
				RecordOpusChannels: recordOpusChannels,
			})
		},
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
			em.outboundHTTPSrv = srv
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
				em.campaignHTTPSrv = srv
			}
		}
	}
	conversation.SetWebSeatTransfer(conversation.StartWebSeatHandoff)

	if err := sipServerPtr.Start(); err != nil {
		return nil, fmt.Errorf("sipapp: sip start: %w", err)
	}
	if logger.Lg != nil {
		logger.Lg.Info("sipapp: SIP UDP listening",
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port),
			zap.String("local_ip", cfg.LocalIP),
		)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "sipapp: listening on udp %s:%d (SDP local-ip=%s)\n", cfg.Host, cfg.Port, cfg.LocalIP)
	}

	if dt, ok := resolveOutboundDialTarget(sipRegStore); ok {
		if logger.Lg != nil {
			logger.Lg.Info("sipapp: outbound target from env",
				zap.String("uri", dt.RequestURI),
				zap.String("signaling", dt.SignalingAddr),
			)
		} else {
			_, _ = fmt.Fprintf(os.Stdout, "sipapp: outbound target from env: uri=%s signaling=%s\n", dt.RequestURI, dt.SignalingAddr)
		}
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
						_, _ = fmt.Fprintf(os.Stderr, "sipapp: outbound auto-dial: %v\n", err)
					}
					return
				}
				if logger.Lg != nil {
					logger.Lg.Info("sip outbound auto-dial started", zap.String("call_id", cid))
				} else {
					_, _ = fmt.Fprintf(os.Stdout, "sipapp: outbound auto-dial call_id=%s\n", cid)
				}
			}()
		}
	} else {
		if strings.TrimSpace(utils.GetEnv(outbound.EnvSIPTargetNumber)) != "" {
			_, _ = fmt.Fprintf(os.Stderr, "sipapp: SIP_TARGET_NUMBER is set but outbound target is incomplete; set SIP_OUTBOUND_HOST (or SIP_OUTBOUND_REQUEST_URI + SIP_SIGNALING_ADDR). See docs/SIP_OUTBOUND_MODULE.md\n")
		}
	}

	return em, nil
}

// Shutdown stops outbound/campaign HTTP servers, campaign worker, and SIP UDP.
func (e *Embedded) Shutdown(ctx context.Context) {
	if e == nil {
		return
	}
	if logger.Lg != nil {
		logger.Lg.Info("sipapp: shutting down")
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "sipapp: shutting down...")
	}
	if e.outboundHTTPSrv != nil {
		shCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		_ = e.outboundHTTPSrv.Shutdown(shCtx)
		cancel()
	}
	if e.campaignHTTPSrv != nil {
		shCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		_ = e.campaignHTTPSrv.Shutdown(shCtx)
		cancel()
	}
	if e.campaignSvc != nil {
		e.campaignSvc.StopWorker()
	}
	if e.sipServer != nil {
		_ = e.sipServer.Stop()
	}
}

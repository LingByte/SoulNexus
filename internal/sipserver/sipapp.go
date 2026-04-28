package sipserver

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
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
	DB      *gorm.DB // DB when non-nil is reused (e.g. same pool as the web app). When nil, a connection is opened from GlobalConfig if DSN is set.
}

// Embedded holds started subsystems for graceful shutdown.
type Embedded struct {
	sipServer   *server.SIPServer
	campaignSvc *CampaignService
	outMgr      *outbound.Manager
}

func (e *Embedded) CampaignService() *CampaignService {
	if e == nil {
		return nil
	}
	return e.campaignSvc
}

func resolveOutboundDialTarget(store *GormStore) (outbound.DialTarget, bool) {
	if store != nil {
		n := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTargetNumber))
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

// Start wires outbound manager, SIP server, DB persistence, WebSeat hub, and starts UDP.
func Start(cfg Config) (*Embedded, error) {
	// SDP c=/Call-ID host: CLI (cfg.LocalIP) first; empty → SIP_LOCAL_IP (.env / process env).
	// If both miss, outbound/server still default to 127.0.0.1 inside their packages.
	localIP := strings.TrimSpace(cfg.LocalIP)
	if localIP == "" {
		localIP = strings.TrimSpace(utils.GetEnv("SIP_LOCAL_IP"))
	}

	sipHost := cfg.Host
	if sipHost == "0.0.0.0" {
		sipHost = localIP
	}

	var sipServerPtr *server.SIPServer
	var sipRegStore *GormStore
	var sipCallPersist *Store
	var campaignSvc *CampaignService
	var acdDB *gorm.DB

	callerUser, callerDisplay := config.CallerIdentityFromEnv()
	outMgr := outbound.NewManager(outbound.ManagerConfig{
		LocalIP:         localIP,
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
			sipCallPersist.OnInvite(ctx, server.InvitePersistParams{
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
		LocalIP:       localIP,
		OnSIPResponse: outMgr.HandleSIPResponse,
	})

	em := &Embedded{
		sipServer: sipServerPtr,
		outMgr:    outMgr,
	}

	if cfg.DB != nil {
		acdDB = cfg.DB
		campaignSvc = NewCampaignService(cfg.DB)
		sipRegStore = NewGormStore(cfg.DB)
		campaignSvc.SetDialTargetResolver(func(ctx context.Context, phone string) (outbound.DialTarget, bool) {
			return sipRegStore.DialTargetForUsername(ctx, phone)
		})
		campaignSvc.StartWorker(outMgr)
		sipServerPtr.SetRegisterStore(sipRegStore)
		sipCallPersist = New(cfg.DB, logger.Lg)
		sipServerPtr.SetCallPersist(sipCallPersist)
		conversation.SetSIPTurnPersist(func(ctx context.Context, callID string, turn conversation.DialogTurn) {
			sipCallPersist.SaveConversationTurn(ctx, callID, turn)
		})
		conversation.SetTransferDialTargetResolver(func(ctx context.Context, inboundCallID string) (outbound.DialTarget, bool) {
			return PickTransferDialTarget(ctx, acdDB, sipRegStore, inboundCallID)
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
				campaignSvc = NewCampaignService(db)
				sipRegStore = NewGormStore(db)
				campaignSvc.SetDialTargetResolver(func(ctx context.Context, phone string) (outbound.DialTarget, bool) {
					return sipRegStore.DialTargetForUsername(ctx, phone)
				})
				campaignSvc.StartWorker(outMgr)
				sipServerPtr.SetRegisterStore(sipRegStore)
				sipCallPersist = New(db, logger.Lg)
				sipServerPtr.SetCallPersist(sipCallPersist)
				conversation.SetSIPTurnPersist(func(ctx context.Context, callID string, turn conversation.DialogTurn) {
					sipCallPersist.SaveConversationTurn(ctx, callID, turn)
				})
				conversation.SetTransferDialTargetResolver(func(ctx context.Context, inboundCallID string) (outbound.DialTarget, bool) {
					return PickTransferDialTarget(ctx, acdDB, sipRegStore, inboundCallID)
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
		SetACDWebSeatWorkState: func(ctx context.Context, targetID uint, workState string) error {
			if acdDB == nil || targetID == 0 {
				return nil
			}
			return models.UpdateACDPoolTargetWorkState(ctx, acdDB, targetID, workState, "sip")
		},
		FinalizeInboundPersist: func(ctx context.Context, callID, initiator string, raw []byte, codecName string, recordSampleRate, recordOpusChannels int) {
			if sipCallPersist == nil {
				return
			}
			sipCallPersist.OnBye(ctx, server.ByePersistParams{
				CallID:             callID,
				RawPayload:         raw,
				CodecName:          codecName,
				Initiator:          initiator,
				RecordSampleRate:   recordSampleRate,
				RecordOpusChannels: recordOpusChannels,
			})
		},
	})
	conversation.SetWebSeatTransfer(conversation.StartWebSeatHandoff)

	if err := sipServerPtr.Start(); err != nil {
		return nil, fmt.Errorf("sipapp: sip start: %w", err)
	}
	if logger.Lg != nil {
		logger.Lg.Info("sipapp: SIP UDP listening",
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port),
			zap.String("local_ip_effective", localIP),
			zap.String("local_ip_from_cli", strings.TrimSpace(cfg.LocalIP)),
		)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "sipapp: listening on udp %s:%d (SDP local-ip effective=%q cli=%q)\n", cfg.Host, cfg.Port, localIP, strings.TrimSpace(cfg.LocalIP))
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
		if config.AutoDialFromEnv() {
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
		if strings.TrimSpace(utils.GetEnv(constants.EnvSIPTargetNumber)) != "" {
			_, _ = fmt.Fprintf(os.Stderr, "sipapp: SIP_TARGET_NUMBER is set but outbound target is incomplete; set SIP_OUTBOUND_HOST (or SIP_OUTBOUND_REQUEST_URI + SIP_SIGNALING_ADDR). See docs/SIP_OUTBOUND_MODULE.md\n")
		}
	}

	return em, nil
}

// Shutdown stops the campaign worker and SIP UDP.
func (e *Embedded) Shutdown(ctx context.Context) {
	if e == nil {
		return
	}
	if logger.Lg != nil {
		logger.Lg.Info("sipapp: shutting down")
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "sipapp: shutting down...")
	}
	if e.campaignSvc != nil {
		e.campaignSvc.StopWorker()
	}
	if e.sipServer != nil {
		_ = e.sipServer.Stop()
	}
}

// PickTransferDialTarget selects one row from acd_pool_targets for blind transfer (DTMF).
// Eligible: not deleted, weight > 0, work_state = available, route_type sip or web.
// Ordering: weight DESC, id ASC (highest weight wins; tie-break lower id first).
//   - web → WebSeat (browser agent leg).
//   - sip trunk → DialTargetFromACDTrunk; sip internal → reg.DialTargetForUsername.
//
// SIP rows: sipCallerId / sipCallerDisplayName copied onto DialTarget when set.
// Web rows: when inboundCallID is set, marks the row ringing and binds call_id → row for webseat ACD state updates.
func PickTransferDialTarget(ctx context.Context, db *gorm.DB, reg *GormStore, inboundCallID string) (outbound.DialTarget, bool) {
	if db == nil {
		return outbound.DialTarget{}, false
	}
	row, err := models.PickEligibleACDPoolTargetForTransfer(ctx, db)
	if err != nil {
		return outbound.DialTarget{}, false
	}

	if row.RouteType == models.ACDPoolRouteTypeWeb {
		if strings.TrimSpace(inboundCallID) != "" {
			if err := models.UpdateACDPoolTargetWorkState(ctx, db, row.ID, models.ACDWorkStateRinging, "sip-transfer"); err == nil {
				webseat.BindInboundCallToWebACD(strings.TrimSpace(inboundCallID), row.ID)
			}
		}
		return outbound.DialTarget{WebSeat: true}, true
	}

	var dt outbound.DialTarget
	src := strings.ToLower(strings.TrimSpace(row.SipSource))
	switch src {
	case models.ACDSipSourceTrunk:
		t, ok := outbound.DialTargetFromACDTrunk(row.TargetValue, row.SipTrunkHost, row.SipTrunkSignalingAddr, row.SipTrunkPort)
		if !ok {
			return outbound.DialTarget{}, false
		}
		dt = t
	default:
		if reg == nil {
			return outbound.DialTarget{}, false
		}
		u := strings.TrimSpace(row.TargetValue)
		if u == "" {
			return outbound.DialTarget{}, false
		}
		t, ok := reg.DialTargetForUsername(ctx, u)
		if !ok {
			return outbound.DialTarget{}, false
		}
		dt = t
	}
	dt.CallerUser = strings.TrimSpace(row.SipCallerID)
	dt.CallerDisplayName = strings.TrimSpace(row.SipCallerDisplayName)
	return dt, true
}

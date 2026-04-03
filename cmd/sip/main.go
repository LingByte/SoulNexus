package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/internal/sipreg"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/outbound"
	"github.com/LingByte/SoulNexus/pkg/sip/server"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
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

	sipHost := *host
	if sipHost == "0.0.0.0" {
		sipHost = *localIP
	}

	var sipServer *server.SIPServer
	var sipRegStore *sipreg.GormStore

	outMgr := outbound.NewManager(outbound.ManagerConfig{
		LocalIP: *localIP,
		SIPHost: sipHost,
		SIPPort: *port,
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
	})

	sipServer = server.New(server.Config{
		Host:          *host,
		Port:          *port,
		LocalIP:       *localIP,
		OnSIPResponse: outMgr.HandleSIPResponse,
	})

	if strings.TrimSpace(utils.GetEnv(constants.ENV_DSN)) != "" {
		db, err := utils.InitDatabase(nil, "", "")
		if err != nil {
			if logger.Lg != nil {
				logger.Lg.Warn("sip: database unavailable, REGISTER not persisted", zap.Error(err))
			}
		} else if err := utils.MakeMigrates(db, []any{&models.SIPUser{}}); err != nil {
			if logger.Lg != nil {
				logger.Lg.Warn("sip: sip_users migrate failed", zap.Error(err))
			}
		} else {
			sipRegStore = sipreg.NewGormStore(db)
			sipServer.SetRegisterStore(sipRegStore)
			conversation.SetTransferDialTargetResolver(func(ctx context.Context) (outbound.DialTarget, bool) {
				if sipRegStore == nil {
					return outbound.DialTarget{}, false
				}
				u := strings.TrimSpace(utils.GetEnv(outbound.EnvSIPTransferNumber))
				if u == "" {
					return outbound.DialTarget{}, false
				}
				return sipRegStore.DialTargetForUsername(ctx, u)
			})
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
	_ = sipServer.Stop()
}

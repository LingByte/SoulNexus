// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voice

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/sfu"
)

// mountSFU registers the WebSocket-Web (SFU) multi-party signaling
// endpoints. Independent from the 1v1 WebRTC handler because the SFU
// does not bridge to a dialog plane — it only forwards media between
// human peers. Both share the WEBRTC_* STUN/TURN env so operators don't
// need a parallel set of variables.
//
// Mounted endpoints:
//   - {SFUPath}                                   WS upgrade
//   - {root}/token  (when not anon and admin secret is set)  dev-only token minter
//
// Returns true on successful mount, false when configuration prevents
// startup. The HTTP listener keeps running so misconfiguration here
// doesn't kill SIP/xiaozhi.
func (h *Handlers) mountSFU(r gin.IRoutes) bool {
	cfg := h.cfg
	if !cfg.SFUAllowAnon && strings.TrimSpace(cfg.SFUSecret) == "" {
		log.Printf("[sfu] disabled: VOICE_SFU_SECRET is required (or set VOICE_SFU_ALLOW_ANON=true for local dev)")
		return false
	}

	logger, _ := zap.NewProduction()
	if logger == nil {
		logger = zap.NewNop()
	}

	mgrCfg := &sfu.Config{
		AuthSecret:             cfg.SFUSecret,
		AllowUnauthenticated:   cfg.SFUAllowAnon,
		MaxParticipantsPerRoom: cfg.SFUMaxParticipants,
		MaxRooms:               cfg.SFUMaxRooms,
		AllowedOrigins:         app.SplitCSV(cfg.SFUAllowedOrigins),
		EnableRecording:        cfg.SFURecord,
		RecordBucket:           cfg.SFURecordBucket,
		WebhookURL:             cfg.SFUWebhookURL,
		PublicIPs:              app.SplitCSV(utils.GetEnv("WEBRTC_PUBLIC_IPS")),
		SinglePort:             app.ParseSinglePort(utils.GetEnv("WEBRTC_UDP_PORT")),
	}
	// Reuse the WebRTC ICE servers env so operators only configure
	// STUN/TURN once. Fall back to pion's normalise default (Google
	// public STUN) when nothing is set.
	if servers, err := app.ParseICEServerList(utils.GetEnv("WEBRTC_ICE_SERVERS")); err == nil && len(servers) > 0 {
		mgrCfg.ICEServers = servers
	}

	mgr, err := sfu.NewManager(mgrCfg, logger.Named("sfu"))
	if err != nil {
		log.Printf("[sfu] init failed: %v", err)
		return false
	}

	wsPath := cfg.SFUPath
	if wsPath == "" {
		wsPath = "/sfu/v1/ws"
	}
	r.GET(wsPath, gin.WrapF(mgr.ServeWS))

	rootPath := wsPath
	if i := strings.LastIndex(wsPath, "/"); i > 0 {
		rootPath = wsPath[:i]
	}

	// Dev-only token minter: signs an SFU access token from query
	// params (?room=&identity=). In anon mode returns an empty token
	// because the WS handler accepts an empty token when
	// AllowUnauthenticated is on.
	//
	// Security: when VOICE_SFU_TOKEN_ADMIN_SECRET is set, callers must
	// present `X-SFU-Admin: <secret>` header to mint tokens. This is
	// the supported path for production deployments where you want the
	// SFU itself to sign tokens (rather than your business backend).
	// Without the env var, the endpoint refuses requests in non-anon
	// mode — preventing accidental public exposure.
	tokenPath := rootPath + "/token"
	adminSecret := strings.TrimSpace(cfg.SFUTokenAdminSecret)
	r.GET(tokenPath, func(c *gin.Context) {
		if cfg.SFUAllowAnon {
			c.JSON(http.StatusOK, gin.H{"token": "", "anon": true})
			return
		}
		if adminSecret == "" {
			c.String(http.StatusForbidden, "token endpoint disabled (set VOICE_SFU_TOKEN_ADMIN_SECRET to enable)")
			return
		}
		if c.GetHeader("X-SFU-Admin") != adminSecret {
			c.String(http.StatusForbidden, "forbidden")
			return
		}
		room := strings.TrimSpace(c.Query("room"))
		identity := strings.TrimSpace(c.Query("identity"))
		if room == "" || identity == "" {
			c.String(http.StatusBadRequest, "room and identity required")
			return
		}
		tok, err := sfu.NewAccessToken(cfg.SFUSecret, sfu.AccessTokenClaims{
			Room:     room,
			Identity: identity,
			Name:     strings.TrimSpace(c.Query("name")),
		})
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tok})
	})

	// Register Manager.Close for context-cancel teardown so the webhook
	// goroutine doesn't leak when the parent server shuts down.
	app.RegisterSFUManagerForShutdown(mgr)

	log.Printf("[sfu] mounted: ws=%s anon=%v record=%v webhook=%s",
		wsPath, cfg.SFUAllowAnon, cfg.SFURecord, cfg.SFUWebhookURL)
	return true
}

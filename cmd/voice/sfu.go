// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

// SFU mount glue. The SFU itself lives in pkg/voice/sfu — this file is
// just the cmd-side wiring: turn flags / env into a sfu.Config, spin
// up a sfu.Manager, and attach its WS handler to the merged HTTP mux.
//
// Independent from the 1v1 WebRTC handler (cmd/voiceserver/webrtc.go)
// because the SFU does not bridge to a dialog plane — it only forwards
// media between human peers. They share STUN/TURN env (WEBRTC_*) so
// operators don't need a parallel set of variables.

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/sfu"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// sfuDemoHTML is a minimal browser test page: capture mic + camera,
// open an SFU session, mount remote tracks as <video>/<audio> tags as
// peers join. Works against -sfu-allow-anon for zero-config demos.
//
//go:embed sfu_demo.html
var sfuDemoHTML []byte

// mountSFUHandlers creates the SFU manager and attaches:
//
//   - {SFUPath}            WS upgrade endpoint
//   - {root}/demo          embedded test page
//   - {root}/token         dev-only token minter (active when
//                           SFUAllowAnon=false AND a non-empty secret;
//                           refuses requests when SFUAllowAnon=true to
//                           avoid the demo page implying tokens are
//                           needed)
//
// Returns true on successful mount, false otherwise; the listener
// keeps running so misconfiguration here doesn't kill SIP/xiaozhi.
func mountSFUHandlers(mux *http.ServeMux, cfg httpListenerConfig) bool {
	if !cfg.SFUAllowAnon && strings.TrimSpace(cfg.SFUSecret) == "" {
		log.Printf("[sfu] disabled: -sfu-secret is required (or pass -sfu-allow-anon for dev)")
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
		AllowedOrigins:         splitCSV(cfg.SFUAllowedOrigins),
		EnableRecording:        cfg.SFURecord,
		RecordBucket:           cfg.SFURecordBucket,
		WebhookURL:             cfg.SFUWebhookURL,
		PublicIPs:              splitCSV(getEnv("WEBRTC_PUBLIC_IPS")),
		SinglePort:             parseSinglePort(getEnv("WEBRTC_UDP_PORT")),
	}
	// Reuse the WebRTC ICE servers env so operators only configure
	// STUN/TURN once. Fall back to pion's normalise default (Google
	// public STUN) when nothing is set.
	if servers, err := parseICEServerList(getEnv("WEBRTC_ICE_SERVERS")); err == nil && len(servers) > 0 {
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
	mux.HandleFunc(wsPath, mgr.ServeWS)

	rootPath := wsPath
	if i := strings.LastIndex(wsPath, "/"); i > 0 {
		rootPath = wsPath[:i]
	}
	demoPath := rootPath + "/demo"
	mux.HandleFunc(demoPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(sfuDemoHTML)
	})

	// Dev-only token minter: signs an SFU access token from query
	// params (?room=&identity=). In anon mode returns an empty token
	// so the bundled demo's fetch never errors out (the WS handler
	// accepts an empty token when AllowUnauthenticated is on).
	//
	// Security: when -sfu-token-admin-secret is set, callers must
	// present `X-SFU-Admin: <secret>` header to mint tokens. This is
	// the supported path for production deployments where you want the
	// SFU itself to sign tokens (rather than your business backend).
	// Without the env var, the endpoint refuses requests in non-anon
	// mode — preventing accidental public exposure.
	tokenPath := rootPath + "/token"
	adminSecret := strings.TrimSpace(cfg.SFUTokenAdminSecret)
	mux.HandleFunc(tokenPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if cfg.SFUAllowAnon {
			_, _ = w.Write([]byte(`{"token":"","anon":true}`))
			return
		}
		if adminSecret == "" {
			http.Error(w, "token endpoint disabled (set -sfu-token-admin-secret to enable)", http.StatusForbidden)
			return
		}
		if r.Header.Get("X-SFU-Admin") != adminSecret {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		room := strings.TrimSpace(r.URL.Query().Get("room"))
		identity := strings.TrimSpace(r.URL.Query().Get("identity"))
		if room == "" || identity == "" {
			http.Error(w, "room and identity required", http.StatusBadRequest)
			return
		}
		tok, err := sfu.NewAccessToken(cfg.SFUSecret, sfu.AccessTokenClaims{
			Room:     room,
			Identity: identity,
			Name:     strings.TrimSpace(r.URL.Query().Get("name")),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"token":"` + tok + `"}`))
	})

	// Register Manager.Close for context-cancel teardown so the webhook
	// goroutine doesn't leak when the parent server shuts down.
	registerSFUManagerForShutdown(mgr)

	log.Printf("[sfu] mounted: ws=%s demo=%s anon=%v record=%v webhook=%s",
		wsPath, demoPath, cfg.SFUAllowAnon, cfg.SFURecord, cfg.SFUWebhookURL)
	return true
}

// sfuManagers tracks every Manager constructed by mountSFUHandlers so
// the process shutdown path can Close() them and reclaim the webhook
// goroutine + active participants. Slice + mutex over a sync.Map
// because we only iterate it once at shutdown and the call count is
// tiny (typically 1).
var (
	sfuManagersMu sync.Mutex
	sfuManagers   []*sfu.Manager
)

func registerSFUManagerForShutdown(m *sfu.Manager) {
	sfuManagersMu.Lock()
	sfuManagers = append(sfuManagers, m)
	sfuManagersMu.Unlock()
}

// shutdownAllSFUManagers is called by the HTTP listener teardown to
// stop every registered Manager. Idempotent on Manager.Close.
func shutdownAllSFUManagers() {
	sfuManagersMu.Lock()
	mgrs := append([]*sfu.Manager(nil), sfuManagers...)
	sfuManagers = nil
	sfuManagersMu.Unlock()
	for _, m := range mgrs {
		m.Close()
	}
}

// parseICEServerList reuses voicertc.ParseICEServers semantics in a
// pion-typed shape. Local fork to avoid importing the 1v1 webrtc
// package just for one helper.
func parseICEServerList(spec string) ([]pionwebrtc.ICEServer, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	out := make([]pionwebrtc.ICEServer, 0)
	for _, raw := range strings.Split(spec, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		// Format: <url>[|user|pass]
		parts := strings.Split(raw, "|")
		s := pionwebrtc.ICEServer{URLs: []string{parts[0]}}
		if len(parts) >= 3 {
			s.Username = parts[1]
			s.Credential = parts[2]
		}
		out = append(out, s)
	}
	return out, nil
}

// getEnv is a tiny os.Getenv wrapper with TrimSpace so empty-after-
// whitespace env vars are treated as unset.
func getEnv(key string) string {
	return strings.TrimSpace(osGetenv(key))
}

// osGetenv is split out so the splitCSV / parseSinglePort helpers in
// webrtc.go can be reused without an import cycle.
var osGetenv = func(key string) string { return os.Getenv(key) }

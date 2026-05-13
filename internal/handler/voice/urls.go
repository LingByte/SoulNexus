// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voice

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/stores"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/metrics"
)

// Register mounts every voice transport handler on r and returns the
// number of transports that successfully mounted (excluding /healthz,
// /metrics and /media which are always-on infra endpoints). The caller
// is expected to keep the listener up even when zero transports
// mounted, so /healthz still answers — useful for kubernetes probes
// during a misconfigured deployment.
//
// Mounted endpoints:
//   - cfg.XiaozhiPath                          → WS-Hardware (xiaozhi)
//   - cfg.SoulnexusHardwarePath                → WS-Hardware (binding wrapper)
//   - cfg.WebRTCOfferPath / cfg.WebRTCOfferPath/../hangup → WebRTC
//   - cfg.SFUPath / .../token                  → WS-Web (SFU)
//   - /healthz                                 → liveness probe
//   - /metrics                                 → Prometheus scrape
//   - /media/* (when stores.Default() is local) → recording file server
func (h *Handlers) Register(r gin.IRoutes) int {
	mounted := 0

	// WebSocket-Hardware: xiaozhi WS adapter (ESP32 firmware + browser web client).
	if h.cfg.EnableXiaozhi {
		if h.mountXiaozhi(r) {
			mounted++
		}
	}

	// WebSocket-Hardware: SoulNexus binding wrapper (legacy ESP32 path).
	hwPath := strings.TrimSpace(h.cfg.SoulnexusHardwarePath)
	hwBind := strings.TrimSpace(h.cfg.SoulnexusHardwareBindingURL)
	switch {
	case h.xiaozhiSrv != nil && hwPath != "" && hwBind != "":
		h.mountSoulnexusHardware(r)
		mounted++
	case hwPath != "" && hwBind == "":
		log.Printf("[lingecho-hw] not mounted: SoulnexusHardwareBindingURL empty (set VOICE_SOULNEXUS_HW_BINDING_URL or LINGECHO_HARDWARE_BINDING_URL)")
	case hwPath != "" && h.xiaozhiSrv == nil:
		log.Printf("[lingecho-hw] not mounted: xiaozhi server unavailable (fix ASR/TTS / VOICE_DIALOG_WS)")
	}

	// WebRTC: 1v1 browser ↔ AI signaling.
	if h.cfg.EnableWebRTC {
		if h.mountWebRTC(r) {
			mounted++
		}
	}

	// WS-Web: SFU multi-party signaling.
	if h.cfg.EnableSFU {
		if h.mountSFU(r) {
			mounted++
		}
	}

	// Always-on infra endpoints.
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok\n") })
	// Prometheus scrape endpoint — exposes active-call gauges, total
	// counters, and latency quantile summaries (E2E first-byte, TTS
	// TTFB, LLM TTFB). Mounted on the same port as the protocol
	// handlers so one firewall hole / one scrape target per host.
	// No auth by default: treat this port as internal-only or put a
	// reverse proxy with IP allowlist in front for public deployments.
	r.GET("/metrics", gin.WrapH(metrics.Handler()))
	// /media/ serves files written by stores.LocalStore so that
	// LocalStore.PublicURL paths actually resolve. Cloud backends bypass
	// this (their PublicURL is already absolute), so we only mount when
	// the active default store is local.
	if local, ok := stores.Default().(*stores.LocalStore); ok {
		mountMediaFileServer(r, local)
	}

	return mounted
}

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

// http_listener.go owns the single HTTP listener that hosts BOTH the
// xiaozhi WebSocket adapter AND the WebRTC signaling/demo endpoints.
// Earlier revisions ran two separate net.Listeners on different ports;
// that meant two TCP sockets, two TLS certificates (when fronted by a
// reverse proxy), two firewall rules, and two health checks — for
// protocols that always run together. Merging them means:
//
//   - One -http addr to configure (e.g. 0.0.0.0:7080)
//   - One TLS / auth boundary at the proxy
//   - Each protocol mounts on its own URL path, e.g. /xiaozhi/v1/ and
//     /webrtc/v1/{offer,hangup,demo}, so they don't collide
//   - Either protocol can be disabled independently with -xiaozhi=false
//     or -webrtc=false; the listener still starts as long as ≥1 mount
//     succeeds

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/stores"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/metrics"
	"gorm.io/gorm"
)

// httpListenerConfig wires up the merged HTTP server. Zero-value Addr
// disables the listener entirely (SIP keeps running).
type httpListenerConfig struct {
	Addr            string
	DialogWS        string
	EnableXiaozhi   bool
	XiaozhiPath     string
	EnableWebRTC    bool
	WebRTCOfferPath string
	// SFU multi-party signaling. EnableSFU=false keeps the rest of the
	// HTTP listener running; SFUSecret is required when EnableSFU=true
	// and SFUAllowAnon=false.
	EnableSFU            bool
	SFUPath              string
	SFUSecret            string
	SFUAllowAnon         bool
	SFUMaxParticipants   int
	SFUMaxRooms          int
	SFUAllowedOrigins    string // CSV; "*" allows any
	SFUTokenAdminSecret  string // required to call /token outside anon mode
	SFURecord            bool
	SFURecordBucket      string
	SFUWebhookURL        string
	DB                 *gorm.DB
	Record             bool
	RecordBucket       string
}

// startHTTPListener mounts the requested protocols on a single mux and
// starts a background HTTP server. Idempotent on context cancel — the
// server is shut down with a 3 s grace period when ctx fires.
func startHTTPListener(ctx context.Context, cfg httpListenerConfig) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		// No -http set: xiaozhi & WebRTC are both disabled. Print a
		// hint only if someone tried to enable them via flag without
		// supplying an address.
		if cfg.EnableXiaozhi || cfg.EnableWebRTC {
			log.Printf("[http] disabled: -http is empty (xiaozhi/webrtc require an HTTP listener)")
		}
		return
	}

	mux := http.NewServeMux()
	mounted := 0
	if cfg.EnableXiaozhi {
		if mountXiaozhiHandlers(mux, cfg.XiaozhiPath, cfg.DialogWS, cfg.DB, cfg.Record, cfg.RecordBucket) {
			mounted++
		}
	}
	if cfg.EnableWebRTC {
		if mountWebRTCHandlers(mux, cfg.WebRTCOfferPath, cfg.DialogWS, cfg.DB, cfg.Record, cfg.RecordBucket) {
			mounted++
		}
	}
	if cfg.EnableSFU {
		if mountSFUHandlers(mux, cfg) {
			mounted++
		}
	}
	// Health probe — useful for kubernetes / docker healthchecks. Always
	// mounted, even when both protocols failed to attach, so a load
	// balancer can still verify the process is up.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	// Prometheus scrape endpoint — exposes active-call gauges, total
	// counters, and latency quantile summaries (E2E first-byte, TTS
	// TTFB, LLM TTFB). Mounted on the same port as the protocol
	// handlers so one firewall hole / one scrape target per host.
	// No auth by default: treat this port as internal-only or put a
	// reverse proxy with IP allowlist in front for public deployments.
	mux.Handle("/metrics", metrics.Handler())
	// /media/ serves files written by stores.LocalStore so that
	// LocalStore.PublicURL paths actually resolve. Cloud backends bypass
	// this (their PublicURL is already absolute), so we only mount when
	// the active default store is local. The on-disk Root and the URL
	// prefix can be tuned with UPLOAD_DIR and LOCAL_MEDIA_URL_PREFIX.
	if local, ok := stores.Default().(*stores.LocalStore); ok {
		mountMediaFileServer(mux, local)
	}
	// Root index — discoverability for humans hitting http://host:7080/
	// in a browser. Lists both demos with the protocols they exercise so
	// you don't have to remember the paths.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(rootIndexHTML))
	})

	if mounted == 0 {
		log.Printf("[http] %s reachable, but no protocol handlers attached (xiaozhi=%v webrtc=%v); only /healthz will respond",
			addr, cfg.EnableXiaozhi, cfg.EnableWebRTC)
	}

	httpSrv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		log.Printf("[http] listening on %s (xiaozhi=%v webrtc=%v)", addr, cfg.EnableXiaozhi, cfg.EnableWebRTC)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[http] server error: %v", err)
		}
	}()
	go func() {
		<-ctx.Done()
		// Stop SFU first so active participants get clean
		// "manager_shutdown" disconnect frames, then drain HTTP.
		shutdownAllSFUManagers()
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()
}

// mountMediaFileServer wires an http.FileServer at the LocalStore's URL
// prefix (default /media/) rooted at its on-disk Root. Read-only by design
// — POST/PUT are rejected to keep the surface tiny. Add an auth middleware
// here if recordings ever carry private data; today the assumption is that
// this listener already sits behind your edge proxy / auth boundary.
func mountMediaFileServer(mux *http.ServeMux, local *stores.LocalStore) {
	prefix := strings.TrimSpace(local.URLPrefix)
	if prefix == "" {
		prefix = stores.DefaultLocalURLPrefix
	}
	prefix = "/" + strings.Trim(prefix, "/") + "/"
	root := strings.TrimSpace(local.Root)
	if root == "" {
		root = stores.UploadDir
	}
	fs := http.StripPrefix(strings.TrimSuffix(prefix, "/"), http.FileServer(http.Dir(root)))
	mux.Handle(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Cache headers tuned for immutable WAV recordings — same key never
		// gets rewritten because the recorder embeds a timestamp.
		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
		fs.ServeHTTP(w, r)
	}))
	log.Printf("[http] media file server mounted: %s -> %s", prefix, root)
}

// rootIndexHTML is served at "/" so a fresh visit to the merged HTTP
// listener lands on a tiny menu page linking the two browser demos
// (xiaozhi WS + WebRTC). Inlined as a string rather than embedded
// because the markup is small enough to read in-place and avoids
// growing the //go:embed surface for a 1 KB file.
const rootIndexHTML = `<!doctype html>
<html lang="zh"><head><meta charset="utf-8"><title>VoiceServer</title>
<style>
body{font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
     background:#0e0f12;color:#e6e6e6;max-width:640px;margin:60px auto;padding:0 24px}
h1{font-size:20px;margin:0 0 6px}.sub{color:#8b8e95;margin-bottom:32px;font-size:13px}
a.card{display:block;text-decoration:none;color:inherit;background:#14161a;
       border:1px solid #25282e;border-radius:8px;padding:18px;margin-bottom:14px}
a.card:hover{border-color:#4ea1ff}
a.card h3{margin:0 0 4px;color:#4ea1ff;font-size:15px}
a.card p{margin:0;color:#8b8e95;font-size:12px}
code{background:#0a0b0e;padding:2px 6px;border-radius:3px;font-size:12px}
</style></head><body>
<h1>VoiceServer</h1>
<div class="sub">三种通话方式共用一份 dialog 后端：SIP / xiaozhi / WebRTC。下面两个浏览器 demo 不需要安装任何依赖。</div>
<a class="card" href="/xiaozhi/demo"><h3>xiaozhi WS demo</h3>
<p>浏览器模拟 ESP32：getUserMedia 抓 16 kHz PCM → xiaozhi WS → 接收 AI 文本 + 语音播放。<br>
WS endpoint: <code>/xiaozhi/v1/</code></p></a>
<a class="card" href="/webrtc/v1/demo"><h3>WebRTC 1v1 AI 通话 demo</h3>
<p>浏览器与 voiceserver 走完整 WebRTC（ICE / DTLS-SRTP / Opus）。麦克风输入 + 实时统计面板。<br>
Offer endpoint: <code>/webrtc/v1/offer</code></p></a>
<a class="card" href="/sfu/v1/demo"><h3>SFU 多人通话 demo</h3>
<p>多端浏览器加入同一个房间，通过 Selective Forwarding Unit 转发音视频。支持 simulcast / ICE restart / per-track 录音。<br>
WS endpoint: <code>/sfu/v1/ws</code> · 启动需 <code>-sfu -sfu-secret=xxx</code> 或 <code>-sfu-allow-anon</code></p></a>
<p style="color:#8b8e95;font-size:11px;margin-top:32px">健康检查 <code>/healthz</code> · SIP 端口 见 voiceserver 启动日志</p>
</body></html>`

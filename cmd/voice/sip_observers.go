// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
 
package main
 
import (
	"log"
)

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

// ---------- merged cmd/voice files ----------

// startHTTPListener owns the single Gin HTTP listener that hosts BOTH the
// xiaozhi WebSocket adapter AND the WebRTC signaling endpoints.
// Earlier revisions ran two separate net.Listeners on different ports;
// that meant two TCP sockets, two TLS certificates (when fronted by a
// reverse proxy), two firewall rules, and two health checks — for
// protocols that always run together. Merging them means:
//
//   - One VOICE_HTTP_ADDR to configure (e.g. 0.0.0.0:7080)
//   - One TLS / auth boundary at the proxy
//   - Each protocol mounts on its own URL path, e.g. /xiaozhi/v1/ and
//     /webrtc/v1/{offer,hangup}, so they don't collide
//   - Either protocol can be disabled independently with VOICE_ENABLE_XIAOZHI
//     or VOICE_ENABLE_WEBRTC; the listener still starts as long as ≥1 mount succeeds.

// SoulNexus hardware (formerly "LingEcho" ESP32 firmware) historically opened
// WebSocket on SoulNexus /api/voice/lingecho/v1/ with Device-Id. The preferred
// path is the same xiaozhi adapter as the browser (cmd/voice) with dialog
// credentials merged via ?payload= — this file mounts an extra WS URL that
// performs the SoulNexus binding HTTP call before delegating to
// xiaozhi.Server.Handle. URL paths and the X-Lingecho-Voice-Secret header are
// preserved for backward compatibility with deployed firmware.

// outboundConfig configures a one-shot UAC call from the CLI.

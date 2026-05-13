// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	vsutils "github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"gorm.io/gorm"
)

// SplitCSV parses a comma-separated list, trimming whitespace and
// dropping empty entries. Returns nil for empty input so callers can
// pass the result straight into "len(...) > 0" checks.
func SplitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseSinglePort returns the integer port encoded in s, or 0 when s is
// empty / unparseable. Used for WEBRTC_UDP_PORT-style single-port
// muxing config.
func ParseSinglePort(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var p int
	_, _ = fmt.Sscanf(s, "%d", &p)
	return p
}

// SplitHostPort wraps net.SplitHostPort with port range validation and
// numeric parsing. Returns the host and the integer port, or an error
// if the address is malformed.
func SplitHostPort(addr string) (string, int, error) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return "", 0, fmt.Errorf("parse port %q: %w", p, err)
	}
	if port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("port out of range: %d", port)
	}
	return h, port, nil
}

// ParseICEServerList parses a comma-separated list of ICE server specs
// of the form `<url>[|user|pass]` into pion ICEServer values. Returns
// (nil, nil) on empty input; an error means the spec was malformed at
// the URL level (currently never produced — kept for future
// validation).
func ParseICEServerList(spec string) ([]pionwebrtc.ICEServer, error) {
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

// VoiceLogger builds a zap logger that pipes Info+ entries to stderr
// with a `[<tag>]` prefix. Earlier we passed zap.NewNop() into the
// xiaozhi/WebRTC servers because a real prod logger needed structured
// JSON; the side-effect was that any error inside startPipelines was
// swallowed and operators only saw "pipeline-error" with no cause. A
// thin console logger keeps the noise low (Info+) while preserving the
// diagnostic value of Error logs.
func VoiceLogger(tag string) *zap.Logger {
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(os.Stderr),
		zapcore.InfoLevel,
	)
	return zap.New(core).Named(tag)
}

// OpenVoiceServerDB initialises the SQLite voiceserver.db (default file
// path); the path can be overridden with VOICESERVER_DB. Returns
// (nil, nil) when persistence is disabled via VOICESERVER_DB="off"|
// "none".
func OpenVoiceServerDB() (*gorm.DB, error) {
	dsn := strings.TrimSpace(utils.GetEnv("VOICESERVER_DB"))
	if dsn == "" {
		dsn = "voiceserver.db"
	}
	if strings.EqualFold(dsn, "off") || strings.EqualFold(dsn, "none") {
		return nil, nil
	}
	db, err := vsutils.InitDatabase(nil, "", dsn)
	if err != nil {
		return nil, err
	}
	if err := persist.Migrate(db); err != nil {
		_ = CloseGormDB(db)
		return nil, err
	}
	return db, nil
}

// CloseGormDB closes the underlying *sql.DB handle. nil-safe.
func CloseGormDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// LogStartupBanner prints a single line summarising the voice server
// listener layout, useful for spotting misconfigurations at boot.
func LogStartupBanner(host string, port, rtpStart, rtpEnd int, localIP string, record, asrEcho bool) {
	log.Printf("voiceserver ready: sip=udp:%s:%d local_ip=%s rtp=%d..%d record=%v asr-echo=%v",
		host, port, localIP, rtpStart, rtpEnd, record, asrEcho)
}

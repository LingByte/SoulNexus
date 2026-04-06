package transport

import (
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

func webrtcDebug() bool {
	v := strings.TrimSpace(os.Getenv("WEBRTC_DEBUG"))
	return v == "1" || strings.EqualFold(v, "true")
}

func dbg(msg string, fields ...zap.Field) {
	if !webrtcDebug() || logger.Lg == nil {
		return
	}
	logger.Lg.Debug("webrtc-ai "+msg, fields...)
}

func aiInfo(msg string, fields ...zap.Field) {
	if logger.Lg == nil {
		return
	}
	logger.Lg.Info("webrtc-ai "+msg, fields...)
}

func aiWarn(msg string, fields ...zap.Field) {
	if logger.Lg == nil {
		return
	}
	logger.Lg.Warn("webrtc-ai "+msg, fields...)
}

// legacyLog bridges old printf-style lines to zap Info (avoids std log spam).
func legacyLog(format string, v ...interface{}) {
	if logger.Lg == nil {
		return
	}
	logger.Lg.Sugar().Infof("webrtc-ai "+format, v...)
}

// dbgLog prints verbose diagnostics only when WEBRTC_DEBUG is set.
func dbgLog(format string, v ...interface{}) {
	if !webrtcDebug() || logger.Lg == nil {
		return
	}
	logger.Lg.Sugar().Debugf("webrtc-ai "+format, v...)
}

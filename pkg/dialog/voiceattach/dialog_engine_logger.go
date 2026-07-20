package voiceattach

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

func bridgeZapLogger(lg engine.Logger) *zap.Logger {
	if zw, ok := lg.(*zapEngineLogger); ok && zw != nil && zw.z != nil {
		return zw.z
	}
	if logger.Lg != nil {
		return logger.Lg
	}
	z, _ := zap.NewDevelopment()
	return z
}

type zapEngineLogger struct{ z *zap.Logger }

// NewZapEngineLogger wraps z as an engine.Logger.
func NewZapEngineLogger(z *zap.Logger) engine.Logger {
	if z == nil {
		return engine.NopLogger{}
	}
	return &zapEngineLogger{z: z}
}

func (l *zapEngineLogger) Debug(msg string, fields ...engine.Field) {
	l.z.Debug(msg, zapFields(fields)...)
}
func (l *zapEngineLogger) Info(msg string, fields ...engine.Field) {
	l.z.Info(msg, zapFields(fields)...)
}
func (l *zapEngineLogger) Warn(msg string, fields ...engine.Field) {
	l.z.Warn(msg, zapFields(fields)...)
}
func (l *zapEngineLogger) Error(msg string, fields ...engine.Field) {
	l.z.Error(msg, zapFields(fields)...)
}
func (l *zapEngineLogger) With(fields ...engine.Field) engine.Logger {
	if len(fields) == 0 {
		return l
	}
	return &zapEngineLogger{z: l.z.With(zapFields(fields)...)}
}

func zapFields(in []engine.Field) []zap.Field {
	if len(in) == 0 {
		return nil
	}
	out := make([]zap.Field, 0, len(in))
	for _, f := range in {
		key := strings.TrimSpace(f.Key)
		if key == "" {
			key = "field"
		}
		out = append(out, zap.Any(key, f.Value))
	}
	return out
}

package logger

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HTTP header and Gin context key for request correlation (aligned with root logger package).
const (
	HeaderXReqID   = "X-Reqid"
	GinCtxReqIDKey = "x-reqid"
)

var genReqPID = uint32(time.Now().UnixNano() % 4294967291)

// GenReqID generates a URL-safe request id (same algorithm as github.com/.../logger at repo root).
func GenReqID() string {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[:], genReqPID)
	binary.LittleEndian.PutUint64(b[4:], uint64(time.Now().UnixNano()))
	return base64.URLEncoding.EncodeToString(b[:])
}

// WithRequestID stores req id on context for InfoCtx / ErrorCtx and downstream RPC.
func WithRequestID(ctx context.Context, reqID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if reqID == "" {
		return ctx
	}
	return context.WithValue(ctx, RequestIDKey, reqID)
}

// RequestIDFromContext reads the request id from context.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(RequestIDKey); v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// ZapReqID returns a zap field for x-reqid when present on context.
func ZapReqID(ctx context.Context) zap.Field {
	id := RequestIDFromContext(ctx)
	if id == "" {
		return zap.Skip()
	}
	return zap.String("x-reqid", id)
}

// ZapReqIDString returns a zap field for a raw request id string.
func ZapReqIDString(reqID string) zap.Field {
	if reqID == "" {
		return zap.Skip()
	}
	return zap.String("x-reqid", reqID)
}

// reqIDFieldKeys are the field keys that carry the per-request id.
// Kept in sync with HeaderXReqID / GinCtxReqIDKey defined in reqid.go.
// "reqid" is accepted as a short alias if callers ever start emitting it
// — we strip both to avoid double printing.
var reqIDFieldKeys = map[string]struct{}{
	"x-reqid": {},
	"reqid":   {},
}

type reqIDPrefixCore struct {
	zapcore.Core
	reqid string // accumulated via With(...)
}

// WrapCoreWithReqIDPrefix returns a Core that strips request-id fields
// and prepends "[reqid:<value>] " to each entry's message.
// inner must be non-nil; nil short-circuits to inner to avoid hiding
// configuration mistakes.
func WrapCoreWithReqIDPrefix(inner zapcore.Core) zapcore.Core {
	if inner == nil {
		return inner
	}
	return &reqIDPrefixCore{Core: inner}
}

// With extracts the latest x-reqid from `fields` (if any), keeps it on
// the returned core, and forwards the remaining fields to the inner
// core. The previously accumulated reqid is preserved when the With
// batch carries none, so nested With(...) calls don't drop it.
func (c *reqIDPrefixCore) With(fields []zapcore.Field) zapcore.Core {
	reqid, filtered := extractReqID(fields, c.reqid)
	return &reqIDPrefixCore{
		Core:  c.Core.With(filtered),
		reqid: reqid,
	}
}

// Check must add this wrapping core (not the inner one) so Write
// observes per-call fields. Without overriding Check, zap would route
// writes straight to the inner core and our message rewriting would be
// skipped.
func (c *reqIDPrefixCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write strips x-reqid from per-call fields, falls back to the
// accumulated value from With(...), and rewrites the message prefix.
func (c *reqIDPrefixCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	reqid, filtered := extractReqID(fields, c.reqid)
	if reqid != "" {
		ent.Message = "[reqid:" + reqid + "] " + ent.Message
	}
	return c.Core.Write(ent, filtered)
}

func (c *reqIDPrefixCore) Sync() error { return c.Core.Sync() }

// extractReqID scans fields for any request-id key and returns
//   - the latest value (or fallback when none found),
//   - the field slice with reqid entries removed.
//
// Returns fields unchanged (same underlying array) when nothing matches
// to avoid an allocation on the hot path.
func extractReqID(fields []zapcore.Field, fallback string) (string, []zapcore.Field) {
	hit := false
	for _, f := range fields {
		if _, ok := reqIDFieldKeys[f.Key]; ok {
			hit = true
			break
		}
	}
	if !hit {
		return fallback, fields
	}
	reqid := fallback
	filtered := make([]zapcore.Field, 0, len(fields))
	for _, f := range fields {
		if _, ok := reqIDFieldKeys[f.Key]; ok {
			if f.Type == zapcore.StringType && f.String != "" {
				reqid = f.String
			}
			continue
		}
		filtered = append(filtered, f)
	}
	return reqid, filtered
}

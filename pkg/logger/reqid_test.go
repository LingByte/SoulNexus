package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestGenReqID_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, GenReqID())
}

func TestWithRequestID_ContextRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "rid-1")
	assert.Equal(t, "rid-1", RequestIDFromContext(ctx))
}

func TestExtractReqID_NoHit(t *testing.T) {
	reqid, filtered := extractReqID([]zapcore.Field{zap.String("user", "x")}, "fb")
	assert.Equal(t, "fb", reqid)
	assert.Len(t, filtered, 1)
}

func TestExtractReqID_MultipleKeys(t *testing.T) {
	reqid, filtered := extractReqID([]zapcore.Field{
		zap.String("x-reqid", "abc"),
		zap.String("reqid", "def"),
	}, "")
	assert.Equal(t, "def", reqid)
	assert.Len(t, filtered, 0)
}

func TestExtractReqID_EmptyStringValue(t *testing.T) {
	reqid, filtered := extractReqID([]zapcore.Field{zap.String("x-reqid", "")}, "fb")
	assert.Equal(t, "fb", reqid)
	assert.Len(t, filtered, 0)
}

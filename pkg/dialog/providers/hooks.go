package providers

import (
	"context"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
	"github.com/LingByte/SoulNexus/pkg/logger"
)

var (
	turnPersist func(ctx context.Context, callID string, t turn.Turn)
	hangupFn    func(callID string)

	warnTurnPersistOnce sync.Once
)

// SetTurnPersist registers dialog turn persistence (wired from bootstrap).
func SetTurnPersist(fn func(ctx context.Context, callID string, t turn.Turn)) {
	turnPersist = fn
}

// SetHangup registers server-side hangup (wired from host when available).
func SetHangup(fn func(callID string)) {
	hangupFn = fn
}

// RecordTurn appends one dialog turn when persistence is wired.
func RecordTurn(ctx context.Context, callID string, t turn.Turn) {
	if callID == "" {
		return
	}
	if turnPersist == nil {
		warnTurnPersistOnce.Do(func() {
			if logger.Lg != nil {
				logger.Lg.Warn("voice: AI dialog turns are not persisted (providers.SetTurnPersist not wired)")
			}
		})
		return
	}
	turnPersist(ctx, callID, t)
}

// RequestHangup ends the session when hangup is wired.
func RequestHangup(callID string) {
	if callID == "" || hangupFn == nil {
		return
	}
	hangupFn(callID)
}

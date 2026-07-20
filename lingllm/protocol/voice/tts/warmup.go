package tts

import (
	"context"
	"time"
)

// Warmup issues a tiny synthesis request so the next real segment avoids cold-start latency.
func Warmup(ctx context.Context, svc TTSService) {
	if svc == nil {
		return
	}
	go func() {
		c, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		_ = svc.Synthesize(c, "好", func([]byte) error { return nil })
	}()
}

// WarmupPipeline warms the pipeline's configured TTS service.
func (p *TTSPipeline) Warmup(ctx context.Context) {
	if p == nil {
		return
	}
	Warmup(ctx, p.ttsService)
}

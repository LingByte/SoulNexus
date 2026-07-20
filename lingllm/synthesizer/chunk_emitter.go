package synthesizer

import (
	"context"
	"time"

	"github.com/LingByte/lingllm/media"
)

// PCMEmitConfig controls fixed-frame PCM delivery to handlers.
type PCMEmitConfig struct {
	SampleRate int
	BitDepth   int
	Channels   int
	FrameMS    int
}

// PCMEmitConfigFromFormat builds emit config from a stream format.
func PCMEmitConfigFromFormat(f media.StreamFormat) PCMEmitConfig {
	cfg := PCMEmitConfig{
		SampleRate: f.SampleRate,
		BitDepth:   f.BitDepth,
		Channels:   f.Channels,
		FrameMS:    20,
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}
	if cfg.BitDepth <= 0 {
		cfg.BitDepth = 16
	}
	if cfg.Channels <= 0 {
		cfg.Channels = 1
	}
	if f.FrameDuration > 0 {
		ms := int(f.FrameDuration / time.Millisecond)
		if ms > 0 {
			cfg.FrameMS = ms
		}
	}
	return cfg
}

// FrameBytes returns PCM bytes for one frame from cfg.
func FrameBytes(cfg PCMEmitConfig) int {
	if cfg.FrameMS <= 0 {
		cfg.FrameMS = 20
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 16000
	}
	if cfg.BitDepth <= 0 {
		cfg.BitDepth = 16
	}
	if cfg.Channels <= 0 {
		cfg.Channels = 1
	}
	return cfg.SampleRate * cfg.BitDepth / 8 * cfg.Channels * cfg.FrameMS / 1000
}

// EmitPCMChunks delivers batch PCM to handler as fixed-size frames.
// Batch vendors should use this so all engines share the same push-chunk contract.
func EmitPCMChunks(ctx context.Context, handler AudioSynthesisHandler, pcm []byte, cfg PCMEmitConfig) error {
	if handler == nil {
		return nil
	}
	if len(pcm) == 0 {
		handler.OnMessage(nil)
		return nil
	}
	chunk := FrameBytes(cfg)
	if chunk <= 0 {
		chunk = 640
	}
	for i := 0; i < len(pcm); i += chunk {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		end := i + chunk
		if end > len(pcm) {
			end = len(pcm)
		}
		handler.OnMessage(pcm[i:end])
	}
	return nil
}

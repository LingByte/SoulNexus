package tts

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
)

// Service provides streaming PCM synthesis.
// callback receives raw PCM bytes (little-endian int16).
type Service interface {
	SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error
}

type Config struct {
	Service Service

	// Output PCM format (most of SoulNexus uses 16k/16bit/mono as internal).
	SampleRate int
	Channels   int

	// FrameDuration controls how PCM is chunked for downstream consumers.
	FrameDuration time.Duration

	// SendPCMFrame is called for every framed PCM chunk (FrameDuration).
	SendPCMFrame func(frame []byte) error

	// PaceRealtime, when true, sleeps FrameDuration after each emitted PCM frame so
	// wall-clock send rate matches the audio duration. Without this, TTS callbacks
	// often push many frames into the RTP path in one burst; RTP timestamps are still
	// correct, but some endpoints sound garbled or "layered" when playout is driven
	// by arrival time or small jitter buffers.
	PaceRealtime bool

	Logger *zap.Logger
}

// Pipeline is a minimal TTS streaming pipeline modeled after pkg/hardware/stream,
// but it outputs PCM frames directly (no opus re-encoding here).
type Pipeline struct {
	cfg Config

	mu     sync.Mutex
	playID string
	seq    uint32

	ctx    context.Context
	cancel context.CancelFunc
}

func New(cfg Config) (*Pipeline, error) {
	if cfg.Service == nil {
		return nil, fmt.Errorf("sip/tts: Service is required")
	}
	if cfg.SendPCMFrame == nil {
		return nil, fmt.Errorf("sip/tts: SendPCMFrame is required")
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 16000
	}
	if cfg.Channels == 0 {
		cfg.Channels = 1
	}
	if cfg.FrameDuration == 0 {
		cfg.FrameDuration = 60 * time.Millisecond
	}
	if cfg.Logger == nil {
		cfg.Logger, _ = zap.NewDevelopment()
	}
	return &Pipeline{cfg: cfg}, nil
}

func (p *Pipeline) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	p.playID = fmt.Sprintf("play-%d", time.Now().UnixNano())
	p.seq = 0
}

func (p *Pipeline) Stop() {
	p.mu.Lock()
	cancel := p.cancel
	p.cancel = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

type Frame struct {
	Data       []byte
	SampleRate int
	Channels   int
	PlayID     string
	Sequence   uint32
}

// Speak streams TTS audio for the full text and emits fixed-duration PCM frames.
func (p *Pipeline) Speak(text string) error {
	p.mu.Lock()
	ctx := p.ctx
	cfg := p.cfg
	playID := p.playID
	p.mu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	text = trimSpace(text)
	if text == "" {
		return nil
	}

	bytesPerFrame := pcmBytesPerFrame(cfg.SampleRate, cfg.Channels, cfg.FrameDuration)
	if bytesPerFrame <= 0 {
		return fmt.Errorf("sip/tts: invalid frame size")
	}

	// Pre-size buffer to cut reallocations on long TTS (rough PCM estimate from text length).
	bufCap := bytesPerFrame * 16
	if n := utf8.RuneCountInString(text); n > 0 {
		if c := n * 1200; c > bufCap {
			bufCap = c
		}
	}
	const bufCapMax = 4 << 20
	if bufCap > bufCapMax {
		bufCap = bufCapMax
	}
	buffer := make([]byte, 0, bufCap)
	err := cfg.Service.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(pcm) == 0 {
			return nil
		}
		buffer = append(buffer, pcm...)
		for len(buffer) >= bytesPerFrame {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			frameData := make([]byte, bytesPerFrame)
			copy(frameData, buffer[:bytesPerFrame])
			buffer = buffer[bytesPerFrame:]

			seq := p.nextSeq()
			if err := cfg.SendPCMFrame(frameData); err != nil {
				return err
			}
			p.paceAfterFrame(ctx, cfg)
			_ = Frame{Data: frameData, SampleRate: cfg.SampleRate, Channels: cfg.Channels, PlayID: playID, Sequence: seq}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(buffer) > 0 {
		// Pad last partial frame so downstream encoders (e.g. Opus) always receive full 20ms PCM.
		if len(buffer) < bytesPerFrame {
			padded := make([]byte, bytesPerFrame)
			copy(padded, buffer)
			buffer = padded
		}
		_ = p.nextSeq()
		if err := cfg.SendPCMFrame(buffer); err != nil {
			return err
		}
		p.paceAfterFrame(ctx, cfg)
	}
	return nil
}

func (p *Pipeline) paceAfterFrame(ctx context.Context, cfg Config) {
	if !cfg.PaceRealtime || cfg.FrameDuration <= 0 {
		return
	}
	if ctx.Err() != nil {
		return
	}
	// Avoid allocating a Timer per 20ms frame on long utterances (reduces GC jitter).
	time.Sleep(cfg.FrameDuration)
}

func (p *Pipeline) nextSeq() uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	seq := p.seq
	p.seq++
	return seq
}

func pcmBytesPerFrame(sampleRate, channels int, dur time.Duration) int {
	if sampleRate <= 0 || channels <= 0 || dur <= 0 {
		return 0
	}
	samples := int64(sampleRate) * dur.Milliseconds() / 1000
	if samples <= 0 {
		samples = int64(sampleRate) * int64(dur) / int64(time.Second)
	}
	if samples <= 0 {
		return 0
	}
	// 16-bit
	return int(samples) * channels * 2
}

func trimSpace(s string) string {
	// avoid importing strings in hot paths elsewhere; keep tiny helper here.
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\r' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

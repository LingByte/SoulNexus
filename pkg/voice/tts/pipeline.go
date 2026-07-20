package tts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"go.uber.org/zap"
)

// Service is provided by service.go (ported from VoiceServer pkg/voice/tts).
// The interface signature is identical to the previous in-pipeline declaration;
// service.go also adds ServiceFunc / FromSynthesisService adapters so callers
// can plug in arbitrary synthesizers without writing boilerplate.

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

	InvocationTenantID uint
	InvocationCallID   string
	InvocationProvider string
	InvocationSource   string
	InvocationModel    string
}

// Pipeline is a minimal TTS streaming pipeline modeled after pkg/hardware/stream,
// but it outputs PCM frames directly (no opus re-encoding here).
type Pipeline struct {
	cfg Config

	mu     sync.Mutex
	playID string
	seq    uint32

	// residual carries the sub-frame PCM tail from the previous Speak()
	// across calls so consecutive Speak()s on the same Pipeline produce
	// a CONTINUOUS audio stream. Without this, each Speak() ended by
	// zero-padding the last partial frame, leaving an audible silence
	// cliff at every sentence boundary — heard as "滋滋" hiss when LEX
	// streams an LLM reply sentence-by-sentence. See Finalize() for the
	// end-of-turn drain.
	residual []byte

	ctx    context.Context
	cancel context.CancelFunc
}

func New(cfg Config) (*Pipeline, error) {
	if cfg.Service == nil {
		return nil, fmt.Errorf("voice/tts: Service is required")
	}
	if cfg.SendPCMFrame == nil {
		return nil, fmt.Errorf("voice/tts: SendPCMFrame is required")
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
	// Discard any sub-frame residual when the pipeline is stopped
	// (barge-in, turn-cancel, etc.) so the next Start()+Speak() does
	// NOT splice last turn's tail into the new utterance — that would
	// produce a tiny audible "glitch" at the start of every reply
	// after a barge-in.
	p.residual = nil
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
	start := time.Now()
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
		return fmt.Errorf("voice/tts: invalid frame size")
	}

	var synthBytes int64

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
	// Seed the per-call buffer with whatever sub-frame tail the previous
	// Speak() left behind. This preserves audio continuity across the
	// segmented Speak() calls that LEX's streamLLMToTTS makes per LLM
	// sentence (eliminates the "滋滋" cliff between segments).
	p.mu.Lock()
	carry := p.residual
	p.residual = nil
	p.mu.Unlock()
	buffer := make([]byte, 0, bufCap+len(carry))
	if len(carry) > 0 {
		buffer = append(buffer, carry...)
	}
	err := cfg.Service.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(pcm) == 0 {
			return nil
		}
		synthBytes += int64(len(pcm))
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
		p.recordInvocation(cfg, start, text, synthBytes, err)
		return err
	}
	if ctx.Err() != nil {
		p.recordInvocation(cfg, start, text, synthBytes, ctx.Err())
		return ctx.Err()
	}
	// Stash any sub-frame tail for the next Speak() instead of
	// zero-padding it here. The old code emitted a half-frame followed
	// by silence at every Speak() boundary; when LLM streaming triggers
	// many short Speak() calls per turn (one per sentence in
	// streamLLMToTTS), those silence cliffs stack up and are heard as
	// the "每说一字滋滋" hiss the user reported. Finalize() drains the
	// residual with a single fade-padded frame at end-of-turn.
	if len(buffer) > 0 {
		p.mu.Lock()
		p.residual = append(p.residual[:0], buffer...)
		p.mu.Unlock()
	}
	p.recordInvocation(cfg, start, text, synthBytes, nil)
	return nil
}

func (p *Pipeline) recordInvocation(cfg Config, start time.Time, text string, synthBytes int64, err error) {
	callID := strings.TrimSpace(cfg.InvocationCallID)
	if callID == "" && cfg.InvocationTenantID == 0 {
		return
	}
	entry := callbinding.AIInvocationRecord{
		TenantID:    cfg.InvocationTenantID,
		Component:   callbinding.AIComponentTTS,
		Provider:    cfg.InvocationProvider,
		Model:       cfg.InvocationModel,
		CallID:      callID,
		Source:      cfg.InvocationSource,
		LatencyMs:   time.Since(start).Milliseconds(),
		InputChars:  utf8.RuneCountInString(text),
		AudioBytes:  synthBytes,
		RequestText: text,
	}
	if cfg.InvocationSource == "" {
		entry.Source = "voice"
	}
	if cfg.SampleRate > 0 && synthBytes >= 2 {
		samples := synthBytes / 2 / int64(max(cfg.Channels, 1))
		entry.AudioMs = samples * 1000 / int64(cfg.SampleRate)
	}
	if err != nil {
		entry.Status = callbinding.AIStatusError
		entry.ErrorMsg = err.Error()
	} else {
		entry.Status = callbinding.AIStatusOK
	}
	callbinding.RecordAIInvocation(entry)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Finalize emits any residual sub-frame audio left over from prior
// Speak() calls, padding the last partial frame with zeros so
// downstream encoders receive a full FrameDuration of PCM. Call this
// at end-of-turn (e.g. after barge-in cancels or after the entire LLM
// reply has been spoken) — NOT between LLM-stream sentences, since
// that is exactly the boundary where the residual carry is needed to
// avoid audible cliffs.
//
// Idempotent: safe to call multiple times; only the first call after
// non-empty residual emits a frame.
func (p *Pipeline) Finalize() error {
	p.mu.Lock()
	cfg := p.cfg
	ctx := p.ctx
	tail := p.residual
	p.residual = nil
	p.mu.Unlock()
	if len(tail) == 0 {
		return nil
	}
	bytesPerFrame := pcmBytesPerFrame(cfg.SampleRate, cfg.Channels, cfg.FrameDuration)
	if bytesPerFrame <= 0 {
		return nil
	}
	if len(tail) < bytesPerFrame {
		padded := make([]byte, bytesPerFrame)
		copy(padded, tail)
		tail = padded
	}
	if cfg.SendPCMFrame == nil {
		return nil
	}
	if err := cfg.SendPCMFrame(tail); err != nil {
		return err
	}
	if ctx != nil {
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

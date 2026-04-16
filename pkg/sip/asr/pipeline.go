package asr

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"go.uber.org/zap"
)

// PipelineComponent is a minimal processing stage.
// It mirrors the hardware ASR pipeline shape but is SIP-focused:
// input is expected to be PCM bytes (16-bit little-endian).
type PipelineComponent interface {
	Name() string
	Process(ctx context.Context, data interface{}) (out interface{}, shouldContinue bool, err error)
}

type Metrics struct {
	mu sync.RWMutex

	FirstPacketTime time.Time
	LastPacketTime  time.Time
	ASRFirstResult  time.Time
	ASRLatency      time.Duration

	TotalAudioBytes int
}

type Options struct {
	ASR recognizer.TranscribeService
	// DecodeSource is the negotiated RTP codec (opus / pcmu / pcma / g722 …).
	// If Codec is empty or "pcm", no decode stage is added — input must already be PCM
	// (typical when pkg/media/MediaSession has already decoded RTP).
	DecodeSource media.CodecConfig

	// PCMForASR is the target PCM format for ASR after decode (default: 16kHz mono 16-bit).
	PCMForASR media.CodecConfig

	// SampleRate / Channels are legacy hints; prefer PCMForASR when set.
	SampleRate int
	Channels   int

	Logger *zap.Logger
}

type Pipeline struct {
	asr recognizer.TranscribeService

	hasDecode bool

	inputStages  []PipelineComponent
	outputStages []PipelineComponent

	onText func(text string, isFinal bool)
	onErr  func(err error, fatal bool)

	metrics *Metrics
	logger  *zap.Logger
}

func New(options Options) (*Pipeline, error) {
	if options.ASR == nil {
		return nil, fmt.Errorf("sip/asr: ASR service is required")
	}
	lg := options.Logger
	if lg == nil {
		lg, _ = zap.NewDevelopment()
	}

	p := &Pipeline{
		asr:     options.ASR,
		metrics: &Metrics{},
		logger:  lg,
	}

	dst := options.PCMForASR
	if strings.TrimSpace(dst.Codec) == "" {
		dst.Codec = "pcm"
	}
	if dst.SampleRate == 0 {
		if options.SampleRate > 0 {
			dst.SampleRate = options.SampleRate
		} else {
			dst.SampleRate = 16000
		}
	}
	if dst.Channels == 0 {
		if options.Channels > 0 {
			dst.Channels = options.Channels
		} else {
			dst.Channels = 1
		}
	}
	if dst.BitDepth == 0 {
		dst.BitDepth = 16
	}

	src := options.DecodeSource
	if codecNeedsDecode(src) {
		decStage, err := newCodecDecodeStage(src, dst, lg)
		if err != nil {
			return nil, err
		}
		p.hasDecode = true
		p.inputStages = append(p.inputStages, decStage)
	}

	p.inputStages = append(p.inputStages, &asrInputStage{asr: p.asr, metrics: p.metrics, logger: p.logger})

	// Match recognizer.WithTranscribeFilter: connect before registering callbacks (e.g. QCloud).
	_ = options.ASR.ConnAndReceive("")
	p.asr.Init(p.onASRResult, p.onASRError)
	return p, nil
}

func codecNeedsDecode(src media.CodecConfig) bool {
	c := strings.ToLower(strings.TrimSpace(src.Codec))
	return c != "" && c != "pcm"
}

func (p *Pipeline) SetTextCallback(cb func(text string, isFinal bool)) {
	p.onText = cb
}

func (p *Pipeline) SetErrorCallback(cb func(err error, fatal bool)) {
	p.onErr = cb
}

func (p *Pipeline) GetMetrics() Metrics {
	if p == nil || p.metrics == nil {
		return Metrics{}
	}
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	return *p.metrics
}

// ProcessPCM feeds one PCM chunk (no decode stage). Use when MediaSession already decoded RTP.
func (p *Pipeline) ProcessPCM(ctx context.Context, pcm []byte) error {
	if p != nil && p.hasDecode {
		return fmt.Errorf("sip/asr: DecodeSource is set; use Process or ProcessEncoded with RTP payload bytes")
	}
	return p.Process(ctx, pcm)
}

// ProcessEncoded feeds one encoded RTP payload chunk (requires DecodeSource in Options).
func (p *Pipeline) ProcessEncoded(ctx context.Context, rtpPayload []byte) error {
	if p == nil || !p.hasDecode {
		return fmt.Errorf("sip/asr: no DecodeSource configured; use ProcessPCM")
	}
	return p.Process(ctx, rtpPayload)
}

// Process runs the pipeline: decode (if configured) → ASR.
func (p *Pipeline) Process(ctx context.Context, audio []byte) error {
	if p == nil {
		return fmt.Errorf("sip/asr: nil pipeline")
	}
	if len(audio) == 0 {
		return nil
	}
	p.metrics.mu.Lock()
	if p.metrics.FirstPacketTime.IsZero() {
		p.metrics.FirstPacketTime = time.Now()
	}
	p.metrics.LastPacketTime = time.Now()
	p.metrics.mu.Unlock()

	current := interface{}(audio)
	for _, stage := range p.inputStages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		out, cont, err := stage.Process(ctx, current)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
		current = out
	}
	return nil
}

func (p *Pipeline) onASRResult(text string, isFinal bool, _ time.Duration, _ string) {
	if p == nil {
		return
	}
	if text == "" {
		return
	}
	p.metrics.mu.Lock()
	if p.metrics.ASRFirstResult.IsZero() {
		p.metrics.ASRFirstResult = time.Now()
		if !p.metrics.LastPacketTime.IsZero() {
			p.metrics.ASRLatency = p.metrics.ASRFirstResult.Sub(p.metrics.LastPacketTime)
		}
	}
	p.metrics.mu.Unlock()

	if p.onText != nil {
		p.onText(text, isFinal)
	}
}

func (p *Pipeline) onASRError(err error, fatal bool) {
	if p == nil {
		return
	}
	if p.onErr != nil {
		p.onErr(err, fatal)
	} else if p.logger != nil {
		p.logger.Error("sip/asr error", zap.Error(err), zap.Bool("fatal", fatal))
	}
}

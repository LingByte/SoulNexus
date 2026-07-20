package dialog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/tts"
)

// Config wires the voice plane for a single call.
type Config struct {
	CallID string
	Meta   StartMeta

	Engine asr.Engine

	InputCodec    string
	PCMSampleRate int

	TTSService tts.TTSService
	// TTSCache wraps TTSService with process-level PCM caching when set.
	TTSCache   *tts.CacheConfig
	OnAudioOut func([]byte) error

	OnEvent      EventHandler
	OnHangup     func(reason string)
	OnTurn       func(TurnEvent)
	OnFirstAudio func(FirstAudioEvent)

	// EnableVAD enables barge-in during downlink playback (default true).
	EnableVAD *bool
	// VADConfig overrides barge-in thresholds; nil uses DefaultBargeInVADConfig.
	VADConfig *asr.VADConfig
	// EnableEchoFilter suppresses uplink while TTS active (default true).
	EnableEchoFilter *bool
	// EchoTail extends echo suppression after playback ends (default 150ms).
	EchoTail time.Duration
	// Denoiser optional uplink noise/AEC (RNNoise, WebRTC AEC3, hardware AEC).
	// When non-nil, runs after decode and before VAD.
	Denoiser asr.Denoiser
	// CoalesceUplink buffers small PCM chunks before ASR (default true).
	CoalesceUplink *bool
	// PaceRealtime paces TTS frames at wall-clock rate for smooth playback (default true).
	PaceRealtime *bool
	// TTSFrameDuration sets downlink frame size (default 60ms).
	TTSFrameDuration time.Duration
	// OutputCodec is the downlink wire codec ("pcm" default, "opus" uses AudioSender).
	OutputCodec string

	// EnableSentenceFilter enables ASR sentence-boundary filtering (default true).
	EnableSentenceFilter *bool
	// SentenceFilterSimilarity sets dedup threshold; nil → 0.85, explicit 0 disables dedup only.
	SentenceFilterSimilarity *float64
	// SentenceFilter overrides auto-created filter when set explicitly.
	SentenceFilter *asr.SentenceFilter

	// EnableStreamSegmenter wires TextSegmenter for tts.stream commands (default true).
	EnableStreamSegmenter *bool
	TextSegmenterConfig   *tts.TextSegmenterConfig
}

func boolOr(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func defaultSentenceFilterSimilarity(ptr *float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return 0.85
}

func resolveOutputCodec(cfg Config) string {
	codec := strings.ToLower(strings.TrimSpace(cfg.OutputCodec))
	if codec == "" {
		codec = strings.ToLower(strings.TrimSpace(cfg.Meta.Codec))
	}
	if codec == "" {
		codec = "pcm"
	}
	return codec
}

// NewSession builds and returns a voice session from cfg.
func NewSession(ctx context.Context, cfg Config) (*Session, error) {
	if cfg.CallID == "" {
		return nil, fmt.Errorf("dialog: empty call_id")
	}
	if cfg.Engine == nil {
		return nil, fmt.Errorf("dialog: nil ASR engine")
	}
	if cfg.TTSService == nil {
		return nil, fmt.Errorf("dialog: nil TTS service")
	}
	if cfg.OnAudioOut == nil {
		return nil, fmt.Errorf("dialog: nil OnAudioOut")
	}
	if cfg.OnEvent == nil {
		return nil, fmt.Errorf("dialog: nil OnEvent")
	}

	sr := cfg.PCMSampleRate
	if sr <= 0 {
		sr = 16000
	}

	echoTail := cfg.EchoTail
	if echoTail == 0 {
		echoTail = 150 * time.Millisecond
	}

	frameDur := cfg.TTSFrameDuration
	if frameDur <= 0 {
		frameDur = 60 * time.Millisecond
	}

	ttsService := cfg.TTSService
	if cfg.TTSCache != nil {
		cached, err := tts.NewCachingTTSService(ttsService, *cfg.TTSCache)
		if err != nil {
			return nil, fmt.Errorf("dialog: tts cache: %w", err)
		}
		ttsService = cached
	}

	outputCodec := resolveOutputCodec(cfg)
	sendCallback := cfg.OnAudioOut
	var audioSender *tts.AudioSender
	if outputCodec != "pcm" {
		var err error
		audioSender, err = tts.NewAudioSender(tts.AudioSenderConfig{
			OutputCodec:      outputCodec,
			TargetSampleRate: sr,
			FrameDuration:    frameDur,
			SendCallback:     cfg.OnAudioOut,
		})
		if err != nil {
			return nil, fmt.Errorf("dialog: audio sender: %w", err)
		}
		sendCallback = func(pcm []byte) error {
			return audioSender.ProcessFrame(tts.AudioFrame{
				Data:       pcm,
				SampleRate: sr,
				Channels:   1,
			})
		}
	}

	ttsPipeline, err := tts.NewTTSPipeline(tts.TTSPipelineConfig{
		TTSService:       ttsService,
		TargetSampleRate: sr,
		FrameDuration:    frameDur,
		PaceRealtime:     boolOr(cfg.PaceRealtime, false),
		SendCallback:     sendCallback,
	})
	if err != nil {
		return nil, err
	}

	speaker, err := tts.NewSpeaker(tts.SpeakerConfig{Pipeline: ttsPipeline})
	if err != nil {
		return nil, err
	}

	gate := asr.NewPlaybackGate(speaker.IsPlaying, speaker.QueueDepth, echoTail)

	recognizer, err := asr.NewRecognizerComponent(cfg.Engine)
	if err != nil {
		return nil, err
	}

	inputStages := make([]asr.PipelineComponent, 0, 7)
	codec := cfg.InputCodec
	if codec == "" {
		codec = "pcm"
	}
	switch codec {
	case "pcm":
		inputStages = append(inputStages, &asr.PCMInputComponent{})
	default:
		dec, err := asr.NewDecoderComponent(asr.DecoderConfig{
			SourceCodec:      codec,
			TargetSampleRate: sr,
		})
		if err != nil {
			return nil, fmt.Errorf("dialog: decoder: %w", err)
		}
		inputStages = append(inputStages, dec)
	}

	if cfg.Denoiser != nil {
		inputStages = append(inputStages, asr.NewDenoiserComponent(cfg.Denoiser))
	}

	if boolOr(cfg.EnableVAD, true) {
		vadCfg := asr.DefaultBargeInVADConfig()
		if cfg.VADConfig != nil {
			vadCfg = *cfg.VADConfig
		}
		inputStages = append(inputStages, asr.NewVADComponent(vadCfg, gate))
	}

	if boolOr(cfg.EnableEchoFilter, true) {
		inputStages = append(inputStages, asr.NewEchoFilterComponent(gate))
	}

	if boolOr(cfg.CoalesceUplink, true) {
		inputStages = append(inputStages, asr.NewPCMCoalesceComponent(sr, 20))
	}
	inputStages = append(inputStages, recognizer)

	asrPipeline, err := asr.NewStandardPipeline(inputStages, nil)
	if err != nil {
		return nil, err
	}
	asrPipeline.WirePlaybackGate(gate)

	var filter *asr.SentenceFilter
	switch {
	case cfg.SentenceFilter != nil:
		filter = cfg.SentenceFilter
	case boolOr(cfg.EnableSentenceFilter, true):
		filter = asr.NewSentenceFilter(defaultSentenceFilterSimilarity(cfg.SentenceFilterSimilarity))
	}

	sess := &Session{
		cfg:         cfg,
		asrPipeline: asrPipeline,
		recognizer:  recognizer,
		ttsPipeline: ttsPipeline,
		speaker:     speaker,
		gate:        gate,
		filter:      filter,
		denoiser:    cfg.Denoiser,
		audioSender: audioSender,
		turnMeta:    make(map[string]*CommandMeta),
	}

	if boolOr(cfg.EnableStreamSegmenter, true) {
		segCfg := tts.DefaultTextSegmenterConfig()
		if cfg.TextSegmenterConfig != nil {
			segCfg = *cfg.TextSegmenterConfig
		} else {
			caps := tts.CapabilitiesFrom(ttsService)
			if caps.FirstMaxChars > 0 {
				segCfg.FirstMaxChars = caps.FirstMaxChars
			}
			if caps.FirstMinChars > 0 {
				segCfg.FirstMinChars = caps.FirstMinChars
			}
		}
		sess.segmenter = tts.NewTextSegmenterComponent(segCfg, sess.enqueueSegment)
	}

	sess.wireCallbacks()
	return sess, nil
}

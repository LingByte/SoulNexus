package tts

import (
	"context"
	"errors"
	"time"

	"github.com/LingByte/lingllm/utils"
)

// ErrInterrupted is returned when Speak is cancelled via Interrupt.
var ErrInterrupted = errors.New("tts: interrupted")

// Speak synthesizes text with per-utterance cancellation (barge-in).
// Blocks until synthesis completes, is interrupted, or the pipeline stops.
func (p *TTSPipeline) Speak(text string) error {
	if p == nil {
		return errors.New("tts: nil pipeline")
	}
	if text == "" {
		return nil
	}

	p.mu.Lock()
	if p.ctx == nil {
		p.mu.Unlock()
		return ErrPipelineNotStarted
	}
	parent := p.ctx
	speakCtx, speakCancel := context.WithCancel(parent)
	p.speakMu.Lock()
	if p.speakCancel != nil {
		p.speakCancel()
	}
	p.speakCtx = speakCtx
	p.speakCancel = speakCancel
	p.speakMu.Unlock()
	p.mu.Unlock()

	p.resetPaceClock()
	p.playing.Store(true)
	defer func() {
		p.playing.Store(false)
		p.speakMu.Lock()
		p.speakCancel = nil
		p.speakCtx = nil
		p.speakMu.Unlock()
	}()

	err := p.synthesize(speakCtx, text)
	if errors.Is(err, context.Canceled) {
		return ErrInterrupted
	}
	return err
}

// Interrupt cancels the in-flight Speak. The pipeline remains usable.
func (p *TTSPipeline) Interrupt() {
	if p == nil {
		return
	}
	p.speakMu.Lock()
	if p.speakCancel != nil {
		p.speakCancel()
	}
	p.speakMu.Unlock()
	p.playing.Store(false)
}

// IsPlaying reports whether a Speak call is streaming audio.
func (p *TTSPipeline) IsPlaying() bool {
	return p != nil && p.playing.Load()
}

// ArmFirstFrameHook installs a one-shot callback fired on the first emitted frame.
func (p *TTSPipeline) ArmFirstFrameHook(fn func()) {
	if p == nil {
		return
	}
	if fn == nil {
		p.firstFrameHook.Store(nil)
		return
	}
	p.firstFrameHook.Store(&fn)
}

func (p *TTSPipeline) fireFirstFrameHook() {
	if hook := p.firstFrameHook.Swap(nil); hook != nil && *hook != nil {
		(*hook)()
	}
}

// synthesize is the internal synthesis path used by Speak and Synthesize.
func (p *TTSPipeline) synthesize(ctx context.Context, text string) error {
	return p.synthesizeFrames(ctx, text, func(frame []byte) error {
		audioBytes, err := p.processAndSendFrame(ctx, frame)
		if err != nil {
			return err
		}
		if len(audioBytes) > 0 {
			p.fireFirstFrameHook()
		}
		return nil
	})
}

// synthesizeFrames runs TTS and invokes emit for each paced output frame.
// Used by Speak (inline playback) and the pipelined Speaker (prefetch path).
func (p *TTSPipeline) synthesizeFrames(ctx context.Context, text string, emit func([]byte) error) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.ctx == nil {
		return ErrPipelineNotStarted
	}

	processedText, err := p.prepareSpeechText(ctx, text)
	if err != nil {
		return err
	}
	if processedText == "" {
		return nil
	}

	frameSizeBytes := p.frameBytes()
	if frameSizeBytes < 2 {
		frameSizeBytes = 1920
	}

	buffer := make([]byte, 0, len(processedText)*100)
	err = p.ttsService.Synthesize(ctx, processedText, func(pcmData []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		buffer = append(buffer, pcmData...)
		for len(buffer) >= frameSizeBytes {
			frameData := make([]byte, frameSizeBytes)
			copy(frameData, buffer[:frameSizeBytes])
			buffer = buffer[frameSizeBytes:]

			if err := emit(frameData); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(buffer) > 0 {
		if err := emit(buffer); err != nil {
			return err
		}
	}
	return nil
}

// PlayFrames emits pre-collected frames with pacing and hooks.
func (p *TTSPipeline) PlayFrames(ctx context.Context, frames [][]byte) error {
	if p == nil {
		return errors.New("tts: nil pipeline")
	}
	for _, frame := range frames {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		audioBytes, err := p.processAndSendFrame(ctx, frame)
		if err != nil {
			return err
		}
		if len(audioBytes) > 0 {
			p.fireFirstFrameHook()
		}
	}
	return nil
}

func (p *TTSPipeline) prepareSpeechText(ctx context.Context, text string) (string, error) {
	text = utils.SanitizeForSpeech(text)
	if text == "" || !utils.HasSpeakableContent(text) {
		return "", nil
	}

	processedText := text
	for _, processor := range p.textProcessors {
		result, shouldContinue, err := processor.Process(ctx, processedText)
		if err != nil {
			return "", err
		}
		if !shouldContinue {
			return "", nil
		}
		if resultStr, ok := result.(string); ok {
			processedText = resultStr
		}
	}
	return processedText, nil
}

func (p *TTSPipeline) frameBytes() int {
	dur := p.config.FrameDuration
	if dur <= 0 {
		dur = 60 * time.Millisecond
	}
	sr := p.config.TargetSampleRate
	if sr <= 0 {
		sr = 16000
	}
	samples := int(float64(sr) * dur.Seconds())
	if samples < 1 {
		samples = 1
	}
	return samples * 2
}

func (p *TTSPipeline) processAndSendFrame(ctx context.Context, frame []byte) ([]byte, error) {
	audioData := interface{}(frame)
	for _, processor := range p.audioProcessors {
		result, shouldContinue, err := processor.Process(ctx, audioData)
		if err != nil {
			return nil, err
		}
		if !shouldContinue {
			return nil, nil
		}
		audioData = result
	}

	var audioBytes []byte
	if resultBytes, ok := audioData.([]byte); ok {
		audioBytes = resultBytes
	} else {
		audioBytes = frame
	}

	if p.config.RecordCallback != nil {
		_ = p.config.RecordCallback(audioBytes)
	}
	if p.config.PaceRealtime {
		p.waitForPaceSlot()
	}
	if err := p.config.SendCallback(audioBytes); err != nil {
		return nil, err
	}
	return audioBytes, nil
}

func (p *TTSPipeline) resetPaceClock() {
	if p == nil || !p.config.PaceRealtime {
		return
	}
	p.paceMu.Lock()
	p.nextFrameAt = time.Time{}
	p.paceMu.Unlock()
}

func (p *TTSPipeline) waitForPaceSlot() {
	frameDur := p.config.FrameDuration
	if frameDur <= 0 {
		frameDur = 60 * time.Millisecond
	}
	p.paceMu.Lock()
	defer p.paceMu.Unlock()
	now := time.Now()
	if p.nextFrameAt.IsZero() {
		p.nextFrameAt = now
	}
	if now.Before(p.nextFrameAt) {
		time.Sleep(p.nextFrameAt.Sub(now))
	}
	p.nextFrameAt = p.nextFrameAt.Add(frameDur)
}

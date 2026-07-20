package tts

import (
	"context"
	"fmt"
	"sync"
)

// TTSWorkerConfig contains configuration for TTS workers.
type TTSWorkerConfig struct {
	// TTSService: the TTS service to use
	TTSService TTSService
	// WorkerCount: number of worker goroutines (default: 1)
	WorkerCount int
	// Logger: optional logging callback
	Logger func(string)
}

// TTSWorkerPool manages multiple TTS worker goroutines.
type TTSWorkerPool struct {
	mu           sync.RWMutex
	config       TTSWorkerConfig
	ttsService   TTSService
	inputCh      <-chan TextSegment
	outputCh     chan<- AudioFrame
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	logger       func(string)
	sequenceMu   sync.Mutex
	globalSeq    uint32
	ttsServiceMu sync.RWMutex
}

// NewTTSWorkerPool creates a new TTS worker pool.
func NewTTSWorkerPool(
	config TTSWorkerConfig,
	inputCh <-chan TextSegment,
	outputCh chan<- AudioFrame,
) (*TTSWorkerPool, error) {
	if config.TTSService == nil {
		return nil, ErrTTSServiceRequired
	}

	if config.WorkerCount <= 0 {
		config.WorkerCount = 1
	}

	return &TTSWorkerPool{
		config:     config,
		ttsService: config.TTSService,
		inputCh:    inputCh,
		outputCh:   outputCh,
		logger:     config.Logger,
	}, nil
}

// Start starts the worker pool.
func (p *TTSWorkerPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ctx, p.cancel = context.WithCancel(ctx)

	for i := 0; i < p.config.WorkerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	if p.logger != nil {
		p.logger(fmt.Sprintf("[TTSWorkerPool] Started with %d workers", p.config.WorkerCount))
	}

	return nil
}

// Stop stops the worker pool.
func (p *TTSWorkerPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}

	p.wg.Wait()

	if p.logger != nil {
		p.logger("[TTSWorkerPool] Stopped")
	}

	return nil
}

// worker processes text segments and synthesizes audio.
func (p *TTSWorkerPool) worker(id int) {
	defer p.wg.Done()

	if p.logger != nil {
		p.logger(fmt.Sprintf("[TTSWorker-%d] Started", id))
	}

	for {
		select {
		case <-p.ctx.Done():
			if p.logger != nil {
				p.logger(fmt.Sprintf("[TTSWorker-%d] Stopped", id))
			}
			return

		case segment := <-p.inputCh:
			p.synthesize(id, segment)
		}
	}
}

// synthesize synthesizes a text segment to audio.
func (p *TTSWorkerPool) synthesize(workerID int, segment TextSegment) {
	if p.logger != nil {
		p.logger(fmt.Sprintf("[TTSWorker-%d] Synthesizing: %q", workerID, segment.Text))
	}

	const frameSizeBytes = 1920 // 60ms @ 16kHz, 16bit, mono

	// Pre-allocate buffer
	estimatedSize := len(segment.Text) * 100
	if estimatedSize < frameSizeBytes*2 {
		estimatedSize = frameSizeBytes * 2
	}
	buffer := make([]byte, 0, estimatedSize)

	// Get TTS service (thread-safe)
	p.ttsServiceMu.RLock()
	ttsService := p.ttsService
	p.ttsServiceMu.RUnlock()

	err := ttsService.Synthesize(p.ctx, segment.Text, func(pcmData []byte) error {
		buffer = append(buffer, pcmData...)

		for len(buffer) >= frameSizeBytes {
			frameData := make([]byte, frameSizeBytes)
			copy(frameData, buffer[:frameSizeBytes])
			buffer = buffer[frameSizeBytes:]

			// Get global sequence
			p.sequenceMu.Lock()
			sequence := p.globalSeq
			p.globalSeq++
			p.sequenceMu.Unlock()

			frame := AudioFrame{
				Data:       frameData,
				SampleRate: 16000,
				Channels:   1,
				PlayID:     segment.PlayID,
				Sequence:   sequence,
			}

			select {
			case p.outputCh <- frame:
			case <-p.ctx.Done():
				return nil
			}
		}

		return nil
	})

	if err != nil {
		if p.logger != nil {
			p.logger(fmt.Sprintf("[TTSWorker-%d] Synthesis error: %v", workerID, err))
		}
		return
	}

	// Handle remaining data
	if len(buffer) > 0 {
		p.sequenceMu.Lock()
		sequence := p.globalSeq
		p.globalSeq++
		p.sequenceMu.Unlock()

		frame := AudioFrame{
			Data:       buffer,
			SampleRate: 16000,
			Channels:   1,
			PlayID:     segment.PlayID,
			Sequence:   sequence,
		}

		select {
		case p.outputCh <- frame:
		case <-p.ctx.Done():
		}
	}

	if p.logger != nil {
		p.logger(fmt.Sprintf("[TTSWorker-%d] Synthesis complete", workerID))
	}
}

// UpdateTTSService updates the TTS service (for speaker switching).
func (p *TTSWorkerPool) UpdateTTSService(newService TTSService) error {
	if newService == nil {
		return ErrTTSServiceRequired
	}

	p.ttsServiceMu.Lock()
	defer p.ttsServiceMu.Unlock()

	p.ttsService = newService

	if p.logger != nil {
		p.logger("[TTSWorkerPool] TTS service updated")
	}

	return nil
}

// GetGlobalSequence returns the current global sequence number.
func (p *TTSWorkerPool) GetGlobalSequence() uint32 {
	p.sequenceMu.Lock()
	defer p.sequenceMu.Unlock()
	return p.globalSeq
}

// Close closes the worker pool.
func (p *TTSWorkerPool) Close() error {
	return p.Stop()
}

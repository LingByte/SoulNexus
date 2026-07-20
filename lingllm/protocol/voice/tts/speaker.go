package tts

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Speaker serializes TTS playback and pipelines synthesis ahead of playback so
// LLM stream segments can be synthesized while the previous segment is playing.
type Speaker struct {
	pipeline   *TTSPipeline
	textQueue  chan speakJob
	readyQueue chan *segmentStream

	runCtx    context.Context
	runCancel context.CancelFunc
	wg        sync.WaitGroup
	closed    atomic.Bool

	onStarted    func(utteranceID, text string, chained bool)
	onEnded      func(utteranceID string, ok bool, duration time.Duration, ttsFirstMs, e2eFirstMs int, moreQueued bool)
	onFirstFrame func(utteranceID string, ttsFirstMs, e2eFirstMs int)

	turnMu      sync.Mutex
	turnLatency map[string]*utteranceLatency
}

type utteranceLatency struct {
	ttsFirstMs int
	e2eFirstMs int
	started    time.Time
	totalDur   time.Duration
	ok         bool
}

type speakJob struct {
	text              string
	utteranceID       string
	e2eAnchor         *time.Time
	chainFromPrevious bool
}

type segmentStream struct {
	job    speakJob
	frames chan []byte
	done   chan struct{}
	err    error
}

// SpeakerConfig configures a serial TTS speaker.
type SpeakerConfig struct {
	Pipeline     *TTSPipeline
	QueueSize    int
	Prefetch     int // buffered segments between synth and play (default 2)
	OnStarted    func(utteranceID, text string, chained bool)
	OnEnded      func(utteranceID string, ok bool, duration time.Duration, ttsFirstMs, e2eFirstMs int, moreQueued bool)
	OnFirstFrame func(utteranceID string, ttsFirstMs, e2eFirstMs int)
}

// NewSpeaker creates a pipelined TTS speaker.
func NewSpeaker(cfg SpeakerConfig) (*Speaker, error) {
	if cfg.Pipeline == nil {
		return nil, errors.New("tts: nil pipeline")
	}
	size := cfg.QueueSize
	if size <= 0 {
		size = 64
	}
	prefetch := cfg.Prefetch
	if prefetch <= 0 {
		prefetch = 3
	}
	return &Speaker{
		pipeline:     cfg.Pipeline,
		textQueue:    make(chan speakJob, size),
		readyQueue:   make(chan *segmentStream, prefetch),
		onStarted:    cfg.OnStarted,
		onEnded:      cfg.OnEnded,
		onFirstFrame: cfg.OnFirstFrame,
		turnLatency:  make(map[string]*utteranceLatency),
	}, nil
}

// SetCallbacks updates lifecycle hooks (safe before Start).
func (s *Speaker) SetCallbacks(
	onStarted func(utteranceID, text string, chained bool),
	onEnded func(utteranceID string, ok bool, duration time.Duration, ttsFirstMs, e2eFirstMs int, moreQueued bool),
) {
	s.SetCallbacksWithFirstFrame(onStarted, onEnded, nil)
}

// SetCallbacksWithFirstFrame also wires an optional first-frame hook (once per utterance).
func (s *Speaker) SetCallbacksWithFirstFrame(
	onStarted func(utteranceID, text string, chained bool),
	onEnded func(utteranceID string, ok bool, duration time.Duration, ttsFirstMs, e2eFirstMs int, moreQueued bool),
	onFirstFrame func(utteranceID string, ttsFirstMs, e2eFirstMs int),
) {
	if s == nil {
		return
	}
	s.onStarted = onStarted
	s.onEnded = onEnded
	s.onFirstFrame = onFirstFrame
}

// Start begins synth and play workers.
func (s *Speaker) Start(ctx context.Context) {
	if s == nil || s.closed.Load() {
		return
	}
	s.runCtx, s.runCancel = context.WithCancel(ctx)
	s.wg.Add(2)
	go s.synthLoop()
	go s.playLoop()
}

// Stop drains and shuts down the speaker.
func (s *Speaker) Stop() {
	if s == nil || s.closed.Swap(true) {
		return
	}
	s.Interrupt()
	if s.runCancel != nil {
		s.runCancel()
	}
	close(s.textQueue)
	s.wg.Wait()
}

// Enqueue schedules text for synthesis. Non-blocking; drops when queue is full.
func (s *Speaker) Enqueue(text, utteranceID string, e2eAnchor *time.Time) bool {
	text = SanitizeSpeech(text)
	if s == nil || s.closed.Load() || text == "" {
		return false
	}
	chain := s.IsActive()
	job := speakJob{
		text:              text,
		utteranceID:       utteranceID,
		e2eAnchor:         e2eAnchor,
		chainFromPrevious: chain,
	}
	select {
	case s.textQueue <- job:
		return true
	default:
		return false
	}
}

// Interrupt stops the current utterance and drains pending jobs.
func (s *Speaker) Interrupt() {
	if s == nil {
		return
	}
	s.pipeline.Interrupt()
	s.drainTextQueue(true)
	s.drainReadyQueue(true)
	s.turnMu.Lock()
	s.turnLatency = make(map[string]*utteranceLatency)
	s.turnMu.Unlock()
}

// DrainQueue drops pending jobs without interrupting the current utterance.
func (s *Speaker) DrainQueue() {
	if s != nil {
		s.drainTextQueue(true)
	}
}

func (s *Speaker) drainTextQueue(notify bool) {
	for {
		select {
		case job, ok := <-s.textQueue:
			if !ok {
				return
			}
			if notify && s.onEnded != nil {
				s.onEnded(job.utteranceID, false, 0, 0, 0, false)
			}
		default:
			return
		}
	}
}

func (s *Speaker) drainReadyQueue(notify bool) {
	for {
		select {
		case seg, ok := <-s.readyQueue:
			if !ok {
				return
			}
			s.drainSegmentFrames(seg)
			if notify && s.onEnded != nil {
				s.onEnded(seg.job.utteranceID, false, 0, 0, 0, false)
			}
		default:
			return
		}
	}
}

func (s *Speaker) drainSegmentFrames(seg *segmentStream) {
	if seg == nil {
		return
	}
	for range seg.frames {
	}
	<-seg.done
}

// IsPlaying reports whether TTS audio is actively streaming.
func (s *Speaker) IsPlaying() bool {
	if s == nil {
		return false
	}
	return s.pipeline.IsPlaying()
}

// IsActive is true while streaming or while utterances remain queued.
func (s *Speaker) IsActive() bool {
	if s == nil {
		return false
	}
	return s.IsPlaying() || s.QueueDepth() > 0 || len(s.readyQueue) > 0
}

// QueueDepth returns pending text segment count.
func (s *Speaker) QueueDepth() int {
	if s == nil {
		return 0
	}
	return len(s.textQueue)
}

func (s *Speaker) synthLoop() {
	defer s.wg.Done()
	defer close(s.readyQueue)

	for {
		select {
		case <-s.runCtx.Done():
			return
		case job, ok := <-s.textQueue:
			if !ok {
				return
			}
			if s.runCtx.Err() != nil {
				return
			}
			job.text = SanitizeSpeech(job.text)
			if job.text == "" {
				continue
			}
			seg := s.startSegmentSynth(job)
			select {
			case s.readyQueue <- seg:
			case <-s.runCtx.Done():
				s.drainSegmentFrames(seg)
				return
			}
		}
	}
}

func (s *Speaker) startSegmentSynth(job speakJob) *segmentStream {
	seg := &segmentStream{
		job:    job,
		frames: make(chan []byte, 48),
		done:   make(chan struct{}),
	}
	go func() {
		defer close(seg.done)
		ctx := s.runCtx
		if ctx == nil {
			ctx = context.Background()
		}

		seg.err = s.pipeline.synthesizeFrames(ctx, job.text, func(frame []byte) error {
			select {
			case seg.frames <- frame:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		if errors.Is(seg.err, context.Canceled) {
			seg.err = ErrInterrupted
		}
		close(seg.frames)
	}()
	return seg
}

func (s *Speaker) playLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.runCtx.Done():
			return
		case seg, ok := <-s.readyQueue:
			if !ok {
				return
			}
			if s.runCtx.Err() != nil {
				s.drainSegmentFrames(seg)
				return
			}
			s.playSegment(seg)
		}
	}
}

func (s *Speaker) playSegment(seg *segmentStream) {
	job := seg.job
	start := time.Now()
	var firstByteAt atomic.Pointer[time.Time]

	playCtx, playCancel := context.WithCancel(s.runCtx)
	s.pipeline.speakMu.Lock()
	if s.pipeline.speakCancel != nil {
		s.pipeline.speakCancel()
	}
	s.pipeline.speakCtx = playCtx
	s.pipeline.speakCancel = playCancel
	s.pipeline.speakMu.Unlock()

	if !job.chainFromPrevious {
		s.pipeline.resetPaceClock()
	}
	s.pipeline.playing.Store(true)
	defer func() {
		s.pipeline.playing.Store(false)
		s.pipeline.speakMu.Lock()
		s.pipeline.speakCancel = nil
		s.pipeline.speakCtx = nil
		s.pipeline.speakMu.Unlock()
		playCancel()
	}()

	playStart := start
	s.pipeline.ArmFirstFrameHook(func() {
		ts := time.Now()
		firstByteAt.Store(&ts)
		if s.onFirstFrame != nil && !job.chainFromPrevious {
			ttsMs := int(ts.Sub(playStart).Milliseconds())
			e2eMs := 0
			if job.e2eAnchor != nil {
				if d := ts.Sub(*job.e2eAnchor).Milliseconds(); d > 0 {
					e2eMs = int(d)
				}
			}
			s.onFirstFrame(job.utteranceID, ttsMs, e2eMs)
		}
	})

	if s.onStarted != nil {
		s.onStarted(job.utteranceID, job.text, job.chainFromPrevious)
	}

	var playErr error
playback:
	for frame := range seg.frames {
		if playCtx.Err() != nil {
			playErr = ErrInterrupted
			break playback
		}
		audioBytes, err := s.pipeline.processAndSendFrame(playCtx, frame)
		if err != nil {
			playErr = err
			break playback
		}
		if len(audioBytes) > 0 {
			s.pipeline.fireFirstFrameHook()
		}
	}
	<-seg.done
	s.pipeline.ArmFirstFrameHook(nil)

	if playErr == nil && seg.err != nil {
		playErr = seg.err
	}
	if errors.Is(playErr, context.Canceled) {
		playErr = ErrInterrupted
	}

	dur := time.Since(start)
	ok := playErr == nil
	if playErr != nil && !errors.Is(playErr, ErrInterrupted) {
		log.Printf("[voice] tts segment failed utter=%s text=%q err=%v", job.utteranceID, job.text, playErr)
	}

	ttsFirstMs := 0
	e2eFirstMs := 0
	if fb := firstByteAt.Load(); fb != nil {
		ttsFirstMs = int(fb.Sub(start).Milliseconds())
		if job.e2eAnchor != nil {
			if d := fb.Sub(*job.e2eAnchor).Milliseconds(); d > 0 {
				e2eFirstMs = int(d)
			}
		}
	}

	moreQueued := len(s.textQueue) > 0 || len(s.readyQueue) > 0
	s.recordTurnLatency(job, ok, dur, ttsFirstMs, e2eFirstMs)

	if s.onEnded != nil && !moreQueued {
		reportTTS, reportE2E, reportDur, reportOK := s.finishTurnLatency(job.utteranceID)
		s.onEnded(job.utteranceID, reportOK, reportDur, reportTTS, reportE2E, false)
	}
}

func (s *Speaker) recordTurnLatency(job speakJob, ok bool, dur time.Duration, ttsFirstMs, e2eFirstMs int) {
	if job.utteranceID == "" {
		return
	}
	s.turnMu.Lock()
	defer s.turnMu.Unlock()
	lat := s.turnLatency[job.utteranceID]
	if lat == nil {
		lat = &utteranceLatency{started: time.Now(), ok: true}
		s.turnLatency[job.utteranceID] = lat
	}
	if !job.chainFromPrevious {
		lat.ttsFirstMs = ttsFirstMs
		lat.e2eFirstMs = e2eFirstMs
	}
	lat.totalDur += dur
	if !ok {
		lat.ok = false
	} else if lat.ok {
		lat.ok = true
	}
}

func (s *Speaker) finishTurnLatency(utteranceID string) (ttsFirstMs, e2eFirstMs int, dur time.Duration, ok bool) {
	s.turnMu.Lock()
	defer s.turnMu.Unlock()
	lat := s.turnLatency[utteranceID]
	delete(s.turnLatency, utteranceID)
	if lat == nil {
		return 0, 0, 0, true
	}
	return lat.ttsFirstMs, lat.e2eFirstMs, lat.totalDur, lat.ok
}

package xiaozhi

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// realtimeOutPacer buffers assistant PCM from realtime and emits
// fixed-size frames to the device at wall-clock rate.
type realtimeOutPacer struct {
	s *wsSession

	mu         sync.Mutex
	pcmBuf     []byte
	frameBytes int
	frameDur   time.Duration
	turnBytes  int // PCM bytes pushed for current assistant turn

	playCtx    context.Context
	playCancel context.CancelFunc
	playWG     sync.WaitGroup

	turnActive atomic.Bool
}

func newRealtimeOutPacer(s *wsSession) *realtimeOutPacer {
	ms := s.ttsWireFrameMs
	if ms <= 0 {
		ms = s.inFrameMs
	}
	if ms <= 0 {
		ms = 60
	}
	dur := time.Duration(ms) * time.Millisecond
	sr := s.outSR
	if sr <= 0 {
		sr = 16000
	}
	samples := int(float64(sr) * dur.Seconds())
	fb := samples * 2
	if fb < 2 {
		fb = 640
	}
	return &realtimeOutPacer{
		s:          s,
		frameBytes: fb,
		frameDur:   dur,
	}
}

func (p *realtimeOutPacer) close() {
	p.interrupt(false)
	p.playWG.Wait()
}

func (p *realtimeOutPacer) push(pcm []byte) {
	if p == nil || len(pcm) == 0 || p.s.closed.Load() {
		return
	}
	p.mu.Lock()
	p.pcmBuf = append(p.pcmBuf, pcm...)
	p.turnBytes += len(pcm)
	if p.playCancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		p.playCancel = cancel
		p.playCtx = ctx
		p.playWG.Add(1)
		go p.playLoop(ctx)
	}
	p.mu.Unlock()
}

func (p *realtimeOutPacer) endTurn() (playbackTail time.Duration) {
	if p == nil {
		return 0
	}
	p.turnActive.Store(false)
	p.mu.Lock()
	cancel := p.playCancel
	p.playCancel = nil
	p.playCtx = nil
	turnBytes := p.turnBytes
	p.turnBytes = 0
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.playWG.Wait()
	sr := p.s.outSR
	if sr <= 0 {
		sr = 16000
	}
	if turnBytes > 0 {
		playbackTail = time.Duration(turnBytes) * time.Second / time.Duration(sr*2)
	}
	return playbackTail
}

func (p *realtimeOutPacer) interrupt(sendStop bool) {
	if p == nil {
		return
	}
	p.turnActive.Store(false)
	p.mu.Lock()
	p.pcmBuf = nil
	p.turnBytes = 0
	cancel := p.playCancel
	p.playCancel = nil
	p.playCtx = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.playWG.Wait()
	if sendStop && p.s.ttsActive.CompareAndSwap(true, false) {
		p.s.writeText(MakeTTSStateReply(p.s.sessionID, "stop", p.s.outFormat))
	}
}

func (p *realtimeOutPacer) playLoop(ctx context.Context) {
	defer p.playWG.Done()
	defer func() {
		p.mu.Lock()
		p.playCancel = nil
		p.playCtx = nil
		p.mu.Unlock()
	}()

	nextDeadline := time.Now()

	flushPaced := func() {
		for {
			if ctx.Err() != nil {
				return
			}
			p.mu.Lock()
			if len(p.pcmBuf) < p.frameBytes {
				p.mu.Unlock()
				return
			}
			frame := make([]byte, p.frameBytes)
			copy(frame, p.pcmBuf[:p.frameBytes])
			p.pcmBuf = p.pcmBuf[p.frameBytes:]
			p.mu.Unlock()

			wait := time.Until(nextDeadline)
			if wait > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(wait):
				}
			}
			nextDeadline = nextDeadline.Add(p.frameDur)
			_ = p.s.emitOutboundPCMFrame(frame)
		}
	}

	for {
		if ctx.Err() != nil {
			p.drainRemaining()
			return
		}
		p.mu.Lock()
		ready := len(p.pcmBuf) >= p.frameBytes
		p.mu.Unlock()
		if ready {
			flushPaced()
			continue
		}
		select {
		case <-ctx.Done():
			p.drainRemaining()
			return
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// drainRemaining sends all buffered PCM immediately (no wall-clock pacing).
// Called when the model turn ends while audio is still queued.
func (p *realtimeOutPacer) drainRemaining() {
	for {
		p.mu.Lock()
		if len(p.pcmBuf) == 0 {
			p.mu.Unlock()
			return
		}
		var frame []byte
		if len(p.pcmBuf) >= p.frameBytes {
			frame = make([]byte, p.frameBytes)
			copy(frame, p.pcmBuf[:p.frameBytes])
			p.pcmBuf = p.pcmBuf[p.frameBytes:]
		} else {
			frame = make([]byte, p.frameBytes)
			copy(frame, p.pcmBuf)
			p.pcmBuf = nil
		}
		p.mu.Unlock()
		_ = p.s.emitOutboundPCMFrame(frame)
	}
}

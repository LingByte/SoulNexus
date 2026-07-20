package webrtc

import (
	"context"
	"sync"
	"time"

	pionmedia "github.com/pion/webrtc/v3/pkg/media"
)

// sampleWriter is satisfied by *webrtc.TrackLocalStaticSample.
type sampleWriter interface {
	WriteSample(pionmedia.Sample) error
}

// kickBurstFrames is how many Opus packets to stamp immediately after an
// idle gap so the browser jitter buffer can start playout without waiting
// for a full realtime ramp (5×20ms ≈ 100ms of audio).
const kickBurstFrames = 5

// defaultPacerDepth is ~6s of 20ms frames. Cascaded TTS expects Speak to
// return when synthesis finishes (PaceRealtime=false) so the next segment
// can dial while prior audio still plays. A shallow queue (~1s) made
// Speak block on playout and delayed later segments by seconds.
const defaultPacerDepth = 300

// downlinkPacer paces Opus WriteSample calls to ~realtime so the browser
// jitter buffer is not flooded. Cascaded TTS uses PaceRealtime=false
// (Speak returns when synthesis finishes so the next segment can dial
// early); without a transport-side pacer, many seconds of Opus are
// stamped into the track in tens of milliseconds and later phrases are
// dropped by the peer.
//
// After a silence gap, the first kickBurstFrames packets are written
// immediately (priming), then pacing resumes.
//
// Enqueue blocks once the queue holds ~depth frames (backpressure onto
// Speak). Flush discards queued samples on barge-in.
type downlinkPacer struct {
	track    sampleWriter
	writeMu  *sync.Mutex
	duration time.Duration
	ch       chan []byte
	quit     chan struct{}
	done     chan struct{}

	closeOnce sync.Once
}

func newDownlinkPacer(ctx context.Context, track sampleWriter, writeMu *sync.Mutex, frameDur time.Duration, depth int) *downlinkPacer {
	if frameDur <= 0 {
		frameDur = 20 * time.Millisecond
	}
	if depth <= 0 {
		depth = defaultPacerDepth
	}
	p := &downlinkPacer{
		track:    track,
		writeMu:  writeMu,
		duration: frameDur,
		ch:       make(chan []byte, depth),
		quit:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go p.loop(ctx)
	return p
}

func (p *downlinkPacer) Enqueue(payload []byte) error {
	if p == nil || len(payload) == 0 {
		return nil
	}
	select {
	case <-p.quit:
		return nil
	default:
	}
	select {
	case p.ch <- append([]byte(nil), payload...):
		return nil
	case <-p.quit:
		return nil
	}
}

// Flush drops queued samples so barge-in does not keep playing the old reply.
func (p *downlinkPacer) Flush() {
	if p == nil {
		return
	}
	for {
		select {
		case <-p.ch:
		default:
			return
		}
	}
}

func (p *downlinkPacer) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		close(p.quit)
	})
	<-p.done
}

func (p *downlinkPacer) loop(ctx context.Context) {
	defer close(p.done)
	var next time.Time
	burstLeft := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.quit:
			return
		case payload := <-p.ch:
			if len(payload) == 0 {
				continue
			}
			now := time.Now()
			idleGap := next.IsZero() || next.Before(now.Add(-2*p.duration))
			if idleGap {
				// New utterance / post-Flush: prime JB, then pace.
				burstLeft = kickBurstFrames
				next = now
			}
			if burstLeft > 0 {
				burstLeft--
			} else if d := next.Sub(now); d > 0 {
				timer := time.NewTimer(d)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-p.quit:
					timer.Stop()
					return
				case <-timer.C:
				}
				now = time.Now()
			}
			p.writeMu.Lock()
			err := p.track.WriteSample(pionmedia.Sample{Data: payload, Duration: p.duration})
			p.writeMu.Unlock()
			if err != nil {
				return
			}
			// Schedule from wall clock after write so burst packs don't
			// leave next stuck in the past (which would disable pacing).
			next = time.Now().Add(p.duration)
		}
	}
}

// Package pcm provides a generic engine.MediaPort over buffered PCM channels.
package pcm

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

// ErrClosed is returned after the port is closed.
var ErrClosed = errors.New("dialog/transport/pcm: closed")

// Port implements engine.MediaPort for WebSocket / WebRTC / tests.
type Port struct {
	sessionID  string
	tenantID   string
	sampleRate int
	codec      engine.CodecSpec

	in chan engine.PCMFrame

	// OutputFn delivers AI PCM to the transport (WebSocket / WebRTC).
	OutputFn func(engine.PCMFrame) error

	bargeMu sync.RWMutex
	barge   func()

	closeOnce sync.Once
	closed    atomic.Bool
}

// Config for NewPort.
type Config struct {
	SessionID  string
	TenantID   string
	SampleRate int
	Buffer     int
}

// NewPort builds a PCM MediaPort. TenantID is string for engine contract.
func NewPort(cfg Config) *Port {
	sr := cfg.SampleRate
	if sr <= 0 {
		sr = 16000
	}
	buf := cfg.Buffer
	if buf <= 0 {
		buf = 32
	}
	return &Port{
		sessionID:  cfg.SessionID,
		tenantID:   cfg.TenantID,
		sampleRate: sr,
		codec: engine.CodecSpec{
			Name:       "PCM",
			SampleRate: sr,
			Channels:   1,
		},
		in: make(chan engine.PCMFrame, buf),
	}
}

var _ engine.MediaPort = (*Port)(nil)

func (p *Port) InputPCM() <-chan engine.PCMFrame { return p.in }

func (p *Port) SendOutputPCM(f engine.PCMFrame) error {
	if p == nil || p.closed.Load() {
		return ErrClosed
	}
	if p.OutputFn != nil {
		return p.OutputFn(f)
	}
	return nil
}

func (p *Port) OnBargeIn(fn func()) {
	if p == nil {
		return
	}
	p.bargeMu.Lock()
	p.barge = fn
	p.bargeMu.Unlock()
}

func (p *Port) Codec() engine.CodecSpec {
	if p == nil {
		return engine.CodecSpec{}
	}
	return p.codec
}

func (p *Port) SampleRate() int {
	if p == nil {
		return 16000
	}
	return p.sampleRate
}

func (p *Port) CallID() string {
	if p == nil {
		return ""
	}
	return p.sessionID
}

func (p *Port) TenantID() string {
	if p == nil {
		return ""
	}
	return p.tenantID
}

// PushInput enqueues caller PCM into the engine input channel.
func (p *Port) PushInput(data []byte) error {
	if p == nil || p.closed.Load() {
		return ErrClosed
	}
	if len(data) == 0 {
		return nil
	}
	frame := engine.PCMFrame{
		Data:       append([]byte(nil), data...),
		SampleRate: p.sampleRate,
		Timestamp:  time.Now(),
	}
	select {
	case p.in <- frame:
		return nil
	default:
		// drop-oldest
		select {
		case <-p.in:
		default:
		}
		select {
		case p.in <- frame:
			return nil
		default:
			return errors.New("dialog/transport/pcm: input overflow")
		}
	}
}

// FireBargeIn invokes the registered barge-in callback.
func (p *Port) FireBargeIn() {
	if p == nil {
		return
	}
	p.bargeMu.RLock()
	fn := p.barge
	p.bargeMu.RUnlock()
	if fn != nil {
		fn()
	}
}

// TriggerBargeIn implements the barge-in hook expected by session.AttachEngine
// (StreamingPort naming).
func (p *Port) TriggerBargeIn() {
	p.FireBargeIn()
}

// Close shuts the port and closes InputPCM.
func (p *Port) Close() error {
	if p == nil {
		return nil
	}
	p.closeOnce.Do(func() {
		p.closed.Store(true)
		close(p.in)
	})
	return nil
}

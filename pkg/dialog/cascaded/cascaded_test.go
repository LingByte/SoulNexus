// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

// fakeMediaPort is a synchronous, in-process MediaPort for tests.
// Exposes the input/output channels directly so test bodies can
// drive frames and assert on what the engine emits.
type fakeMediaPort struct {
	in       chan engine.PCMFrame
	outMu    sync.Mutex
	out      []engine.PCMFrame
	sendErr  error
	codec    engine.CodecSpec
	sr       int
	callID   string
	tenantID string

	bargeMu sync.Mutex
	barge   func()
}

func newFakePort() *fakeMediaPort {
	return &fakeMediaPort{
		in:       make(chan engine.PCMFrame, 8),
		codec:    engine.CodecSpec{Name: "PCMU", SampleRate: 8000, Channels: 1},
		sr:       8000,
		callID:   "call-fake",
		tenantID: "tenant-fake",
	}
}

func (p *fakeMediaPort) InputPCM() <-chan engine.PCMFrame { return p.in }
func (p *fakeMediaPort) SendOutputPCM(f engine.PCMFrame) error {
	if p.sendErr != nil {
		return p.sendErr
	}
	p.outMu.Lock()
	p.out = append(p.out, f)
	p.outMu.Unlock()
	return nil
}
func (p *fakeMediaPort) OnBargeIn(fn func()) {
	p.bargeMu.Lock()
	p.barge = fn
	p.bargeMu.Unlock()
}
func (p *fakeMediaPort) Codec() engine.CodecSpec { return p.codec }
func (p *fakeMediaPort) SampleRate() int         { return p.sr }
func (p *fakeMediaPort) CallID() string          { return p.callID }
func (p *fakeMediaPort) TenantID() string        { return p.tenantID }

// ----- Engine lifecycle ---------------------------------------------

func TestEngine_Mode(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	if got := e.Mode(); got != engine.ModeCascaded {
		t.Errorf("Mode = %q, want cascaded", got)
	}
}

func TestEngine_AttachHappyPath(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	port := newFakePort()
	detach, err := e.Attach(context.Background(), port, engine.NopLogger{})
	if err != nil {
		t.Fatalf("Attach err = %v", err)
	}
	if detach == nil {
		t.Fatal("Attach returned nil Detach")
	}

	// Feed one frame so asrStub increments its counter and emits a
	// transcript when the input closes.
	port.in <- engine.PCMFrame{Data: []byte{0, 0}, SampleRate: 8000}
	close(port.in)

	// Wait for the pipeline to drain on its own (asrStub exits when
	// input closes, then llm/tts cascade through to completion).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-e.done:
			goto drained
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("pipeline did not drain within deadline")
drained:

	// Detach after natural drain must still return nil (idempotent
	// and observes the recorded pipeErr, which should also be nil).
	if err := detach(context.Background()); err != nil {
		t.Errorf("detach after drain err = %v, want nil", err)
	}
}

func TestEngine_AttachTwiceRejected(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	port := newFakePort()
	if _, err := e.Attach(context.Background(), port, nil); err != nil {
		t.Fatalf("first Attach err = %v", err)
	}
	_, err := e.Attach(context.Background(), port, nil)
	if !errors.Is(err, ErrAlreadyAttached) {
		t.Errorf("second Attach err = %v, want ErrAlreadyAttached", err)
	}
}

func TestEngine_AttachNilPortRejected(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	if _, err := e.Attach(context.Background(), nil, nil); err == nil {
		t.Error("Attach(nil port) should error")
	}
}

func TestEngine_DetachIdempotent(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	port := newFakePort()
	detach, err := e.Attach(context.Background(), port, nil)
	if err != nil {
		t.Fatalf("Attach err = %v", err)
	}

	close(port.in)
	// First detach may race ahead of natural drain → returns
	// context.Canceled. That's expected; the IDEMPOTENCY contract is
	// about subsequent calls being no-ops, not about the first call's
	// error code.
	if err := detach(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("first detach err = %v, want nil or context.Canceled", err)
	}
	if err := detach(context.Background()); err != nil {
		t.Errorf("second detach err = %v, want nil (idempotent)", err)
	}
	if err := detach(context.Background()); err != nil {
		t.Errorf("third detach err = %v, want nil (idempotent)", err)
	}
}

func TestEngine_DetachCancelsCtx(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	port := newFakePort()
	// Long-running call: don't close port.in; the engine must shut
	// down via Detach.
	detach, err := e.Attach(context.Background(), port, nil)
	if err != nil {
		t.Fatalf("Attach err = %v", err)
	}

	// Push a couple of frames; nothing comes back (TTS stub emits
	// no PCM), but the engine should be in the steady running state.
	port.in <- engine.PCMFrame{Data: []byte{0, 0}, SampleRate: 8000}
	port.in <- engine.PCMFrame{Data: []byte{0, 0}, SampleRate: 8000}

	if err := detach(context.Background()); err != nil {
		// Detach may surface ctx.Canceled from the pipeline stages —
		// that's acceptable because cancellation IS the teardown
		// signal. We only flag unexpected errors.
		if !errors.Is(err, context.Canceled) {
			t.Errorf("detach err = %v, want nil or context.Canceled", err)
		}
	}
}

func TestEngine_UplinkPCMNotEchoedToOutput(t *testing.T) {
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	port := newFakePort()
	detach, err := e.Attach(context.Background(), port, nil)
	if err != nil {
		t.Fatalf("Attach err = %v", err)
	}
	defer func() { _ = detach(context.Background()) }()

	port.in <- engine.PCMFrame{Data: []byte{1, 2, 3, 4}, SampleRate: 8000}
	port.in <- engine.PCMFrame{Data: []byte{5, 6, 7, 8}, SampleRate: 8000}
	time.Sleep(120 * time.Millisecond)

	port.outMu.Lock()
	n := len(port.out)
	port.outMu.Unlock()
	if n != 0 {
		t.Fatalf("uplink passthrough echoed %d PCM frames to output, want 0", n)
	}
}

func TestEngine_CtxCancelTearsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	e := New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1", TenantID: "t1"})
	port := newFakePort()
	_, err := e.Attach(ctx, port, nil)
	if err != nil {
		t.Fatalf("Attach err = %v", err)
	}

	cancel()

	// done must close shortly after ctx cancel.
	select {
	case <-e.done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("ctx cancel did not tear down engine within deadline")
	}
}

// ----- Factory ------------------------------------------------------

func TestFactory_BuildsCascadedEngine(t *testing.T) {
	f := NewFactory()
	eng, err := f.Build(engine.Config{Mode: engine.ModeCascaded, CallID: "c1"})
	if err != nil {
		t.Fatalf("Build err = %v", err)
	}
	if eng.Mode() != engine.ModeCascaded {
		t.Errorf("built engine.Mode = %q, want cascaded", eng.Mode())
	}
}

func TestFactory_RejectsWrongMode(t *testing.T) {
	f := NewFactory()
	_, err := f.Build(engine.Config{Mode: engine.ModeRealtime, CallID: "c1"})
	if err == nil {
		t.Error("Build with realtime mode should error")
	}
}

func TestRegisterForTesting_HappyAndDuplicate(t *testing.T) {
	// Clean slate so this test doesn't depend on whether other tests
	// have registered modes.
	engine.ResetRegistryForTest()
	defer engine.ResetRegistryForTest()

	if err := RegisterForTesting(); err != nil {
		t.Fatalf("first RegisterForTesting err = %v", err)
	}
	modes := engine.RegisteredModes()
	found := false
	for _, m := range modes {
		if m == engine.ModeCascaded {
			found = true
		}
	}
	if !found {
		t.Errorf("after RegisterForTesting, modes = %v; want cascaded present", modes)
	}

	// Second call must not propagate the panic from engine.Register.
	if err := RegisterForTesting(); err != nil {
		t.Errorf("second RegisterForTesting err = %v; want recovered nil", err)
	}

	// engine.New through the registry should now build a cascaded.Engine.
	built, err := engine.New(engine.Config{Mode: engine.ModeCascaded, CallID: "c1"})
	if err != nil {
		t.Fatalf("engine.New err = %v", err)
	}
	if _, ok := built.(*Engine); !ok {
		t.Errorf("engine.New returned %T, want *cascaded.Engine", built)
	}
}

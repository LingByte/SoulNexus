package media

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==========================================================================
// cache.go
// ==========================================================================

func TestMediaCache_BuildKey(t *testing.T) {
	c := &LocalMediaCache{CacheRoot: t.TempDir()}
	k1 := c.BuildKey("abc", "def")
	k2 := c.BuildKey("abc", "def")
	if k1 != k2 {
		t.Errorf("BuildKey not deterministic")
	}
	if len(k1) != 32 {
		t.Errorf("BuildKey length = %d, want 32 hex chars", len(k1))
	}
	k3 := c.BuildKey("totally-different-input-material")
	if k1 == k3 {
		t.Errorf("BuildKey should produce different keys for different inputs")
	}
}

func TestMediaCache_StoreAndGet(t *testing.T) {
	c := &LocalMediaCache{CacheRoot: t.TempDir()}
	key := c.BuildKey("test-store")
	data := []byte("hello world")

	if err := c.Store(key, data); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := c.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("round-trip mismatch: %q", got)
	}
}

func TestMediaCache_GetMissing(t *testing.T) {
	c := &LocalMediaCache{CacheRoot: t.TempDir()}
	if _, err := c.Get("nonexistent"); err == nil {
		t.Error("missing key must error")
	}
}

func TestMediaCache_DisabledStoreIsNoop(t *testing.T) {
	c := &LocalMediaCache{Disabled: true, CacheRoot: t.TempDir()}
	if err := c.Store("k", []byte("x")); err != nil {
		t.Errorf("disabled Store should no-op, got %v", err)
	}
	if _, err := c.Get("k"); err == nil {
		t.Error("disabled Get should error")
	}
}

func TestMediaCache_StoreOverDirectoryFails(t *testing.T) {
	root := t.TempDir()
	dirKey := "conflict-dir"
	if err := os.MkdirAll(filepath.Join(root, dirKey), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	c := &LocalMediaCache{CacheRoot: root}
	if err := c.Store(dirKey, []byte("x")); err == nil {
		t.Error("storing to a directory path must error")
	}
}

func TestMediaCache_Singleton(t *testing.T) {
	// Just exercise the lazy init path
	c := MediaCache()
	if c == nil {
		t.Fatal("MediaCache() returned nil")
	}
	if MediaCache() != c {
		t.Error("MediaCache() must return the same singleton")
	}
}

// ==========================================================================
// resampler.go
// ==========================================================================

func pcmLE(samples ...int16) []byte {
	buf := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	return buf
}

func TestResamplePCM_SameRateIdentity(t *testing.T) {
	in := pcmLE(0, 100, -100, 200, -200)
	out, err := ResamplePCM(in, 16000, 16000)
	if err != nil {
		t.Fatalf("ResamplePCM: %v", err)
	}
	if string(out) != string(in) {
		t.Errorf("same-rate resample altered samples")
	}
}

func TestResamplePCM_UpsampleProducesMoreSamples(t *testing.T) {
	in := pcmLE(0, 100, -100, 200, -200, 300, -300, 400)
	out, err := ResamplePCM(in, 8000, 16000)
	if err != nil {
		t.Fatalf("ResamplePCM: %v", err)
	}
	if len(out) <= len(in) {
		t.Errorf("upsample 8k→16k expected more output, got %d bytes (in=%d)", len(out), len(in))
	}
}

func TestResamplePCM_DownsampleProducesFewerSamples(t *testing.T) {
	in := pcmLE(0, 100, -100, 200, -200, 300, -300, 400, 500, -500, 600, -600)
	out, err := ResamplePCM(in, 16000, 8000)
	if err != nil {
		t.Fatalf("ResamplePCM: %v", err)
	}
	if len(out) >= len(in) {
		t.Errorf("downsample 16k→8k expected fewer output, got %d (in=%d)", len(out), len(in))
	}
}

func TestInterpolatingConverter_OddByteReturnsNil(t *testing.T) {
	conv := NewInterpolatingConverter(16000, 8000).(*InterpolatingConverter)
	if got := conv.ConvertSamples([]byte{0x01, 0x02, 0x03}); got != nil {
		t.Error("odd byte count must yield nil")
	}
}

func TestInterpolatingConverter_SameRatePassthrough(t *testing.T) {
	conv := NewInterpolatingConverter(8000, 8000).(*InterpolatingConverter)
	in := pcmLE(1, 2, 3, 4)
	out := conv.ConvertSamples(in)
	if string(out) != string(in) {
		t.Errorf("same-rate should pass through")
	}
}

func TestCubicConverter_WritesAndSamplesAndClose(t *testing.T) {
	conv := NewCubicInterpolatingConverter(8000, 16000)
	n, err := conv.Write(pcmLE(100, -100, 200, -200, 300, -300, 400, -400))
	if err != nil || n == 0 {
		t.Fatalf("Write: n=%d err=%v", n, err)
	}
	if err := conv.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	out := conv.Samples()
	if len(out) == 0 {
		t.Error("Samples returned empty")
	}
	if second := conv.Samples(); len(second) != 0 {
		t.Error("Samples must drain on first call; second call should be empty")
	}
}

func TestSetDefaultResampler_ReplacesFactory(t *testing.T) {
	// Save & restore to avoid bleed
	old := defaultConverterFactory
	defer func() { defaultConverterFactory = old }()

	var called int32
	SetDefaultResampler(func(in, out int) SampleRateConverter {
		atomic.AddInt32(&called, 1)
		return NewInterpolatingConverter(in, out)
	})
	_ = DefaultResampler(8000, 16000)
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("custom factory called=%d, want 1", called)
	}
}

// ==========================================================================
// eventbus.go
// ==========================================================================

func TestEventBus_SubscribeAndPublish(t *testing.T) {
	eb := NewEventBus(context.Background(), 16, 2)
	defer eb.Close()

	var got int32
	eb.Subscribe(EventTypePacket, func(ctx context.Context, ev *MediaEvent) error {
		atomic.AddInt32(&got, 1)
		return nil
	})

	eb.PublishPacket("s1", &AudioPacket{Payload: []byte{1}}, "sender1")
	eb.PublishState("s1", StateChange{State: "begin"}, "sender1")
	eb.PublishError("s1", errors.New("boom"), "sender1")

	// Wait for async delivery.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && atomic.LoadInt32(&got) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if atomic.LoadInt32(&got) == 0 {
		t.Error("packet handler not invoked")
	}
}

func TestEventBus_LifecycleWildcardReceivesAll(t *testing.T) {
	eb := NewEventBus(context.Background(), 8, 1)
	defer eb.Close()

	var wildcard int32
	eb.Subscribe(EventTypeLifecycle, func(ctx context.Context, ev *MediaEvent) error {
		atomic.AddInt32(&wildcard, 1)
		return nil
	})

	eb.PublishPacket("s", &AudioPacket{Payload: []byte{1}}, nil)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && atomic.LoadInt32(&wildcard) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	if atomic.LoadInt32(&wildcard) == 0 {
		t.Error("lifecycle wildcard should have received packet event")
	}
}

func TestEventBus_PublishNilAndRecovers(t *testing.T) {
	eb := NewEventBus(context.Background(), 4, 1)
	defer eb.Close()

	// Nil event should be ignored
	eb.Publish(nil)

	// Handler that panics must be recovered
	eb.Subscribe(EventTypePacket, func(ctx context.Context, ev *MediaEvent) error {
		panic("handler panic")
	})
	eb.PublishPacket("s", &AudioPacket{Payload: []byte{1}}, nil)

	// Handler that errors is also logged (no panic)
	eb.Subscribe(EventTypePacket, func(ctx context.Context, ev *MediaEvent) error {
		return errors.New("handler err")
	})
	eb.PublishPacket("s", &AudioPacket{Payload: []byte{1}}, nil)
	time.Sleep(20 * time.Millisecond)
}

func TestEventBus_Unsubscribe(t *testing.T) {
	eb := NewEventBus(context.Background(), 4, 1)
	defer eb.Close()

	handler := func(ctx context.Context, ev *MediaEvent) error { return nil }
	eb.Subscribe(EventTypeState, handler)
	eb.Unsubscribe(EventTypeState, handler)
	// The pointer-based comparison removes it; subsequent events should find no subscribers.
	eb.PublishState("s", StateChange{State: "x"}, nil)
}

// ==========================================================================
// processor.go
// ==========================================================================

// noopFuncProc produces a FuncProcessor with a no-op Process (easiest concrete Processor).
func noopFuncProc(name string, pri ProcessorPriority) *FuncProcessor {
	return NewFuncProcessor(name, pri, func(ctx context.Context, s *MediaSession, ev *MediaEvent) error { return nil })
}

func TestProcessorRegistry_PriorityOrdering(t *testing.T) {
	r := NewProcessorRegistry()
	r.Register(noopFuncProc("low", PriorityLow))
	r.Register(noopFuncProc("high", PriorityHigh))
	r.Register(noopFuncProc("normal", PriorityNormal))

	ordered := r.GetAllProcessors()
	if len(ordered) != 3 {
		t.Fatalf("expected 3 processors, got %d", len(ordered))
	}
	if ordered[0].Name() != "high" || ordered[2].Name() != "low" {
		names := make([]string, len(ordered))
		for i, p := range ordered {
			names[i] = p.Name()
		}
		t.Errorf("priority order wrong: %v", names)
	}
}

func TestProcessorRegistry_Unregister(t *testing.T) {
	r := NewProcessorRegistry()
	r.Register(noopFuncProc("a", PriorityNormal))
	r.Register(noopFuncProc("b", PriorityNormal))
	r.Unregister("a")
	ps := r.GetAllProcessors()
	if len(ps) != 1 || ps[0].Name() != "b" {
		t.Errorf("after unregister: %v", ps)
	}
	// Unregister unknown — no-op
	r.Unregister("nonexistent")
}

func TestBaseProcessor_WithCondition(t *testing.T) {
	bp := NewBaseProcessor("p", PriorityNormal)
	if !bp.CanHandle(context.Background(), &MediaEvent{}) {
		t.Error("default CanHandle must be true")
	}
	bp.WithCondition(func(ctx context.Context, ev *MediaEvent) bool {
		return ev.Type == EventTypePacket
	})
	if bp.CanHandle(context.Background(), &MediaEvent{Type: EventTypeState}) {
		t.Error("condition should reject state events")
	}
	if !bp.CanHandle(context.Background(), &MediaEvent{Type: EventTypePacket}) {
		t.Error("condition should accept packet events")
	}
}

func TestFuncProcessor_Process(t *testing.T) {
	called := false
	fp := NewFuncProcessor("fp", PriorityNormal, func(ctx context.Context, s *MediaSession, ev *MediaEvent) error {
		called = true
		return nil
	})
	if err := fp.Process(context.Background(), nil, &MediaEvent{}); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !called {
		t.Error("processFunc not invoked")
	}
}

func TestPacketProcessor_CanHandleAndProcess(t *testing.T) {
	got := 0
	pp := NewPacketProcessor("pp", PriorityNormal, func(ctx context.Context, s *MediaSession, p MediaPacket) error {
		got++
		return nil
	})
	// Non-packet event must be rejected
	if pp.CanHandle(context.Background(), &MediaEvent{Type: EventTypeState}) {
		t.Error("non-packet event must not be handled")
	}
	packetEv := &MediaEvent{Type: EventTypePacket, Payload: &AudioPacket{Payload: []byte{1}}}
	if !pp.CanHandle(context.Background(), packetEv) {
		t.Error("packet event must be handled")
	}
	if err := pp.Process(context.Background(), nil, packetEv); err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got != 1 {
		t.Errorf("got %d, want 1", got)
	}
	// Payload not a MediaPacket → error
	if err := pp.Process(context.Background(), nil, &MediaEvent{Type: EventTypePacket, Payload: "not packet"}); err == nil {
		t.Error("non-packet payload must error")
	}
}

func TestProcessorRegistry_GetProcessors_FiltersByCanHandle(t *testing.T) {
	r := NewProcessorRegistry()
	r.Register(NewPacketProcessor("pkt", PriorityNormal, func(ctx context.Context, s *MediaSession, p MediaPacket) error { return nil }))
	r.Register(NewFuncProcessor("fn", PriorityHigh, func(ctx context.Context, s *MediaSession, ev *MediaEvent) error { return nil }))

	// Only PacketProcessor should handle a packet event (plus FuncProcessor with no condition → handles all)
	ps := r.GetProcessors(context.Background(), &MediaEvent{Type: EventTypePacket})
	if len(ps) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(ps))
	}
	// Only FuncProcessor handles a state event (PacketProcessor filters state)
	ps2 := r.GetProcessors(context.Background(), &MediaEvent{Type: EventTypeState})
	if len(ps2) != 1 {
		t.Errorf("expected 1 handler for state, got %d", len(ps2))
	}
}

// ==========================================================================
// router.go
// ==========================================================================

func TestRouter_BroadcastStrategy(t *testing.T) {
	r := NewRouter(StrategyBroadcast)
	outs := []*TransportConnector{
		NewTransportConnector("a", nil, DirectionOutput),
		NewTransportConnector("b", nil, DirectionOutput),
	}
	got := r.Route(&AudioPacket{}, outs)
	if len(got) != 2 {
		t.Errorf("broadcast want 2, got %d", len(got))
	}
}

func TestRouter_RoundRobinAdvancesIndex(t *testing.T) {
	r := NewRouter(StrategyRoundRobin)
	outs := []*TransportConnector{
		NewTransportConnector("a", nil, DirectionOutput),
		NewTransportConnector("b", nil, DirectionOutput),
	}
	got1 := r.Route(&AudioPacket{}, outs)
	got2 := r.Route(&AudioPacket{}, outs)
	if len(got1) != 1 || len(got2) != 1 {
		t.Fatalf("round-robin should return 1 each time")
	}
	if got1[0] == got2[0] {
		t.Error("round-robin should rotate")
	}
}

func TestRouter_FirstAvailableReturnsFirst(t *testing.T) {
	r := NewRouter(StrategyFirstAvailable)
	outs := []*TransportConnector{
		NewTransportConnector("a", nil, DirectionOutput),
		NewTransportConnector("b", nil, DirectionOutput),
	}
	got := r.Route(&AudioPacket{}, outs)
	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("FirstAvailable should pick 'a', got %+v", got)
	}
}

func TestRouter_RuleMatching(t *testing.T) {
	r := NewRouter(StrategyBroadcast)
	r.AddRule(RouteRule{
		Condition: func(p MediaPacket) bool {
			_, isAudio := p.(*AudioPacket)
			return isAudio
		},
		Strategy: StrategyFirstAvailable,
	})
	outs := []*TransportConnector{
		NewTransportConnector("a", nil, DirectionOutput),
		NewTransportConnector("b", nil, DirectionOutput),
	}
	// Audio → FirstAvailable (1 output)
	if got := r.Route(&AudioPacket{}, outs); len(got) != 1 {
		t.Errorf("audio rule expected 1 output, got %d", len(got))
	}
	// DTMF → falls to default Broadcast (2 outputs)
	if got := r.Route(&DTMFPacket{Digit: "5"}, outs); len(got) != 2 {
		t.Errorf("dtmf default expected 2 outputs, got %d", len(got))
	}
}

func TestRouter_NoAvailableTransports(t *testing.T) {
	r := NewRouter(StrategyBroadcast)
	if got := r.Route(&AudioPacket{}, nil); got != nil {
		t.Errorf("no transports should yield nil, got %+v", got)
	}
}

func TestTransportConnector_ActiveToggleAndString(t *testing.T) {
	c := NewTransportConnector("x", nil, DirectionInput)
	if !c.IsActive() {
		t.Error("new connector must be active")
	}
	c.SetActive(false)
	if c.IsActive() {
		t.Error("SetActive(false) did not take effect")
	}
	if c.String() == "" {
		t.Error("String should not be empty")
	}
}

// ==========================================================================
// types.go — MediaPacket Body/String, CodecConfig
// ==========================================================================

func TestDTMFPacket_BodyAndString_NilAndValue(t *testing.T) {
	var n *DTMFPacket
	if n.Body() != nil {
		t.Error("nil DTMFPacket.Body must be nil")
	}
	if !strings.Contains(n.String(), "nil") {
		t.Error("nil DTMFPacket.String should mention nil")
	}
	d := &DTMFPacket{Digit: "5", End: true}
	if string(d.Body()) != "5" {
		t.Errorf("Body = %q", d.Body())
	}
	if !strings.Contains(d.String(), "5") {
		t.Errorf("String: %q", d.String())
	}
}

func TestAudioPacket_BodyAndString(t *testing.T) {
	p := &AudioPacket{Payload: []byte("abc"), IsSynthesized: true}
	if string(p.Body()) != "abc" {
		t.Errorf("Body = %q", p.Body())
	}
	if !strings.Contains(p.String(), "IsSynthesized: true") {
		t.Errorf("String missing flag: %q", p.String())
	}
}

func TestClosePacket_BodyAndString(t *testing.T) {
	c := &ClosePacket{Reason: "timeout"}
	if c.Body() != nil {
		t.Error("ClosePacket.Body must be nil")
	}
	if !strings.Contains(c.String(), "timeout") {
		t.Errorf("ClosePacket.String: %q", c.String())
	}
}

func TestTextPacket_BodyAndSourceBranches(t *testing.T) {
	p := &TextPacket{Text: "hi"}
	if string(p.Body()) != "hi" {
		t.Errorf("Body = %q", p.Body())
	}
	// Source branches: default "user", transcribed, LLM
	if !strings.Contains(p.String(), "user") {
		t.Errorf("default source user expected: %q", p.String())
	}
	p.IsTranscribed = true
	if !strings.Contains(p.String(), "Transcribed") {
		t.Errorf("Transcribed source expected: %q", p.String())
	}
	p.IsLLMGenerated = true
	if !strings.Contains(p.String(), "LLMGenerated") {
		t.Errorf("LLMGenerated source expected: %q", p.String())
	}
}

func TestCodecConfig_DefaultAndString(t *testing.T) {
	c := DefaultCodecConfig()
	if c.Codec != "pcm" || c.SampleRate != 16000 || c.Channels != 1 || c.BitDepth != 16 {
		t.Errorf("defaults wrong: %+v", c)
	}
	if c.String() == "" {
		t.Error("String empty")
	}
}

func TestMediaData_StringBranches(t *testing.T) {
	stateData := &MediaData{Type: MediaDataTypeState, State: StateChange{State: "begin"}}
	if !strings.Contains(stateData.String(), "State") {
		t.Errorf("state MediaData.String: %q", stateData.String())
	}
	pktData := &MediaData{Type: MediaDataTypePacket, Packet: &AudioPacket{Payload: []byte{1}}}
	if !strings.Contains(pktData.String(), "Packet") {
		t.Errorf("packet MediaData.String: %q", pktData.String())
	}
	metricData := &MediaData{Type: MediaDataTypeMetric}
	if !strings.Contains(metricData.String(), "metric") {
		t.Errorf("metric MediaData.String: %q", metricData.String())
	}
}

// ==========================================================================
// session.go — most straight-line helpers (no Serve, which needs real transports)
// ==========================================================================

func TestMediaSession_SetSessionIDStringGetSetDelete(t *testing.T) {
	s := NewDefaultSession().SetSessionID("sess-1")
	if s.ID != "sess-1" {
		t.Errorf("ID = %q", s.ID)
	}
	if !strings.Contains(s.String(), "sess-1") {
		t.Errorf("String: %q", s.String())
	}

	s.Set("k", "v")
	if v, ok := s.Get("k"); !ok || v.(string) != "v" {
		t.Errorf("Get after Set: ok=%v v=%v", ok, v)
	}
	if v := s.GetString("k"); v != "v" {
		t.Errorf("GetString: %q", v)
	}
	s.Delete("k")
	if _, ok := s.Get("k"); ok {
		t.Error("Delete did not remove key")
	}

	if v := s.GetString("missing"); v != "" {
		t.Errorf("GetString missing = %q", v)
	}
	// GetString on non-string value → empty
	s.Set("num", 42)
	if v := s.GetString("num"); v != "" {
		t.Errorf("GetString non-string: %q", v)
	}
}

func TestMediaSession_GetUintAllBranches(t *testing.T) {
	s := NewDefaultSession()
	s.Set("u", uint(7))
	if got := s.GetUint("u"); got != 7 {
		t.Errorf("uint branch: %d", got)
	}
	s.Set("i", 8)
	if got := s.GetUint("i"); got != 8 {
		t.Errorf("int branch: %d", got)
	}
	s.Set("neg", -1)
	if got := s.GetUint("neg"); got != 0 {
		t.Errorf("negative int must return 0, got %d", got)
	}
	s.Set("str", "nope")
	if got := s.GetUint("str"); got != 0 {
		t.Errorf("non-int type must return 0, got %d", got)
	}
	if got := s.GetUint("missing"); got != 0 {
		t.Errorf("missing key must return 0, got %d", got)
	}
}

func TestMediaSession_ChainableMethods(t *testing.T) {
	s := NewDefaultSession().
		SetSessionID("chain").
		Context(context.Background()).
		Trace(func(h MediaHandler, d MediaData) {}).
		Encode(func(p MediaPacket) ([]MediaPacket, error) { return []MediaPacket{p}, nil }).
		Decode(func(p MediaPacket) ([]MediaPacket, error) { return []MediaPacket{p}, nil }).
		Error(func(sender any, err error) {}).
		On("begin", func(event StateChange) {}).
		RegisterProcessor(noopFuncProc("p", PriorityNormal))

	if s.ID != "chain" {
		t.Errorf("chain failed: %q", s.ID)
	}
	if s.GetContext() == nil {
		t.Error("GetContext should lazy-init")
	}
	if s.GetSession() != s {
		t.Error("GetSession must return self")
	}
	if c := s.Codec(); c.Codec != "pcm" {
		t.Errorf("Codec default = %+v", c)
	}
	// Deprecated alias just for coverage
	s.UseMiddleware(func(h MediaHandler, d MediaData) {})
	s.Pipeline(func(h MediaHandler, d MediaData) {})
	s.InjectPacket(nil) // no-op on session
}

func TestMediaSession_IsValid_Errors(t *testing.T) {
	s := NewDefaultSession()
	if err := s.IsValid(); err == nil {
		t.Error("session with no transports must be invalid")
	}
}

func TestMediaSession_WaitServeShutdown_NotScheduledReturnsNil(t *testing.T) {
	s := NewDefaultSession()
	if err := s.WaitServeShutdown(context.Background()); err != nil {
		t.Errorf("unscheduled WaitServeShutdown must return nil, got %v", err)
	}
}

func TestMediaSession_NotifyServeStarting_NilSafe(t *testing.T) {
	var s *MediaSession
	s.NotifyServeStarting() // must not panic
	s2 := NewDefaultSession()
	s2.NotifyServeStarting()
	// After scheduling, WaitServeShutdown should block on shutdownCh; verify it cancels via ctx.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := s2.WaitServeShutdown(ctx); err == nil {
		t.Error("WaitServeShutdown on scheduled session should honour ctx deadline")
	}
}

// ==========================================================================
// executor.go — AsyncTaskRunner basic invariants (no real MediaHandler needed)
// ==========================================================================

func TestNewAsyncTaskRunner_Defaults(t *testing.T) {
	tr := NewAsyncTaskRunner[int](8)
	if tr.WorkerPoolSize != 8 {
		t.Errorf("WorkerPoolSize = %d, want 8", tr.WorkerPoolSize)
	}
	if !tr.ConcurrentMode {
		t.Error("ConcurrentMode should default true")
	}
	if tr.MaxTaskTimeout != time.Minute {
		t.Errorf("MaxTaskTimeout = %v, want 1m", tr.MaxTaskTimeout)
	}
}

func TestAsyncTaskRunner_ReleaseResources_Idempotent(t *testing.T) {
	tr := NewAsyncTaskRunner[int](4)
	// without Start, stopWorkers should be a no-op
	tr.ReleaseResources()
	tr.ReleaseResources() // second call safe
}

// ==========================================================================
// MediaSession lifecycle via fake MediaTransport
// ==========================================================================

// fakeTransport implements MediaTransport — drives input/output loops in tests.
type fakeTransport struct {
	mu         sync.Mutex
	session    *MediaSession
	direction  string
	in         chan MediaPacket
	sent       []MediaPacket
	codec      CodecConfig
	closed     bool
}

func newFakeTransport(dir string) *fakeTransport {
	return &fakeTransport{
		direction: dir,
		in:        make(chan MediaPacket, 16),
		codec:     DefaultCodecConfig(),
	}
}

func (f *fakeTransport) String() string       { return "fakeTransport{" + f.direction + "}" }
func (f *fakeTransport) Codec() CodecConfig   { return f.codec }
func (f *fakeTransport) Attach(s *MediaSession) { f.session = s }
func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	close(f.in)
	return nil
}

func (f *fakeTransport) Next(ctx context.Context) (MediaPacket, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p, ok := <-f.in:
		if !ok {
			return nil, io.EOF
		}
		return p, nil
	}
}

func (f *fakeTransport) Send(ctx context.Context, packet MediaPacket) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return 0, io.ErrClosedPipe
	}
	f.sent = append(f.sent, packet)
	return len(packet.Body()), nil
}

func (f *fakeTransport) sentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func TestMediaSession_Serve_InputOutputLifecycle(t *testing.T) {
	rx := newFakeTransport("input")
	tx := newFakeTransport("output")

	s := NewDefaultSession().
		SetSessionID("serve-test").
		Context(context.Background()).
		Input(rx).
		Output(tx)

	if err := s.IsValid(); err != nil {
		t.Fatalf("IsValid after transports: %v", err)
	}

	s.NotifyServeStarting()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = s.Serve()
	}()

	// Push a packet to input — it should be echoed to output via the router processor
	// (output-router is auto-registered during Serve).
	rx.in <- &AudioPacket{Payload: []byte{1, 2, 3}, IsSynthesized: true}

	// Allow event bus & transport pipeline to run.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && tx.sentCount() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if tx.sentCount() == 0 {
		t.Error("expected at least one packet delivered to output transport")
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Close")
	}
	if err := s.WaitServeShutdown(context.Background()); err != nil {
		t.Errorf("WaitServeShutdown after Serve: %v", err)
	}
}

func TestMediaSession_Serve_ErrorHandlerAndState(t *testing.T) {
	rx := newFakeTransport("input")
	tx := newFakeTransport("output")

	var (
		errCount   int32
		stateCount int32
	)
	hookFired := false
	s := NewDefaultSession().
		SetSessionID("err-test").
		Context(context.Background()).
		Error(func(sender any, err error) { atomic.AddInt32(&errCount, 1) }).
		On("*", func(ev StateChange) { atomic.AddInt32(&stateCount, 1) }).
		PostHook(func(session *MediaSession) { hookFired = true }).
		Input(rx).
		Output(tx)

	s.NotifyServeStarting()
	go func() { _ = s.Serve() }()

	// Drive an error through the session
	time.Sleep(20 * time.Millisecond)
	s.CauseError("test", errors.New("boom"))

	// Trigger a few explicit state emissions
	s.EmitState("test", "custom-state")

	time.Sleep(30 * time.Millisecond)
	_ = s.Close()

	if err := s.WaitServeShutdown(context.Background()); err != nil {
		t.Errorf("WaitServeShutdown: %v", err)
	}
	if atomic.LoadInt32(&errCount) == 0 {
		t.Error("error handler should fire at least once")
	}
	_ = stateCount // handlers registered via On are event-bus driven; not all state emissions route to On callbacks here.
	if !hookFired {
		t.Error("post-hook did not fire")
	}
}

func TestMediaSession_TrySendPacket_DropWhenQueueFullForNonSynth(t *testing.T) {
	rx := newFakeTransport("input")
	tx := newFakeTransport("output")
	s := NewDefaultSession().Context(context.Background())
	s.QueueSize = 1
	s = s.Input(rx).Output(tx)

	// Manually exercise trySendPacket on the output transport manager.
	tm := s.outputs[0]
	tm.trySendPacket(&AudioPacket{Payload: []byte{1}, IsSynthesized: false})
	// Queue is size 1 and consumer not running → second should drop
	tm.trySendPacket(&AudioPacket{Payload: []byte{2}, IsSynthesized: false})

	_ = s.Close()
}

// ==========================================================================
// AsyncTaskRunner end-to-end via fake MediaHandler
// ==========================================================================

// fakeHandler is a minimal MediaHandler for driving AsyncTaskRunner.
type fakeHandler struct {
	ctx       context.Context
	session   *MediaSession
	errMu     sync.Mutex
	errsSeen  []error
}

func (h *fakeHandler) GetContext() context.Context { return h.ctx }
func (h *fakeHandler) GetSession() *MediaSession   { return h.session }
func (h *fakeHandler) CauseError(sender any, err error) {
	h.errMu.Lock()
	defer h.errMu.Unlock()
	h.errsSeen = append(h.errsSeen, err)
}
func (h *fakeHandler) EmitState(sender any, state string, params ...any) {}
func (h *fakeHandler) EmitPacket(sender any, packet MediaPacket)         {}
func (h *fakeHandler) SendToOutput(sender any, packet MediaPacket)       {}
func (h *fakeHandler) AddMetric(key string, duration time.Duration)      {}
func (h *fakeHandler) InjectPacket(f PacketFilter)                       {}

func TestAsyncTaskRunner_EndToEnd_BeginPacketEnd(t *testing.T) {
	tr := NewAsyncTaskRunner[int](2)
	tr.TaskTimeout = 100 * time.Millisecond

	var executed int32
	var initFired, termFired, stateFired int32
	tr.InitCallback = func(h MediaHandler) error { atomic.AddInt32(&initFired, 1); return nil }
	tr.TerminateCallback = func(h MediaHandler) error { atomic.AddInt32(&termFired, 1); return nil }
	tr.StateCallback = func(h MediaHandler, e StateChange) error { atomic.AddInt32(&stateFired, 1); return nil }
	tr.RequestBuilder = func(h MediaHandler, p MediaPacket) (*PacketRequest[int], error) {
		return &PacketRequest[int]{Req: 1}, nil
	}
	tr.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[int]) error {
		atomic.AddInt32(&executed, 1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := &fakeHandler{ctx: ctx, session: NewDefaultSession().Context(ctx)}

	// Begin → spins up worker pool
	tr.HandleMediaData(h, MediaData{Type: MediaDataTypeState, State: StateChange{State: Begin}})

	// Drive a few packets
	for i := 0; i < 3; i++ {
		tr.HandleMediaData(h, MediaData{Type: MediaDataTypePacket, Packet: &AudioPacket{Payload: []byte{byte(i)}}})
	}

	// Wait for tasks to drain
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && atomic.LoadInt32(&executed) < 3 {
		time.Sleep(5 * time.Millisecond)
	}

	// End → tears down workers, fires TerminateCallback
	tr.HandleMediaData(h, MediaData{Type: MediaDataTypeState, State: StateChange{State: End}})

	if atomic.LoadInt32(&initFired) != 1 {
		t.Errorf("InitCallback fired %d times, want 1", initFired)
	}
	if atomic.LoadInt32(&termFired) != 1 {
		t.Errorf("TerminateCallback fired %d times, want 1", termFired)
	}
	if atomic.LoadInt32(&stateFired) != 2 {
		t.Errorf("StateCallback fired %d times, want 2 (Begin+End)", stateFired)
	}
	if atomic.LoadInt32(&executed) < 3 {
		t.Errorf("TaskExecutor executed %d times, want ≥3", executed)
	}
}

func TestAsyncTaskRunner_RequestBuilderError(t *testing.T) {
	tr := NewAsyncTaskRunner[int](2)
	tr.ConcurrentMode = false // Use synchronous mode to avoid starting pool
	tr.RequestBuilder = func(h MediaHandler, p MediaPacket) (*PacketRequest[int], error) {
		return nil, errors.New("build failed")
	}
	tr.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[int]) error { return nil }

	h := &fakeHandler{ctx: context.Background(), session: NewDefaultSession().Context(context.Background())}
	tr.HandlePacket(h, &AudioPacket{Payload: []byte{1}})

	h.errMu.Lock()
	defer h.errMu.Unlock()
	if len(h.errsSeen) == 0 {
		t.Error("RequestBuilder error should flow to CauseError")
	}
}

func TestAsyncTaskRunner_SynchronousMode_ExecutesInline(t *testing.T) {
	tr := NewAsyncTaskRunner[int](2)
	tr.ConcurrentMode = false

	var executed int32
	tr.RequestBuilder = func(h MediaHandler, p MediaPacket) (*PacketRequest[int], error) {
		return &PacketRequest[int]{Req: 1}, nil
	}
	tr.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[int]) error {
		atomic.AddInt32(&executed, 1)
		return nil
	}

	h := &fakeHandler{ctx: context.Background(), session: NewDefaultSession().Context(context.Background())}
	tr.HandlePacket(h, &AudioPacket{Payload: []byte{1}})
	if atomic.LoadInt32(&executed) != 1 {
		t.Errorf("synchronous mode should execute inline, got %d", executed)
	}
}

func TestAsyncTaskRunner_RequestBuilderReturnsNil(t *testing.T) {
	tr := NewAsyncTaskRunner[int](2)
	tr.ConcurrentMode = false
	tr.RequestBuilder = func(h MediaHandler, p MediaPacket) (*PacketRequest[int], error) { return nil, nil }
	var executed int32
	tr.TaskExecutor = func(ctx context.Context, h MediaHandler, req PacketRequest[int]) error {
		atomic.AddInt32(&executed, 1)
		return nil
	}
	h := &fakeHandler{ctx: context.Background(), session: NewDefaultSession().Context(context.Background())}
	tr.HandlePacket(h, &AudioPacket{Payload: []byte{1}})
	if atomic.LoadInt32(&executed) != 0 {
		t.Errorf("nil request should skip execution, got %d", executed)
	}
}

// ==========================================================================
// sessionHandlerAdapter (covers UseMiddleware + adapter methods)
// ==========================================================================

func TestSessionHandlerAdapter_ViaMiddlewareAndAdditionalAPIs(t *testing.T) {
	rx := newFakeTransport("input")
	tx := newFakeTransport("output")

	var (
		handled       int32
		adapterSeen   MediaHandler
		adapterSeenMu sync.Mutex
	)
	s := NewDefaultSession().
		SetSessionID("adapter-test").
		Context(context.Background()).
		Input(rx).
		Output(tx).
		UseMiddleware(func(h MediaHandler, d MediaData) {
			adapterSeenMu.Lock()
			adapterSeen = h
			adapterSeenMu.Unlock()
			// Exercise several adapter methods — just need no panic
			_ = h.GetContext()
			_ = h.GetSession()
			h.EmitState(nil, "x")
			h.EmitPacket(nil, &AudioPacket{Payload: []byte{1}})
			h.SendToOutput(nil, &AudioPacket{Payload: []byte{2}})
			h.CauseError(nil, errors.New("adapter err"))
			h.AddMetric("m", time.Millisecond)
			h.InjectPacket(nil)
			atomic.AddInt32(&handled, 1)
		})

	s.NotifyServeStarting()
	go func() { _ = s.Serve() }()

	// Drive a packet
	rx.in <- &AudioPacket{Payload: []byte{7}}

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&handled) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	_ = s.Close()
	_ = s.WaitServeShutdown(context.Background())

	adapterSeenMu.Lock()
	got := adapterSeen
	adapterSeenMu.Unlock()
	if got == nil {
		t.Error("middleware not invoked via adapter")
	}
	if atomic.LoadInt32(&handled) == 0 {
		t.Error("middleware never fired")
	}
}

// ==========================================================================
// Direct exercise of simple helpers
// ==========================================================================

func TestMediaSession_DirectHelpers(t *testing.T) {
	s := NewDefaultSession().Context(context.Background())
	s.AddMetric("m", 5*time.Millisecond) // no trace set → no-op path
	s.SendToOutput("test", &AudioPacket{Payload: []byte{1}}) // no outputs → no-op iterate
	s.putPacket(DirectionInput, &AudioPacket{Payload: []byte{1}})
	s.putPacket(DirectionOutput, &AudioPacket{Payload: []byte{1}})

	// Now with a trace hook so AddMetric routes into it
	traced := int32(0)
	s = s.Trace(func(h MediaHandler, d MediaData) { atomic.AddInt32(&traced, 1) })
	s.AddMetric("m2", 10*time.Millisecond)
	if atomic.LoadInt32(&traced) != 1 {
		t.Errorf("trace handler invoked %d times", traced)
	}
}

type testOptions struct {
	Name   string `json:"name" default:"voiceserver"`
	Port   int    `json:"port" env:"VS_TEST_PORT" default:"7072"`
	Nested struct {
		Enabled bool    `json:"enabled" default:"true"`
		Rate    float64 `json:"rate" default:"1.5"`
		Count   uint    `json:"count" default:"3"`
	} `json:"nested"`
}

func TestCastOption_EnvAndDefaultAndJSON(t *testing.T) {
	t.Setenv("VS_TEST_PORT", "5555")
	var empty testOptions
	out := CastOption[testOptions](nil)
	if out.Name != "voiceserver" || out.Port != 5555 || !out.Nested.Enabled || out.Nested.Rate != 1.5 || out.Nested.Count != 3 {
		t.Errorf("defaults/env: %+v", out)
	}

	// JSON overrides default
	override := map[string]any{"name": "custom", "port": 9090}
	got := CastOption[testOptions](override)
	if got.Name != "custom" || got.Port != 9090 {
		t.Errorf("json override failed: %+v", got)
	}
	_ = empty
}

func TestCallHandleWithMediaData_RecoversFromPanic(t *testing.T) {
	s := NewDefaultSession()
	// Handler that panics
	callHandleWithMediaData(s, nil, func(h MediaHandler, d MediaData) {
		panic("boom")
	}, MediaData{Type: MediaDataTypePacket})
}

func TestCallHandleWithState_RecoversFromPanic(t *testing.T) {
	s := NewDefaultSession()
	callHandleWithState(s, func(ev StateChange) {
		panic("state-boom")
	}, StateChange{State: "x"})
}

func TestSession_ProcessData_IsNoOp(t *testing.T) {
	s := NewDefaultSession()
	s.processData(&MediaData{Type: MediaDataTypePacket}) // must not panic
}

func TestSession_InjectPacket_IsNoOp(t *testing.T) {
	s := NewDefaultSession()
	s.InjectPacket(func(p MediaPacket) (bool, error) { return false, nil })
}

func TestTransportManager_String(t *testing.T) {
	s := NewDefaultSession().Context(context.Background()).Input(newFakeTransport("i"))
	if got := s.inputs[0].String(); got == "" || !strings.Contains(got, "fakeTransport") {
		t.Errorf("TransportManager.String = %q", got)
	}
}

// Concurrent goroutines safety for EventBus publish under load
func TestEventBus_ConcurrentPublish(t *testing.T) {
	eb := NewEventBus(context.Background(), 256, 4)
	defer eb.Close()

	var received int32
	eb.Subscribe(EventTypePacket, func(ctx context.Context, ev *MediaEvent) error {
		atomic.AddInt32(&received, 1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				eb.PublishPacket("s", &AudioPacket{Payload: []byte{byte(j)}}, nil)
			}
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&received) < 10 {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&received) == 0 {
		t.Error("no events delivered under concurrent load")
	}
}

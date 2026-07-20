package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestMode_IsValid is the canonical list of supported modes.
func TestMode_IsValid(t *testing.T) {
	cases := []struct {
		mode Mode
		want bool
	}{
		{ModeCascaded, true},
		{ModeRealtime, true},
		{ModeHybrid, true},
		{Mode(""), false},
		{Mode("unknown"), false},
		{Mode("cascade"), false}, // typo guard
	}
	for _, tc := range cases {
		if got := tc.mode.IsValid(); got != tc.want {
			t.Errorf("Mode(%q).IsValid() = %v, want %v", string(tc.mode), got, tc.want)
		}
	}
}

// fakeFactory is a no-op factory used by registry tests. It returns
// a sentinel engine so callers can verify which factory ran.
type fakeFactory struct {
	mode Mode
	tag  string
}

func (f fakeFactory) Build(cfg Config) (Engine, error) {
	return fakeEngine{mode: f.mode, tag: f.tag, cfg: cfg}, nil
}

type fakeEngine struct {
	mode Mode
	tag  string
	cfg  Config
}

func (e fakeEngine) Attach(context.Context, MediaPort, Logger) (Detach, error) {
	return func(context.Context) error { return nil }, nil
}
func (e fakeEngine) Mode() Mode { return e.mode }

// Helper to reset the global registry between tests. We intentionally
// don't expose this in the public API — tests in the engine package
// can reach into the unexported state.
func resetRegistry(t *testing.T) {
	t.Helper()
	registryMu.Lock()
	registry = map[Mode]Factory{}
	registryMu.Unlock()
}

func TestRegister_PanicsOnInvalidMode(t *testing.T) {
	resetRegistry(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid mode")
		}
	}()
	Register(Mode("nope"), fakeFactory{})
}

func TestRegister_PanicsOnNilFactory(t *testing.T) {
	resetRegistry(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil factory")
		}
	}()
	Register(ModeCascaded, nil)
}

func TestRegister_PanicsOnDuplicate(t *testing.T) {
	resetRegistry(t)
	Register(ModeCascaded, fakeFactory{mode: ModeCascaded, tag: "first"})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate")
		}
		if !strings.Contains(toString(r), "already registered") {
			t.Errorf("unexpected panic value: %v", r)
		}
	}()
	Register(ModeCascaded, fakeFactory{mode: ModeCascaded, tag: "second"})
}

func TestNew_DispatchesByMode(t *testing.T) {
	resetRegistry(t)
	Register(ModeCascaded, fakeFactory{mode: ModeCascaded, tag: "casc"})
	Register(ModeRealtime, fakeFactory{mode: ModeRealtime, tag: "rt"})

	cases := []struct {
		mode Mode
		want string
	}{
		{ModeCascaded, "casc"},
		{ModeRealtime, "rt"},
	}
	for _, tc := range cases {
		e, err := New(Config{Mode: tc.mode, CallID: "c1", TenantID: "t1"})
		if err != nil {
			t.Fatalf("Mode=%s: New err: %v", tc.mode, err)
		}
		fe, ok := e.(fakeEngine)
		if !ok {
			t.Fatalf("Mode=%s: expected fakeEngine, got %T", tc.mode, e)
		}
		if fe.tag != tc.want {
			t.Errorf("Mode=%s: tag=%q, want %q", tc.mode, fe.tag, tc.want)
		}
		if fe.cfg.CallID != "c1" || fe.cfg.TenantID != "t1" {
			t.Errorf("Mode=%s: config not propagated: %+v", tc.mode, fe.cfg)
		}
	}
}

func TestNew_UnknownMode(t *testing.T) {
	resetRegistry(t)
	_, err := New(Config{Mode: Mode("")})
	if !errors.Is(err, ErrUnknownMode) {
		t.Errorf("expected ErrUnknownMode, got %v", err)
	}
	Register(ModeCascaded, fakeFactory{mode: ModeCascaded})
	_, err = New(Config{Mode: ModeRealtime}) // valid mode, but not registered
	if !errors.Is(err, ErrUnknownMode) {
		t.Errorf("expected ErrUnknownMode for unregistered, got %v", err)
	}
}

func TestRegisteredModes(t *testing.T) {
	resetRegistry(t)
	if got := RegisteredModes(); len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
	Register(ModeCascaded, fakeFactory{mode: ModeCascaded})
	Register(ModeRealtime, fakeFactory{mode: ModeRealtime})
	got := RegisteredModes()
	if len(got) != 2 {
		t.Errorf("expected 2 modes, got %d: %v", len(got), got)
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	if s, ok := v.(string); ok {
		return s
	}
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	return ""
}

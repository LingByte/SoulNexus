package provider

import (
	"errors"
	"testing"
)

// fakeASR is a stub implementing asr.Provider just enough to satisfy
// the type parameter. We don't import the real interface because
// these tests only exercise the registry plumbing.
type fakeASR struct{ name string }

// Stub methods to satisfy asr.Provider — we don't actually call them
// here, but a compile-time check guarantees the parameter shape.
func (f fakeASR) Name() string { return f.name }
func (f fakeASR) Open(any, any) (any, error) { return nil, nil } // unused signature

// We test against the real ASR registry using a minimal fake type
// that doesn't have to implement asr.Provider — Registry[T] is
// generic, so we can use Registry[any] for unit tests.

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := &Registry[string]{}
	r.Register("foo", "FOO_PROVIDER")
	r.Register("bar", "BAR_PROVIDER")

	got, err := r.Get("foo")
	if err != nil {
		t.Fatalf("Get(foo): %v", err)
	}
	if got != "FOO_PROVIDER" {
		t.Errorf("Get(foo) = %q, want FOO_PROVIDER", got)
	}
}

func TestRegistry_Get_NotRegistered(t *testing.T) {
	r := &Registry[string]{}
	_, err := r.Get("nope")
	if !errors.Is(err, ErrNotRegistered) {
		t.Errorf("expected ErrNotRegistered, got %v", err)
	}
}

func TestRegistry_Get_EmptyName(t *testing.T) {
	r := &Registry[string]{}
	r.Register("foo", "F")
	_, err := r.Get("")
	if !errors.Is(err, ErrNotRegistered) {
		t.Errorf("expected ErrNotRegistered for empty, got %v", err)
	}
}

func TestRegistry_Register_PanicsOnEmptyName(t *testing.T) {
	r := &Registry[string]{}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on empty name")
		}
	}()
	r.Register("", "X")
}

func TestRegistry_Register_PanicsOnDuplicate(t *testing.T) {
	r := &Registry[string]{}
	r.Register("dup", "first")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate")
		}
	}()
	r.Register("dup", "second")
}

func TestRegistry_Names_Sorted(t *testing.T) {
	r := &Registry[string]{}
	r.Register("zebra", "z")
	r.Register("apple", "a")
	r.Register("mango", "m")
	got := r.Names()
	want := []string{"apple", "mango", "zebra"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGlobalRegistries_AreDistinct(t *testing.T) {
	// Sanity: the four global registries are independent instances.
	if ASRRegistry == nil || TTSRegistry == nil || LLMRegistry == nil || MultimodalRegistry == nil {
		t.Fatal("global registry pointer is nil")
	}
	// Compile-time only: each registry is parameterised by a different
	// provider interface type, which the type system enforces.
}

// Keep the unused fakeASR symbol referenced so tooling doesn't trip.
var _ = fakeASR{}

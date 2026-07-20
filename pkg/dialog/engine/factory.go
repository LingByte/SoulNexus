package engine

import (
	"errors"
	"fmt"
	"sync"
)

// Factory builds an Engine for a given call configuration. Each Mode
// has exactly one Factory registered; concrete factory implementations
// live in their respective sub-packages
// (cascaded.factory, realtime.factory, ...) and self-register via init().
type Factory interface {
	// Build creates a new Engine instance for one call. Implementations
	// SHOULD be cheap (no I/O) — heavy lifting (provider WS handshake,
	// model warmup) belongs inside Engine.Attach.
	Build(cfg Config) (Engine, error)
}

// Config bundles everything needed to construct an Engine. Fields are
// optional unless documented otherwise; concrete factories fail with
// a clear error when a required field is missing for their mode.
type Config struct {
	// Mode selects the engine implementation. REQUIRED.
	Mode Mode

	// TenantID is the owning tenant. REQUIRED for billing & metrics.
	TenantID string

	// CallID is the unique call identifier. REQUIRED.
	CallID string

	// Providers references the ASR / TTS / LLM / Realtime providers
	// to use for this call. Cascaded mode reads ASR + LLM + TTS;
	// realtime mode reads Realtime; hybrid reads all four.
	Providers ProviderRefs

	// Tools is the list of LLM/realtime function-call tools available
	// in this call (e.g. hangup_call). nil = no
	// tools registered.
	Tools []ToolRef

	// Welcome is the optional welcome audio specification. Empty Path
	// and nil Speak = no welcome played.
	Welcome WelcomeSpec

	// Hooks is a sink for engine lifecycle events (turn-completed,
	// barge-in, error). nil = no hooks fired.
	Hooks Hooks
}

// ProviderRefs identifies which Provider builders to invoke per call.
// The actual Provider instance is constructed lazily inside the
// Engine to keep Config a plain data type.
//
// Each field is "<provider name>:<model id>" (e.g. "qcloud:nova",
// "openai:gpt-4o-realtime"). Empty string = use tenant default.
type ProviderRefs struct {
	ASR      string
	TTS      string
	LLM      string
	Realtime string
}

// WelcomeSpec describes the welcome audio for a call. Engines pick
// the best-fit field for their mode.
type WelcomeSpec struct {
	// AudioURL is a remote WAV URL pre-validated by the admin layer.
	// Empty = no remote welcome.
	AudioURL string

	// LocalPath is a filesystem WAV path (relative to working dir).
	// Used as a final fallback.
	LocalPath string

	// Speak is a TTS prompt rendered through the configured TTS
	// provider when neither AudioURL nor LocalPath is available.
	Speak string
}

// ToolRef is an opaque reference to a registered tool. The actual
// resolution happens in pkg/dialog/tools.
type ToolRef struct {
	Name string
}

// Hooks is the engine-level observer. All callbacks are invoked from
// engine-owned goroutines and SHOULD return quickly; long-running
// work belongs in a queue/worker on the caller side.
type Hooks struct {
	// OnTurnCompleted fires after a complete (user → AI) round trip.
	// nil = silent.
	OnTurnCompleted func(turnID string, userText, aiText string)

	// OnBargeIn fires when the user interrupts AI output.
	OnBargeIn func()

	// OnError fires on non-fatal engine errors (provider hiccups,
	// transient network). Fatal errors are returned from Attach
	// directly or surface via Detach error.
	OnError func(err error)
}

// Errors common to engine assembly.
var (
	ErrUnknownMode    = errors.New("dialog/engine: unknown Mode")
	ErrModeRegistered = errors.New("dialog/engine: Mode already registered")
)

// modeRegistry holds the global Mode → Factory map. Populated by
// each engine sub-package's init().
var (
	registryMu sync.RWMutex
	registry   = map[Mode]Factory{}
)

// Register installs f as the factory for mode m. Panics if mode is
// already registered — this is fail-fast init() time, not runtime.
//
// Typically called from an engine sub-package's init():
//
//	func init() {
//	    engine.Register(engine.ModeCascaded, factory{})
//	}
func Register(m Mode, f Factory) {
	if !m.IsValid() {
		panic(fmt.Errorf("dialog/engine: invalid mode %q", string(m)))
	}
	if f == nil {
		panic(fmt.Errorf("dialog/engine: nil factory for mode %q", string(m)))
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[m]; exists {
		panic(fmt.Errorf("%w: %q", ErrModeRegistered, string(m)))
	}
	registry[m] = f
}

// New builds an Engine for the given config. Looks up the registered
// Factory by cfg.Mode and delegates.
func New(cfg Config) (Engine, error) {
	if !cfg.Mode.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrUnknownMode, string(cfg.Mode))
	}
	registryMu.RLock()
	f, ok := registry[cfg.Mode]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q (no factory registered)", ErrUnknownMode, string(cfg.Mode))
	}
	return f.Build(cfg)
}

// ResetRegistryForTest clears the global Mode → Factory registry.
// ONLY for use in tests; calling it from production code voids the
// warranty. Exists in a regular file (not _test.go) so cross-package
// tests can reset between cases.
func ResetRegistryForTest() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[Mode]Factory{}
}

// RegisteredModes returns the set of currently-registered modes.
// Order is unspecified. Useful for diagnostics / health checks.
func RegisteredModes() []Mode {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Mode, 0, len(registry))
	for m := range registry {
		out = append(out, m)
	}
	return out
}

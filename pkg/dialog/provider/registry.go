// Package provider hosts the typed Registry used by all Provider
// kinds (ASR / TTS / LLM / Multimodal). Sub-packages define the
// interface for their kind; concrete vendor implementations register
// themselves in their own init().
package provider

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/provider/asr"
	"github.com/LingByte/SoulNexus/pkg/dialog/provider/llm"
	"github.com/LingByte/SoulNexus/pkg/dialog/provider/multimodal"
	"github.com/LingByte/SoulNexus/pkg/dialog/provider/tts"
)

// Registry is a thread-safe name→Provider table. Generics keep the
// API type-safe: ASRRegistry stores asr.Provider, LLMRegistry stores
// llm.Provider, etc.
//
// Lookup is O(1) and never allocates. Register panics on duplicate
// names — vendor authors are expected to catch this at init-time
// rather than discover at runtime.
type Registry[T any] struct {
	mu        sync.RWMutex
	providers map[string]T
}

// Register installs p under name. Panics if name is empty or already
// registered. Intended for init() use.
func (r *Registry[T]) Register(name string, p T) {
	if name == "" {
		panic("dialog/provider: empty name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers == nil {
		r.providers = map[string]T{}
	}
	if _, exists := r.providers[name]; exists {
		panic(fmt.Errorf("dialog/provider: %q already registered", name))
	}
	r.providers[name] = p
}

// Get returns the provider registered under name, or ErrNotRegistered.
func (r *Registry[T]) Get(name string) (T, error) {
	var zero T
	if name == "" {
		return zero, ErrNotRegistered
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return zero, fmt.Errorf("%w: %q", ErrNotRegistered, name)
	}
	return p, nil
}

// Names returns the registered names in sorted order. Useful for
// diagnostics / admin listings.
func (r *Registry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for n := range r.providers {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// ErrNotRegistered is returned by Get when the requested provider
// name is unknown.
var ErrNotRegistered = errors.New("dialog/provider: not registered")

// Global registries — one per provider kind. Vendor packages call
// the appropriate Register at init().
var (
	ASRRegistry        = &Registry[asr.Provider]{}
	TTSRegistry        = &Registry[tts.Provider]{}
	LLMRegistry        = &Registry[llm.Provider]{}
	MultimodalRegistry = &Registry[multimodal.Provider]{}
)

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import (
	"context"
	"sync"
)

// VoiceConfigBundle is the raw voice configuration for one tenant call leg.
// Populated from assistants (preferred) or legacy tenant JSON columns.
type VoiceConfigBundle struct {
	Asr, Tts, Llm, Realtime []byte
	VoiceMode                 string

	Welcome            string
	Prompt             string
	KnowledgeNamespace string
	KnowledgeNamespaceID uint
	KnowledgeCollection  string

	// NLU profile paths when assistant binds a ready tenant NLU model.
	NluModelID        uint
	NluModelPath      string
	NluTokenizerPath  string
	NluIntentsPath    string
	NluPrototypesPath string
	NluMinConfidence  float64

	VadConfig, AgentConfig []byte
	HotWords, InterruptionConfig, AudioTrackConfig []byte
	AudioProcessConfig, QueryRewriter []byte
	McpServers         []byte
	VoiceDialogWsURL string

	AssistantID        uint
	AssistantVersionID uint
}

// JSONLoader fetches voice config for one tenant.
//
// Wired from the storage layer (handlers / bootstrap, tests). Return ok=false
// when the tenant row is missing — the call leg then falls back to env-driven
// config or scripts/config_error.wav playback.
type JSONLoader func(ctx context.Context, tenantID uint, callID string) (VoiceConfigBundle, bool)

var (
	loaderMu sync.RWMutex
	loader   JSONLoader
)

// SetLoader installs the DB-backed tenant voice JSON loader.
func SetLoader(fn JSONLoader) {
	loaderMu.Lock()
	loader = fn
	loaderMu.Unlock()
}

// Loader returns the currently installed JSONLoader (or nil).
func Loader() JSONLoader {
	loaderMu.RLock()
	fn := loader
	loaderMu.RUnlock()
	return fn
}

// Resolve loads the tenant's voice JSON via the installed loader and
// parses it into a VoiceEnv. callID scopes assistant binding to inbound DID.
func Resolve(ctx context.Context, tenantID uint, callID string) (VoiceEnv, bool, error) {
	if tenantID == 0 {
		return VoiceEnv{}, false, nil
	}
	fn := Loader()
	if fn == nil {
		return VoiceEnv{}, false, nil
	}
	bundle, ok := fn(ctx, tenantID, callID)
	if !ok {
		return VoiceEnv{}, false, nil
	}
	env, err := VoiceEnvFromJSON(bundle.Asr, bundle.Tts, bundle.Llm, bundle.Realtime, bundle.VoiceMode)
	if err != nil {
		return VoiceEnv{}, false, err
	}
	env.TenantID = tenantID
	env = ApplyVoiceConfigBundle(env, bundle)
	return env, true, nil
}

// ApplyVoiceConfigBundle merges assistant-level overlay fields onto VoiceEnv.
func ApplyVoiceConfigBundle(env VoiceEnv, bundle VoiceConfigBundle) VoiceEnv {
	if p := trim(bundle.Prompt); p != "" {
		env.LLMInstructions = p
	}
	env.AssistantWelcome = trim(bundle.Welcome)
	env.KnowledgeNamespace = trim(bundle.KnowledgeNamespace)
	env.KnowledgeNamespaceID = bundle.KnowledgeNamespaceID
	env.KnowledgeCollection = trim(bundle.KnowledgeCollection)
	env.AssistantID = bundle.AssistantID
	env.AssistantVersionID = bundle.AssistantVersionID
	env.NluModelID = bundle.NluModelID
	env.NluModelPath = trim(bundle.NluModelPath)
	env.NluTokenizerPath = trim(bundle.NluTokenizerPath)
	env.NluIntentsPath = trim(bundle.NluIntentsPath)
	env.NluPrototypesPath = trim(bundle.NluPrototypesPath)
	env.NluMinConfidence = bundle.NluMinConfidence
	if raw, err := parseObj(bundle.VadConfig); err == nil && len(raw) > 0 {
		env.VadConfigRaw = raw
	}
	if raw, err := parseObj(bundle.AgentConfig); err == nil && len(raw) > 0 {
		env.AgentConfigRaw = raw
	}
	if len(bundle.HotWords) > 0 {
		env.HotWordsRaw = append([]byte(nil), bundle.HotWords...)
	}
	if raw, err := parseObj(bundle.InterruptionConfig); err == nil && len(raw) > 0 {
		env.InterruptionConfigRaw = raw
	}
	if raw, err := parseObj(bundle.AudioTrackConfig); err == nil && len(raw) > 0 {
		env.AudioTrackConfigRaw = raw
	}
	if raw, err := parseObj(bundle.AudioProcessConfig); err == nil && len(raw) > 0 {
		env.AudioProcessConfigRaw = raw
	}
	if raw, err := parseObj(bundle.QueryRewriter); err == nil && len(raw) > 0 {
		env.QueryRewriterRaw = raw
	}
	if len(bundle.McpServers) > 0 {
		env.McpServersRaw = append([]byte(nil), bundle.McpServers...)
	}
	env.VoiceDialogWsURL = trim(bundle.VoiceDialogWsURL)
	return env
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\t' && last != '\n' && last != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

package knowledge

import (
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

var (
	callSearchCfgMu sync.RWMutex
	callSearchCfg   = map[string]SearchConfig{}
)

// CacheSearchConfig pins per-call KB search tuning from assistant config.
func CacheSearchConfig(callID string, cfg SearchConfig) {
	callID = trim(callID)
	if callID == "" {
		return
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 3
	}
	callSearchCfgMu.Lock()
	callSearchCfg[callID] = cfg
	callSearchCfgMu.Unlock()
}

// ResolveSearchConfig returns per-call config or defaults.
func ResolveSearchConfig(callID string) SearchConfig {
	callID = trim(callID)
	cfg := SearchConfig{TopK: 3, Threshold: 0.25, AutoEnrich: true}
	if callID == "" {
		return cfg
	}
	callSearchCfgMu.RLock()
	if c, ok := callSearchCfg[callID]; ok {
		cfg = c
	}
	callSearchCfgMu.RUnlock()
	b := ResolveBinding(callID)
	// Merge field-by-field so a resolver Binding with zero AutoEnrich cannot wipe defaults.
	if b.SearchConfig.TopK > 0 {
		cfg.TopK = b.SearchConfig.TopK
	}
	if b.SearchConfig.Threshold > 0 {
		cfg.Threshold = b.SearchConfig.Threshold
	}
	if b.SearchConfig.MinScore > 0 {
		cfg.MinScore = b.SearchConfig.MinScore
	}
	if b.SearchConfig.UsePreviousRoundsSlice > 0 {
		cfg.UsePreviousRoundsSlice = b.SearchConfig.UsePreviousRoundsSlice
	}
	if b.SearchConfig.UseMemoEnhanceQuery {
		cfg.UseMemoEnhanceQuery = true
	}
	// AutoEnrich: keep true unless explicitly disabled via cached callSearchCfg / overlay.
	if callID != "" {
		callSearchCfgMu.RLock()
		if c, ok := callSearchCfg[callID]; ok {
			cfg.AutoEnrich = c.AutoEnrich
		}
		callSearchCfgMu.RUnlock()
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 3
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 0.25
	}
	return cfg
}

// ClearSearchConfig removes per-call tuning on hangup.
func ClearSearchConfig(callID string) {
	callID = trim(callID)
	if callID == "" {
		return
	}
	callSearchCfgMu.Lock()
	delete(callSearchCfg, callID)
	callSearchCfgMu.Unlock()
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\t' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

// RecallTopKForCall returns configured topK for a call leg.
func RecallTopKForCall(callID string) int {
	cfg := ResolveSearchConfig(callID)
	if cfg.TopK <= 0 {
		return 3
	}
	if cfg.TopK > 20 {
		return 20
	}
	return cfg.TopK
}

// RecallTopK is the default recall limit when no per-call config exists.
func RecallTopK() int { return 3 }

// SearchConfigFromVoiceEnv parses assistant knowledgeConfig from VoiceEnv.
func SearchConfigFromVoiceEnv(env tenantcfg.VoiceEnv) SearchConfig {
	cfg := SearchConfig{TopK: 3, Threshold: 0.25, AutoEnrich: true}
	if len(env.AgentConfigRaw) == 0 {
		return cfg
	}
	return searchConfigFromMap(env.AgentConfigRaw)
}

func searchConfigFromMap(raw map[string]any) SearchConfig {
	cfg := SearchConfig{TopK: 3, Threshold: 0.25, AutoEnrich: true}
	if raw == nil {
		return cfg
	}
	kc, ok := raw["knowledgeConfig"].(map[string]any)
	if !ok {
		if v, ok := raw["topK"]; ok {
			cfg.TopK = utils.IntDefault(v, cfg.TopK)
		}
		if v, ok := raw["threshold"]; ok {
			cfg.Threshold = utils.Float64Default(v, cfg.Threshold)
			cfg.MinScore = cfg.Threshold
		}
		return cfg
	}
	if v, ok := kc["topK"]; ok {
		cfg.TopK = utils.IntDefault(v, cfg.TopK)
	}
	if v, ok := kc["sliceMinimumScore"]; ok {
		cfg.MinScore = utils.Float64Default(v, cfg.MinScore)
	}
	if v, ok := kc["threshold"]; ok {
		cfg.Threshold = utils.Float64Default(v, cfg.Threshold)
		if cfg.MinScore == 0 {
			cfg.MinScore = cfg.Threshold
		}
	}
	if v, ok := kc["useMemoEnhanceQuery"]; ok {
		cfg.UseMemoEnhanceQuery = utils.BoolFromAny(v)
	}
	if v, ok := kc["usePreviousRoundsSlice"]; ok {
		cfg.UsePreviousRoundsSlice = utils.IntDefault(v, 0)
	}
	if v, ok := kc["autoEnrich"]; ok {
		cfg.AutoEnrich = utils.BoolFromAny(v)
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 3
	}
	return cfg
}

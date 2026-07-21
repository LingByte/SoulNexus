package models

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"gorm.io/gorm"
)

func voiceIDFromLegJSON(leg []byte) string {
	raw := bytesTrimJSON(leg)
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return ""
	}
	provider, _ := m["provider"].(string)
	keys := []string{
		tenantcfg.VoiceFieldForTTSProvider(provider),
		tenantcfg.VoiceFieldForRealtimeProvider(provider),
		"voice", "voiceId", "voice_id", "voiceType", "assetId", "asset_id",
		"speaker", "reference_id", "referenceId",
	}
	seen := map[string]struct{}{}
	for _, k := range keys {
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		if v, ok := m[k].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func extractTransitVoices(bundle tenantcfg.VoiceConfigBundle) (ttsVoice, rtVoice string) {
	ttsVoice = voiceIDFromLegJSON(bundle.Tts)
	rtVoice = voiceIDFromLegJSON(bundle.Realtime)
	if ttsVoice == "" {
		ttsVoice = rtVoice
	}
	return ttsVoice, rtVoice
}

// stripProviderLegsForTransit removes tenant/assistant vendor JSON so 号池 is the only provider source.
func stripProviderLegsForTransit(bundle tenantcfg.VoiceConfigBundle) tenantcfg.VoiceConfigBundle {
	bundle.Asr = nil
	bundle.Tts = nil
	bundle.Llm = nil
	bundle.Realtime = nil
	return bundle
}

// ApplyAIProviderPoolsToBundle routes legs through tenant-granted 号池 (中转站). Records pool ids on callID when set.
func ApplyAIProviderPoolsToBundle(db *gorm.DB, tenantID uint, callID string, bundle tenantcfg.VoiceConfigBundle, transitOnly bool) tenantcfg.VoiceConfigBundle {
	if db == nil || tenantID == 0 {
		return bundle
	}
	ttsVoice, rtVoice := extractTransitVoices(bundle)
	if transitOnly {
		bundle = stripProviderLegsForTransit(bundle)
	}

	var usedPools []uint
	applyLeg := func(modality string, voice string, set func([]byte)) {
		p, ok := SelectAIProviderPoolForTenant(db, tenantID, modality, voice)
		if !ok || len(bytesTrimJSON(p.Config)) == 0 {
			return
		}
		usedPools = append(usedPools, p.ID)
		raw := append([]byte(nil), p.Config...)
		if voice != "" {
			switch modality {
			case constants.AIPoolModalityTTS:
				raw = tenantcfg.ApplyTTSVoice(raw, voice)
			case constants.AIPoolModalityRealtime:
				raw = tenantcfg.ApplyRealtimeVoice(raw, voice)
			}
		}
		set(raw)
	}

	applyLeg(constants.AIPoolModalityASR, "", func(b []byte) { bundle.Asr = b })
	applyLeg(constants.AIPoolModalityLLM, "", func(b []byte) { bundle.Llm = b })
	applyLeg(constants.AIPoolModalityTTS, ttsVoice, func(b []byte) { bundle.Tts = b })
	applyLeg(constants.AIPoolModalityRealtime, rtVoice, func(b []byte) { bundle.Realtime = b })

	if callID = strings.TrimSpace(callID); callID != "" && len(usedPools) > 0 {
		callbinding.SetTransitPoolIDs(callID, usedPools)
	}
	return bundle
}

package handlers

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	"gorm.io/datatypes"
)

func credentialKindFromCreate(req dto.CredentialCreateReq) string {
	return models.NormalizeCredentialKind(req.Kind)
}

func credentialAIFieldsFromCreate(req dto.CredentialCreateReq) (voiceMode string, asr, tts, llm, rt datatypes.JSON, err error) {
	voiceMode = strings.TrimSpace(req.VoiceMode)
	asr, err = models.ParseOptionalJSONColumnNullable(string(req.AsrConfig))
	if err != nil {
		return "", nil, nil, nil, nil, err
	}
	tts, err = models.ParseOptionalJSONColumnNullable(string(req.TtsConfig))
	if err != nil {
		return "", nil, nil, nil, nil, err
	}
	llm, err = models.ParseOptionalJSONColumnNullable(string(req.LlmConfig))
	if err != nil {
		return "", nil, nil, nil, nil, err
	}
	rt, err = models.ParseOptionalJSONColumnNullable(string(req.RealtimeConfig))
	if err != nil {
		return "", nil, nil, nil, nil, err
	}
	return voiceMode, asr, tts, llm, rt, nil
}

func fixedCredentialAuth() (permJSON, routeJSON string, err error) {
	return models.FixedCredentialAuth()
}

func credentialAIUpdatesFromPatch(req dto.CredentialUpdateReq) (map[string]any, error) {
	out := map[string]any{}
	if req.VoiceMode != nil {
		out["voice_mode"] = strings.TrimSpace(*req.VoiceMode)
	}
	for _, pair := range []struct {
		raw []byte
		col string
	}{
		{req.AsrConfig, "asr_config"},
		{req.TtsConfig, "tts_config"},
		{req.LlmConfig, "llm_config"},
		{req.RealtimeConfig, "realtime_config"},
	} {
		if len(pair.raw) == 0 {
			continue
		}
		j, err := models.ParseOptionalJSONColumn(string(pair.raw))
		if err != nil {
			return nil, err
		}
		out[pair.col] = j
	}
	return out, nil
}

func refreshCredentialFixedRoutes(updates map[string]any) error {
	permJSON, routeJSON, err := fixedCredentialAuth()
	if err != nil {
		return err
	}
	updates["permission_codes"] = permJSON
	updates["allowed_route_ids"] = routeJSON
	return nil
}

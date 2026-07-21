package models

import (
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// DefaultAIBundleRouteIDs are the only HTTP capabilities exposed via tenant API keys.
func DefaultAIBundleRouteIDs() []string {
	return NormalizeAKSKRouteIDs([]string{
		"voice.session.create",
		"voice.session.end",
		"voice.session.webrtc",
		"voice.session.ws",
		"assistants.list",
		"assistants.get",
	})
}

// ExternalAPIKeyRouteIDs is an alias for the fixed external surface.
func ExternalAPIKeyRouteIDs() []string {
	return DefaultAIBundleRouteIDs()
}

// FixedCredentialAuth returns permission JSON and route JSON for new API keys.
func FixedCredentialAuth() (permJSON, routeJSON string, err error) {
	routeJSON, err = MarshalCredentialAllowedRouteIDs(ExternalAPIKeyRouteIDs())
	if err != nil {
		return "", "", err
	}
	permJSON, err = utils.MarshalStringSliceJSON([]string{constants.CredentialPermissionWildcard}, nil)
	return permJSON, routeJSON, err
}

// CredentialHasAIConfig reports whether the row carries any provider JSON.
func CredentialHasAIConfig(row Credential) bool {
	return len(bytesTrimJSON(row.AsrConfig)) > 0 ||
		len(bytesTrimJSON(row.TtsConfig)) > 0 ||
		len(bytesTrimJSON(row.LlmConfig)) > 0 ||
		len(bytesTrimJSON(row.RealtimeConfig)) > 0
}

func bytesTrimJSON(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

// NormalizeCredentialKind maps API / legacy values to platform_bundle or user_bundle.
func NormalizeCredentialKind(kind string) string {
	k := strings.TrimSpace(strings.ToLower(kind))
	switch k {
	case constants.CredentialKindPlatformBundle, "platform":
		return constants.CredentialKindPlatformBundle
	case constants.CredentialKindUserBundle, "user", "ai_bundle", "api_access":
		return constants.CredentialKindUserBundle
	default:
		return constants.CredentialKindUserBundle
	}
}

// CredentialUsesTenantAIConfig reports whether voice calls resolve provider JSON from the tenant row.
func CredentialUsesTenantAIConfig(row Credential) bool {
	return NormalizeCredentialKind(row.Kind) == constants.CredentialKindPlatformBundle
}

// ApplyCredentialVoiceOverlay merges credential provider JSON onto a resolved bundle.
// Non-empty credential fields replace tenant/assistant base configs for that leg.
func ApplyCredentialVoiceOverlay(db *gorm.DB, bundle tenantcfg.VoiceConfigBundle, credentialID uint, callID string) tenantcfg.VoiceConfigBundle {
	if db == nil || credentialID == 0 {
		return bundle
	}
	var row Credential
	if err := db.Where("id = ? AND status = ?", credentialID, constants.CredentialStatusActive).First(&row).Error; err != nil {
		return bundle
	}
	if CredentialUsesTenantAIConfig(row) {
		bundle = ApplyAIProviderPoolsToBundle(db, row.TenantID, callID, bundle, true)
		if vm := strings.TrimSpace(row.VoiceMode); vm != "" {
			bundle.VoiceMode = vm
		}
		return bundle
	}
	if !CredentialHasAIConfig(row) {
		return bundle
	}
	if raw := bytesTrimJSON(row.AsrConfig); len(raw) > 0 {
		bundle.Asr = append([]byte(nil), raw...)
	}
	if raw := bytesTrimJSON(row.TtsConfig); len(raw) > 0 {
		bundle.Tts = append([]byte(nil), raw...)
	}
	if raw := bytesTrimJSON(row.LlmConfig); len(raw) > 0 {
		bundle.Llm = mergeLLMCredentials(bundle.Llm, raw)
	}
	if raw := bytesTrimJSON(row.RealtimeConfig); len(raw) > 0 {
		bundle.Realtime = append([]byte(nil), raw...)
	}
	if vm := strings.TrimSpace(row.VoiceMode); vm != "" {
		bundle.VoiceMode = vm
	}
	return bundle
}

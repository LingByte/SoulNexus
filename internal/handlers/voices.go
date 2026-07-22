package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	siptts "github.com/LingByte/SoulNexus/pkg/voice/tts"
	voicepreview "github.com/LingByte/SoulNexus/pkg/voice/preview"
	"github.com/LingByte/lingllm/realtime/aliyunomni"
	"github.com/LingByte/lingllm/synthesizer"
	"github.com/gin-gonic/gin"
)

// listVoiceCatalog returns timbre options for a TTS/realtime provider.
//
// Query: provider (required), mode=tts|realtime (default tts)
func (h *Handlers) listVoiceCatalog(c *gin.Context) {
	provider := strings.TrimSpace(c.Query("provider"))
	if provider == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyProviderRequired))
		return
	}
	mode := strings.TrimSpace(c.Query("mode"))
	if mode == "" {
		mode = "tts"
	}
	if mode != "tts" && mode != "realtime" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyModeInvalid))
		return
	}
	out, err := listVoiceCatalog(provider, mode)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeNotFound, err))
		return
	}
	for i := range out.Voices {
		if key, ok, err := voicepreview.ResolveObjectKey(out.Provider, out.Mode, out.Voices[i].ID); err == nil && ok {
			out.Voices[i].PreviewURL = resolvePreviewPublicURL(c, key)
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

// getTenantVoiceProviders returns voiceMode + TTS/realtime provider slugs for the
// authenticated tenant (no API keys). Platform admins may pass ?tenantId=.
// When the tenant has 号池 grants, provider + optional voiceId allow-list come from pools.
func (h *Handlers) getTenantVoiceProviders(c *gin.Context) {
	tenantID := middleware.AuthTenantID(c)
	if tid := strings.TrimSpace(c.Query("tenantId")); tid != "" {
		if middleware.AuthPlatformAdminID(c) > 0 {
			parsed, err := utils.ParseID(tid)
			if err != nil {
				response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidTenantID))
				return
			}
			tenantID = parsed
		}
	}
	if tenantID == 0 {
		response.Render(c, response.NewI18n(response.CodeUnauthorized, i18n.KeyTenantRequired))
		return
	}
	var tenant models.Tenant
	if err := h.db.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyTenantNotFound))
		return
	}
	out := tenantcfg.VoiceProvidersFromTenant(
		tenant.VoiceMode,
		tenant.AsrConfig,
		tenant.TtsConfig,
		tenant.RealtimeConfig,
	)
	source := "tenant"
	ttsScope := models.ListTenantPoolVoiceScope(h.db, tenantID, constants.AIPoolModalityTTS)
	if ttsScope.Provider != "" {
		out.TtsProvider = ttsScope.Provider
		source = "pool"
		if !ttsScope.Wildcard && len(ttsScope.VoiceIDs) > 0 {
			out.TtsVoiceIDs = ttsScope.VoiceIDs
		}
	}
	rtScope := models.ListTenantPoolVoiceScope(h.db, tenantID, constants.AIPoolModalityRealtime)
	if rtScope.Provider != "" {
		out.RealtimeProvider = rtScope.Provider
		source = "pool"
		if !rtScope.Wildcard && len(rtScope.VoiceIDs) > 0 {
			out.RealtimeVoiceIDs = rtScope.VoiceIDs
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"voiceMode":          out.VoiceMode,
		"ttsProvider":        out.TtsProvider,
		"realtimeProvider":   out.RealtimeProvider,
		"source":             source,
		"ttsVoiceIds":        out.TtsVoiceIDs,
		"realtimeVoiceIds":   out.RealtimeVoiceIDs,
		"hasPoolGrant":       models.TenantHasActivePoolGrant(h.db, tenantID),
	})
}

type previewVoiceReq struct {
	Provider     string `json:"provider" binding:"required"`
	Mode         string `json:"mode"`
	VoiceID      string `json:"voiceId" binding:"required"`
	Text         string `json:"text"`
	TenantID     string `json:"tenantId"`
	CredentialID string `json:"credentialId"`
}

const defaultVoicePreviewText = "您好，欢迎致电，我是您的智能客服助手。"

// previewVoice synthesizes a short sample using credential / 号池 / tenant voice credentials.
func (h *Handlers) previewVoice(c *gin.Context) {
	var req previewVoiceReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "tts"
	}
	if mode != "tts" && mode != "realtime" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyModeInvalid))
		return
	}
	voiceID := strings.TrimSpace(req.VoiceID)
	if voiceID == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyVoiceIDRequired))
		return
	}

	tenantID := middleware.AuthTenantID(c)
	if tid := strings.TrimSpace(req.TenantID); tid != "" && middleware.AuthPlatformAdminID(c) > 0 {
		parsed, err := utils.ParseID(tid)
		if err != nil {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidTenantID))
			return
		}
		tenantID = parsed
	}
	if tenantID == 0 {
		response.Render(c, response.NewI18n(response.CodeUnauthorized, i18n.KeyTenantRequired))
		return
	}

	var tenant models.Tenant
	if err := h.db.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyTenantNotFound))
		return
	}

	credID := utils.ParseOptionalID(req.CredentialID)
	if credID > 0 {
		if _, err := models.GetCredentialByIDForTenant(h.db, credID, tenantID); err != nil {
			ginutil.WriteGORMError(c, err, "credential not found")
			return
		}
	}

	catalog, err := listVoiceCatalog(req.Provider, mode)
	if err != nil {
		response.Render(c, response.WrapErr(response.CodeNotFound, err))
		return
	}

	cfg, err := h.resolveVoicePreviewConfig(tenant, mode, voiceID, credID)
	if err != nil {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyVoicePreviewConfigMissing))
		return
	}
	overlayVoiceID(cfg, catalog.VoiceField, voiceID)

	resolved := resolveVoiceCatalogProvider(catalog.Provider)
	sample := strings.TrimSpace(req.Text)
	if sample == "" {
		sample = defaultVoicePreviewText
	}
	// Only cache the default sample phrase — custom text always synthesizes live.
	useCache := sample == defaultVoicePreviewText
	if useCache {
		if key, ok, err := voicepreview.ResolveObjectKey(resolved, mode, voiceID); err == nil && ok && key != "" {
			audioURL := resolvePreviewPublicURL(c, key)
			response.SuccessI18n(c, i18n.KeySuccess, gin.H{
				"audioUrl":   audioURL,
				"sampleRate": 24000,
				"format":     "wav",
				"cached":     true,
			})
			return
		}
	}

	var buf bytes.Buffer
	sampleRate := 16000
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	if mode == "realtime" {
		switch resolved {
		case "aliyun_omni":
			pcm, sr, err := aliyunomni.PreviewSpeech(ctx, cfg, voiceID, sample)
			if err != nil {
				response.Render(c, response.Wrap(response.CodeBadRequest, "voice preview failed", err))
				return
			}
			if len(pcm) == 0 {
				response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyVoicePreviewEmpty))
				return
			}
			buf.Write(pcm)
			if sr > 0 {
				sampleRate = sr
			}
		default:
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyVoicePreviewRealtimeUnsupported))
			return
		}
	} else {
		handle, err := siptts.NewFromCredential(synthesizer.TTSCredentialConfig(cfg))
		if err != nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "voice preview failed", err))
			return
		}
		defer handle.Engine.Close()

		if err := handle.Service.SynthesizeStream(ctx, sample, func(chunk []byte) error {
			if len(chunk) > 0 {
				buf.Write(chunk)
			}
			return nil
		}); err != nil && ctx.Err() == nil {
			response.Render(c, response.Wrap(response.CodeBadRequest, "voice preview failed", err))
			return
		}
		if buf.Len() == 0 {
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyVoicePreviewEmpty))
			return
		}
		if handle.SampleRate > 0 {
			sampleRate = handle.SampleRate
		} else {
			env, _ := tenantcfg.VoiceEnvFromJSON(tenant.AsrConfig, tenant.TtsConfig, tenant.LlmConfig, tenant.RealtimeConfig, tenant.VoiceMode)
			if env.TTSSampleRate > 0 {
				sampleRate = env.TTSSampleRate
			}
		}
	}

	out := gin.H{
		"sampleRate": sampleRate,
	}
	if useCache {
		key, err := voicepreview.UploadPCM(resolved, mode, voiceID, buf.Bytes(), sampleRate)
		if err != nil {
			response.Render(c, response.Wrap(response.CodeInternal, "voice preview upload failed", err))
			return
		}
		audioURL := resolvePreviewPublicURL(c, key)
		if audioURL == "" {
			response.Render(c, response.Wrap(response.CodeInternal, "voice preview upload failed", fmt.Errorf("empty public url for key %s", key)))
			return
		}
		_ = voicepreview.SetCachedObjectKey(resolved, mode, voiceID, key)
		out["audioUrl"] = audioURL
		out["format"] = "wav"
		out["cached"] = false
	} else {
		out["pcmBase64"] = base64.StdEncoding.EncodeToString(buf.Bytes())
		out["format"] = "pcm_s16le"
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

// resolveVoicePreviewConfig picks TTS/Realtime JSON for preview:
// credential (platform → 号池 / user → key bundle) → tenant 号池 → tenant AI columns.
func (h *Handlers) resolveVoicePreviewConfig(tenant models.Tenant, mode, voiceID string, credentialID uint) (map[string]any, error) {
	modality := constants.AIPoolModalityTTS
	if mode == "realtime" {
		modality = constants.AIPoolModalityRealtime
	}

	tryRaw := func(raw []byte) (map[string]any, error) {
		return cloneJSONMap(raw)
	}

	if credentialID > 0 {
		var row models.Credential
		if err := h.db.Where("id = ? AND tenant_id = ? AND status = ?", credentialID, tenant.ID, constants.CredentialStatusActive).
			First(&row).Error; err == nil {
			if models.CredentialUsesTenantAIConfig(row) {
				if raw, _, ok := models.ResolvePoolConfigForVoice(h.db, tenant.ID, modality, voiceID); ok {
					if cfg, err := tryRaw(raw); err == nil {
						return cfg, nil
					}
				}
			} else {
				raw := row.TtsConfig
				if mode == "realtime" {
					raw = row.RealtimeConfig
				}
				if cfg, err := tryRaw(raw); err == nil {
					return cfg, nil
				}
			}
		}
	}

	if raw, _, ok := models.ResolvePoolConfigForVoice(h.db, tenant.ID, modality, voiceID); ok {
		if cfg, err := tryRaw(raw); err == nil {
			return cfg, nil
		}
	}

	cfgRaw := tenant.TtsConfig
	if mode == "realtime" {
		cfgRaw = tenant.RealtimeConfig
	}
	return tryRaw(cfgRaw)
}

func cloneJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty config")
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return nil, fmt.Errorf("nil config")
	}
	return out, nil
}

func overlayVoiceID(cfg map[string]any, voiceField, voiceID string) {
	field := strings.TrimSpace(voiceField)
	if field == "" {
		field = "voiceType"
	}
	val := coerceVoiceValue(field, voiceID)
	cfg[field] = val
	switch field {
	case "voiceType":
		cfg["voice_type"] = val
	case "voiceId":
		cfg["voice_id"] = val
	case "assetId":
		cfg["asset_id"] = val
	case "speaker":
		cfg["speaker"] = val
	case "voice":
		cfg["voice"] = val
		cfg["voiceId"] = val
		cfg["voice_id"] = val
	}
}

func coerceVoiceValue(field, voiceID string) any {
	if field == "voiceType" || field == "voice_type" {
		if n, err := strconv.ParseInt(voiceID, 10, 64); err == nil {
			return n
		}
	}
	return voiceID
}

func resolvePreviewPublicURL(c *gin.Context, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if isAbsoluteHTTPURL(raw) {
		return raw
	}
	if key := previewObjectKeyFromStoredURL(raw); key != "" {
		return ginutil.UploadURL(c, key)
	}
	return raw
}

func isAbsoluteHTTPURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func previewObjectKeyFromStoredURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if i := strings.Index(raw, "/uploads/"); i >= 0 {
		return strings.TrimPrefix(raw[i+len("/uploads/"):], "/")
	}
	if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "/") {
		return strings.TrimPrefix(raw, "/")
	}
	return ""
}

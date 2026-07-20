package cloneconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/voiceclone"
)

const envProvider = "VOICE_CLONE_PROVIDER"

// ResolveEnabled returns the configured clone slug (xunfei|volcengine) when env + credentials are valid.
func ResolveEnabled() (slug string, provider voiceclone.Provider, ok bool) {
	p := strings.ToLower(strings.TrimSpace(utils.GetEnv(envProvider)))
	if p == "" || p == "none" || p == "off" || p == "disabled" {
		return "", "", false
	}
	switch p {
	case "xunfei", "iflytek", "讯飞":
		provider = voiceclone.ProviderXunfei
		slug = "xunfei"
	case "volcengine", "volc", "volcengine_clone", "火山", "doubao":
		provider = voiceclone.ProviderVolcengine
		slug = "volcengine"
	default:
		return "", "", false
	}
	if ValidateEnv(provider) != nil {
		return "", "", false
	}
	return slug, provider, true
}

// DefaultProvider reads VOICE_CLONE_PROVIDER when enabled; otherwise returns empty provider.
func DefaultProvider() voiceclone.Provider {
	if _, p, ok := ResolveEnabled(); ok {
		return p
	}
	return ""
}

// NewService builds a voice clone client when voice clone is enabled in env.
func NewService() (voiceclone.VoiceCloneService, voiceclone.Provider, error) {
	_, provider, ok := ResolveEnabled()
	if !ok {
		return nil, "", fmt.Errorf("voice clone not enabled or misconfigured")
	}
	factory := voiceclone.NewFactory()
	cfg := &voiceclone.Config{Provider: provider, Options: optionsFromEnv(provider)}
	svc, err := factory.CreateService(cfg)
	if err != nil {
		return nil, provider, err
	}
	return svc, provider, nil
}

func optionsFromEnv(provider voiceclone.Provider) map[string]any {
	switch provider {
	case voiceclone.ProviderXunfei:
		appID, apiKey := resolveXunfeiTrainCredentials()
		opts := map[string]any{
			"app_id":  appID,
			"api_key": apiKey,
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_BASE_URL")); v != "" {
			opts["base_url"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_WS_APP_ID")); v != "" {
			opts["ws_app_id"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_WS_API_KEY")); v != "" {
			opts["ws_api_key"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_WS_API_SECRET")); v != "" {
			opts["ws_api_secret"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_ENGINE_VERSION")); v != "" {
			opts["engine_version"] = v
		} else {
			opts["engine_version"] = "omni_v1"
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_VCN")); v != "" {
			opts["vcn"] = v
		} else {
			opts["vcn"] = "x6_clone"
		}
		if t := strings.TrimSpace(utils.GetEnv("XUNFEI_TIMEOUT")); t != "" {
			if n, err := strconv.Atoi(t); err == nil && n > 0 {
				opts["timeout"] = n
			}
		}
		return opts
	default:
		opts := map[string]any{
			"app_id": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("VOLCENGINE_APP_ID"),
				utils.GetEnv("VOLCENGINE_CLONE_APP_ID"),
			)),
			"token": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("VOLCENGINE_TOKEN"),
				utils.GetEnv("VOLCENGINE_ACCESS_TOKEN"),
				utils.GetEnv("VOLCENGINE_CLONE_TOKEN"),
			)),
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_CLUSTER")); v != "" {
			opts["cluster"] = v
		} else if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLUSTER")); v != "" {
			opts["cluster"] = v
		}
		if v := strings.TrimSpace(firstNonEmpty(
			utils.GetEnv("VOLCENGINE_CLONE_RESOURCE_ID"),
			utils.GetEnv("VOLCENGINE_RESOURCE_ID"),
		)); v != "" {
			opts["resource_id"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_MODEL_TYPE")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				opts["model_type"] = n
			}
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_VOICE_TYPE")); v != "" {
			opts["voice_type"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_ENCODING")); v != "" {
			opts["encoding"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_FRAME_DURATION")); v != "" {
			opts["frame_duration"] = v
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_SAMPLE_RATE")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts["sample_rate"] = n
			}
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_BIT_DEPTH")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts["bit_depth"] = n
			}
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_CHANNELS")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts["channels"] = n
			}
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_SPEED_RATIO")); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				opts["speed_ratio"] = f
			}
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_CLONE_TRAINING_TIMES")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts["training_times"] = n
			}
		}
		return opts
	}
}

func ProviderLabel(p voiceclone.Provider) string {
	switch p {
	case voiceclone.ProviderXunfei:
		return "讯飞星火"
	case voiceclone.ProviderVolcengine:
		return "火山引擎"
	default:
		return string(p)
	}
}

func ConfigSummary(provider voiceclone.Provider) map[string]any {
	out := map[string]any{
		"provider": string(provider),
		"label":    ProviderLabel(provider),
		"enabled":  true,
	}
	switch provider {
	case voiceclone.ProviderXunfei:
		out["supportsCreateTask"] = true
		out["supportsTrainingTexts"] = true
		out["speakerField"] = "assetId"
		out["ttsProvider"] = "xunfei_clone"
	case voiceclone.ProviderVolcengine:
		out["supportsCreateTask"] = false
		out["supportsTrainingTexts"] = false
		out["speakerField"] = "speakerId"
		out["ttsProvider"] = "volcengine_clone"
	}
	return out
}

func ValidateEnv(provider voiceclone.Provider) error {
	opts := optionsFromEnv(provider)
	switch provider {
	case voiceclone.ProviderXunfei:
		if strings.TrimSpace(fmt.Sprint(opts["app_id"])) == "" || strings.TrimSpace(fmt.Sprint(opts["api_key"])) == "" {
			return fmt.Errorf("XUNFEI_APP_ID and XUNFEI_API_KEY are required for voice clone")
		}
	case voiceclone.ProviderVolcengine:
		if strings.TrimSpace(fmt.Sprint(opts["token"])) == "" {
			return fmt.Errorf("VOLCENGINE_CLONE_TOKEN or VOLCENGINE_TOKEN is required for voice clone")
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// resolveXunfeiTrainCredentials returns credentials for 一句话复刻 HTTP API (aiauth / traintext).
// SoulNexus 仅使用 XUNFEI_APP_ID + XUNFEI_API_KEY。XUNFEI_TRAIN_* 仅在成对配置且 AppID 形态合理时覆盖，
// 避免把 32 位 APIKey 误填进 XUNFEI_TRAIN_APP_ID 导致 retcode=000007。
func resolveXunfeiTrainCredentials() (appID, apiKey string) {
	mainAppID := strings.TrimSpace(utils.GetEnv("XUNFEI_APP_ID"))
	mainAPIKey := strings.TrimSpace(utils.GetEnv("XUNFEI_API_KEY"))
	trainAppID := strings.TrimSpace(utils.GetEnv("XUNFEI_TRAIN_APP_ID"))
	trainAPIKey := strings.TrimSpace(utils.GetEnv("XUNFEI_TRAIN_API_KEY"))
	if trainAppID != "" && trainAPIKey != "" && looksLikeXunfeiAppID(trainAppID) {
		return trainAppID, trainAPIKey
	}
	return mainAppID, mainAPIKey
}

func looksLikeXunfeiAppID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	// 讯飞 AppID 通常 8 位左右；32 位字符串多为 APIKey 误填
	return len(id) < 20
}

// BuildTTSConfigForProfile builds pipeline TTS JSON for a trained clone profile (env credentials + voice id).
func BuildTTSConfigForProfile(profile voiceclone.Provider, assetID, speakerID string) (map[string]any, bool) {
	voiceID := strings.TrimSpace(assetID)
	if voiceID == "" {
		voiceID = strings.TrimSpace(speakerID)
	}
	if voiceID == "" {
		return nil, false
	}
	switch profile {
	case voiceclone.ProviderVolcengine:
		opts := optionsFromEnv(voiceclone.ProviderVolcengine)
		appID := strings.TrimSpace(fmt.Sprint(opts["app_id"]))
		token := strings.TrimSpace(fmt.Sprint(opts["token"]))
		if token == "" {
			return nil, false
		}
		out := map[string]any{
			"provider": "volcengine_clone",
			"assetId":  voiceID,
		}
		if appID != "" {
			out["appId"] = appID
		}
		out["token"] = token
		if v := strings.TrimSpace(fmt.Sprint(opts["cluster"])); v != "" {
			out["cluster"] = v
		}
		if v := strings.TrimSpace(fmt.Sprint(opts["resource_id"])); v != "" {
			out["resourceId"] = v
		}
		return out, true
	case voiceclone.ProviderXunfei:
		appID := strings.TrimSpace(firstNonEmpty(
			utils.GetEnv("XUNFEI_WS_APP_ID"),
			utils.GetEnv("XUNFEI_APP_ID"),
		))
		apiKey := strings.TrimSpace(firstNonEmpty(
			utils.GetEnv("XUNFEI_WS_API_KEY"),
			utils.GetEnv("XUNFEI_API_KEY"),
		))
		apiSecret := strings.TrimSpace(utils.GetEnv("XUNFEI_WS_API_SECRET"))
		if appID == "" || apiKey == "" || apiSecret == "" {
			return nil, false
		}
		vcn := strings.TrimSpace(utils.GetEnv("XUNFEI_VCN"))
		if vcn == "" {
			vcn = "x6_clone"
		}
		return map[string]any{
			"provider":  "xunfei",
			"appId":     appID,
			"apiKey":    apiKey,
			"apiSecret": apiSecret,
			"resId":     voiceID,
			"vcn":       vcn,
		}, true
	default:
		return nil, false
	}
}

// BuildTTSJSONForCloneProfile serializes pipeline TTS config for assistant/call binding.
func BuildTTSJSONForCloneProfile(row ProviderCloneRow) ([]byte, bool) {
	cfg, ok := BuildTTSConfigForProfile(voiceclone.Provider(row.Provider), row.AssetID, row.SpeakerID)
	if !ok {
		return nil, false
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, false
	}
	return raw, true
}

// ProviderCloneRow is a minimal view of VoiceCloneProfile for TTS wiring.
type ProviderCloneRow struct {
	Provider  string
	AssetID   string
	SpeakerID string
}

// EnvFileHint returns a short note for operators.
func EnvFileHint() string {
	return envProvider + "=xunfei|volcengine"
}

// LoadDotEnv is a no-op placeholder; server loads .env at startup.
func LoadDotEnv() { _ = os.Getenv(envProvider) }

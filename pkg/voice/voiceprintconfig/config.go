package voiceprintconfig

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/voiceprint"
)

const envProvider = "VOICEPRINT_PROVIDER"

// ResolveEnabled returns provider slug when VOICEPRINT_PROVIDER is http|xunfei|volcengine and credentials are valid.
// Empty or unknown value means disabled.
func ResolveEnabled() (slug string, provider voiceprint.Provider, ok bool) {
	p := strings.ToLower(strings.TrimSpace(utils.GetEnv(envProvider)))
	if p == "" {
		return "", "", false
	}
	switch p {
	case "http":
		provider = voiceprint.ProviderHTTP
		slug = "http"
	case "xunfei":
		provider = voiceprint.ProviderXunfei
		slug = "xunfei"
	case "volcengine":
		provider = voiceprint.ProviderVolcengine
		slug = "volcengine"
	default:
		return "", "", false
	}
	if ValidateEnv(provider) != nil {
		return "", "", false
	}
	return slug, provider, true
}

// DefaultProvider reads VOICEPRINT_PROVIDER when enabled; otherwise returns empty provider.
func DefaultProvider() voiceprint.Provider {
	if _, p, ok := ResolveEnabled(); ok {
		return p
	}
	return ""
}

// NewProvider builds a native cloud/HTTP adapter when voiceprint is enabled.
func NewProvider() (voiceprint.VoiceprintProvider, voiceprint.Provider, error) {
	_, provider, ok := ResolveEnabled()
	if !ok {
		return nil, "", fmt.Errorf("voiceprint not enabled or misconfigured")
	}
	factory := voiceprint.NewFactory()
	cfg := &voiceprint.ProviderConfig{Provider: provider, Options: optionsFromEnv(provider)}
	prov, err := factory.CreateProvider(cfg)
	if err != nil {
		return nil, provider, err
	}
	return prov, provider, nil
}

func optionsFromEnv(provider voiceprint.Provider) map[string]any {
	switch provider {
	case voiceprint.ProviderHTTP:
		opts := map[string]any{
			"base_url": defaultHTTPBaseURL(),
			"api_key": strings.TrimSpace(utils.GetEnv("VOICEPRINT_API_KEY")),
		}
		return opts
	case voiceprint.ProviderXunfei:
		opts := map[string]any{
			"app_id": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("XUNFEI_VOICEPRINT_APP_ID"),
				utils.GetEnv("XUNFEI_WS_APP_ID"),
				utils.GetEnv("XUNFEI_APP_ID"),
			)),
			"api_key": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("XUNFEI_VOICEPRINT_API_KEY"),
				utils.GetEnv("XUNFEI_WS_API_KEY"),
				utils.GetEnv("XUNFEI_API_KEY"),
			)),
			"api_secret": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("XUNFEI_VOICEPRINT_API_SECRET"),
				utils.GetEnv("XUNFEI_WS_API_SECRET"),
			)),
		}
		if v := strings.TrimSpace(utils.GetEnv("XUNFEI_VOICEPRINT_BASE_URL")); v != "" {
			opts["base_url"] = v
		} else if v := strings.TrimSpace(utils.GetEnv("XUNFEI_BASE_URL")); v != "" {
			opts["base_url"] = v
		}
		return opts
	default:
		opts := map[string]any{
			"access_key": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("VOLCENGINE_ACCESS_KEY"),
				utils.GetEnv("VOLCENGINE_VOICEPRINT_ACCESS_KEY"),
			)),
			"secret_key": strings.TrimSpace(firstNonEmpty(
				utils.GetEnv("VOLCENGINE_SECRET_KEY"),
				utils.GetEnv("VOLCENGINE_VOICEPRINT_SECRET_KEY"),
			)),
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_REGION")); v != "" {
			opts["region"] = v
		} else {
			opts["region"] = "cn-north-1"
		}
		if v := strings.TrimSpace(utils.GetEnv("VOLCENGINE_VOICEPRINT_BASE_URL")); v != "" {
			opts["base_url"] = v
		}
		return opts
	}
}

// HTTPServiceConfig builds lingllm HTTP microservice config from env.
func HTTPServiceConfig() *voiceprint.Config {
	get := func(k string) string { return strings.TrimSpace(utils.GetEnv(k)) }
	cfg := &voiceprint.Config{
		Enabled:             true,
		BaseURL:             defaultHTTPBaseURL(),
		APIKey:              get("VOICEPRINT_API_KEY"),
		Timeout:             30 * time.Second,
		ConnectTimeout:      10 * time.Second,
		SimilarityThreshold: 0.6,
		MaxCandidates:       10,
	}
	if v := get("VOICEPRINT_SIMILARITY_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.SimilarityThreshold = f
		}
	}
	if v := get("VOICEPRINT_MAX_CANDIDATES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.MaxCandidates = n
		}
	}
	if v := get("VOICEPRINT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
		}
	}
	if v := get("VOICEPRINT_CONNECT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.ConnectTimeout = d
		}
	}
	return cfg
}

func ProviderLabel(p voiceprint.Provider) string {
	switch p {
	case voiceprint.ProviderHTTP:
		return "HTTP 微服务"
	case voiceprint.ProviderXunfei:
		return "讯飞声纹"
	case voiceprint.ProviderVolcengine:
		return "火山引擎声纹"
	default:
		return string(p)
	}
}

func ConfigSummary(provider voiceprint.Provider) map[string]any {
	out := map[string]any{
		"provider": string(provider),
		"label":    ProviderLabel(provider),
		"enabled":  true,
	}
	switch provider {
	case voiceprint.ProviderHTTP:
		out["supportsEnroll"] = true
		out["supportsIdentify"] = true
		out["supportsVerify"] = true
	case voiceprint.ProviderXunfei:
		out["supportsEnroll"] = true
		out["supportsIdentify"] = true
		out["supportsVerify"] = true
		out["groupScoped"] = true
	case voiceprint.ProviderVolcengine:
		out["supportsEnroll"] = true
		out["supportsIdentify"] = true
		out["supportsVerify"] = true
	}
	if cfg := HTTPServiceConfig(); provider == voiceprint.ProviderHTTP {
		out["similarityThreshold"] = cfg.SimilarityThreshold
		out["maxCandidates"] = cfg.MaxCandidates
	}
	return out
}

func ValidateEnv(provider voiceprint.Provider) error {
	opts := optionsFromEnv(provider)
	switch provider {
	case voiceprint.ProviderHTTP:
		if strings.TrimSpace(fmt.Sprint(opts["api_key"])) == "" {
			return fmt.Errorf("VOICEPRINT_API_KEY is required for voiceprint http provider")
		}
	case voiceprint.ProviderXunfei:
		for _, k := range []string{"app_id", "api_key", "api_secret"} {
			if strings.TrimSpace(fmt.Sprint(opts[k])) == "" {
				return fmt.Errorf("xunfei voiceprint requires app_id, api_key and api_secret")
			}
		}
	case voiceprint.ProviderVolcengine:
		if strings.TrimSpace(fmt.Sprint(opts["access_key"])) == "" || strings.TrimSpace(fmt.Sprint(opts["secret_key"])) == "" {
			return fmt.Errorf("VOLCENGINE_ACCESS_KEY and VOLCENGINE_SECRET_KEY are required for voiceprint")
		}
	}
	return nil
}

func TenantGroupID(tenantID uint) string {
	return fmt.Sprintf("lingecho-tenant-%d", tenantID)
}

func TenantIDString(tenantID uint) string {
	return fmt.Sprintf("%d", tenantID)
}

func AssistantIDString(assistantID *uint) string {
	if assistantID == nil || *assistantID == 0 {
		return ""
	}
	return fmt.Sprintf("%d", *assistantID)
}

func ProfileIDString(profileID uint) string {
	if profileID == 0 {
		return ""
	}
	return fmt.Sprintf("%d", profileID)
}

// AgentID is deprecated; kept for backward compatibility with older configs.
func AgentID(tenantID uint) string {
	return TenantIDString(tenantID)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func defaultHTTPBaseURL() string {
	if v := firstNonEmpty(
		utils.GetEnv("VOICEPRINT_BASE_URL"),
		utils.GetEnv("VOICEPRINT_SERVICE_URL"),
	); v != "" {
		return v
	}
	return "http://127.0.0.1:8005"
}

package rerank

import "strings"

// Provider constants matching root rerank package identifiers.
const (
	ProviderAliyun      = "aliyun"
	ProviderZhipu       = "zhipu"
	ProviderJina        = "jina"
	ProviderNvidia      = "nvidia"
	ProviderLKEAP       = "lkeap"
	ProviderOpenAI      = "openai"
	ProviderSiliconFlow = "siliconflow"
	ProviderCohere      = "cohere"
	ProviderLocal       = "local"
)

// Legacy provider aliases kept for backward compatibility.
const (
	ProviderJinaAI   = "jinaai"
	ProviderCohereAI = "cohereai"
)

// ProviderName normalizes a provider string to a canonical identifier.
func ProviderName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "generic":
		return ""
	case "jinaai", "jina-ai":
		return ProviderJina
	case "cohereai", "cohere-ai":
		return ProviderCohere
	case "silicon-flow":
		return ProviderSiliconFlow
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

// DetectProvider infers the rerank provider from a base URL.
func DetectProvider(baseURL string) string {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case strings.Contains(u, "dashscope.aliyuncs.com"), strings.Contains(u, "aliyuncs.com"):
		return ProviderAliyun
	case strings.Contains(u, "bigmodel.cn"):
		return ProviderZhipu
	case strings.Contains(u, "jina.ai"):
		return ProviderJina
	case strings.Contains(u, "nvidia.com"):
		return ProviderNvidia
	case strings.Contains(u, "tencentcloudapi.com"), strings.Contains(u, "lkeap"):
		return ProviderLKEAP
	case strings.Contains(u, "siliconflow"):
		return ProviderSiliconFlow
	case strings.Contains(u, "cohere"):
		return ProviderCohere
	default:
		return ProviderOpenAI
	}
}

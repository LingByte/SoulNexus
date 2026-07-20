package embedder

import "strings"

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

const (
	ProviderOpenAI      = "openai"
	ProviderAliyun      = "aliyun"
	ProviderVolcengine  = "volcengine"
	ProviderJina        = "jina"
	ProviderAzureOpenAI = "azure_openai"
	ProviderNvidia      = "nvidia"
	ProviderGemini      = "gemini"
	ProviderZhipu       = "zhipu"
	ProviderOllama      = "ollama"
	ProviderDashscope   = "dashscope"
	ProviderLocal       = "local"
)

// NormalizeProvider maps legacy or alias provider names to canonical identifiers.
func NormalizeProvider(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "generic":
		return ""
	case "dashscope", "tongyi", "qwen":
		return ProviderDashscope
	case "jinaai", "jina-ai":
		return ProviderJina
	case "azure-openai", "azure":
		return ProviderAzureOpenAI
	case "google", "google-ai", "google-gemini":
		return ProviderGemini
	case "silicon-flow":
		return "siliconflow"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

// DetectProvider infers the embedding provider from a base URL.
func DetectProvider(baseURL string) string {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case strings.Contains(u, "dashscope.aliyuncs.com"), strings.Contains(u, "aliyuncs.com"):
		return ProviderAliyun
	case strings.Contains(u, "open.bigmodel.cn"), strings.Contains(u, "zhipu"):
		return ProviderZhipu
	case strings.Contains(u, "jina.ai"):
		return ProviderJina
	case strings.Contains(u, "openai.azure.com"):
		return ProviderAzureOpenAI
	case strings.Contains(u, "api.openai.com"):
		return ProviderOpenAI
	case strings.Contains(u, "generativelanguage.googleapis.com"):
		return ProviderGemini
	case strings.Contains(u, "volces.com"), strings.Contains(u, "volcengine"):
		return ProviderVolcengine
	case strings.Contains(u, "nvidia.com"):
		return ProviderNvidia
	case strings.Contains(u, "11434"):
		return ProviderOllama
	default:
		return ProviderOpenAI
	}
}

func isKnownProvider(name string) bool {
	switch name {
	case ProviderOpenAI, ProviderAliyun, ProviderDashscope, ProviderVolcengine,
		ProviderJina, ProviderAzureOpenAI, ProviderNvidia, ProviderGemini,
		ProviderZhipu, ProviderOllama, ProviderLocal:
		return true
	default:
		return false
	}
}

func isAliyunProvider(name string) bool {
	return name == ProviderAliyun || name == ProviderDashscope
}

// OpenAICompatStringInputBaseURL reports whether a base URL is known to require
// string (not array) "input" in OpenAI-compatible /embeddings requests.
func OpenAICompatStringInputBaseURL(baseURL string) bool {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	return strings.Contains(u, "ai.gitee.com")
}

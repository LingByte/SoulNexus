package rerank

import (
	"net/http"
	"testing"
)

func TestProviderName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"jinaai", ProviderJina},
		{"cohereai", ProviderCohere},
		{"silicon-flow", ProviderSiliconFlow},
		{"aliyun", ProviderAliyun},
		{"", ""},
	}
	for _, tt := range tests {
		if got := ProviderName(tt.input); got != tt.want {
			t.Errorf("ProviderName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://dashscope.aliyuncs.com/api/v1", ProviderAliyun},
		{"https://open.bigmodel.cn/api/paas/v4/rerank", ProviderZhipu},
		{"https://api.jina.ai/v1", ProviderJina},
		{"https://ai.api.nvidia.com/v1/retrieval/nvidia/reranking", ProviderNvidia},
		{"https://lkeap.tencentcloudapi.com", ProviderLKEAP},
		{"https://api.siliconflow.cn/v1", ProviderSiliconFlow},
		{"https://api.cohere.ai/v1", ProviderCohere},
		{"https://api.openai.com/v1", ProviderOpenAI},
	}
	for _, tt := range tests {
		if got := DetectProvider(tt.baseURL); got != tt.want {
			t.Errorf("DetectProvider(%q) = %q, want %q", tt.baseURL, got, tt.want)
		}
	}
}

func TestApplyCustomHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/rerank", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	ApplyCustomHeaders(req, map[string]string{
		"X-Gateway": "edge",
		"":          "ignored",
	})
	if got := req.Header.Get("X-Gateway"); got != "edge" {
		t.Fatalf("X-Gateway = %q, want edge", got)
	}
}

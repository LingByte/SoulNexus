package embedder

import (
	"net/http"
	"testing"
)

func TestNormalizeProvider(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"dashscope", ProviderDashscope},
		{"aliyun", ProviderAliyun},
		{"jinaai", ProviderJina},
		{"azure", ProviderAzureOpenAI},
		{"google", ProviderGemini},
		{"", ""},
		{"generic", ""},
	}
	for _, tt := range tests {
		if got := NormalizeProvider(tt.input); got != tt.want {
			t.Errorf("NormalizeProvider(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://dashscope.aliyuncs.com/compatible-mode/v1", ProviderAliyun},
		{"https://open.bigmodel.cn/api/paas/v4", ProviderZhipu},
		{"https://api.jina.ai/v1", ProviderJina},
		{"https://integrate.api.nvidia.com/v1", ProviderNvidia},
		{"https://example-resource.openai.azure.com", ProviderAzureOpenAI},
		{"https://generativelanguage.googleapis.com/v1beta", ProviderGemini},
		{"https://ark.cn-beijing.volces.com", ProviderVolcengine},
		{"https://api.openai.com/v1", ProviderOpenAI},
		{"http://localhost:11434", ProviderOllama},
	}
	for _, tt := range tests {
		if got := DetectProvider(tt.baseURL); got != tt.want {
			t.Errorf("DetectProvider(%q) = %q, want %q", tt.baseURL, got, tt.want)
		}
	}
}

func TestApplyCustomHeadersSkipsReserved(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.com/embeddings", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer original")
	req.Header.Set("Content-Type", "application/json")

	ApplyCustomHeaders(req, map[string]string{
		"Authorization": "Bearer override",
		"Content-Type":  "text/plain",
		"X-Gateway":     "edge",
	})

	if got := req.Header.Get("Authorization"); got != "Bearer original" {
		t.Fatalf("Authorization = %q, want original preserved", got)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want original preserved", got)
	}
	if got := req.Header.Get("X-Gateway"); got != "edge" {
		t.Fatalf("X-Gateway = %q, want edge", got)
	}
}

func TestSanitizeEmbedInputs(t *testing.T) {
	got := SanitizeEmbedInputs([]string{" hello ", "", "world"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0] != "hello" || got[1] != " " || got[2] != "world" {
		t.Fatalf("unexpected sanitized values: %#v", got)
	}
}

func TestOpenAICompatStringInputBaseURL(t *testing.T) {
	if !OpenAICompatStringInputBaseURL("https://ai.gitee.com/v1") {
		t.Fatal("expected Gitee AI base URL to require string input")
	}
	if OpenAICompatStringInputBaseURL("https://api.openai.com/v1") {
		t.Fatal("expected OpenAI base URL to use array input")
	}
}

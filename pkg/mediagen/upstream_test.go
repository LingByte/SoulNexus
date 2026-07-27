package mediagen

import (
	"strings"
	"testing"
)

func TestParseVolcengineErrorDetail(t *testing.T) {
	body := []byte(`{"error":{"code":"InvalidEndpointOrModel.NotFound","message":"The model ` + "`doubao-seedance-1-5-pro-251215`" + ` does not exist","type":"NotFound"}}`)
	got := parseVolcengineErrorDetail(body)
	want := "InvalidEndpointOrModel.NotFound: The model `doubao-seedance-1-5-pro-251215` does not exist"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUpstreamHTTPError404Hint(t *testing.T) {
	err := upstreamHTTPError("Seedance 创建任务失败", 404, []byte(`{"error":{"code":"InvalidEndpointOrModel.NotFound","message":"model not found"}}`))
	if err == nil || !strings.Contains(err.Error(), "InvalidEndpointOrModel.NotFound") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "SEEDANCE_MODEL_ID") {
		t.Fatalf("expected config hint, got: %v", err)
	}
}

package workflowdef

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
)

func TestExtractClientMeta_JSONMap(t *testing.T) {
	meta := models.JSONMap{"ip": "127.0.0.1", "userAgent": "curl/8.0"}
	params := map[string]interface{}{
		"_client_meta": meta,
		"city":         "成都",
	}
	got := ExtractClientMeta(params)
	if got == nil || got["ip"] != "127.0.0.1" {
		t.Fatalf("expected client meta, got %#v", got)
	}
}

func TestExtractClientMeta_Map(t *testing.T) {
	params := map[string]interface{}{
		"_client_meta": map[string]interface{}{
			"ip":        "10.0.0.1",
			"userAgent": "Reqable",
		},
	}
	got := ExtractClientMeta(params)
	if got == nil || got["userAgent"] != "Reqable" {
		t.Fatalf("expected client meta, got %#v", got)
	}
}

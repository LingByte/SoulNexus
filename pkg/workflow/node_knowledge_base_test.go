package workflow

import "testing"

func TestParseKnowledgeBaseConfig_StringNumbers(t *testing.T) {
	cfg, err := ParseKnowledgeBaseConfig(map[string]interface{}{
		"namespaceId":    "16285369153940881920",
		"inputVariable":  "query",
		"outputVariable": "kb_result",
		"topK":           "5",
		"minScore":       "0",
		"outputFormat":   "text_block",
	})
	if err != nil {
		t.Fatalf("ParseKnowledgeBaseConfig() error = %v", err)
	}
	if cfg.TopK != 5 {
		t.Fatalf("TopK = %d, want 5", cfg.TopK)
	}
	if cfg.MinScore != 0 {
		t.Fatalf("MinScore = %v, want 0", cfg.MinScore)
	}
}

package nlu

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
)

func TestConfigReady(t *testing.T) {
	c := Config{
		Enabled:       true,
		ModelPath:     "/tmp/model.onnx",
		TokenizerPath: "/tmp/tokenizer.json",
		ORTLibPath:    "/tmp/libonnxruntime.dylib",
	}
	if !c.Ready() {
		t.Fatal("expected ready")
	}
	c.ModelPath = ""
	if c.Ready() {
		t.Fatal("expected not ready without model")
	}
}

func TestDeployEnabledEnvKey(t *testing.T) {
	if constants.ENVNLUEnabled != "NLU_ENABLED" {
		t.Fatalf("env key=%q", constants.ENVNLUEnabled)
	}
}

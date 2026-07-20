package intentonnx

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ort "github.com/yalue/onnxruntime_go"
)

//go:embed default_intents.json
var defaultIntentsJSON []byte

// DefaultIntentConfigJSON returns the embedded default intents JSON bytes.
func DefaultIntentConfigJSON() []byte {
	return append([]byte(nil), defaultIntentsJSON...)
}

// IntentConfig maps each logits index to a display name, canned reply, and optional keywords.
// Order of Intents must match the model's id2label / training class order.
type IntentConfig struct {
	MinSoftmaxProb    float64       `json:"min_softmax_prob"`
	KeywordLogitBonus float64       `json:"keyword_logit_bonus"`
	MinTopMargin      float64       `json:"min_top_margin"`
	DefaultReply      string        `json:"default_reply"`
	Intents           []IntentEntry `json:"intents"`
}

// IntentEntry is one output class (one logits index).
type IntentEntry struct {
	Name           string   `json:"name"`
	Reply          string   `json:"reply"`
	ReplyVariants  []string `json:"reply_variants,omitempty"` // optional extra canned lines; one is chosen at random with Reply
	Keywords       []string `json:"keywords"`
	KeywordBonus   float64  `json:"keyword_bonus"`
}

// LoadIntentConfig parses JSON. If path is empty, uses the embedded default.
func LoadIntentConfig(path string) (*IntentConfig, error) {
	raw := defaultIntentsJSON
	if strings.TrimSpace(path) != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	var cfg IntentConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ValidateIntentConfig checks intent rows match the model class count.
func ValidateIntentConfig(cfg *IntentConfig, nClass int) error {
	if cfg == nil {
		return fmt.Errorf("nil intent config")
	}
	if nClass <= 0 {
		return fmt.Errorf("invalid class count %d", nClass)
	}
	if len(cfg.Intents) != nClass {
		if nClass == 384 || nClass == 768 {
			return fmt.Errorf("intent config has %d entries but model output dim is %d (looks like an embedding model, not sequence classification; see data/nlu/README.md)", len(cfg.Intents), nClass)
		}
		return fmt.Errorf("intent config has %d entries but model has %d classes (intents JSON order must match model id2label; see data/nlu/README.md)", len(cfg.Intents), nClass)
	}
	for i := range cfg.Intents {
		if strings.TrimSpace(cfg.Intents[i].Reply) == "" {
			return fmt.Errorf("intents[%d] missing reply", i)
		}
	}
	return nil
}

func logitsClassCount(outputs []ort.InputOutputInfo) int {
	for _, o := range outputs {
		if o.Name != "logits" || o.OrtValueType != ort.ONNXTypeTensor || o.DataType != ort.TensorElementDataTypeFloat {
			continue
		}
		if len(o.Dimensions) == 0 {
			continue
		}
		last := o.Dimensions[len(o.Dimensions)-1]
		if last > 0 {
			return int(last)
		}
	}
	for _, o := range outputs {
		if o.OrtValueType != ort.ONNXTypeTensor || o.DataType != ort.TensorElementDataTypeFloat || len(o.Dimensions) == 0 {
			continue
		}
		last := o.Dimensions[len(o.Dimensions)-1]
		if last > 0 {
			return int(last)
		}
	}
	return 0
}

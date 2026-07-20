package nlu

import (
	"os"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// Config holds ONNX intent settings (env + optional system-config overrides).
type Config struct {
	Enabled       bool
	Mode          string // "embedding" (dynamic intents) or "classifier" (fixed ONNX head)
	ModelPath     string
	TokenizerPath string
	IntentsPath   string
	ORTLibPath    string
	SeqLen        int
	MinConfidence float64
	UseCoreML     bool
}

const (
	ModeEmbedding   = "embedding"
	ModeClassifier  = "classifier"
)

// IsEmbeddingMode reports whether tenant models use sentence-similarity routing.
func (c Config) IsEmbeddingMode() bool {
	return strings.ToLower(strings.TrimSpace(c.Mode)) != ModeClassifier
}

var (
	current Config
	loaded  bool
)

// DeployEnabled reports whether NLU lab is turned on at deploy time (.env NLU_ENABLED).
func DeployEnabled() bool {
	return utils.GetBoolEnv(constants.ENVNLUEnabled)
}

func envOrConfig(db *gorm.DB, envKey, configKey string) string {
	if v := strings.TrimSpace(utils.GetEnv(envKey)); v != "" {
		return v
	}
	if db != nil && configKey != "" {
		return strings.TrimSpace(utils.GetValue(db, configKey))
	}
	return ""
}

// Load reads NLU settings into the process-wide snapshot.
func Load(db *gorm.DB) Config {
	cfg := Config{
		Enabled:       DeployEnabled(),
		Mode:          ModeEmbedding,
		SeqLen:        128,
		MinConfidence: 0.85,
	}
	mode := strings.ToLower(strings.TrimSpace(utils.GetEnv(constants.ENVNLUMode)))
	if mode == ModeClassifier {
		cfg.Mode = ModeClassifier
		cfg.MinConfidence = 0.85
	}
	cfg.ModelPath = envOrConfig(db, constants.ENVNLUModel, constants.KEY_NLU_MODEL)
	cfg.TokenizerPath = envOrConfig(db, constants.ENVNLUTokenizer, constants.KEY_NLU_TOKENIZER)
	cfg.IntentsPath = envOrConfig(db, constants.ENVNLUIntentsConfig, constants.KEY_NLU_INTENTS_CONFIG)
	cfg.ORTLibPath = strings.TrimSpace(utils.GetEnv(constants.ENVNLUORTLib))
	if cfg.ORTLibPath == "" {
		cfg.ORTLibPath = strings.TrimSpace(os.Getenv("ONNXRUNTIME_SHARED_LIBRARY_PATH"))
	}
	if v := strings.TrimSpace(utils.GetEnv(constants.ENVNLUSeq)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.SeqLen = n
		}
	}
	if db != nil {
		if v := strings.TrimSpace(utils.GetValue(db, constants.KEY_NLU_MIN_CONFIDENCE)); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
				cfg.MinConfidence = f
			}
		}
	}
	v := strings.ToLower(strings.TrimSpace(utils.GetEnv(constants.ENVNLUCoreML)))
	cfg.UseCoreML = v == "1" || v == "true" || v == "yes"
	current = cfg
	loaded = true
	return current
}

// Get returns the last Load snapshot.
func Get() Config {
	if !loaded {
		return Config{Enabled: DeployEnabled()}
	}
	return current
}

// SetForTest replaces the global snapshot (tests only).
func SetForTest(cfg Config) {
	current = cfg
	loaded = true
}

// Ready reports whether ONNX inference may be attempted.
func (c Config) Ready() bool {
	return c.Enabled && c.ModelPath != "" && c.TokenizerPath != "" && c.ORTLibPath != ""
}

package providers

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/LingByte/lingllm/recognizer"
	"go.uber.org/zap"
)

// NewHotwordCorrector builds optional ASR hotword correction (env fallback).
func NewHotwordCorrector(lg *zap.Logger) *HotwordCorrector {
	return NewHotwordCorrectorWithPairs(lg, nil)
}

// NewHotwordCorrectorWithPairs builds ASR correction from assistant pairs or env.
func NewHotwordCorrectorWithPairs(lg *zap.Logger, pairs map[string]string) *HotwordCorrector {
	if len(pairs) == 0 {
		pairs = loadHotwordPairs()
	}
	if len(pairs) == 0 {
		return &HotwordCorrector{logger: lg}
	}
	if lg != nil {
		lg.Info("hotword corrector enabled", zap.Int("pairs", len(pairs)))
	}
	return newHotwordCorrector(lg, pairs, nil)
}

func loadHotwordPairs() map[string]string {
	out := map[string]string{}

	if raw := strings.TrimSpace(os.Getenv("LINGECHO_HOTWORD_CORRECTIONS_JSON")); raw != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			for k, v := range m {
				out[k] = v
			}
		}
	}

	// CSV-ish fallback: "foo=bar,baz=qux"
	if raw := strings.TrimSpace(os.Getenv("LINGECHO_HOTWORD_CORRECTIONS")); raw != "" {
		items := strings.Split(raw, ",")
		for _, it := range items {
			kv := strings.SplitN(strings.TrimSpace(it), "=", 2)
			if len(kv) != 2 {
				continue
			}
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			if k != "" && v != "" {
				out[k] = v
			}
		}
	}
	return out
}

type hotWordJSON struct {
	Word             string   `json:"word"`
	Weight           int      `json:"weight"`
	ReplacedWords    []string `json:"replacedWords"`
	EnableFuzzyMatch bool     `json:"enableFuzzyMatch"`
}

// ParseRecognizerHotWords decodes assistant hotWords JSON for ASR biasing.
func ParseRecognizerHotWords(raw []byte) []recognizer.HotWord {
	return ParseHotwordBundle(raw, nil, nil).ASRHotWords
}

// HotwordPairsFromJSON builds post-ASR correction pairs from assistant hotWords.
func HotwordPairsFromJSON(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var rows []hotWordJSON
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	out := map[string]string{}
	for _, w := range rows {
		canon := strings.TrimSpace(w.Word)
		if canon == "" {
			continue
		}
		if display, _ := phonemeHintsFromHotword(canon); display != "" {
			canon = display
		}
		for _, rep := range w.ReplacedWords {
			r := strings.TrimSpace(rep)
			if r != "" && r != canon {
				out[r] = canon
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

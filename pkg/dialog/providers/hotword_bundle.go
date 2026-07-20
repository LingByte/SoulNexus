package providers

import (
	"encoding/json"
	"strings"

	"github.com/LingByte/lingllm/recognizer"
	"go.uber.org/zap"
)

// HotwordBundle groups ASR biasing, post-ASR correction, and TTS text prep.
type HotwordBundle struct {
	ASRHotWords []recognizer.HotWord
	Corrector   *HotwordCorrector
	TTS         *TTSTextOptimizer
}

// HotwordCorrector applies exact replacedWords mapping and optional pinyin fuzzy match.
type HotwordCorrector struct {
	logger       *zap.Logger
	replaceWords map[string]string
	fuzzyCanon   map[string]string
}

// ParseHotwordBundle decodes assistant hotWords JSON into runtime ASR/TTS helpers.
func ParseHotwordBundle(raw []byte, lg *zap.Logger, ttsConfig map[string]any) HotwordBundle {
	if len(raw) == 0 {
		return HotwordBundle{}
	}
	var rows []hotWordJSON
	if err := json.Unmarshal(raw, &rows); err != nil {
		return HotwordBundle{}
	}

	asr := make([]recognizer.HotWord, 0, len(rows))
	replacePairs := map[string]string{}
	fuzzyCanon := map[string]string{}
	phonemeByDisplay := map[string]string{}

	for _, w := range rows {
		word := strings.TrimSpace(w.Word)
		if word == "" {
			continue
		}
		display, phoneme := phonemeHintsFromHotword(word)
		if display == "" {
			continue
		}
		weight := w.Weight
		if weight <= 0 {
			weight = 10
		}
		asr = append(asr, recognizer.HotWord{Word: display, Weight: weight})
		if phoneme != "" {
			phonemeByDisplay[display] = phoneme
		}
		for _, rep := range w.ReplacedWords {
			r := strings.TrimSpace(rep)
			if r != "" && r != display {
				replacePairs[r] = display
			}
		}
		if w.EnableFuzzyMatch {
			fuzzyCanon[display] = display
		}
	}

	var corrector *HotwordCorrector
	if len(replacePairs) > 0 || len(fuzzyCanon) > 0 {
		corrector = newHotwordCorrector(lg, replacePairs, fuzzyCanon)
	}
	tts := NewTTSTextOptimizer(replacePairs, phonemeByDisplay, ttsConfig)

	if lg != nil && (len(asr) > 0 || corrector != nil || tts != nil) {
		lg.Info("assistant hotwords loaded",
			zap.Int("asr_count", len(asr)),
			zap.Int("replace_pairs", len(replacePairs)),
			zap.Int("fuzzy_words", len(fuzzyCanon)),
			zap.Int("tts_phoneme", len(phonemeByDisplay)),
		)
	}

	return HotwordBundle{
		ASRHotWords: asr,
		Corrector:   corrector,
		TTS:         tts,
	}
}

func newHotwordCorrector(lg *zap.Logger, replacePairs, fuzzyCanon map[string]string) *HotwordCorrector {
	return &HotwordCorrector{
		logger:       lg,
		replaceWords: replacePairs,
		fuzzyCanon:   fuzzyCanon,
	}
}

// Correct implements cascaded.TextRewriter.
func (c *HotwordCorrector) Correct(text string) string {
	t := strings.TrimSpace(text)
	if t == "" || c == nil {
		return t
	}
	if len(c.replaceWords) == 0 && len(c.fuzzyCanon) == 0 {
		return t
	}
	return correctByTokens(t, c.replaceWords, c.fuzzyCanon)
}

// PrepareTTS returns TTS-safe text using bundle TTS optimizer when present.
func (b HotwordBundle) PrepareTTS(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if b.TTS != nil {
		return b.TTS.Prepare(text)
	}
	return text
}

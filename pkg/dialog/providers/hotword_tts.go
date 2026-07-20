package providers

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ssmlSpeakOpen  = "<speak>"
	ssmlSpeakClose = "</speak>"
)

var ssmlPhonemeRegex = regexp.MustCompile(`[\{｛]([^|｜\}｝]+)[|｜]([^\}｝]+)[\}｝]`)

// TTSTextOptimizer rewrites assistant reply text before synthesis.
type TTSTextOptimizer struct {
	replacer         *strings.Replacer
	phonemeByDisplay map[string]string
	ssmlEnabled      bool
	ssmlReplacements []ssmlReplacement
}

type ssmlReplacement struct {
	key   string
	value string
}

// NewTTSTextOptimizer builds TTS-side hotword normalization and optional SSML phoneme tags.
func NewTTSTextOptimizer(replacePairs map[string]string, phonemeByDisplay map[string]string, ttsConfig map[string]any) *TTSTextOptimizer {
	rp := make([]string, 0, len(replacePairs)*2)
	for from, to := range replacePairs {
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" || to == "" || from == to {
			continue
		}
		rp = append(rp, from, to)
	}
	opt := &TTSTextOptimizer{phonemeByDisplay: phonemeByDisplay}
	if len(rp) > 0 {
		opt.replacer = strings.NewReplacer(rp...)
	}
	if ttsConfig != nil {
		if v, ok := ttsConfig["ssml"].(bool); ok {
			opt.ssmlEnabled = v
		}
		if rows, ok := ttsConfig["ssmlReplaces"].([]any); ok {
			for _, row := range rows {
				m, ok := row.(map[string]any)
				if !ok {
					continue
				}
				key := strings.TrimSpace(fmt.Sprint(m["key"]))
				val := strings.TrimSpace(fmt.Sprint(m["value"]))
				if key == "" || val == "" {
					continue
				}
				opt.ssmlReplacements = append(opt.ssmlReplacements, ssmlReplacement{
					key:   key,
					value: convertToSSMLPhoneme(val),
				})
			}
		}
	}
	if opt.replacer == nil && len(opt.phonemeByDisplay) == 0 && len(opt.ssmlReplacements) == 0 {
		return nil
	}
	return opt
}

// Prepare applies hotword canonicalization and SSML phoneme wrapping for TTS.
func (o *TTSTextOptimizer) Prepare(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || o == nil {
		return text
	}
	if o.replacer != nil {
		text = o.replacer.Replace(text)
	}
	changed := false
	for display, phoneme := range o.phonemeByDisplay {
		if display == "" || phoneme == "" || !strings.Contains(text, display) {
			continue
		}
		text = strings.ReplaceAll(text, display, phoneme)
		changed = true
	}
	for _, r := range o.ssmlReplacements {
		if r.key == "" || !strings.Contains(text, r.key) {
			continue
		}
		text = strings.ReplaceAll(text, r.key, r.value)
		changed = true
	}
	if (changed || o.ssmlEnabled) && needsSSMLWrapper(text) {
		if !strings.HasPrefix(text, ssmlSpeakOpen) {
			text = ssmlSpeakOpen + text + ssmlSpeakClose
		}
	}
	return strings.TrimSpace(text)
}

func needsSSMLWrapper(text string) bool {
	return strings.Contains(text, "<phoneme") || strings.Contains(text, "<speak")
}

func convertToSSMLPhoneme(text string) string {
	return ssmlPhonemeRegex.ReplaceAllStringFunc(text, func(match string) string {
		matches := ssmlPhonemeRegex.FindStringSubmatch(match)
		if len(matches) == 3 {
			character := matches[1]
			py := matches[2]
			return fmt.Sprintf(`<phoneme alphabet="py" ph="%s">%s</phoneme>`, py, character)
		}
		return match
	})
}

// phonemeHintsFromHotword extracts display text and SSML phoneme tag from "{字|pin1}" syntax.
func phonemeHintsFromHotword(word string) (display, phoneme string) {
	word = strings.TrimSpace(word)
	if word == "" {
		return "", ""
	}
	if ssmlPhonemeRegex.MatchString(word) {
		display = ssmlPhonemeRegex.ReplaceAllString(word, "$1")
		phoneme = convertToSSMLPhoneme(word)
		return display, phoneme
	}
	return word, ""
}

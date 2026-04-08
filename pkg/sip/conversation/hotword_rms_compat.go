package conversation

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

// SIPHotwordCorrector applies optional phrase replacements on ASR incremental text.
// Backward-compatible fallback used by voice.go.
type SIPHotwordCorrector struct {
	logger   *zap.Logger
	replacer *strings.Replacer
}

// NewSIPHotwordCorrector builds a lightweight corrector from env.
//
// Supported env:
// - SIP_HOTWORD_CORRECTIONS_JSON: {"错词":"正词","錯詞":"正詞"}
// - SIP_HOTWORD_CORRECTIONS: "错词=正词,錯詞=正詞"
func NewSIPHotwordCorrector(lg *zap.Logger) *SIPHotwordCorrector {
	pairs := loadHotwordPairs()
	if len(pairs) == 0 {
		return &SIPHotwordCorrector{logger: lg}
	}
	rp := make([]string, 0, len(pairs)*2)
	for from, to := range pairs {
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" || to == "" || from == to {
			continue
		}
		rp = append(rp, from, to)
	}
	if len(rp) == 0 {
		return &SIPHotwordCorrector{logger: lg}
	}
	if lg != nil {
		lg.Info("sip hotword corrector enabled", zap.Int("pairs", len(rp)/2))
	}
	return &SIPHotwordCorrector{
		logger:   lg,
		replacer: strings.NewReplacer(rp...),
	}
}

func loadHotwordPairs() map[string]string {
	out := map[string]string{}

	if raw := strings.TrimSpace(utils.GetEnv("SIP_HOTWORD_CORRECTIONS_JSON")); raw != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err == nil {
			for k, v := range m {
				out[k] = v
			}
		}
	}

	// CSV-ish fallback: "foo=bar,baz=qux"
	if raw := strings.TrimSpace(utils.GetEnv("SIP_HOTWORD_CORRECTIONS")); raw != "" {
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

func (c *SIPHotwordCorrector) Correct(text string) string {
	t := strings.TrimSpace(text)
	if t == "" || c == nil || c.replacer == nil {
		return t
	}
	return c.replacer.Replace(t)
}

// rmsPCM16LE computes RMS of signed 16-bit little-endian PCM.
func rmsPCM16LE(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	n := len(pcm) / 2
	var sum float64
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(uint16(pcm[i]) | (uint16(pcm[i+1]) << 8))
		f := float64(v)
		sum += f * f
	}
	return math.Sqrt(sum / float64(n))
}

// Welcome prompt barge-in RMS threshold. Default lower than general VAD threshold
// to make "interrupt welcome prompt" more responsive.
func sipWelcomeBargeInThresholdFromEnv() float64 {
	s := strings.TrimSpace(utils.GetEnv("SIP_WELCOME_BARGE_IN_THRESHOLD"))
	if s == "" {
		return 1800.0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 1800.0
	}
	return f
}


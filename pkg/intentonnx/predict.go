package intentonnx

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
)

func buildPrediction(text string, rawLogits []float32, cfg *IntentConfig, opts RouteOptions) Prediction {
	p := Prediction{
		Text:   text,
		Logits: append([]float32(nil), rawLogits...),
	}
	adj := append([]float32(nil), rawLogits...)
	if cfg != nil && !opts.DisableKeywordBias {
		var hit bool
		adj, hit = applyKeywordLogitBonus(rawLogits, text, cfg, opts)
		p.KeywordBiasApplied = hit
		if hit {
			p.AdjustedLogits = append([]float32(nil), adj...)
		}
	}
	ix := argmaxFloat32(adj)
	p.IntentIndex = ix
	if ix < len(opts.LabelOverrides) {
		p.IntentName = opts.LabelOverrides[ix]
	}
	p.Softmax = softmaxFloat32(adj)
	if len(p.Softmax) > 0 && ix < len(p.Softmax) {
		p.Confidence = float64(p.Softmax[ix])
	}
	return p
}

func applyKeywordLogitBonus(raw []float32, text string, cfg *IntentConfig, opts RouteOptions) (adj []float32, anyHit bool) {
	adj = append([]float32(nil), raw...)
	if cfg == nil {
		return adj, false
	}
	base := cfg.KeywordLogitBonus
	suppressQueryKW := false
	if opts.VoiceASRHints {
		human := strings.Contains(text, "人工") || strings.Contains(text, "真人")
		helpOrQuery := strings.Contains(text, "帮") || strings.Contains(text, "查")
		if human && helpOrQuery {
			suppressQueryKW = true
		}
	}
	for i := range cfg.Intents {
		if suppressQueryKW && strings.TrimSpace(cfg.Intents[i].Name) == "查询" {
			continue
		}
		bonus := cfg.Intents[i].KeywordBonus
		if bonus <= 0 {
			bonus = base
		}
		if bonus <= 0 {
			continue
		}
		for _, kw := range cfg.Intents[i].Keywords {
			if kw == "" {
				continue
			}
			if strings.Contains(text, kw) {
				adj[i] += float32(bonus)
				anyHit = true
				break
			}
		}
	}
	return adj, anyHit
}

func finalizeRoute(pred *Prediction, cfg *IntentConfig, opts RouteOptions) (reply string, ch AnswerChannel) {
	if pred == nil || cfg == nil {
		return "", AnswerChannelLLM
	}
	if cfg.MinTopMargin > 0 && len(pred.Softmax) >= 2 && !pred.KeywordBiasApplied {
		top := pred.IntentIndex
		if top >= 0 && top < len(pred.Softmax) {
			sec := largestSoftmaxExcept(pred.Softmax, top)
			if float64(pred.Softmax[top]-sec) < cfg.MinTopMargin {
				pred.UsedConfigFallback = true
				if opts.UncertainMeansLLM {
					return "", AnswerChannelLLM
				}
				return cfg.DefaultReply, AnswerChannelIntent
			}
		}
	}
	top := pred.IntentIndex
	if top < 0 || top >= len(cfg.Intents) {
		pred.UsedConfigFallback = true
		if opts.UncertainMeansLLM {
			return "", AnswerChannelLLM
		}
		return cfg.DefaultReply, AnswerChannelIntent
	}
	if len(pred.Softmax) > top && float64(pred.Softmax[top]) < cfg.MinSoftmaxProb {
		pred.UsedConfigFallback = true
		if opts.UncertainMeansLLM {
			return "", AnswerChannelLLM
		}
		return cfg.DefaultReply, AnswerChannelIntent
	}
	ent := cfg.Intents[top]
	if pred.IntentName == "" {
		pred.IntentName = ent.Name
	}
	return pickIntentCannedReply(ent), AnswerChannelIntent
}

// pickIntentCannedReply returns Reply, or a random pick from Reply plus ReplyVariants (deduped).
func pickIntentCannedReply(ent IntentEntry) string {
	candidates := make([]string, 0, 1+len(ent.ReplyVariants))
	if s := strings.TrimSpace(ent.Reply); s != "" {
		candidates = append(candidates, s)
	}
	for _, v := range ent.ReplyVariants {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		dup := false
		for _, c := range candidates {
			if c == v {
				dup = true
				break
			}
		}
		if !dup {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		return strings.TrimSpace(ent.Reply)
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	return candidates[rand.IntN(len(candidates))]
}

func largestSoftmaxExcept(probs []float32, exclude int) float32 {
	var best float32 = -1
	for i, p := range probs {
		if i == exclude {
			continue
		}
		if p > best {
			best = p
		}
	}
	return best
}

func softmaxFloat32(logits []float32) []float32 {
	if len(logits) == 0 {
		return nil
	}
	mx := logits[0]
	for _, v := range logits[1:] {
		if v > mx {
			mx = v
		}
	}
	out := make([]float32, len(logits))
	var sum float64
	for i, v := range logits {
		out[i] = float32(math.Exp(float64(v - mx)))
		sum += float64(out[i])
	}
	if sum <= 0 {
		return out
	}
	inv := 1.0 / sum
	for i := range out {
		out[i] = float32(float64(out[i]) * inv)
	}
	return out
}

func argmaxFloat32(xs []float32) int {
	best := 0
	bestV := float32(math.Inf(-1))
	for i, v := range xs {
		if v > bestV {
			bestV = v
			best = i
		}
	}
	return best
}

// ValidateSingleChannel checks the mutual-exclusion contract for [RouteOutput].
func ValidateSingleChannel(r *RouteOutput) error {
	if r == nil {
		return fmt.Errorf("nil route output")
	}
	if r.Channel == AnswerChannelIntent && strings.TrimSpace(r.Reply) == "" {
		return fmt.Errorf("AnswerChannelIntent requires non-empty Reply")
	}
	if r.Channel == AnswerChannelLLM && r.Reply != "" {
		return fmt.Errorf("AnswerChannelLLM requires empty Reply (do not merge with intent text)")
	}
	return nil
}

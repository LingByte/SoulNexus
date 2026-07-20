package intentonnx

import (
	"strings"
)

// RouteByEmbedding matches user text to the nearest intent prototype.
func (e *EmbedEngine) RouteByEmbedding(text string, cfg *IntentConfig, store *EmbedPrototypeStore, opts RouteOptions) (*RouteOutput, error) {
	if e == nil || store == nil {
		return nil, errNilEmbedRoute
	}
	text = NormalizeTranscript(text)
	if text == "" {
		return nil, errEmptyText
	}
	if err := ValidateIntentConfigFlexible(cfg); err != nil {
		return nil, err
	}
	if len(store.Intents) != len(cfg.Intents) {
		return nil, errPrototypeIntentMismatch
	}

	vec, err := e.Embed(text)
	if err != nil {
		return nil, err
	}

	scores := make([]float32, len(store.Intents))
	for i, proto := range store.Intents {
		scores[i] = float32(cosineSimilarity(vec, proto.Vector))
	}

	pred := buildEmbedPrediction(text, scores, cfg, opts)
	reply, ch := finalizeEmbedRoute(&pred, cfg, opts)
	out := &RouteOutput{
		Channel:    ch,
		Reply:      reply,
		Prediction: pred,
	}
	if err := ValidateSingleChannel(out); err != nil {
		return nil, err
	}
	return out, nil
}

var (
	errNilEmbedRoute            = errString("intentonnx: nil embed route inputs")
	errEmptyText                = errString("intentonnx: empty text")
	errPrototypeIntentMismatch  = errString("intentonnx: prototypes count != intents count")
)

type errString string

func (e errString) Error() string { return string(e) }

func buildEmbedPrediction(text string, scores []float32, cfg *IntentConfig, opts RouteOptions) Prediction {
	adj := append([]float32(nil), scores...)
	var hit bool
	if cfg != nil && !opts.DisableKeywordBias {
		adj, hit = applyKeywordScoreBonus(scores, text, cfg, opts)
	}
	ix := argmaxFloat32(adj)
	p := Prediction{
		Text:               text,
		Logits:             append([]float32(nil), scores...),
		IntentIndex:        ix,
		KeywordBiasApplied: hit,
	}
	if hit {
		p.AdjustedLogits = append([]float32(nil), adj...)
	}
	if ix < len(adj) {
		p.Confidence = float64(adj[ix])
	}
	if ix < len(cfg.Intents) {
		p.IntentName = cfg.Intents[ix].Name
	}
	p.Softmax = append([]float32(nil), adj...)
	return p
}

func applyKeywordScoreBonus(raw []float32, text string, cfg *IntentConfig, opts RouteOptions) (adj []float32, anyHit bool) {
	adj = append([]float32(nil), raw...)
	if cfg == nil {
		return adj, false
	}
	base := cfg.KeywordLogitBonus
	var bonusF float32
	if base <= 0 {
		bonusF = 0.12
	} else {
		bonusF = float32(base / 30.0)
		if bonusF > 0.25 {
			bonusF = 0.25
		}
	}
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
		var add float32
		if bonus <= 0 {
			add = bonusF
		} else {
			add = float32(bonus / 30.0)
			if add > 0.25 {
				add = 0.25
			}
		}
		for _, kw := range cfg.Intents[i].Keywords {
			if kw == "" {
				continue
			}
			if strings.Contains(text, kw) {
				adj[i] += add
				anyHit = true
				break
			}
		}
	}
	return adj, anyHit
}

func finalizeEmbedRoute(pred *Prediction, cfg *IntentConfig, opts RouteOptions) (reply string, ch AnswerChannel) {
	if pred == nil || cfg == nil {
		return "", AnswerChannelLLM
	}
	minSim := cfg.MinSoftmaxProb
	if minSim <= 0 {
		minSim = 0.55
	}
	margin := cfg.MinTopMargin
	if margin <= 0 {
		margin = 0.08
	}

	scores := pred.Softmax
	if pred.KeywordBiasApplied && len(pred.AdjustedLogits) > 0 {
		scores = pred.AdjustedLogits
	}
	top := pred.IntentIndex
	if top < 0 || top >= len(cfg.Intents) || len(scores) <= top {
		pred.UsedConfigFallback = true
		if opts.UncertainMeansLLM {
			return "", AnswerChannelLLM
		}
		return cfg.DefaultReply, AnswerChannelIntent
	}
	if float64(scores[top]) < minSim {
		pred.UsedConfigFallback = true
		if opts.UncertainMeansLLM {
			return "", AnswerChannelLLM
		}
		return cfg.DefaultReply, AnswerChannelIntent
	}
	if len(scores) >= 2 && !pred.KeywordBiasApplied {
		sec := largestSoftmaxExcept(scores, top)
		if float64(scores[top]-sec) < margin {
			pred.UsedConfigFallback = true
			if opts.UncertainMeansLLM {
				return "", AnswerChannelLLM
			}
			return cfg.DefaultReply, AnswerChannelIntent
		}
	}
	ent := cfg.Intents[top]
	if pred.IntentName == "" {
		pred.IntentName = ent.Name
	}
	return pickIntentCannedReply(ent), AnswerChannelIntent
}

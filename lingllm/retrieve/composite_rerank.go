package retrieve

import "math"

// CompositeRerankScore blends rerank model score with the pre-rerank retrieval score.
// Base scores are min-max normalized to [0,1] within the candidate batch when bounds differ.
func CompositeRerankScore(modelScore, baseScore, baseMin, baseMax, modelWeight, baseWeight float64) float64 {
	modelW, baseW := effectiveCompositeWeights(modelWeight, baseWeight)
	normBase := normalizeBaseScore(baseScore, baseMin, baseMax)
	return modelW*modelScore + baseW*normBase
}

func effectiveCompositeWeights(modelW, baseW float64) (float64, float64) {
	if modelW <= 0 && baseW <= 0 {
		return 0.6, 0.3
	}
	if modelW <= 0 {
		modelW = 0.6
	}
	if baseW <= 0 {
		baseW = 0.3
	}
	return modelW, baseW
}

func normalizeBaseScore(score, minScore, maxScore float64) float64 {
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0
	}
	if maxScore <= minScore {
		return clamp01(score)
	}
	return clamp01((score - minScore) / (maxScore - minScore))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func baseScoreBounds(docs []*Document) (minScore, maxScore float64) {
	first := true
	for _, d := range docs {
		if d == nil {
			continue
		}
		if first {
			minScore, maxScore = d.Score, d.Score
			first = false
			continue
		}
		if d.Score < minScore {
			minScore = d.Score
		}
		if d.Score > maxScore {
			maxScore = d.Score
		}
	}
	return minScore, maxScore
}

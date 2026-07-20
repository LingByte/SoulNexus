package retrieve

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// RetrievalParams holds WeKnora-style retrieval tuning (RRF, thresholds, over-retrieval).
type RetrievalParams struct {
	EmbeddingTopK    int
	VectorThreshold  float64
	KeywordThreshold float64
	RerankTopK       int
	RerankThreshold  float64
	RRFK             int
	RRFVectorWeight  float64
	RRFKeywordWeight float64
	OverRetrieveMult int
	OverRetrieveMin  int
	OverRetrieveMax  int
	MMRLambda        float64
	EnableMMR        bool
	EnableDedup      bool
	OverlapThreshold float64
	// Composite rerank: final = modelWeight*modelScore + baseWeight*normalizedBaseScore
	EnableCompositeRerank bool
	RerankModelWeight     float64
	RerankBaseWeight      float64
}

// DefaultRetrievalParams returns sensible defaults aligned with WeKnora RetrievalConfig.
func DefaultRetrievalParams() RetrievalParams {
	return RetrievalParams{
		EmbeddingTopK:    50,
		VectorThreshold:  0.15,
		KeywordThreshold: 0,
		RerankTopK:       10,
		RerankThreshold:  0.2,
		RRFK:             60,
		RRFVectorWeight:  0.7,
		RRFKeywordWeight: 0.3,
		OverRetrieveMult: 5,
		OverRetrieveMin:  50,
		OverRetrieveMax:  500,
		MMRLambda:        0.7,
		EnableMMR:             true,
		EnableDedup:           true,
		OverlapThreshold:      0.85,
		EnableCompositeRerank: true,
		RerankModelWeight:     0.6,
		RerankBaseWeight:      0.3,
	}
}

func (p RetrievalParams) effectiveRRFK() int {
	if p.RRFK <= 0 {
		return 60
	}
	return p.RRFK
}

func (p RetrievalParams) effectiveRRFWeights() (vectorW, keywordW float64) {
	v, k := p.RRFVectorWeight, p.RRFKeywordWeight
	if v <= 0 && k <= 0 {
		return 0.7, 0.3
	}
	if v <= 0 {
		v = 0.7
	}
	if k <= 0 {
		k = 0.3
	}
	return v, k
}

// OverRetrieveK expands recall depth before rerank (WeKnora: max(topK*5, 50), cap 500).
func OverRetrieveK(topK int, p RetrievalParams) int {
	if topK <= 0 {
		topK = 10
	}
	mult := p.OverRetrieveMult
	if mult <= 0 {
		mult = 5
	}
	minK := p.OverRetrieveMin
	if minK <= 0 {
		minK = 50
	}
	maxK := p.OverRetrieveMax
	if maxK <= 0 {
		maxK = 500
	}
	k := topK * mult
	if k < minK {
		k = minK
	}
	if k > maxK {
		k = maxK
	}
	return k
}

func fuseRRF(vecDocs, kwDocs []*Document, p RetrievalParams) []*Document {
	if len(kwDocs) == 0 {
		return dedupeByID(vecDocs)
	}
	if len(vecDocs) == 0 {
		return dedupeByID(kwDocs)
	}

	rrfK := p.effectiveRRFK()
	vectorW, keywordW := p.effectiveRRFWeights()

	vectorRanks := rankMap(vecDocs)
	keywordRanks := rankMap(kwDocs)

	merged := make(map[string]*Document)
	for _, d := range vecDocs {
		if d == nil {
			continue
		}
		id := docKey(d)
		if cur, ok := merged[id]; !ok || d.Score > cur.Score {
			copy := *d
			merged[id] = &copy
		}
	}
	for _, d := range kwDocs {
		if d == nil {
			continue
		}
		id := docKey(d)
		if _, ok := merged[id]; !ok {
			copy := *d
			merged[id] = &copy
		}
	}

	out := make([]*Document, 0, len(merged))
	for id, doc := range merged {
		score := 0.0
		if rank, ok := vectorRanks[id]; ok {
			score += vectorW / float64(rrfK+rank)
		}
		if rank, ok := keywordRanks[id]; ok {
			score += keywordW / float64(rrfK+rank)
		}
		copy := *doc
		if copy.Metadata == nil {
			copy.Metadata = map[string]string{}
		}
		if rank, ok := vectorRanks[id]; ok {
			copy.Metadata["vector_rank"] = fmt.Sprintf("%d", rank)
			copy.Metadata["vector_rrf"] = fmt.Sprintf("%.6f", vectorW/float64(rrfK+rank))
		}
		if rank, ok := keywordRanks[id]; ok {
			copy.Metadata["keyword_rank"] = fmt.Sprintf("%d", rank)
			copy.Metadata["keyword_rrf"] = fmt.Sprintf("%.6f", keywordW/float64(rrfK+rank))
		}
		copy.Metadata["rrf_score"] = fmt.Sprintf("%.6f", score)
		copy.Score = score
		out = append(out, &copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func rankMap(docs []*Document) map[string]int {
	ranks := make(map[string]int, len(docs))
	for i, d := range docs {
		if d == nil {
			continue
		}
		id := docKey(d)
		if _, exists := ranks[id]; !exists {
			ranks[id] = i + 1
		}
	}
	return ranks
}

func dedupeByID(docs []*Document) []*Document {
	best := make(map[string]*Document)
	for _, d := range docs {
		if d == nil {
			continue
		}
		id := docKey(d)
		if cur, ok := best[id]; !ok || d.Score > cur.Score {
			copy := *d
			best[id] = &copy
		}
	}
	out := make([]*Document, 0, len(best))
	for _, d := range best {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func docKey(d *Document) string {
	if d.ID != "" {
		return d.ID
	}
	return d.Content
}

func applyThreshold(docs []*Document, minScore float64) []*Document {
	if minScore <= 0 || len(docs) == 0 {
		return docs
	}
	out := make([]*Document, 0, len(docs))
	for _, d := range docs {
		if d != nil && d.Score >= minScore {
			out = append(out, d)
		}
	}
	return out
}

// ApplyMMR selects diverse top-k documents using Maximal Marginal Relevance.
func ApplyMMR(docs []*Document, k int, lambda float64) []*Document {
	if k <= 0 || len(docs) == 0 {
		return nil
	}
	if lambda <= 0 {
		lambda = 0.7
	}
	if lambda > 1 {
		lambda = 1
	}

	tokenSets := make([]map[string]struct{}, len(docs))
	for i, d := range docs {
		tokenSets[i] = tokenizeSimple(CleanPassageForRerank(d.Content))
	}

	selected := make([]*Document, 0, k)
	selectedTokens := make([]map[string]struct{}, 0, k)
	used := make(map[int]struct{})

	for len(selected) < k && len(used) < len(docs) {
		bestIdx := -1
		bestScore := -1.0
		for i, d := range docs {
			if _, ok := used[i]; ok || d == nil {
				continue
			}
			relevance := d.Score
			redundancy := 0.0
			for _, sel := range selectedTokens {
				if sim := jaccard(tokenSets[i], sel); sim > redundancy {
					redundancy = sim
				}
			}
			mmr := lambda*relevance - (1-lambda)*redundancy
			if mmr > bestScore {
				bestScore = mmr
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			break
		}
		selected = append(selected, docs[bestIdx])
		selectedTokens = append(selectedTokens, tokenSets[bestIdx])
		used[bestIdx] = struct{}{}
	}
	return selected
}

var (
	reMarkdownLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	reMarkdownImg  = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	reRawURL       = regexp.MustCompile(`https?://[^\s)\]>]+`)
	reCodeBlock    = regexp.MustCompile("(?s)```.*?```")
	reHeading      = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reWhitespace   = regexp.MustCompile(`\s+`)
)

// CleanPassageForRerank strips markdown noise before sending text to a rerank model.
func CleanPassageForRerank(text string) string {
	text = reCodeBlock.ReplaceAllString(text, " ")
	text = reMarkdownImg.ReplaceAllString(text, " ")
	text = reMarkdownLink.ReplaceAllString(text, "$1")
	text = reRawURL.ReplaceAllString(text, " ")
	text = reHeading.ReplaceAllString(text, "")
	text = reWhitespace.ReplaceAllString(strings.TrimSpace(text), " ")
	if utf8.RuneCountInString(text) > 8000 {
		runes := []rune(text)
		text = string(runes[:8000])
	}
	return text
}

func tokenizeSimple(text string) map[string]struct{} {
	return tokenizeForDedup(text)
}

func tokenizeForDedup(text string) map[string]struct{} {
	text = strings.ToLower(CleanPassageForRerank(text))
	parts := strings.FieldsFunc(text, func(r rune) bool {
		if unicode.Is(unicode.Han, r) {
			return false
		}
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r < 128
	})
	set := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) < 2 {
			continue
		}
		set[p] = struct{}{}
	}
	if len(set) == 0 {
		runes := []rune(text)
		for i := 0; i+1 < len(runes); i++ {
			if unicode.Is(unicode.Han, runes[i]) || unicode.Is(unicode.Han, runes[i+1]) {
				bg := string(runes[i : i+2])
				set[bg] = struct{}{}
			}
		}
	}
	return set
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

package knowledge

import "strings"

var (
	onEnhanceQuery func(callID, rawQuery string, cfg SearchConfig) string
	onFilterHits   func(hits []Hit, minScore float64) []Hit
	onReuseHits    func(callID string, cfg SearchConfig) ([]Hit, bool)
	onRecordUser   func(callID, text string)
	onRecordHits   func(callID string, hits []Hit)
)

// SetQueryEnhancer installs memory-based query rewrite.
func SetQueryEnhancer(fn func(callID, rawQuery string, cfg SearchConfig) string) {
	onEnhanceQuery = fn
}

// SetHitFilter installs min-score filtering.
func SetHitFilter(fn func(hits []Hit, minScore float64) []Hit) {
	onFilterHits = fn
}

// SetHitReuse installs previous-round slice reuse.
func SetHitReuse(fn func(callID string, cfg SearchConfig) ([]Hit, bool)) {
	onReuseHits = fn
}

// SetUserUtteranceRecorder stores user text for query memory.
func SetUserUtteranceRecorder(fn func(callID, text string)) {
	onRecordUser = fn
}

// SetSearchHitRecorder stores retrieval hits for memory / quote validation.
func SetSearchHitRecorder(fn func(callID string, hits []Hit)) {
	onRecordHits = fn
}

func enhanceQuery(callID, raw string, cfg SearchConfig) string {
	if onEnhanceQuery != nil {
		return onEnhanceQuery(callID, raw, cfg)
	}
	return strings.TrimSpace(raw)
}

func filterHits(hits []Hit, minScore float64) []Hit {
	if onFilterHits != nil {
		return onFilterHits(hits, minScore)
	}
	if minScore <= 0 {
		return hits
	}
	out := make([]Hit, 0, len(hits))
	for _, h := range hits {
		if h.Score >= minScore {
			out = append(out, h)
		}
	}
	return out
}

func reuseHits(callID string, cfg SearchConfig) ([]Hit, bool) {
	if onReuseHits != nil {
		return onReuseHits(callID, cfg)
	}
	return nil, false
}

func recordUserUtterance(callID, text string) {
	if onRecordUser != nil {
		onRecordUser(callID, text)
	}
}

func recordSearchHits(callID string, hits []Hit) {
	StoreLastHits(callID, hits)
	if onRecordHits != nil {
		onRecordHits(callID, hits)
	}
}

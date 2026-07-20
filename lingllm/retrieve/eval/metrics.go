package eval

import (
	"math"
	"sort"
	"strings"
)

// QueryMetrics holds per-query retrieval scores.
type QueryMetrics struct {
	Query        string             `json:"query"`
	Namespace    string             `json:"namespace,omitempty"`
	RelevantIDs  []string           `json:"relevant_ids"`
	RetrievedIDs []string           `json:"retrieved_ids"`
	RecallAtK    map[int]float64    `json:"recall_at_k"`
	PrecisionAtK map[int]float64    `json:"precision_at_k"`
	NDCGAtK      map[int]float64    `json:"ndcg_at_k"`
	MRR          float64            `json:"mrr"`
	MAP          float64            `json:"map"`
	HitsAtK      map[int]int        `json:"hits_at_k,omitempty"`
	LatencyMs    int64              `json:"latency_ms"`
	FirstHitRank int                `json:"first_hit_rank"` // 0 if none
}

// Report aggregates metrics across all evaluated queries.
type Report struct {
	Queries         int                `json:"queries"`
	Strategy        string             `json:"strategy,omitempty"`
	Namespace       string             `json:"namespace,omitempty"`
	TopK            int                `json:"top_k"`
	RecallAtK       map[int]float64    `json:"recall_at_k"`
	PrecisionAtK    map[int]float64    `json:"precision_at_k"`
	NDCGAtK         map[int]float64    `json:"ndcg_at_k"`
	MRR             float64            `json:"mrr"`
	MAP             float64            `json:"map"`
	LatencyMeanMs   float64            `json:"latency_mean_ms"`
	LatencyP50Ms    float64            `json:"latency_p50_ms"`
	LatencyP95Ms    float64            `json:"latency_p95_ms"`
	EmptyResultRate float64            `json:"empty_result_rate"`
	PerQuery        []QueryMetrics     `json:"per_query,omitempty"`
}

func toSet(ids []string) map[string]struct{} {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	return set
}

// ComputeQueryMetrics calculates IR metrics for one ranked retrieval result.
func ComputeQueryMetrics(relevantIDs, retrievedIDs []string, kValues []int, ndcgK int) QueryMetrics {
	relevant := toSet(relevantIDs)
	if ndcgK <= 0 {
		ndcgK = 10
	}
	if len(kValues) == 0 {
		kValues = []int{1, 3, 5, 10}
	}

	qm := QueryMetrics{
		RelevantIDs:  append([]string(nil), relevantIDs...),
		RetrievedIDs: append([]string(nil), retrievedIDs...),
		RecallAtK:    make(map[int]float64, len(kValues)),
		PrecisionAtK: make(map[int]float64, len(kValues)),
		NDCGAtK:      make(map[int]float64, len(kValues)),
		HitsAtK:      make(map[int]int, len(kValues)),
	}

	for _, k := range kValues {
		hits := hitsInTopK(retrievedIDs, relevant, k)
		qm.HitsAtK[k] = hits
		qm.RecallAtK[k] = recallAtK(hits, len(relevant))
		qm.PrecisionAtK[k] = precisionAtK(hits, k)
	}

	for _, k := range uniqueSorted(append(kValues, ndcgK)) {
		qm.NDCGAtK[k] = ndcgAtK(retrievedIDs, relevant, k)
	}

	qm.MRR = mrr(retrievedIDs, relevant)
	qm.MAP = averagePrecision(retrievedIDs, relevant)
	qm.FirstHitRank = firstHitRank(retrievedIDs, relevant)
	return qm
}

func hitsInTopK(retrieved []string, relevant map[string]struct{}, k int) int {
	if k <= 0 || len(retrieved) == 0 || len(relevant) == 0 {
		return 0
	}
	limit := k
	if limit > len(retrieved) {
		limit = len(retrieved)
	}
	hits := 0
	for i := 0; i < limit; i++ {
		if _, ok := relevant[strings.TrimSpace(retrieved[i])]; ok {
			hits++
		}
	}
	return hits
}

func recallAtK(hits, relevantCount int) float64 {
	if relevantCount <= 0 {
		return 0
	}
	return float64(hits) / float64(relevantCount)
}

func precisionAtK(hits, k int) float64 {
	if k <= 0 {
		return 0
	}
	return float64(hits) / float64(k)
}

func firstHitRank(retrieved []string, relevant map[string]struct{}) int {
	for i, id := range retrieved {
		if _, ok := relevant[strings.TrimSpace(id)]; ok {
			return i + 1
		}
	}
	return 0
}

func mrr(retrieved []string, relevant map[string]struct{}) float64 {
	rank := firstHitRank(retrieved, relevant)
	if rank == 0 {
		return 0
	}
	return 1.0 / float64(rank)
}

func averagePrecision(retrieved []string, relevant map[string]struct{}) float64 {
	if len(relevant) == 0 {
		return 0
	}
	hitCount := 0
	var sumPrecision float64
	for i, id := range retrieved {
		if _, ok := relevant[strings.TrimSpace(id)]; ok {
			hitCount++
			sumPrecision += float64(hitCount) / float64(i+1)
		}
	}
	if hitCount == 0 {
		return 0
	}
	return sumPrecision / float64(len(relevant))
}

func ndcgAtK(retrieved []string, relevant map[string]struct{}, k int) float64 {
	if k <= 0 || len(relevant) == 0 {
		return 0
	}
	limit := k
	if limit > len(retrieved) {
		limit = len(retrieved)
	}
	top := retrieved[:limit]

	var dcg float64
	for i, id := range top {
		rel := 0.0
		if _, ok := relevant[strings.TrimSpace(id)]; ok {
			rel = 1
		}
		dcg += rel / math.Log2(float64(i+2))
	}

	idealLen := len(relevant)
	if idealLen > k {
		idealLen = k
	}
	var idcg float64
	for i := 0; i < idealLen; i++ {
		idcg += 1.0 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

func aggregateReports(perQuery []QueryMetrics, kValues []int, ndcgK int, meta Report) Report {
	if len(kValues) == 0 {
		kValues = []int{1, 3, 5, 10}
	}
	if ndcgK <= 0 {
		ndcgK = 10
	}
	allK := uniqueSorted(append(kValues, ndcgK))

	out := Report{
		Queries:      len(perQuery),
		Strategy:     meta.Strategy,
		Namespace:    meta.Namespace,
		TopK:         meta.TopK,
		RecallAtK:    make(map[int]float64, len(kValues)),
		PrecisionAtK: make(map[int]float64, len(kValues)),
		NDCGAtK:      make(map[int]float64, len(allK)),
		PerQuery:     perQuery,
	}

	var mrrSum, mapSum float64
	var empty int
	latencies := make([]float64, 0, len(perQuery))

	for _, q := range perQuery {
		mrrSum += q.MRR
		mapSum += q.MAP
		if len(q.RetrievedIDs) == 0 {
			empty++
		}
		latencies = append(latencies, float64(q.LatencyMs))
	}

	for _, k := range kValues {
		var recallSum, precSum float64
		for _, q := range perQuery {
			recallSum += q.RecallAtK[k]
			precSum += q.PrecisionAtK[k]
		}
		n := float64(len(perQuery))
		if n > 0 {
			out.RecallAtK[k] = recallSum / n
			out.PrecisionAtK[k] = precSum / n
		}
	}
	for _, k := range allK {
		var ndcgSum float64
		for _, q := range perQuery {
			ndcgSum += q.NDCGAtK[k]
		}
		if len(perQuery) > 0 {
			out.NDCGAtK[k] = ndcgSum / float64(len(perQuery))
		}
	}

	n := float64(len(perQuery))
	if n > 0 {
		out.MRR = mrrSum / n
		out.MAP = mapSum / n
		out.EmptyResultRate = float64(empty) / n
	}
	out.LatencyMeanMs, out.LatencyP50Ms, out.LatencyP95Ms = latencyStats(latencies)
	return out
}

func latencyStats(values []float64) (mean, p50, p95 float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	sort.Float64s(values)
	p50 = percentile(values, 0.50)
	p95 = percentile(values, 0.95)
	return mean, p50, p95
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func uniqueSorted(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, v := range values {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Ints(out)
	return out
}

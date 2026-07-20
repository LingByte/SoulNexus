package eval

import (
	"context"
	"strings"
	"testing"

	"github.com/LingByte/lingllm/retrieve"
)

func TestComputeQueryMetrics_PerfectRecall(t *testing.T) {
	qm := ComputeQueryMetrics(
		[]string{"a", "b"},
		[]string{"a", "b", "c"},
		[]int{1, 3, 5},
		3,
	)
	if qm.RecallAtK[5] != 1.0 {
		t.Fatalf("recall@5=%v", qm.RecallAtK[5])
	}
	if qm.MRR != 1.0 {
		t.Fatalf("mrr=%v", qm.MRR)
	}
	if qm.FirstHitRank != 1 {
		t.Fatalf("first hit rank=%d", qm.FirstHitRank)
	}
}

func TestComputeQueryMetrics_PartialRecall(t *testing.T) {
	qm := ComputeQueryMetrics(
		[]string{"a", "b", "c"},
		[]string{"x", "a", "y"},
		[]int{3},
		3,
	)
	if qm.RecallAtK[3] != 1.0/3.0 {
		t.Fatalf("recall@3=%v", qm.RecallAtK[3])
	}
	if qm.PrecisionAtK[3] != 1.0/3.0 {
		t.Fatalf("precision@3=%v", qm.PrecisionAtK[3])
	}
	if qm.MRR != 0.5 {
		t.Fatalf("mrr=%v", qm.MRR)
	}
}

func TestComputeQueryMetrics_NoHits(t *testing.T) {
	qm := ComputeQueryMetrics(
		[]string{"a"},
		[]string{"x", "y"},
		[]int{5},
		5,
	)
	if qm.RecallAtK[5] != 0 || qm.MRR != 0 || qm.MAP != 0 {
		t.Fatalf("expected zero metrics: %+v", qm)
	}
}

func TestLoadJSONL(t *testing.T) {
	input := `
# comment
{"query":"hello","relevant_ids":["id1"]}

{"query":"world","relevant_ids":["id2","id3"]}
`
	samples, err := LoadJSONL(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 || samples[0].Query != "hello" {
		t.Fatalf("samples=%+v", samples)
	}
}

func TestNormalizeSamples(t *testing.T) {
	out := NormalizeSamples([]Sample{{Query: "q", RelevantIDs: []string{"a"}}}, "demo")
	if out[0].Namespace != "demo" {
		t.Fatalf("namespace=%q", out[0].Namespace)
	}
}

type stubEvalRetriever struct {
	byQuery map[string][]string
}

func (s stubEvalRetriever) Retrieve(_ context.Context, query string, topK int) ([]*retrieve.Document, error) {
	ids := s.byQuery[query]
	if topK > 0 && len(ids) > topK {
		ids = ids[:topK]
	}
	out := make([]*retrieve.Document, 0, len(ids))
	for _, id := range ids {
		out = append(out, &retrieve.Document{ID: id, Content: id, Score: 1})
	}
	return out, nil
}

func TestRun_Aggregates(t *testing.T) {
	ret := stubEvalRetriever{byQuery: map[string][]string{
		"q1": {"a", "b", "c"},
		"q2": {"x", "a"},
	}}
	report, err := Run(t.Context(), ret, []Sample{
		{Query: "q1", RelevantIDs: []string{"a", "b"}},
		{Query: "q2", RelevantIDs: []string{"a"}},
	}, Options{TopK: 3, KValues: []int{3}, IncludePerQuery: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Queries != 2 {
		t.Fatalf("queries=%d", report.Queries)
	}
	if report.RecallAtK[3] != 1.0 {
		t.Fatalf("recall@3=%v", report.RecallAtK[3])
	}
	if report.MRR != 0.75 {
		t.Fatalf("mrr=%v", report.MRR)
	}
}

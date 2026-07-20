package eval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/lingllm/retrieve"
)

// Retriever runs ranked retrieval for evaluation.
type Retriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]*retrieve.Document, error)
}

// Options configures an evaluation run.
type Options struct {
	TopK      int
	KValues   []int
	NDCGK     int
	Strategy  string
	Namespace string
	MinScore  float64
	IncludePerQuery bool
}

// DefaultOptions returns common K cutoffs for RAG chunk evaluation.
func DefaultOptions() Options {
	return Options{
		TopK:    10,
		KValues: []int{1, 3, 5, 10},
		NDCGK:   10,
	}
}

// Run evaluates retrieval quality on labeled samples.
func Run(ctx context.Context, retriever Retriever, samples []Sample, opts Options) (Report, error) {
	if retriever == nil {
		return Report{}, fmt.Errorf("eval: nil retriever")
	}
	if len(samples) == 0 {
		return Report{}, fmt.Errorf("eval: no samples")
	}
	if opts.TopK <= 0 {
		opts.TopK = DefaultOptions().TopK
	}
	if len(opts.KValues) == 0 {
		opts.KValues = DefaultOptions().KValues
	}
	if opts.NDCGK <= 0 {
		opts.NDCGK = DefaultOptions().NDCGK
	}
	maxK := opts.TopK
	for _, k := range opts.KValues {
		if k > maxK {
			maxK = k
		}
	}
	if opts.NDCGK > maxK {
		maxK = opts.NDCGK
	}
	fetchK := maxK

	perQuery := make([]QueryMetrics, 0, len(samples))
	for _, sample := range samples {
		if err := ctx.Err(); err != nil {
			return Report{}, err
		}
		start := time.Now()
		docs, err := retriever.Retrieve(ctx, sample.Query, fetchK)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			return Report{}, fmt.Errorf("eval: query %q: %w", sample.Query, err)
		}
		retrievedIDs := docIDs(docs)
		qm := ComputeQueryMetrics(sample.RelevantIDs, retrievedIDs, opts.KValues, opts.NDCGK)
		qm.Query = sample.Query
		qm.Namespace = sample.Namespace
		qm.LatencyMs = latency
		perQuery = append(perQuery, qm)
	}

	report := aggregateReports(perQuery, opts.KValues, opts.NDCGK, Report{
		Strategy:  opts.Strategy,
		Namespace: opts.Namespace,
		TopK:      fetchK,
	})
	if !opts.IncludePerQuery {
		report.PerQuery = nil
	} else {
		report.PerQuery = perQuery
	}
	return report, nil
}

func docIDs(docs []*retrieve.Document) []string {
	out := make([]string, 0, len(docs))
	for _, d := range docs {
		if d == nil {
			continue
		}
		id := strings.TrimSpace(d.ID)
		if id == "" {
			continue
		}
		out = append(out, id)
	}
	return out
}

// StrategyRetriever adapts retrieve.StrategyRetriever to eval.Retriever.
type StrategyRetriever struct {
	Inner *retrieve.StrategyRetriever
}

func (s StrategyRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*retrieve.Document, error) {
	if s.Inner == nil {
		return nil, fmt.Errorf("eval: nil strategy retriever")
	}
	return s.Inner.Retrieve(ctx, query, topK)
}

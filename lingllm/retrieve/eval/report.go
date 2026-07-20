package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// FormatReportText renders a human-readable summary.
func FormatReportText(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Knowledge retrieval evaluation\n")
	if r.Strategy != "" {
		fmt.Fprintf(&b, "  strategy:  %s\n", r.Strategy)
	}
	if r.Namespace != "" {
		fmt.Fprintf(&b, "  namespace: %s\n", r.Namespace)
	}
	fmt.Fprintf(&b, "  queries:   %d\n", r.Queries)
	fmt.Fprintf(&b, "  top_k:     %d\n\n", r.TopK)

	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Metric\tValue")
	for _, k := range sortedKeys(r.RecallAtK) {
		fmt.Fprintf(w, "Recall@%d\t%.4f\n", k, r.RecallAtK[k])
	}
	for _, k := range sortedKeys(r.PrecisionAtK) {
		fmt.Fprintf(w, "Precision@%d\t%.4f\n", k, r.PrecisionAtK[k])
	}
	for _, k := range sortedKeys(r.NDCGAtK) {
		fmt.Fprintf(w, "NDCG@%d\t%.4f\n", k, r.NDCGAtK[k])
	}
	fmt.Fprintf(w, "MRR\t%.4f\n", r.MRR)
	fmt.Fprintf(w, "MAP\t%.4f\n", r.MAP)
	fmt.Fprintf(w, "Empty result rate\t%.4f\n", r.EmptyResultRate)
	fmt.Fprintf(w, "Latency mean (ms)\t%.1f\n", r.LatencyMeanMs)
	fmt.Fprintf(w, "Latency p50 (ms)\t%.1f\n", r.LatencyP50Ms)
	fmt.Fprintf(w, "Latency p95 (ms)\t%.1f\n", r.LatencyP95Ms)
	_ = w.Flush()
	return b.String()
}

// FormatComparisonText compares multiple strategy reports side by side.
func FormatComparisonText(reports []Report) string {
	if len(reports) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Strategy comparison (%d queries)\n\n", reports[0].Queries)
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	header := "Metric"
	for _, r := range reports {
		name := r.Strategy
		if name == "" {
			name = "?"
		}
		header += "\t" + name
	}
	fmt.Fprintln(w, header)

	kSet := map[int]struct{}{}
	for _, r := range reports {
		for k := range r.RecallAtK {
			kSet[k] = struct{}{}
		}
	}
	for _, k := range sortedKeySet(kSet) {
		row := fmt.Sprintf("Recall@%d", k)
		for _, r := range reports {
			row += fmt.Sprintf("\t%.4f", r.RecallAtK[k])
		}
		fmt.Fprintln(w, row)
	}
	for _, k := range sortedKeySet(kSet) {
		row := fmt.Sprintf("Precision@%d", k)
		for _, r := range reports {
			row += fmt.Sprintf("\t%.4f", r.PrecisionAtK[k])
		}
		fmt.Fprintln(w, row)
	}
	row := "MRR"
	for _, r := range reports {
		row += fmt.Sprintf("\t%.4f", r.MRR)
	}
	fmt.Fprintln(w, row)
	row = "MAP"
	for _, r := range reports {
		row += fmt.Sprintf("\t%.4f", r.MAP)
	}
	fmt.Fprintln(w, row)
	row = "Latency p50 (ms)"
	for _, r := range reports {
		row += fmt.Sprintf("\t%.1f", r.LatencyP50Ms)
	}
	fmt.Fprintln(w, row)
	_ = w.Flush()
	return b.String()
}

// WriteReportJSON encodes the report as JSON.
func WriteReportJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func sortedKeys(m map[int]float64) []int {
	set := make(map[int]struct{}, len(m))
	for k := range m {
		set[k] = struct{}{}
	}
	return sortedKeySet(set)
}

func sortedKeySet(set map[int]struct{}) []int {
	out := make([]int, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

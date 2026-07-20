package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/LingByte/lingllm/embedder"
)

var namespaceSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// sanitizeNamespace converts a namespace/collection name into a safe identifier suffix.
func sanitizeNamespace(namespace string) string {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return ""
	}
	ns = namespaceSanitizer.ReplaceAllString(ns, "_")
	ns = strings.Trim(ns, "_")
	if ns == "" {
		return "default"
	}
	return strings.ToLower(ns)
}

func metadataJSON(meta map[string]any) string {
	if len(meta) == 0 {
		return "{}"
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func parseMetadataJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	return meta
}

func tagsJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(tags)
	return string(b)
}

func parseTagsJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}

func recordTimestamps(r *Record, now time.Time) {
	if r == nil {
		return
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}
}

// fillMissingVectors embeds content for records without vectors and returns the vector dimension.
func fillMissingVectors(ctx context.Context, emb embedder.Embedder, records []Record) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}
	if emb == nil {
		return 0, ErrHandlerNotFound
	}

	vectorDim := 0
	for i := range records {
		if len(records[i].Vector) > 0 {
			vectorDim = len(records[i].Vector)
			break
		}
	}

	var needIdx []int
	var inputs []string
	for i := range records {
		if len(records[i].Vector) > 0 {
			if vectorDim > 0 && len(records[i].Vector) != vectorDim {
				return 0, ErrInvalidVectorDimension
			}
			continue
		}
		needIdx = append(needIdx, i)
		inputs = append(inputs, records[i].Content)
	}
	if len(needIdx) == 0 {
		if vectorDim <= 0 {
			return 0, ErrInvalidVectorDimension
		}
		return vectorDim, nil
	}

	vecs, err := emb.Embed(ctx, inputs)
	if err != nil {
		return 0, err
	}
	if len(vecs) != len(needIdx) {
		return 0, fmt.Errorf("embedder_vector_mismatch: want=%d got=%d", len(needIdx), len(vecs))
	}
	for k, idx := range needIdx {
		if len(vecs[k]) == 0 {
			return 0, ErrInvalidVectorDimension
		}
		if vectorDim == 0 {
			vectorDim = len(vecs[k])
		}
		if len(vecs[k]) != vectorDim {
			return 0, ErrInvalidVectorDimension
		}
		tmp := make([]float32, vectorDim)
		for j := range tmp {
			tmp[j] = float32(vecs[k][j])
		}
		records[idx].Vector = tmp
	}
	return vectorDim, nil
}

func float64sTo32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i := range v {
		out[i] = float32(v[i])
	}
	return out
}

func float32sTo64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i := range v {
		out[i] = float64(v[i])
	}
	return out
}

func filterQueryResults(results []QueryResult, minScore float64) []QueryResult {
	if minScore <= 0 || len(results) == 0 {
		return results
	}
	out := make([]QueryResult, 0, len(results))
	for _, r := range results {
		if r.Score >= minScore {
			out = append(out, r)
		}
	}
	return out
}

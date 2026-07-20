package intentonnx

import (
	"fmt"
	"math"
	"strings"
)

func meanPoolNormalize(hidden []float32, seqLen, dim int, mask []int) ([]float32, error) {
	if seqLen <= 0 || dim <= 0 {
		return nil, fmt.Errorf("intentonnx: invalid pool dims seq=%d dim=%d", seqLen, dim)
	}
	if len(hidden) < seqLen*dim {
		return nil, fmt.Errorf("intentonnx: hidden size %d < seq*dim %d", len(hidden), seqLen*dim)
	}
	out := make([]float32, dim)
	var denom float32
	for t := 0; t < seqLen; t++ {
		if len(mask) > t && mask[t] == 0 {
			continue
		}
		denom++
		off := t * dim
		for d := 0; d < dim; d++ {
			out[d] += hidden[off+d]
		}
	}
	if denom <= 0 {
		denom = float32(seqLen)
		for t := 0; t < seqLen; t++ {
			off := t * dim
			for d := 0; d < dim; d++ {
				out[d] += hidden[off+d]
			}
		}
	}
	inv := 1 / denom
	for d := range out {
		out[d] *= inv
	}
	return l2Normalize(out), nil
}

func l2Normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum <= 0 {
		return v
	}
	inv := float32(1 / math.Sqrt(sum))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x * inv
	}
	return out
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot
}

func averageVectors(vecs [][]float32) ([]float32, error) {
	if len(vecs) == 0 {
		return nil, fmt.Errorf("intentonnx: no vectors to average")
	}
	dim := len(vecs[0])
	if dim == 0 {
		return nil, fmt.Errorf("intentonnx: empty vector")
	}
	sum := make([]float32, dim)
	for _, v := range vecs {
		if len(v) != dim {
			return nil, fmt.Errorf("intentonnx: vector dim mismatch")
		}
		for i := range v {
			sum[i] += v[i]
		}
	}
	inv := 1 / float32(len(vecs))
	for i := range sum {
		sum[i] *= inv
	}
	return l2Normalize(sum), nil
}

func intentPhrases(name, reply string, keywords, samples []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 1+len(keywords)+len(samples))
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(name)
	for _, s := range keywords {
		add(s)
	}
	for _, s := range samples {
		add(s)
	}
	if len(out) == 0 && strings.TrimSpace(reply) != "" {
		add(reply)
	}
	return out
}

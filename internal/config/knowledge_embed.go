// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	lingembedder "github.com/LingByte/lingllm/embedder"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

const (
	defaultEmbedMaxInputChars = 12000
	defaultEmbedBatchSize     = 16
)

// EmbedMaxInputChars caps each embedding request body (EMBED_MAX_INPUT_CHARS).
func EmbedMaxInputChars() int {
	raw := strings.TrimSpace(utils.GetEnv("EMBED_MAX_INPUT_CHARS"))
	if raw == "" {
		return defaultEmbedMaxInputChars
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultEmbedMaxInputChars
	}
	return n
}

// EmbedBatchSize is how many texts to send per Embed call (EMBED_BATCH_SIZE).
func EmbedBatchSize() int {
	raw := strings.TrimSpace(utils.GetEnv("EMBED_BATCH_SIZE"))
	if raw == "" {
		return defaultEmbedBatchSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultEmbedBatchSize
	}
	return n
}

// ClipEmbedInput shortens a single string before embedding (by runes, safe for UTF-8).
func ClipEmbedInput(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	if maxChars <= 0 || len(text) <= maxChars {
		return text
	}
	r := []rune(text)
	if len(r) <= maxChars {
		return text
	}
	return string(r[:maxChars])
}

// EmbedTextsBatched calls the embedder in batches; each input is clipped to EmbedMaxInputChars.
func EmbedTextsBatched(ctx context.Context, emb lingembedder.Embedder, inputs []string) ([][]float32, error) {
	if emb == nil {
		return nil, fmt.Errorf("embedder is nil")
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}
	maxChars := EmbedMaxInputChars()
	batchSize := EmbedBatchSize()
	out := make([][]float32, 0, len(inputs))
	for start := 0; start < len(inputs); start += batchSize {
		end := start + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		batch := make([]string, end-start)
		for i, in := range inputs[start:end] {
			batch[i] = ClipEmbedInput(in, maxChars)
		}
		vecs, err := emb.Embed(ctx, batch)
		if err != nil {
			return nil, err
		}
		if len(vecs) != len(batch) {
			return nil, fmt.Errorf("embedder returned %d vectors for %d inputs", len(vecs), len(batch))
		}
		out = append(out, vecs...)
	}
	return out, nil
}

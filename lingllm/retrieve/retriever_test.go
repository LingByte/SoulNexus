package retrieve

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// vecStub implements VectorRetriever for testing.
type vecStub struct {
	docs []*Document
}

func (v vecStub) Retrieve(_ context.Context, _ string, topK int) ([]*Document, error) {
	if topK <= 0 || topK >= len(v.docs) {
		return v.docs, nil
	}
	return v.docs[:topK], nil
}

// memSearch implements SearchEngine for testing.
type memSearch struct {
	docs map[string]map[string]interface{}
}

func (m *memSearch) Search(_ context.Context, query string, fields []string, size int) ([]SearchHit, error) {
	var hits []SearchHit

	for id, doc := range m.docs {
		found := false
		for _, field := range fields {
			content, _ := doc[field].(string)
			if query == "" || containsFold(content, query) || containsFold(id, query) {
				found = true
				break
			}
		}
		if found {
			hits = append(hits, SearchHit{ID: id, Score: 1.0, Fields: doc})
		}
	}

	if len(hits) > size && size > 0 {
		hits = hits[:size]
	}

	return hits, nil
}

func containsFold(s, sub string) bool {
	return len(sub) > 0 && (s == sub || len(s) >= len(sub) && indexFold(s, sub) >= 0)
}

func indexFold(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// stubReranker implements Reranker for testing.
type stubReranker struct{}

func (stubReranker) Rerank(_ context.Context, _ string, documents []string, topN int) ([]RerankResult, error) {
	results := make([]RerankResult, len(documents))
	for i, doc := range documents {
		results[i] = RerankResult{Index: i, Score: float64(len(doc))}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if topN <= 0 || topN > len(results) {
		topN = len(results)
	}
	return results[:topN], nil
}

func TestStrategyRetriever_Vector(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyVector,
		Vector: vecStub{docs: []*Document{
			{ID: "a", Content: "machine learning", Score: 0.9},
			{ID: "b", Content: "deep learning", Score: 0.8},
			{ID: "c", Content: "neural networks", Score: 0.7},
		}},
		TopK: 2,
	})

	assert.Nil(t, err)
	assert.NotNil(t, sr)

	out, err := sr.Retrieve(context.Background(), "learning", 2)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(out))
	assert.Equal(t, "a", out[0].ID)
}

func TestStrategyRetriever_Keyword(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyKeyword,
		Search: &memSearch{docs: map[string]map[string]interface{}{
			"x": {"content": "mailbox channel", "title": "Messaging"},
			"y": {"content": "voice streaming", "title": "Audio"},
		}},
		TopK: 1,
	})

	assert.Nil(t, err)
	assert.NotNil(t, sr)

	out, err := sr.Retrieve(context.Background(), "mailbox", 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(out))
	assert.Equal(t, "x", out[0].ID)
}

func TestStrategyRetriever_Hybrid(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyHybrid,
		Vector: vecStub{docs: []*Document{
			{ID: "a", Content: "pregel bsp", Score: 0.9},
			{ID: "b", Content: "a2a jsonrpc", Score: 0.2},
		}},
		Search: &memSearch{docs: map[string]map[string]interface{}{
			"b": {"content": "a2a jsonrpc streaming"},
			"c": {"content": "voice asr tts"},
		}},
		TopK: 2,
	})

	assert.Nil(t, err)
	assert.NotNil(t, sr)

	out, err := sr.Retrieve(context.Background(), "a2a", 2)
	assert.Nil(t, err)
	assert.True(t, len(out) > 0)
}

func TestStrategyRetriever_Rerank(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyVector,
		Vector: vecStub{docs: []*Document{
			{ID: "short", Content: "a2a", Score: 0.1},
			{ID: "long", Content: "pregel mailbox streaming channel", Score: 0.9},
		}},
		TopK:             1,
		Reranker:         stubReranker{},
		RerankCandidates: 2,
	})

	assert.Nil(t, err)
	assert.NotNil(t, sr)

	out, err := sr.Retrieve(context.Background(), "pregel", 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(out))
	assert.Equal(t, "long", out[0].ID)
}

func TestStrategyRetriever_InvalidStrategy(t *testing.T) {
	_, err := New(Config{
		Strategy: Strategy("invalid"),
	})

	assert.NotNil(t, err)
}

func TestStrategyRetriever_VectorRequired(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyVector,
		Vector:   nil,
	})

	assert.NotNil(t, err)
}

func TestStrategyRetriever_SearchRequired(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyKeyword,
		Search:   nil,
	})

	assert.NotNil(t, err)
}

func TestStrategyRetriever_HybridRequired(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyHybrid,
		Vector:   vecStub{},
		Search:   nil,
	})

	assert.NotNil(t, err)
}

func TestStrategyRetriever_DefaultValues(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyVector,
		Vector:   vecStub{},
	})

	assert.Nil(t, err)
	assert.Equal(t, 3, sr.cfg.TopK)
	assert.Equal(t, 0.65, sr.cfg.VectorWeight)
	assert.Equal(t, 3, len(sr.cfg.KeywordFields))
}

func TestStrategyRetriever_MinScore(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyKeyword,
		Search: &memSearch{docs: map[string]map[string]interface{}{
			"x": {"content": "test"},
		}},
		TopK:     1,
		MinScore: 2.0,
	})

	assert.Nil(t, err)

	out, err := sr.Retrieve(context.Background(), "test", 1)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(out))
}

func TestFieldString(t *testing.T) {
	tests := []struct {
		name     string
		fields   map[string]interface{}
		key      string
		expected string
	}{
		{"string", map[string]interface{}{"title": "test"}, "title", "test"},
		{"string with space", map[string]interface{}{"title": "  test  "}, "title", "test"},
		{"string array", map[string]interface{}{"tags": []string{"a", "b"}}, "tags", "a"},
		{"missing key", map[string]interface{}{"title": "test"}, "missing", ""},
		{"nil value", map[string]interface{}{"title": nil}, "title", ""},
		{"nil fields", nil, "title", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldString(tt.fields, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

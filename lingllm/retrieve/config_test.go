package retrieve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNew_DefaultStrategy(t *testing.T) {
	sr, err := New(Config{
		Vector: vecStub{},
	})

	assert.Nil(t, err)
	assert.Equal(t, StrategyVector, sr.cfg.Strategy)
}

func TestNew_DefaultTopK(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyVector,
		Vector:   vecStub{},
	})

	assert.Nil(t, err)
	assert.Equal(t, 3, sr.cfg.TopK)
}

func TestNew_DefaultVectorWeight(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyHybrid,
		Vector:   vecStub{},
		Search:   &memSearch{},
	})

	assert.Nil(t, err)
	assert.Equal(t, 0.65, sr.cfg.VectorWeight)
}

func TestNew_DefaultKeywordFields(t *testing.T) {
	sr, err := New(Config{
		Strategy: StrategyKeyword,
		Search:   &memSearch{},
	})

	assert.Nil(t, err)
	assert.Equal(t, 3, len(sr.cfg.KeywordFields))
	assert.Contains(t, sr.cfg.KeywordFields, "title")
	assert.Contains(t, sr.cfg.KeywordFields, "content")
	assert.Contains(t, sr.cfg.KeywordFields, "body")
}

func TestNew_CustomValues(t *testing.T) {
	sr, err := New(Config{
		Strategy:      StrategyVector,
		Vector:        vecStub{},
		TopK:          10,
		VectorWeight:  0.8,
		KeywordFields: []string{"custom"},
	})

	assert.Nil(t, err)
	assert.Equal(t, 10, sr.cfg.TopK)
	assert.Equal(t, 0.8, sr.cfg.VectorWeight)
	assert.Equal(t, 1, len(sr.cfg.KeywordFields))
	assert.Equal(t, "custom", sr.cfg.KeywordFields[0])
}

func TestNew_VectorStrategyNoVector(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyVector,
	})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "vector retriever required")
}

func TestNew_KeywordStrategyNoSearch(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyKeyword,
	})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "search engine required")
}

func TestNew_HybridStrategyNoVector(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyHybrid,
		Search:   &memSearch{},
	})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "hybrid")
}

func TestNew_HybridStrategyNoSearch(t *testing.T) {
	_, err := New(Config{
		Strategy: StrategyHybrid,
		Vector:   vecStub{},
	})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "hybrid")
}

func TestNew_UnsupportedStrategy(t *testing.T) {
	_, err := New(Config{
		Strategy: Strategy("unsupported"),
	})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unsupported strategy")
}

func TestNew_VectorWeightClamping(t *testing.T) {
	sr, err := New(Config{
		Strategy:     StrategyHybrid,
		Vector:       vecStub{},
		Search:       &memSearch{},
		VectorWeight: 2.0,
	})

	assert.Nil(t, err)
	assert.Equal(t, 2.0, sr.cfg.VectorWeight)
}

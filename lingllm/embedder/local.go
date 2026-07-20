package embedder

import (
	"context"
	"crypto/md5"
	"hash"
	"io"
	"math"
	"strings"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// LocalEmbedder 本地 embedding 实现
type LocalEmbedder struct {
	model     string
	dimension int
	hasher    hash.Hash
}

// NewLocalEmbedder 创建本地 embedder
func NewLocalEmbedder(cfg *Config) *LocalEmbedder {
	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 384
	}

	return &LocalEmbedder{
		model:     cfg.Model,
		dimension: dimension,
		hasher:    md5.New(),
	}
}

func (e *LocalEmbedder) Name() string {
	return "local"
}

func (e *LocalEmbedder) Provider() string {
	return "local"
}

func (e *LocalEmbedder) Dimension() int {
	return e.dimension
}

func (e *LocalEmbedder) Close() error {
	return nil
}

// Embed 批量向量化（基于哈希的确定性向量）
func (e *LocalEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	vectors := make([][]float32, 0, len(texts))

	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			text = " "
		}

		// 基于文本内容生成确定性向量
		vector := e.hashToVector(text)
		vectors = append(vectors, vector)
	}

	return vectors, nil
}

// EmbedSingle 单个文本向量化
func (e *LocalEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyInput
	}

	return e.hashToVector(text), nil
}

// hashToVector 将文本哈希转换为向量
func (e *LocalEmbedder) hashToVector(text string) []float32 {
	// 计算 MD5 哈希
	h := md5.New()
	io.WriteString(h, text)
	hash := h.Sum(nil)

	// 将哈希字节转换为向量
	vector := make([]float32, e.dimension)
	for i := 0; i < e.dimension; i++ {
		// 使用哈希字节循环填充向量
		byteIdx := i % len(hash)
		// 将字节转换为 [-1, 1] 范围的浮点数
		vector[i] = (float32(hash[byteIdx]) - 128.0) / 128.0
	}

	// L2 归一化向量
	var norm float32
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}

	return vector
}

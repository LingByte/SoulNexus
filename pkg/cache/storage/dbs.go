package storage

import "github.com/LingByte/SoulNexus/pkg/cache/constants"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type CacheDb struct {
	id      int
	keys    map[string]CacheObject
	expires map[string]int64
}

func NewCacheDb(id int) *CacheDb {
	return &CacheDb{
		id:      id,
		keys:    make(map[string]CacheObject),
		expires: make(map[string]int64),
	}
}

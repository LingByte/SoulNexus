package cache

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"sync"
	"time"
)

var (
	globalCache Cache
	globalOnce  sync.Once
	globalMu    sync.RWMutex
)

// InitGlobalCache 初始化全局缓存实例
func InitGlobalCache(config Config) error {
	var err error
	globalOnce.Do(func() {
		globalMu.Lock()
		defer globalMu.Unlock()

		globalCache, err = NewCache(config)
		if err != nil {
			return
		}
	})
	return err
}

// InitGlobalCacheWithOptions 使用选项初始化全局缓存实例
func InitGlobalCacheWithOptions(config Config, options *Options) error {
	var err error
	globalOnce.Do(func() {
		globalMu.Lock()
		defer globalMu.Unlock()

		globalCache, err = NewCacheWithOptions(config, options)
		if err != nil {
			return
		}
	})
	return err
}

// GetGlobalCache 获取全局缓存实例
// 如果未初始化，返回一个默认的本地缓存实例
func GetGlobalCache() Cache {
	globalMu.RLock()
	if globalCache != nil {
		globalMu.RUnlock()
		return globalCache
	}
	globalMu.RUnlock()

	// 双重检查，如果未初始化，创建一个默认的本地缓存
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalCache == nil {
		globalCache = NewLocalCache(LocalConfig{
			MaxSize:           1000,
			DefaultExpiration: 5 * time.Minute,
			CleanupInterval:   10 * time.Minute,
		})
	}
	return globalCache
}

// SetGlobalCache 设置全局缓存实例（主要用于测试）
func SetGlobalCache(c Cache) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalCache = c
}

// CloseGlobalCache 关闭全局缓存连接
func CloseGlobalCache() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalCache != nil {
		err := globalCache.Close()
		globalCache = nil
		return err
	}
	return nil
}

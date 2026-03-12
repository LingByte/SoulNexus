package media

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

type LocalMediaCache struct {
	Disabled  bool
	CacheRoot string
}

var _defaultMediaCache *LocalMediaCache

func MediaCache() *LocalMediaCache {
	if _defaultMediaCache == nil {
		rootVal, ok := os.LookupEnv("MEDIA_CACHE_ROOT")
		if !ok {
			rootVal = "/tmp"
		}
		disableVal, ok := os.LookupEnv("MEDIA_CACHE_DISABLED")
		var disable bool
		if ok {
			disable, _ = strconv.ParseBool(disableVal)
		}
		_defaultMediaCache = &LocalMediaCache{
			Disabled:  disable,
			CacheRoot: rootVal,
		}
		if !disable {
			if _, err := os.Stat(rootVal); err != nil {
				os.MkdirAll(rootVal, 0755)
			}
			logger.Info("mediacache: initialized", zap.String("root", rootVal))
		}
	}
	return _defaultMediaCache
}

func (c *LocalMediaCache) BuildKey(params ...string) string {
	md5hash := md5.New()
	for _, p := range params {
		md5hash.Write([]byte(p))
	}
	digest := md5hash.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

func (c *LocalMediaCache) Store(key string, data []byte) error {
	if c.Disabled {
		return nil
	}
	filename := filepath.Join(c.CacheRoot, key)
	if st, err := os.Stat(filename); err == nil {
		if st.IsDir() {
			return os.ErrExist
		}
	}
	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		logger.Error("mediacache: failed to write file", zap.String("filename", filename), zap.Error(err))
		return err
	}
	logger.Info("mediacache: stored", zap.String("filename", filename), zap.Int("datasize", len(data)))
	return nil
}

func (c *LocalMediaCache) Get(key string) ([]byte, error) {
	if c.Disabled {
		return nil, os.ErrNotExist
	}
	filename := filepath.Join(c.CacheRoot, key)
	if st, err := os.Stat(filename); err == nil {
		if st.IsDir() {
			return nil, os.ErrNotExist
		}
	} else {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		logger.Error("mediacache: failed to read file", zap.String("filename", filename), zap.Error(err))
		return nil, err
	}
	return data, nil
}

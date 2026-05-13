package stores

import (
	"io"
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

const (
	KindLocal = "local"
	KindCos   = "cos"   // tencent
	KindMinio = "minio" // minio/s3 compatible
	KindQiNiu = "qiniu"
	KindOss   = "oss"
	KinsS3    = "s3"
)

var ErrInvalidPath = &utils.Error{Code: http.StatusBadRequest, Message: "invalid path"}

var DefaultStoreKind = getDefaultStoreKind()

// getDefaultStoreKind
func getDefaultStoreKind() string {
	kind := utils.GetEnv("STORAGE_KIND")
	if kind == "" {
		return KindLocal
	}
	switch kind {
	case KindLocal, KindCos, KindMinio, KindQiNiu, KindOss, KinsS3:
		return kind
	default:
		// 无效的类型，使用默认值并记录警告
		return KindLocal
	}
}

// Store Common Storage Modules
type Store interface {
	Read(key string) (io.ReadCloser, int64, error)

	Write(key string, r io.Reader) error

	Delete(key string) error

	Exists(key string) (bool, error)

	PublicURL(key string) string
}

func GetStore(kind string) Store {
	switch kind {
	case KindCos:
		return NewCosStore()
	case KindMinio:
		return NewMinioStore()
	case KindQiNiu:
		return NewQiNiuStore()
	case KindOss:
		return NewOSSStore()
	case KinsS3:
		return NewS3Store()
	default:
		return NewLocalStore()
	}
}

func Default() Store {
	return GetStore(DefaultStoreKind)
}

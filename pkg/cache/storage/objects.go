package storage

import (
	"github.com/LingByte/SoulNexus/pkg/cache/constants"
	"github.com/LingByte/SoulNexus/pkg/cache/structure"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

type ObjectType byte

const (
	OBJ_STRING ObjectType = 0
	OBJ_LIST   ObjectType = 1
	OBJ_SET    ObjectType = 2
	OBJ_ZSET   ObjectType = 3
	OBJ_HASH   ObjectType = 4
)

type CacheObject struct {
	Type     ObjectType
	Encoding structure.Encoding
	Ptr      interface{}
	RefCount int
}

// NewStringObject 创建字符串对象
func NewStringObject(value []byte) *CacheObject {
	sds := structure.NewSDSFromBytes(value)
	return &CacheObject{
		Type:     OBJ_STRING,
		Encoding: structure.OBJ_ENCODING_RAW,
		Ptr:      sds,
		RefCount: 1,
	}
}

func (obj *CacheObject) TypeString() string {
	switch obj.Type {
	case OBJ_STRING:
		return constants.OBJ_TYPE_STRING
	case OBJ_LIST:
		return constants.OBJ_TYPE_LIST
	case OBJ_SET:
		return constants.OBJ_TYPE_SET
	case OBJ_ZSET:
		return constants.OBJ_TYPE_ZSET
	case OBJ_HASH:
		return constants.OBJ_TYPE_HASH
	default:
		return constants.OBJ_TYPE_UNKNOWN
	}
}

// EncodingString 返回编码方式的字符串表示
func (obj *CacheObject) EncodingString() string {
	switch obj.Encoding {
	case structure.OBJ_ENCODING_RAW:
		return constants.OBJ_ENCODING_TYPE_RAW
	case structure.OBJ_ENCODING_INT:
		return constants.OBJ_ENCODING_TYPE_INT
	case structure.OBJ_ENCODING_HT:
		return constants.OBJ_ENCODING_TYPE_HT
	case structure.OBJ_ENCODING_INTSET:
		return constants.OBJ_ENCODING_TYPE_INTSET
	case structure.OBJ_ENCODING_SKIPLIST:
		return constants.OBJ_ENCODING_TYPE_SKIPLIST
	case structure.OBJ_ENCODING_QUICKLIST:
		return constants.OBJ_ENCODING_TYPE_QUICKLIST
	case structure.OBJ_ENCODING_LISTPACK:
		return constants.OBJ_ENCODING_TYPE_LISTPACK
	default:
		return constants.OBJ_ENCODING_TYPE_UNKNOWN
	}
}

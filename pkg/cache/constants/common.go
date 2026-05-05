package constants

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

const (
	OBJ_TYPE_STRING  = "string"
	OBJ_TYPE_LIST    = "list"
	OBJ_TYPE_SET     = "set"
	OBJ_TYPE_ZSET    = "zset"
	OBJ_TYPE_HASH    = "hash"
	OBJ_TYPE_UNKNOWN = "unknown"
)

const (
	OBJ_ENCODING_TYPE_RAW       = "raw"
	OBJ_ENCODING_TYPE_INT       = "int"
	OBJ_ENCODING_TYPE_HT        = "hashtable"
	OBJ_ENCODING_TYPE_INTSET    = "intset"
	OBJ_ENCODING_TYPE_SKIPLIST  = "skiplist"
	OBJ_ENCODING_TYPE_QUICKLIST = "quicklist"
	OBJ_ENCODING_TYPE_LISTPACK  = "listpack"
	OBJ_ENCODING_TYPE_UNKNOWN   = "unknown"
)

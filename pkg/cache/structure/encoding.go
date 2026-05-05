package structure

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

/*
 * Cache 对象编码类型定义
 * 使用不同的编码方式来表示同一种数据类型，以优化内存和性能。
 */

// Encoding 通用编码类型
type Encoding byte

const (
	OBJ_ENCODING_RAW        Encoding = 0  // 原始字符串（SDS）
	OBJ_ENCODING_INT        Encoding = 1  // 整数
	OBJ_ENCODING_HT         Encoding = 2  // 哈希表（dict）
	OBJ_ENCODING_ZIPMAP     Encoding = 3  // zipmap（已废弃）
	OBJ_ENCODING_LINKEDLIST Encoding = 4  // 双向链表（已废弃）
	OBJ_ENCODING_ZIPLIST    Encoding = 5  // ziplist（已废弃，被 listpack 替代）
	OBJ_ENCODING_INTSET     Encoding = 6  // 整数集合
	OBJ_ENCODING_SKIPLIST   Encoding = 7  // 跳表
	OBJ_ENCODING_EMBSTR     Encoding = 8  // 嵌入式字符串
	OBJ_ENCODING_QUICKLIST  Encoding = 9  // 快速列表
	OBJ_ENCODING_STREAM     Encoding = 10 // 流
	OBJ_ENCODING_LISTPACK   Encoding = 11 // 列表包
)

// ListEncoding List 编码类型（别名）
type ListEncoding = Encoding

// SetEncoding Set 编码类型（别名）
type SetEncoding = Encoding

// ZSetEncoding ZSet 编码类型（别名）
type ZSetEncoding = Encoding

// HashEncoding Hash 编码类型（别名）
type HashEncoding = Encoding

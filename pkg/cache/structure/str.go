package structure

import (
	"bytes"
	"unsafe"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

/*
 * Redis String 数据结构 - SDS简单动态字符串
 *
 * 【内存布局】
 * ┌─────────────────────────────────────────┐
 * │         SDS 内存布局                     │
 * ├─────────────────────────────────────────┤
 * │  Header (sdshdr8/16/32/64)             │
 * │  ├─ len: 已使用长度                     │
 * │  ├─ alloc: 总容量（不包括 header 和 \0）│
 * │  └─ flags: 类型标志（低3位）            │
 * ├─────────────────────────────────────────┤
 * │  buf[]: 实际字符串数据                  │
 * │  └─ 以 \0 结尾        │
 * └─────────────────────────────────────────┘

 * 【核心原理】
 * SDS 是 Redis 自己实现的字符串类型，相比 C 语言的 char* 有以下优势：
 *
 * 1. O(1) 时间复杂度获取字符串长度
 *    - C 字符串需要 O(n) 遍历到 \0 才能知道长度
 *    - SDS 在 header 中存储了 len 字段，直接返回即可
 *
 * 2. 二进制安全
 *    - C 字符串不能存储包含 \0 的二进制数据（会被误判为字符串结束）
 *    - SDS 使用 len 字段记录长度，可以存储任意二进制数据
 *
 * 3. 预分配机制（减少内存重分配）
 *    - 空间预分配：当 SDS 需要扩展时，不仅分配所需空间，还会额外分配
 *      - 如果 len < 1MB，分配 len * 2 的空间（预留 100%）
 *      - 如果 len >= 1MB，每次多分配 1MB
 *    - 惰性空间释放：缩短字符串时不立即释放内存，而是保留在 alloc 中
 *
 * 4. 多种 header 类型优化内存
 *    - sdshdr5: 已废弃（flags 高 5 位存长度，最多 31 字节）
 *    - sdshdr8: len/alloc 各 1 字节，适合 <= 255 字节的字符串
 *    - sdshdr16: len/alloc 各 2 字节，适合 <= 65535 字节的字符串
 *    - sdshdr32: len/alloc 各 4 字节，适合 <= 4GB 的字符串
 *    - sdshdr64: len/alloc 各 8 字节，适合超大字符串
 */

type SDSType byte

const (
	SDS_TYPE_5  SDSType = 0 // 已废弃但保留
	SDS_TYPE_8  SDSType = 1
	SDS_TYPE_16 SDSType = 2
	SDS_TYPE_32 SDSType = 3
	SDS_TYPE_64 SDSType = 4
)

const (
	SDS_TYPE_MASK    = 7           // 0b111，用于提取类型
	SDS_TYPE_BITS    = 3           // 类型占用的位数
	SDS_MAX_PREALLOC = 1024 * 1024 // 1MB，超过此大小每次多分配 1MB
)

// SDS 指针类型（指向 buf 的起始位置）
type SDS *byte

// sdshdr8 结构体（适合 <= 255 字节的字符串）
type sdshdr8 struct {
	len   uint8 // 已使用长度
	alloc uint8 // 总容量（不包括 header 和 \0）
	flags byte  // SDSType 在内存中就是 byte
}

// sdshdr16 结构体（适合 <= 65535 字节的字符串）
type sdshdr16 struct {
	len   uint16
	alloc uint16
	flags byte
}

// sdshdr32 结构体（适合 <= 4GB 的字符串）
type sdshdr32 struct {
	len   uint32
	alloc uint32
	flags byte
}

// sdshdr64 结构体（适合超大字符串）
type sdshdr64 struct {
	len   uint64
	alloc uint64
	flags byte
}

// 头部地址 = buf地址 - 头部大小
// 内存地址低 →→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→→ 内存地址高
//
//[ sdshdr16 头部 ]  [ buf 数据区 ]
//    ↑                 ↑
//    |                 |
//  你要找的地址      SDS 变量存的地址

// sdsType 获取 SDS 类型（通过 s[-1] 访问 flags）
func sdsType(s SDS) SDSType {
	flags := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) - 1))
	return SDSType(flags & SDS_TYPE_MASK)
}

// sdsHdr8 获取 sdshdr8 的头部指针
func sdsHdr8(s SDS) *sdshdr8 {
	return (*sdshdr8)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) - unsafe.Sizeof(sdshdr8{})))
}

// sdsHdr16 获取 sdshdr16 的头部指针
func sdsHdr16(s SDS) *sdshdr16 {
	return (*sdshdr16)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) - unsafe.Sizeof(sdshdr16{})))
}

// sdsHdr32 获取 sdshdr32 的头部指针
func sdsHdr32(s SDS) *sdshdr32 {
	return (*sdshdr32)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) - unsafe.Sizeof(sdshdr32{})))
}

// sdsHdr64 获取 sdshdr64 的头部指针
func sdsHdr64(s SDS) *sdshdr64 {
	return (*sdshdr64)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) - unsafe.Sizeof(sdshdr64{})))
}

// sdsLen 获取字符串长度 - O(1) 时间复杂度
func sdsLen(s SDS) uint64 {
	switch sdsType(s) {
	case SDS_TYPE_8:
		return uint64(sdsHdr8(s).len)
	case SDS_TYPE_16:
		return uint64(sdsHdr16(s).len)
	case SDS_TYPE_32:
		return uint64(sdsHdr32(s).len)
	case SDS_TYPE_64:
		return sdsHdr64(s).len
	}
	return 0
}

// sdsAvail 获取可用空间
func sdsAvail(s SDS) uint64 {
	switch sdsType(s) {
	case SDS_TYPE_8:
		hdr := sdsHdr8(s)
		return uint64(hdr.alloc - hdr.len)
	case SDS_TYPE_16:
		hdr := sdsHdr16(s)
		return uint64(hdr.alloc - hdr.len)
	case SDS_TYPE_32:
		hdr := sdsHdr32(s)
		return uint64(hdr.alloc - hdr.len)
	case SDS_TYPE_64:
		hdr := sdsHdr64(s)
		return hdr.alloc - hdr.len
	}
	return 0
}

// sdsAlloc 获取总分配空间
func sdsAlloc(s SDS) uint64 {
	switch sdsType(s) {
	case SDS_TYPE_8:
		return uint64(sdsHdr8(s).alloc)
	case SDS_TYPE_16:
		return uint64(sdsHdr16(s).alloc)
	case SDS_TYPE_32:
		return uint64(sdsHdr32(s).alloc)
	case SDS_TYPE_64:
		return sdsHdr64(s).alloc
	}
	return 0
}

// sdsReqType 根据字符串长度选择合适的 SDS 类型
func sdsReqType(size uint64) SDSType {
	if size < 32 {
		return SDS_TYPE_8
	} else if size <= 255 {
		return SDS_TYPE_8
	} else if size <= 65535 {
		return SDS_TYPE_16
	} else if size <= 4294967295 {
		return SDS_TYPE_32
	} else {
		return SDS_TYPE_64
	}
}

// sdsHdrSize 获取 header 大小
func sdsHdrSize(t SDSType) uintptr {
	switch t {
	case SDS_TYPE_5:
		return unsafe.Sizeof(struct {
			flags byte
		}{})
	case SDS_TYPE_8:
		return unsafe.Sizeof(sdshdr8{})
	case SDS_TYPE_16:
		return unsafe.Sizeof(sdshdr16{})
	case SDS_TYPE_32:
		return unsafe.Sizeof(sdshdr32{})
	case SDS_TYPE_64:
		return unsafe.Sizeof(sdshdr64{})
	}
	return 0
}

// NewSDSFromBytes 从字节数组创建 SDS
func NewSDSFromBytes(init []byte) SDS {
	initlen := uint64(len(init))
	if initlen == 0 {
		return NewSDSEmpty()
	}

	sdsType := sdsReqType(initlen)
	hdrSize := sdsHdrSize(sdsType)

	// 分配内存：头部 + 字符串 + \0
	totalSize := hdrSize + uintptr(initlen) + 1
	buf := make([]byte, totalSize)

	// 设置头部
	switch sdsType {
	case SDS_TYPE_8:
		hdr := (*sdshdr8)(unsafe.Pointer(&buf[0]))
		hdr.len = uint8(initlen)
		hdr.alloc = uint8(initlen)
		hdr.flags = byte(sdsType)
		copy(buf[hdrSize:], init)
		buf[totalSize-1] = 0 // \0 结尾
		return (*byte)(unsafe.Pointer(&buf[hdrSize]))
	case SDS_TYPE_16:
		hdr := (*sdshdr16)(unsafe.Pointer(&buf[0]))
		hdr.len = uint16(initlen)
		hdr.alloc = uint16(initlen)
		hdr.flags = byte(sdsType)
		copy(buf[hdrSize:], init)
		buf[totalSize-1] = 0
		return (*byte)(unsafe.Pointer(&buf[hdrSize]))
	case SDS_TYPE_32:
		hdr := (*sdshdr32)(unsafe.Pointer(&buf[0]))
		hdr.len = uint32(initlen)
		hdr.alloc = uint32(initlen)
		hdr.flags = byte(sdsType)
		copy(buf[hdrSize:], init)
		buf[totalSize-1] = 0
		return (*byte)(unsafe.Pointer(&buf[hdrSize]))
	case SDS_TYPE_64:
		hdr := (*sdshdr64)(unsafe.Pointer(&buf[0]))
		hdr.len = initlen
		hdr.alloc = initlen
		hdr.flags = byte(sdsType)
		copy(buf[hdrSize:], init)
		buf[totalSize-1] = 0
		return (*byte)(unsafe.Pointer(&buf[hdrSize]))
	}

	return nil
}

// NewSDSEmpty 创建空的 SDS
func NewSDSEmpty() SDS {
	// 直接创建空的 SDS，避免递归调用
	sdsType := SDS_TYPE_8
	hdrSize := sdsHdrSize(sdsType)

	// 分配内存：头部 + \0
	totalSize := hdrSize + 1
	buf := make([]byte, totalSize)

	// 设置头部
	hdr := (*sdshdr8)(unsafe.Pointer(&buf[0]))
	hdr.len = 0
	hdr.alloc = 0
	hdr.flags = byte(sdsType)
	buf[totalSize-1] = 0 // \0 结尾

	return (*byte)(unsafe.Pointer(&buf[hdrSize]))
}

// SDSCat 连接字符串到 SDS（实现空间预分配）
func SDSCat(s SDS, t string) SDS {
	return SDSCatLen(s, []byte(t), uint64(len(t)))
}

// SDSCatLen 连接指定长度的字节数组到 SDS
func SDSCatLen(s SDS, t []byte, addlen uint64) SDS {
	curlen := sdsLen(s)

	// 检查是否需要扩展空间
	s = sdsMakeRoomFor(s, addlen)
	if s == nil {
		return nil
	}

	// 复制数据
	buf := sdsBuf(s)
	copy(buf[curlen:], t)

	// 更新长度
	sdsSetLen(s, curlen+addlen)
	buf[curlen+addlen] = 0 // 设置新的结束符

	return s
}

// sdsMakeRoomFor 为 SDS 分配更多空间（实现预分配策略）
func sdsMakeRoomFor(s SDS, addlen uint64) SDS {
	avail := sdsAvail(s)

	if avail >= addlen {
		return s
	}

	curlen := sdsLen(s)
	newlen := curlen + addlen
	newtype := sdsReqType(newlen)
	oldtype := sdsType(s)

	// 如果类型不变且新长度在 alloc 范围内，只需要扩展 alloc
	if newtype == oldtype {
		hdrSize := sdsHdrSize(oldtype)
		oldHdrPtr := uintptr(unsafe.Pointer(s)) - hdrSize
		oldTotalSize := hdrSize + uintptr(curlen) + 1

		// 计算新的 alloc（预分配策略）
		var newalloc uint64
		if newlen < SDS_MAX_PREALLOC {
			newalloc = newlen * 2 // 小于 1MB，分配 2 倍
		} else {
			newalloc = newlen + SDS_MAX_PREALLOC // 大于 1MB，多分配 1MB
		}
		newTotalSize := hdrSize + uintptr(newalloc) + 1
		newBuf := make([]byte, newTotalSize)
		copy(newBuf, (*[1 << 30]byte)(unsafe.Pointer(oldHdrPtr))[:oldTotalSize])
		switch oldtype {
		case SDS_TYPE_8:
			hdr := (*sdshdr8)(unsafe.Pointer(&newBuf[0]))
			hdr.alloc = uint8(newalloc)
		case SDS_TYPE_16:
			hdr := (*sdshdr16)(unsafe.Pointer(&newBuf[0]))
			hdr.alloc = uint16(newalloc)
		case SDS_TYPE_32:
			hdr := (*sdshdr32)(unsafe.Pointer(&newBuf[0]))
			hdr.alloc = uint32(newalloc)
		case SDS_TYPE_64:
			hdr := (*sdshdr64)(unsafe.Pointer(&newBuf[0]))
			hdr.alloc = newalloc
		}

		return &newBuf[hdrSize]
	}
	news := NewSDSFromBytes(SdsBytes(s))
	news = SDSCatLen(news, SdsBytes(s), curlen)
	return SDSCatLen(news, nil, addlen)
}

// sdsSetLen 设置 SDS 长度
func sdsSetLen(s SDS, newlen uint64) {
	switch sdsType(s) {
	case SDS_TYPE_8:
		hdr := sdsHdr8(s)
		if newlen <= 255 {
			hdr.len = uint8(newlen)
		}
	case SDS_TYPE_16:
		hdr := sdsHdr16(s)
		if newlen <= 65535 {
			hdr.len = uint16(newlen)
		}
	case SDS_TYPE_32:
		hdr := sdsHdr32(s)
		hdr.len = uint32(newlen)
	case SDS_TYPE_64:
		hdr := sdsHdr64(s)
		hdr.len = newlen
	}
}

// sdsBuf 获取 SDS 的 buf 指针（转换为 []byte）
func sdsBuf(s SDS) []byte {
	len := sdsLen(s)
	if len == 0 {
		return []byte{}
	}
	return (*[1 << 30]byte)(unsafe.Pointer(s))[:len:len]
}

// SdsBytes 获取 SDS 的字节数组（复制）（导出函数）
func SdsBytes(s SDS) []byte {
	buf := sdsBuf(s)
	result := make([]byte, len(buf))
	copy(result, buf)
	return result
}

// SDSCmp 比较两个 SDS 字符串
func SDSCmp(s1, s2 SDS) int {
	l1 := sdsLen(s1)
	l2 := sdsLen(s2)
	minlen := l1
	if l2 < minlen {
		minlen = l2
	}

	buf1 := sdsBuf(s1)
	buf2 := sdsBuf(s2)
	cmp := bytes.Compare(buf1[:minlen], buf2[:minlen])
	if cmp != 0 {
		return cmp
	}

	if l1 < l2 {
		return -1
	} else if l1 > l2 {
		return 1
	}
	return 0
}

// SDSTrim 去除 SDS 两端指定的字符
func SDSTrim(s SDS, cutset string) SDS {
	buf := sdsBuf(s)
	cutsetBytes := []byte(cutset)

	// 找到左边界
	start := 0
	for start < len(buf) {
		found := false
		for _, c := range cutsetBytes {
			if buf[start] == c {
				found = true
				break
			}
		}
		if !found {
			break
		}
		start++
	}

	// 找到右边界
	end := len(buf) - 1
	for end >= start {
		found := false
		for _, c := range cutsetBytes {
			if buf[end] == c {
				found = true
				break
			}
		}
		if !found {
			break
		}
		end--
	}

	// 创建新的 SDS（惰性空间释放：这里简化实现，实际可以原地修改）
	if start > 0 || end < len(buf)-1 {
		newBuf := buf[start : end+1]
		return NewSDSFromBytes(newBuf)
	}

	return s
}

// SDSClear 清空 SDS（惰性释放：只设置 len=0，不释放内存）
func SDSClear(s SDS) {
	sdsSetLen(s, 0)
	if s != nil {
		buf := sdsBuf(s)
		if len(buf) > 0 {
			buf[0] = 0
		}
	}
}

// SDSCpy 复制字符串到 SDS
func SDSCpy(s SDS, t string) SDS {
	return SDSCpyLen(s, []byte(t), uint64(len(t)))
}

// SDSCpyLen 复制指定长度的字节数组到 SDS
func SDSCpyLen(s SDS, t []byte, len uint64) SDS {
	curlen := sdsLen(s)

	// 确保有足够空间
	if curlen < len {
		s = sdsMakeRoomFor(s, len-curlen)
	}

	if s == nil {
		return nil
	}

	buf := sdsBuf(s)
	copy(buf, t)
	sdsSetLen(s, len)
	buf[len] = 0

	return s
}

// SDSFree 释放 SDS（Go 中由 GC 自动管理，这里只是标记）
func SDSFree(s SDS) {
	// Go 的 GC 会自动管理内存
	// 这里可以添加自定义的释放逻辑
}

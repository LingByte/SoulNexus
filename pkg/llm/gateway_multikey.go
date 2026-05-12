// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
package llm

import (
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
)

// 内存级轮询游标。channelID -> 当前已使用次数。
// 进程重启即重置——无需持久化（这就是个负载均衡光标，不是状态机）。
var (
	multiKeyCursors sync.Map // map[int]*uint32
)

// SplitChannelKeys 将渠道的 Key 字段按换行/逗号拆分为 key 数组。
// 会去除空白与重复。供 admin / pickChannelKey 共用。
func SplitChannelKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	// 既支持换行也支持逗号
	normalized := strings.ReplaceAll(raw, "\r", "\n")
	normalized = strings.ReplaceAll(normalized, ",", "\n")
	parts := strings.Split(normalized, "\n")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		k := strings.TrimSpace(p)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

// PickChannelKey 根据渠道是否启用多 Key 与调度模式选取一个 key。
// channelID 仅用于轮询光标隔离，传 0 也能跑（视为非持久化轮询）。
// mode 取值："polling" 或 "random"，其它一律走 polling。
func PickChannelKey(channelID int, rawKey string, multiKey bool, mode string) string {
	rawKey = strings.TrimSpace(rawKey)
	if !multiKey {
		return rawKey
	}
	keys := SplitChannelKeys(rawKey)
	if len(keys) == 0 {
		return rawKey
	}
	if len(keys) == 1 {
		return keys[0]
	}
	if strings.EqualFold(strings.TrimSpace(mode), "random") {
		return keys[rand.Intn(len(keys))]
	}
	v, _ := multiKeyCursors.LoadOrStore(channelID, new(uint32))
	cursor := atomic.AddUint32(v.(*uint32), 1)
	return keys[(int(cursor)-1)%len(keys)]
}

package workflow

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"strconv"
	"time"
)

// TimerNode implements delay functionality
type TimerNode struct {
	Node
	Delay  time.Duration // 延迟时间
	Repeat bool          // 是否重复执行
}

func (t *TimerNode) Base() *Node {
	return &t.Node
}

func (t *TimerNode) Run(ctx *WorkflowContext) ([]string, error) {
	delay := t.Delay
	if delay <= 0 {
		delay = 1 * time.Second
	}

	ctx.AddLog("info", "定时器节点开始执行", t.ID, t.Name)
	ctx.AddLog("info", "延迟时间: "+delay.String(), t.ID, t.Name)

	fires := 1
	if t.Repeat {
		fires = 2
		if t.Properties != nil {
			if nStr, ok := t.Properties["repeatCount"]; ok {
				if n, err := parsePositiveInt(nStr); err == nil && n > 0 && n <= 10 {
					fires = n
				}
			}
		}
		ctx.AddLog("info", fmt.Sprintf("定时器重复执行 %d 次（进程内同步；跨副本需持久化调度，见 docs/distributed.md）", fires), t.ID, t.Name)
	}

	for i := 0; i < fires; i++ {
		if i > 0 {
			ctx.AddLog("info", fmt.Sprintf("定时器重复触发 #%d", i+1), t.ID, t.Name)
		}
		time.Sleep(delay)
	}

	ctx.AddLog("success", "定时器延迟完成", t.ID, t.Name)
	return t.NextNodes, nil
}

func parsePositiveInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid positive int")
	}
	return n, nil
}

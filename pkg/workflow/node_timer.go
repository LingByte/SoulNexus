package workflow

import (
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
	// 如果没有设置延迟时间，使用默认值1秒
	delay := t.Delay
	if delay <= 0 {
		delay = 1 * time.Second
	}

	// 添加日志
	ctx.AddLog("info", "定时器节点开始执行", t.ID, t.Name)
	ctx.AddLog("info", "延迟时间: "+delay.String(), t.ID, t.Name)

	// 执行延迟
	time.Sleep(delay)

	// 添加完成日志
	ctx.AddLog("success", "定时器延迟完成", t.ID, t.Name)

	// 如果设置了重复执行，这里可以添加重复逻辑
	// 目前简单实现为单次延迟
	if t.Repeat {
		ctx.AddLog("info", "定时器设置为重复执行（当前版本仅执行一次）", t.ID, t.Name)
		// TODO: 实现重复执行逻辑
		// 这需要更复杂的调度机制
	}

	return t.NextNodes, nil
}

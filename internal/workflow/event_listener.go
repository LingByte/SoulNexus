package workflowdef

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/events"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkflowEventListener 工作流事件监听器
type WorkflowEventListener struct {
	db             *gorm.DB
	triggerManager *WorkflowTriggerManager
}

// NewWorkflowEventListener 创建事件监听器
func NewWorkflowEventListener(db *gorm.DB) *WorkflowEventListener {
	return &WorkflowEventListener{
		db:             db,
		triggerManager: NewWorkflowTriggerManager(db),
	}
}

// Start 启动事件监听
func (l *WorkflowEventListener) Start() error {
	bus := events.GetEventBus()

	// 订阅所有事件，然后根据工作流配置决定是否触发
	bus.Subscribe("*", l.handleEvent)
	return nil
}

// handleEvent 处理事件
func (l *WorkflowEventListener) handleEvent(event events.Event) error {
	// 获取所有需要监听此事件的工作流
	workflows, err := l.triggerManager.GetActiveWorkflowsByEvent(event.Type)
	if err != nil {
		logger.Error("Failed to get workflows for event",
			zap.String("eventType", event.Type),
			zap.Error(err))
		return err
	}

	if len(workflows) == 0 {
		return nil
	}

	logger.Info("Triggering workflows for event",
		zap.String("eventType", event.Type),
		zap.Int("workflowCount", len(workflows)))

	// 为每个工作流触发执行
	for _, wf := range workflows {
		go func(workflow models.WorkflowDefinition) {
			// 将事件数据作为参数传递给工作流
			parameters := make(map[string]interface{})
			parameters["_event_type"] = event.Type
			parameters["_event_timestamp"] = event.Timestamp
			parameters["_event_source"] = event.Source
			// 将事件数据合并到参数中
			for k, v := range event.Data {
				parameters[k] = v
			}

			_, err := l.triggerManager.TriggerWorkflow(
				workflow.ID,
				parameters,
				fmt.Sprintf("event:%s", event.Type),
			)

			if err != nil {
				logger.Error("Failed to trigger workflow from event",
					zap.Uint("workflowId", workflow.ID),
					zap.String("eventType", event.Type),
					zap.Error(err))
			} else {
				logger.Info("Workflow triggered from event",
					zap.Uint("workflowId", workflow.ID),
					zap.String("eventType", event.Type))
			}
		}(wf)
	}

	return nil
}

package workflowdef

import (
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkflowScheduler workflow scheduled task scheduler
type WorkflowScheduler struct {
	db             *gorm.DB
	triggerManager *WorkflowTriggerManager
	cron           *cron.Cron
	jobIDs         map[uint]cron.EntryID // Workflow ID -> Cron Job ID
	mu             sync.RWMutex
}

var (
	schedulerInstance *WorkflowScheduler
	schedulerOnce     sync.Once
)

// GetWorkflowScheduler gets global scheduler instance
func GetWorkflowScheduler(db *gorm.DB) *WorkflowScheduler {
	schedulerOnce.Do(func() {
		schedulerInstance = &WorkflowScheduler{
			db:             db,
			triggerManager: NewWorkflowTriggerManager(db),
			cron:           cron.New(cron.WithSeconds()),
			jobIDs:         make(map[uint]cron.EntryID),
		}
	})
	return schedulerInstance
}

// Start starts the scheduler
func (s *WorkflowScheduler) Start() error {
	// Load all workflows that need scheduled execution
	workflows, err := s.triggerManager.GetScheduledWorkflows()
	if err != nil {
		return fmt.Errorf("failed to load scheduled workflows: %w", err)
	}

	logger.Info("Starting workflow scheduler",
		zap.Int("workflowCount", len(workflows)))

	// Register scheduled tasks for each workflow
	for _, wf := range workflows {
		if err := s.ScheduleWorkflow(wf.ID); err != nil {
			logger.Error("Failed to schedule workflow",
				zap.Uint("workflowId", wf.ID),
				zap.Error(err))
			continue
		}
	}

	// Start Cron
	s.cron.Start()
	logger.Info("Workflow scheduler started")

	return nil
}

// Stop stops the scheduler
func (s *WorkflowScheduler) Stop() {
	s.cron.Stop()
	logger.Info("Workflow scheduler stopped")
}

// ScheduleWorkflow registers scheduled task for workflow
func (s *WorkflowScheduler) ScheduleWorkflow(workflowID uint) error {
	// Get workflow definition
	var def models.WorkflowDefinition
	if err := s.db.First(&def, workflowID).Error; err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	// Check workflow status
	if def.Status != "active" {
		return fmt.Errorf("workflow is not active")
	}

	// Parse trigger configuration
	config, err := ParseTriggerConfig(&def)
	if err != nil {
		return fmt.Errorf("failed to parse trigger config: %w", err)
	}

	// Check scheduled trigger configuration
	if config.Schedule == nil || !config.Schedule.Enabled || config.Schedule.CronExpr == "" {
		return fmt.Errorf("schedule trigger not enabled or cron expression missing")
	}

	// 如果已经注册过，先移除
	s.UnscheduleWorkflow(workflowID)

	// 创建定时任务
	entryID, err := s.cron.AddFunc(config.Schedule.CronExpr, func() {
		s.executeScheduledWorkflow(workflowID)
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// 保存任务ID
	s.mu.Lock()
	s.jobIDs[workflowID] = entryID
	s.mu.Unlock()

	logger.Info("Workflow scheduled",
		zap.Uint("workflowId", workflowID),
		zap.String("cronExpr", config.Schedule.CronExpr))

	return nil
}

// UnscheduleWorkflow 取消工作流的定时任务
func (s *WorkflowScheduler) UnscheduleWorkflow(workflowID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobIDs[workflowID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobIDs, workflowID)
		logger.Info("Workflow unscheduled",
			zap.Uint("workflowId", workflowID))
	}
}

// executeScheduledWorkflow 执行定时工作流
func (s *WorkflowScheduler) executeScheduledWorkflow(workflowID uint) {
	logger.Info("Executing scheduled workflow",
		zap.Uint("workflowId", workflowID))

	// 使用触发器管理器执行工作流
	_, err := s.triggerManager.TriggerWorkflow(
		workflowID,
		make(map[string]interface{}), // 定时任务通常没有参数
		fmt.Sprintf("schedule:%d", workflowID),
	)

	if err != nil {
		logger.Error("Scheduled workflow execution failed",
			zap.Uint("workflowId", workflowID),
			zap.Error(err))
	} else {
		logger.Info("Scheduled workflow executed successfully",
			zap.Uint("workflowId", workflowID))
	}
}

// ReloadSchedules 重新加载所有定时任务（用于配置更新后）
func (s *WorkflowScheduler) ReloadSchedules() error {
	// 停止当前调度器
	s.Stop()

	// 清空所有任务
	s.mu.Lock()
	s.jobIDs = make(map[uint]cron.EntryID)
	s.mu.Unlock()

	// 重新创建 Cron 实例
	s.cron = cron.New(cron.WithSeconds())

	// 重新加载并启动
	return s.Start()
}

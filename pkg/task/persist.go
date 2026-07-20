package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PersistConfig wires a database into a Scheduler for crash-safe queues.
type PersistConfig[Params, Result any] struct {
	DB           *gorm.DB
	Queue        string
	Marshal      func(Params) ([]byte, error)
	Unmarshal    func([]byte) (Params, error)
	Handler      func(context.Context, Params) (Result, error)
	OnQueued     func(*Task[Params, Result])
	EnrichRecord func(Params, *ExecutionTask)
}

// NewPersistentScheduler builds a worker pool that survives restarts via execution_tasks.
func NewPersistentScheduler[Params, Result any](workerCount int, lg *zap.Logger, cfg PersistConfig[Params, Result]) (*Scheduler[Params, Result], error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("task persist db is required")
	}
	if cfg.Queue == "" {
		return nil, fmt.Errorf("task persist queue name is required")
	}
	if cfg.Handler == nil {
		return nil, fmt.Errorf("task persist handler is required")
	}
	if cfg.Marshal == nil {
		cfg.Marshal = func(p Params) ([]byte, error) { return json.Marshal(p) }
	}
	if cfg.Unmarshal == nil {
		cfg.Unmarshal = func(b []byte) (Params, error) {
			var p Params
			return p, json.Unmarshal(b, &p)
		}
	}
	s := NewScheduler[Params, Result](workerCount, lg)
	s.db = cfg.DB
	s.persistQueue = cfg.Queue
	s.marshal = cfg.Marshal
	s.unmarshal = cfg.Unmarshal
	s.handler = cfg.Handler
	s.onQueued = cfg.OnQueued
	s.enrichRecord = cfg.EnrichRecord
	if err := s.recoverPending(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scheduler[Params, Result]) recoverPending() error {
	if s == nil || s.db == nil {
		return nil
	}
	n, err := ResetRunningExecutionTasksToPending(s.db, s.persistQueue)
	if err != nil {
		return fmt.Errorf("reset interrupted tasks: %w", err)
	}
	if n > 0 && s.lg != nil {
		s.lg.Info("task queue recovered interrupted jobs",
			zap.String("queue", s.persistQueue),
			zap.Int64("count", n),
		)
	}
	rows, err := ListPendingExecutionTasks(s.db, s.persistQueue)
	if err != nil {
		return fmt.Errorf("list pending tasks: %w", err)
	}
	for _, row := range rows {
		params, err := s.unmarshal([]byte(row.ParamsJSON))
		if err != nil {
			_ = MarkExecutionTaskFinished(s.db, row.TaskID, TaskStatusFailed, "unmarshal params: "+err.Error())
			continue
		}
		s.enqueueRecovered(row.TaskID, row.Priority, params, row.SubmitTime)
	}
	if len(rows) > 0 && s.lg != nil {
		s.lg.Info("task queue reloaded pending jobs",
			zap.String("queue", s.persistQueue),
			zap.Int("count", len(rows)),
		)
	}
	return nil
}

func (s *Scheduler[Params, Result]) enqueueRecovered(taskID string, priority int, param Params, submitTime time.Time) {
	handler := s.handler
	if handler == nil {
		return
	}
	s.RequeueExisting(context.Background(), taskID, priority, param, submitTime, handler)
}

func (s *Scheduler[Params, Result]) persistPending(task *Task[Params, Result]) error {
	if s == nil || s.db == nil || task == nil {
		return nil
	}
	raw, err := s.marshal(task.Params)
	if err != nil {
		return fmt.Errorf("marshal task params: %w", err)
	}
	rec := ExecutionTask{
		TaskID:     task.ID,
		QueueName:  s.persistQueue,
		Priority:   task.Priority,
		ParamsJSON: string(raw),
		SubmitTime: task.SubmitTime,
		Source:     s.persistQueue,
	}
	if s.enrichRecord != nil {
		s.enrichRecord(task.Params, &rec)
	}
	return SaveExecutionTaskPending(s.db, rec)
}

func (s *Scheduler[Params, Result]) persistRunning(taskID string) {
	if s == nil || s.db == nil || taskID == "" {
		return
	}
	if err := MarkExecutionTaskRunning(s.db, taskID); err != nil && s.lg != nil {
		s.lg.Warn("task persist mark running failed", zap.String("task_id", taskID), zap.Error(err))
	}
}

func (s *Scheduler[Params, Result]) persistFinished(task *Task[Params, Result]) {
	if s == nil || s.db == nil || task == nil {
		return
	}
	st, _ := task.Status.Load().(TaskStatus)
	var errMsg string
	if task.err != nil {
		errMsg = task.err.Error()
	}
	if err := MarkExecutionTaskFinished(s.db, task.ID, st, errMsg); err != nil && s.lg != nil {
		s.lg.Warn("task persist mark finished failed", zap.String("task_id", task.ID), zap.Error(err))
	}
}

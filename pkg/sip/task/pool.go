package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
)

// PoolOption configures a TaskPool.
type PoolOption struct {
	WorkerCount int
	QueueSize   int
	Logger      *zap.Logger
}

// TaskPool runs handlers on a fixed number of workers fed from a buffered channel.
type TaskPool[Param any, Result any] struct {
	taskChan chan *Task[Param, Result]
	wg       sync.WaitGroup
	logger   *zap.Logger
	running  atomic.Int32
	workers  int
}

// NewTaskPool starts workerCount goroutines consuming from an internal queue.
func NewTaskPool[Param any, Result any](opt *PoolOption) *TaskPool[Param, Result] {
	if opt == nil {
		opt = &PoolOption{}
	}
	if opt.WorkerCount <= 0 {
		opt.WorkerCount = 4
	}
	if opt.QueueSize <= 0 {
		opt.QueueSize = 1024
	}
	lg := opt.Logger
	if lg == nil {
		lg = zap.NewNop()
	}
	tp := &TaskPool[Param, Result]{
		taskChan: make(chan *Task[Param, Result], opt.QueueSize),
		logger:   lg,
		workers:  opt.WorkerCount,
	}
	for i := 0; i < opt.WorkerCount; i++ {
		go tp.worker()
	}
	return tp
}

func (tp *TaskPool[Param, Result]) worker() {
	for task := range tp.taskChan {
		var zero Result
		if task.ctx.Err() != nil {
			task.deliver(zero, task.ctx.Err())
			tp.wg.Done()
			continue
		}
		tp.running.Add(1)
		task.Status.Store(TaskStatusRunning)
		tp.logger.Debug("task start", zap.String("id", task.ID))
		res, err := task.Handler(task.ctx, task.Params)
		tp.logger.Debug("task end", zap.String("id", task.ID))
		tp.running.Add(-1)
		if err != nil {
			task.deliver(zero, err)
		} else {
			task.deliver(res, nil)
		}
		tp.wg.Done()
	}
}

// Enqueue schedules a task that was already constructed (used by Scheduler).
func (tp *TaskPool[Param, Result]) Enqueue(task *Task[Param, Result]) error {
	if task == nil {
		return errors.New("nil task")
	}
	tp.wg.Add(1)
	select {
	case tp.taskChan <- task:
		return nil
	default:
		tp.wg.Done()
		return fmt.Errorf("task pool queue full (cap %d)", cap(tp.taskChan))
	}
}

// AddTask submits work directly to the pool (no priority queue).
func (tp *TaskPool[Param, Result]) AddTask(ctx context.Context, param Param, handler func(ctx context.Context, p Param) (Result, error)) (*Task[Param, Result], error) {
	task := newTask(ctx, 0, param, handler)
	if err := tp.Enqueue(task); err != nil {
		return nil, err
	}
	return task, nil
}

// CancelTask cancels a task's context (running handler should respect ctx).
func (tp *TaskPool[Param, Result]) CancelTask(task *Task[Param, Result]) {
	if task == nil || task.cancel == nil {
		return
	}
	task.cancel()
}

// WorkerCount returns configured workers.
func (tp *TaskPool[Param, Result]) WorkerCount() int { return tp.workers }

// QueueLen is the number of tasks waiting in the channel.
func (tp *TaskPool[Param, Result]) QueueLen() int { return len(tp.taskChan) }

// Running returns in-flight handler count.
func (tp *TaskPool[Param, Result]) Running() int32 { return tp.running.Load() }

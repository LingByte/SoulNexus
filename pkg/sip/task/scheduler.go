package task

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Stats exposes scheduler + pool visibility for dashboards.
type Stats struct {
	Queued     int   // priority queue length
	ChannelLen int   // tasks sitting in pool channel
	Running    int32 // handlers executing
	Unfinished int   // Queued + ChannelLen + Running
}

// Scheduler multiplexes prioritized submissions onto a TaskPool.
type Scheduler[Params, Result any] struct {
	pool      *TaskPool[Params, Result]
	queue     *PriorityQueue
	lg        *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	stopFlag  atomic.Bool
	stoppedCh chan struct{}
}

// NewScheduler builds a scheduler with workerCount workers.
func NewScheduler[Params, Result any](workerCount int, lg *zap.Logger) *Scheduler[Params, Result] {
	if lg == nil {
		lg = zap.NewNop()
	}
	pool := NewTaskPool[Params, Result](&PoolOption{
		WorkerCount: workerCount,
		Logger:      lg,
	})
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler[Params, Result]{
		pool:      pool,
		queue:     &PriorityQueue{},
		lg:        lg,
		ctx:       ctx,
		cancel:    cancel,
		stoppedCh: make(chan struct{}),
	}
	go s.dispatchLoop()
	return s
}

func (s *Scheduler[Params, Result]) dispatchLoop() {
	defer close(s.stoppedCh)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		if s.pool.Running() >= int32(s.pool.WorkerCount()) {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		item := s.queue.Pop()
		if item == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		task, ok := item.(*Task[Params, Result])
		if !ok || task == nil {
			continue
		}
		if err := s.pool.Enqueue(task); err != nil {
			s.lg.Warn("scheduler enqueue failed", zap.String("task_id", task.ID), zap.Error(err))
			var z Result
			task.deliver(z, err)
		}
	}
}

// SubmitTask enqueues by priority (higher runs earlier among waiting tasks).
func (s *Scheduler[Params, Result]) SubmitTask(
	ctx context.Context,
	priority int,
	param Params,
	handler func(ctx context.Context, params Params) (Result, error),
) *Task[Params, Result] {
	task := newTask(ctx, priority, param, handler)
	if s.stopFlag.Load() {
		var z Result
		task.deliver(z, fmt.Errorf("scheduler stopped"))
		return task
	}
	s.queue.Push(task, priority, task.ID)
	return task
}

// CancelTaskByID removes a pending task from the priority queue.
func (s *Scheduler[Params, Result]) CancelTaskByID(taskID string) bool {
	return s.queue.Remove(taskID)
}

// GetTaskPosition returns queue position (0 = next) or -1.
func (s *Scheduler[Params, Result]) GetTaskPosition(taskID string) int {
	return s.queue.GetPosition(taskID)
}

// QueueLen is tasks waiting in the priority queue (not yet in the pool channel).
func (s *Scheduler[Params, Result]) QueueLen() int {
	return s.queue.Len()
}

// RunningCount is handlers currently executing in the pool.
func (s *Scheduler[Params, Result]) RunningCount() int32 {
	return s.pool.Running()
}

// Stats returns queued, channel backlog, running, and unfinished totals.
func (s *Scheduler[Params, Result]) Stats() Stats {
	ql := s.queue.Len()
	cl := s.pool.QueueLen()
	rn := s.pool.Running()
	return Stats{
		Queued:     ql,
		ChannelLen: cl,
		Running:    rn,
		Unfinished: ql + cl + int(rn),
	}
}

// PendingSnapshot returns pending task IDs and priorities in queue order.
func (s *Scheduler[Params, Result]) PendingSnapshot() []QueueItemSnapshot {
	return s.queue.Snapshot()
}

// Stop stops dispatching new tasks. Running tasks continue to completion.
func (s *Scheduler[Params, Result]) Stop() error {
	if s == nil {
		return nil
	}
	if !s.stopFlag.CompareAndSwap(false, true) {
		return errors.New("scheduler already stopped")
	}
	if s.cancel != nil {
		s.cancel()
	}
	<-s.stoppedCh
	return nil
}

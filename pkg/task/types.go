package task

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
)

var errNilTask = errors.New("nil task")

// TaskStatus is the lifecycle of a submitted unit of work.
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusSuccess  TaskStatus = "success"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusCanceled TaskStatus = "canceled"
)

func (t TaskStatus) String() string { return string(t) }

// Task carries one handler invocation. Wait() blocks until completion.
type Task[Params, Result any] struct {
	ID         string
	ctx        context.Context
	cancel     context.CancelFunc
	Priority   int
	Params     Params
	Handler    func(ctx context.Context, params Params) (Result, error)
	Status     atomic.Value
	Progress   atomic.Int32
	SubmitTime time.Time

	queueAhead     atomic.Int32
	queuedTotal    atomic.Int32
	runningWorkers atomic.Int32
	unfinishedEst  atomic.Int32

	finishOnce sync.Once
	done       chan struct{}
	res        Result
	err        error
}

func newTask[Params, Result any](ctx context.Context, priority int, param Params, handler func(ctx context.Context, p Params) (Result, error)) *Task[Params, Result] {
	return newTaskWithID("", ctx, priority, param, time.Time{}, handler)
}

func newTaskWithID[Params, Result any](id string, ctx context.Context, priority int, param Params, submitTime time.Time, handler func(ctx context.Context, p Params) (Result, error)) *Task[Params, Result] {
	ctxCancel, cancel := context.WithCancel(ctx)
	if strings.TrimSpace(id) == "" {
		nid, _ := gonanoid.Nanoid()
		id = "task_" + nid
	}
	if submitTime.IsZero() {
		submitTime = time.Now()
	}
	t := &Task[Params, Result]{
		ID:         id,
		ctx:        ctxCancel,
		cancel:     cancel,
		Priority:   priority,
		Params:     param,
		Handler:    handler,
		SubmitTime: submitTime,
		done:       make(chan struct{}),
	}
	t.Status.Store(TaskStatusPending)
	t.Progress.Store(0)
	return t
}

func (t *Task[Params, Result]) applyQueueLabels(ahead, qlen int, running int32) {
	if t == nil {
		return
	}
	t.queueAhead.Store(int32(ahead))
	t.queuedTotal.Store(int32(qlen))
	t.runningWorkers.Store(running)
	t.unfinishedEst.Store(int32(qlen) + running)
}

// QueueAhead is how many tasks are still in front of this one in the priority wait queue (0 = next).
func (t *Task[Params, Result]) QueueAhead() int {
	if t == nil {
		return -1
	}
	return int(t.queueAhead.Load())
}

// QueuedTotal is the depth of the wait queue at the last relabel (includes this task while waiting).
func (t *Task[Params, Result]) QueuedTotal() int {
	if t == nil {
		return 0
	}
	return int(t.queuedTotal.Load())
}

// RunningWorkers is the snapshot of executing tasks at the last relabel.
func (t *Task[Params, Result]) RunningWorkers() int {
	if t == nil {
		return 0
	}
	return int(t.runningWorkers.Load())
}

// UnfinishedEstimate is queuedTotal + runningWorkers at last relabel (system load hint).
func (t *Task[Params, Result]) UnfinishedEstimate() int {
	if t == nil {
		return 0
	}
	return int(t.unfinishedEst.Load())
}

func (t *Task[Params, Result]) deliver(res Result, err error) {
	if t == nil {
		return
	}
	t.finishOnce.Do(func() {
		switch {
		case err == nil:
			t.Status.Store(TaskStatusSuccess)
		case errors.Is(err, context.Canceled):
			t.Status.Store(TaskStatusCanceled)
		default:
			t.Status.Store(TaskStatusFailed)
		}
		t.res = res
		t.err = err
		close(t.done)
	})
}

// Wait blocks until the handler finishes (success, failure, or cancel).
func (t *Task[Params, Result]) Wait() (Result, error) {
	if t == nil {
		var zero Result
		return zero, errNilTask
	}
	<-t.done
	return t.res, t.err
}

// Cancel requests cooperative stop via the task context.
func (t *Task[Params, Result]) Cancel() {
	if t == nil || t.cancel == nil {
		return
	}
	t.cancel()
}

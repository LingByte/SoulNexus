package task

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
)

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

// Task carries one handler invocation.
type Task[Params, Result any] struct {
	ID         string
	ctx        context.Context
	cancel     context.CancelFunc
	Priority   int
	Params     Params
	Handler    func(ctx context.Context, params Params) (Result, error)
	Result     chan Result
	Err        chan error
	Status     atomic.Value
	Progress   atomic.Int32
	SubmitTime time.Time

	finishOnce sync.Once
}

func newTask[Params, Result any](ctx context.Context, priority int, param Params, handler func(ctx context.Context, p Params) (Result, error)) *Task[Params, Result] {
	ctxCancel, cancel := context.WithCancel(ctx)
	id, _ := gonanoid.Nanoid()
	t := &Task[Params, Result]{
		ID:         "task_" + id,
		ctx:        ctxCancel,
		cancel:     cancel,
		Priority:   priority,
		Params:     param,
		Handler:    handler,
		Result:     make(chan Result, 1),
		Err:        make(chan error, 1),
		SubmitTime: time.Now(),
	}
	t.Status.Store(TaskStatusPending)
	t.Progress.Store(0)
	return t
}

func (t *Task[Params, Result]) deliver(res Result, err error) {
	t.finishOnce.Do(func() {
		switch {
		case err == nil:
			t.Status.Store(TaskStatusSuccess)
		case errors.Is(err, context.Canceled):
			t.Status.Store(TaskStatusCanceled)
		default:
			t.Status.Store(TaskStatusFailed)
		}
		t.Result <- res
		t.Err <- err
		close(t.Result)
		close(t.Err)
	})
}

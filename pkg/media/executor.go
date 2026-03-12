package media

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// PacketRequest is an alias for backward compatibility - maps Interrupt to shouldStop
type PacketRequest[R any] struct {
	H         MediaHandler
	Interrupt bool
	Req       R
}

// AsyncTaskRunner handles asynchronous task execution using true multi-worker pool pattern
type AsyncTaskRunner[T any] struct {
	InitCallback      func(h MediaHandler) error
	TerminateCallback func(h MediaHandler) error
	StateCallback     func(h MediaHandler, event StateChange) error
	RequestBuilder    func(h MediaHandler, packet MediaPacket) (*PacketRequest[T], error)
	TaskExecutor      func(ctx context.Context, h MediaHandler, req PacketRequest[T]) error

	WorkerPoolSize int
	TaskTimeout    time.Duration
	MaxTaskTimeout time.Duration
	ConcurrentMode bool

	taskQueue     chan *PacketRequest[T]
	workers       []context.CancelFunc
	workerWg      sync.WaitGroup
	poolCtx       context.Context
	poolCancel    context.CancelFunc
	workersActive bool
}

// NewAsyncTaskRunner creates a new task runner with worker pool configuration
func NewAsyncTaskRunner[T any](queueSize int) AsyncTaskRunner[T] {
	return AsyncTaskRunner[T]{
		WorkerPoolSize: queueSize,
		MaxTaskTimeout: 1 * time.Minute,
		ConcurrentMode: true,
	}
}

// ReleaseResources frees allocated resources and stops all workers
func (tr *AsyncTaskRunner[T]) ReleaseResources() {
	tr.stopWorkers()
	if tr.taskQueue != nil {
		close(tr.taskQueue)
		tr.taskQueue = nil
	}
}

// CancelActiveTask stops all worker tasks (for multi-worker pool)
func (tr *AsyncTaskRunner[T]) CancelActiveTask() {
	tr.stopWorkers()
}

// stopWorkers stops all worker goroutines
func (tr *AsyncTaskRunner[T]) stopWorkers() {
	if !tr.workersActive {
		return
	}
	if tr.poolCancel != nil {
		tr.poolCancel()
	}
	tr.workerWg.Wait()
	tr.workers = nil
	tr.workersActive = false
}

// HandleMediaData routes media data to appropriate handlers
func (tr *AsyncTaskRunner[T]) HandleMediaData(h MediaHandler, data MediaData) {
	if tr.TaskExecutor == nil {
		panic("TaskExecutor is not set")
	}
	switch data.Type {
	case MediaDataTypePacket:
		tr.HandlePacket(h, data.Packet)
	case MediaDataTypeState:
		tr.HandleState(h, data.State)
	}
}

// HandleState processes state change events
func (tr *AsyncTaskRunner[T]) HandleState(h MediaHandler, event StateChange) {
	switch event.State {
	case Begin:
		if tr.ConcurrentMode {
			tr.startMultiWorkerPool(h.GetContext())
		}
		if tr.InitCallback != nil {
			if err := tr.InitCallback(h); err != nil {
				h.CauseError(tr, err)
			}
		}
	case End:
		tr.CancelActiveTask()
		tr.ReleaseResources()
		if tr.TerminateCallback != nil {
			if err := tr.TerminateCallback(h); err != nil {
				h.CauseError(tr, err)
			}
		}
	}
	if tr.StateCallback != nil {
		if err := tr.StateCallback(h, event); err != nil {
			h.CauseError(tr, err)
		}
	}
}

// HandlePacket processes packet data through task queue
func (tr *AsyncTaskRunner[T]) HandlePacket(h MediaHandler, packet MediaPacket) {
	if tr.RequestBuilder == nil {
		panic("RequestBuilder is not set")
	}
	req, err := tr.RequestBuilder(h, packet)
	if err != nil {
		h.CauseError(tr, err)
		return
	}
	if req == nil {
		return
	}
	req.H = h
	if tr.ConcurrentMode {
		if tr.taskQueue != nil {
			tr.taskQueue <- req
		} else {
			logger.Warn("taskQueue is nil", zap.Any("packet", packet), zap.Any("runner", tr))
		}
	} else {
		tr.executeTask(h.GetContext(), *req)
	}
}

// startMultiWorkerPool starts multiple worker goroutines (true worker pool)
func (tr *AsyncTaskRunner[T]) startMultiWorkerPool(parent context.Context) {
	if tr.workersActive {
		return
	}

	poolSize := tr.WorkerPoolSize
	if poolSize <= 0 {
		poolSize = 4 // Default to 4 workers
	}

	tr.taskQueue = make(chan *PacketRequest[T], poolSize*2)
	tr.poolCtx, tr.poolCancel = context.WithCancel(parent)
	tr.workers = make([]context.CancelFunc, poolSize)
	tr.workersActive = true

	// Start multiple workers
	for i := 0; i < poolSize; i++ {
		workerCtx, workerCancel := context.WithCancel(tr.poolCtx)
		tr.workers[i] = workerCancel
		tr.workerWg.Add(1)
		go tr.workerLoop(workerCtx, i)
	}
}

// workerLoop is the processing loop for a single worker
func (tr *AsyncTaskRunner[T]) workerLoop(ctx context.Context, workerID int) {
	defer tr.workerWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-tr.taskQueue:
			if !ok {
				return
			}
			if req == nil {
				continue
			}
			if req.Interrupt {
				// Interrupt signal, cancel all workers
				tr.poolCancel()
				return
			}
			tr.executeTask(ctx, *req)
		}
	}
}

// executeTask runs a single task with timeout control
func (tr *AsyncTaskRunner[T]) executeTask(ctx context.Context, req PacketRequest[T]) {
	timeout := tr.MaxTaskTimeout
	if tr.TaskTimeout > 0 {
		timeout = tr.TaskTimeout
	}
	taskCtx, taskCancel := context.WithTimeout(ctx, timeout)
	defer taskCancel()

	err := tr.TaskExecutor(taskCtx, req.H, req)
	if err != nil {
		logger.Error("Task execution error", zap.Any("handler", req.H), zap.Error(err))
		req.H.CauseError(tr, err)
	}
}

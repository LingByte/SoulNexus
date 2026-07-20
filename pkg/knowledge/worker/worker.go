package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/task"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultDocumentJobTimeout = 30 * time.Minute
	documentWorkerQueue       = "knowledge.document"
)

// IngestJob is one async parse + vector-index unit of work.
type IngestJob struct {
	DocID        uint
	Namespace    string
	Title        string
	FileName     string
	RawFileURL   string
	TextURL      string
	OldRecordIDs []string
	ConfirmIndex bool
}

// PurgeJob removes vectors and object-storage artifacts for a deleted document.
type PurgeJob struct {
	Namespace  string
	RecordIDs  []string
	DocID      uint
	GroupID    uint
	RawFileURL string
	TextURL    string
}

// DocumentJob is queued on the knowledge background worker pool.
type DocumentJob struct {
	Ingest       *IngestJob
	Purge        *PurgeJob
	SyncSourceID uint
}

// DocumentHandler runs one background knowledge document job.
type DocumentHandler func(ctx context.Context, job DocumentJob) error

// DocumentWorker schedules parse/index/purge work on a fixed worker pool.
type DocumentWorker struct {
	scheduler *task.Scheduler[DocumentJob, struct{}]
	handler   DocumentHandler

	trackMu sync.RWMutex
	byTask  map[string]documentJobTrack
	byDoc   map[uint]string
}

// NewDocumentWorker builds a priority scheduler backed by workerCount goroutines.
// When db is non-nil, pending jobs survive process restarts via execution_tasks persistence.
func NewDocumentWorker(db *gorm.DB, workerCount int, handler DocumentHandler) *DocumentWorker {
	if handler == nil {
		return nil
	}
	if workerCount <= 0 {
		workerCount = 4
	}
	w := &DocumentWorker{handler: handler}
	if db != nil {
		sched, err := task.NewPersistentScheduler(workerCount, logger.Lg, task.PersistConfig[DocumentJob, struct{}]{
			DB:      db,
			Queue:   documentWorkerQueue,
			Handler: w.runJob,
			OnQueued: func(t *task.Task[DocumentJob, struct{}]) {
				if t != nil {
					w.registerTask(t, t.Params)
				}
			},
			EnrichRecord: enrichDocumentJobRecord,
		})
		if err != nil {
			if logger.Lg != nil {
				logger.Lg.Error("knowledge document worker persistence init failed; falling back to in-memory queue", zap.Error(err))
			}
			w.scheduler = task.NewScheduler[DocumentJob, struct{}](workerCount, logger.Lg)
		} else {
			w.scheduler = sched
		}
	} else {
		w.scheduler = task.NewScheduler[DocumentJob, struct{}](workerCount, logger.Lg)
	}
	return w
}

// CancelTaskByID cancels a pending in-memory job and marks the durable row canceled when present.
func (w *DocumentWorker) CancelTaskByID(taskID string) bool {
	if w == nil || w.scheduler == nil || taskID == "" {
		return false
	}
	return w.scheduler.CancelTaskByID(taskID)
}

// RequeueJob puts an existing durable task back onto the worker pool.
func (w *DocumentWorker) RequeueJob(taskID string, job DocumentJob, priority int, submitTime time.Time) {
	if w == nil || w.scheduler == nil || taskID == "" {
		return
	}
	t := w.scheduler.RequeueExisting(context.Background(), taskID, priority, job, submitTime, w.runJob)
	w.registerTaskIfNeeded(t, job)
}

// RetryTaskByID re-enqueues a durable job from its stored params.
func (w *DocumentWorker) RetryTaskByID(db *gorm.DB, taskID string) error {
	if w == nil || db == nil || taskID == "" {
		return fmt.Errorf("worker or task id missing")
	}
	row, err := task.GetExecutionTaskByTaskID(db, taskID)
	if err != nil {
		return err
	}
	var job DocumentJob
	if err := json.Unmarshal([]byte(row.ParamsJSON), &job); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}
	if err := task.ResetExecutionTaskForRetry(db, taskID); err != nil {
		return err
	}
	w.RequeueJob(taskID, job, row.Priority, row.SubmitTime)
	return nil
}

func enrichDocumentJobRecord(job DocumentJob, rec *task.ExecutionTask) {
	if rec == nil {
		return
	}
	rec.Source = "knowledge"
	rec.Kind = string(jobKind(job))
	switch rec.Kind {
	case string(JobKindIngest):
		if job.Ingest != nil {
			rec.Title = job.Ingest.Title
			if rec.Title == "" {
				rec.Title = job.Ingest.FileName
			}
		}
	case string(JobKindPurge):
		rec.Title = "purge document vectors"
		if job.Purge != nil && job.Purge.DocID > 0 {
			rec.Title = fmt.Sprintf("purge doc %d", job.Purge.DocID)
		}
	case string(JobKindSync):
		rec.Title = fmt.Sprintf("sync source %d", job.SyncSourceID)
	}
}

func (w *DocumentWorker) runJob(ctx context.Context, job DocumentJob) (struct{}, error) {
	taskID := task.TaskIDFromContext(ctx)
	defer w.unregisterTask(taskID)
	runCtx, cancel := context.WithTimeout(ctx, defaultDocumentJobTimeout)
	defer cancel()
	err := w.handler(runCtx, job)
	if err != nil && logger.Lg != nil {
		docID := uint(0)
		if job.Ingest != nil {
			docID = job.Ingest.DocID
		}
		logger.Lg.Warn("knowledge document job failed",
			zap.String("taskId", taskID),
			zap.Uint("docId", docID),
			zap.Bool("purge", job.Purge != nil),
			zap.Error(err),
		)
	}
	return struct{}{}, err
}

// Enqueue submits a job without blocking the HTTP handler.
func (w *DocumentWorker) Enqueue(job DocumentJob) {
	if w == nil || w.scheduler == nil {
		return
	}
	t := w.scheduler.SubmitTask(context.Background(), 0, job, w.runJob)
	w.registerTaskIfNeeded(t, job)
}

func (w *DocumentWorker) registerTaskIfNeeded(t *task.Task[DocumentJob, struct{}], job DocumentJob) {
	if w == nil || t == nil {
		return
	}
	w.trackMu.RLock()
	_, exists := w.byTask[t.ID]
	w.trackMu.RUnlock()
	if !exists {
		w.registerTask(t, job)
	}
}

// Stop drains the worker pool (call on process shutdown).
func (w *DocumentWorker) Stop() error {
	if w == nil || w.scheduler == nil {
		return nil
	}
	return w.scheduler.Stop()
}

// Stats exposes queue depth for observability.
func (w *DocumentWorker) Stats() task.Stats {
	if w == nil || w.scheduler == nil {
		return task.Stats{}
	}
	return w.scheduler.Stats()
}

// JobKind identifies a background knowledge worker job.
type JobKind string

const (
	JobKindIngest JobKind = "ingest"
	JobKindPurge  JobKind = "purge"
	JobKindSync   JobKind = "sync"
)

// JobProgress is queue position and lifecycle for one worker job.
type JobProgress struct {
	TaskID             string    `json:"taskId"`
	Kind               JobKind   `json:"kind"`
	DocID              uint      `json:"docId,omitempty"`
	SyncSourceID       uint      `json:"syncSourceId,omitempty"`
	TaskStatus         string    `json:"taskStatus"`
	QueueAhead         int       `json:"queueAhead"`
	QueuedTotal        int       `json:"queuedTotal"`
	RunningWorkers     int       `json:"runningWorkers"`
	UnfinishedEstimate int       `json:"unfinishedEstimate"`
	SubmittedAt        time.Time `json:"submittedAt"`
}

// WorkerSnapshot is queue depth plus all in-flight document jobs.
type WorkerSnapshot struct {
	Queued     int           `json:"queued"`
	Running    int32         `json:"running"`
	Unfinished int           `json:"unfinished"`
	Jobs       []JobProgress `json:"jobs"`
}

type documentJobTrack struct {
	job  DocumentJob
	task *task.Task[DocumentJob, struct{}]
}

func jobKind(job DocumentJob) JobKind {
	if job.SyncSourceID > 0 {
		return JobKindSync
	}
	if job.Purge != nil {
		return JobKindPurge
	}
	return JobKindIngest
}

func docIDFromJob(job DocumentJob) uint {
	if job.Ingest != nil {
		return job.Ingest.DocID
	}
	if job.Purge != nil {
		return job.Purge.DocID
	}
	return 0
}

func taskProgress(t *task.Task[DocumentJob, struct{}], job DocumentJob) JobProgress {
	st := task.TaskStatusPending
	if v := t.Status.Load(); v != nil {
		if s, ok := v.(task.TaskStatus); ok {
			st = s
		}
	}
	return JobProgress{
		TaskID:             t.ID,
		Kind:               jobKind(job),
		DocID:              docIDFromJob(job),
		SyncSourceID:       job.SyncSourceID,
		TaskStatus:         st.String(),
		QueueAhead:         t.QueueAhead(),
		QueuedTotal:        t.QueuedTotal(),
		RunningWorkers:     t.RunningWorkers(),
		UnfinishedEstimate: t.UnfinishedEstimate(),
		SubmittedAt:        t.SubmitTime,
	}
}

func (w *DocumentWorker) registerTask(t *task.Task[DocumentJob, struct{}], job DocumentJob) {
	if w == nil || t == nil {
		return
	}
	w.trackMu.Lock()
	defer w.trackMu.Unlock()
	if w.byTask == nil {
		w.byTask = make(map[string]documentJobTrack)
		w.byDoc = make(map[uint]string)
	}
	w.byTask[t.ID] = documentJobTrack{job: job, task: t}
	if docID := docIDFromJob(job); docID > 0 {
		w.byDoc[docID] = t.ID
	}
}

func (w *DocumentWorker) unregisterTask(taskID string) {
	if w == nil || taskID == "" {
		return
	}
	w.trackMu.Lock()
	defer w.trackMu.Unlock()
	tr, ok := w.byTask[taskID]
	if !ok {
		return
	}
	delete(w.byTask, taskID)
	if docID := docIDFromJob(tr.job); docID > 0 && w.byDoc[docID] == taskID {
		delete(w.byDoc, docID)
	}
}

// DocProgress returns live queue position for a document job, if one is active.
func (w *DocumentWorker) DocProgress(docID uint) (JobProgress, bool) {
	if w == nil || docID == 0 {
		return JobProgress{}, false
	}
	w.trackMu.RLock()
	taskID, ok := w.byDoc[docID]
	tr, ok2 := w.byTask[taskID]
	w.trackMu.RUnlock()
	if !ok || !ok2 || tr.task == nil {
		return JobProgress{}, false
	}
	return taskProgress(tr.task, tr.job), true
}

// Snapshot returns queue totals and all tracked pending/running jobs.
func (w *DocumentWorker) Snapshot() WorkerSnapshot {
	out := WorkerSnapshot{}
	if w == nil || w.scheduler == nil {
		return out
	}
	st := w.scheduler.Stats()
	out.Queued = st.Queued
	out.Running = st.Running
	out.Unfinished = st.Unfinished

	w.trackMu.RLock()
	jobs := make([]JobProgress, 0, len(w.byTask))
	for _, tr := range w.byTask {
		if tr.task == nil {
			continue
		}
		jp := taskProgress(tr.task, tr.job)
		if jp.TaskStatus != task.TaskStatusPending.String() && jp.TaskStatus != task.TaskStatusRunning.String() {
			continue
		}
		jobs = append(jobs, jp)
	}
	w.trackMu.RUnlock()

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].QueueAhead != jobs[j].QueueAhead {
			return jobs[i].QueueAhead < jobs[j].QueueAhead
		}
		return jobs[i].SubmittedAt.Before(jobs[j].SubmittedAt)
	})
	out.Jobs = jobs
	return out
}

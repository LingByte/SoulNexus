package mediagen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/task"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	QueueName           = "mediagen.generate"
	mediaWorkerQueue    = QueueName
	defaultImageTimeout = 3 * time.Minute
	defaultVideoTimeout = 25 * time.Minute
	defaultMediaWorkers = 2
)

// WorkerJob is queued on the mediagen persistent scheduler.
type WorkerJob struct {
	PublicID string `json:"publicId"`
}

// Worker runs image/video generation with execution_tasks durability.
type Worker struct {
	db        *gorm.DB
	svc       *Service
	scheduler *task.Scheduler[WorkerJob, struct{}]
}

func NewWorker(db *gorm.DB, svc *Service, workerCount int) *Worker {
	if svc == nil {
		return nil
	}
	if workerCount <= 0 {
		workerCount = defaultMediaWorkers
	}
	w := &Worker{db: db, svc: svc}
	if db != nil {
		sched, err := task.NewPersistentScheduler(workerCount, logger.Lg, task.PersistConfig[WorkerJob, struct{}]{
			DB:      db,
			Queue:   mediaWorkerQueue,
			Handler: w.runJob,
			EnrichRecord: func(job WorkerJob, rec *task.ExecutionTask) {
				if rec == nil {
					return
				}
				rec.Source = "mediagen"
				rec.Kind = "media_generate"
				rec.Title = "media generate " + strings.TrimSpace(job.PublicID)
			},
		})
		if err != nil {
			if logger.Lg != nil {
				logger.Lg.Error("mediagen worker persistence init failed; falling back to in-memory queue", zap.Error(err))
			}
			w.scheduler = task.NewScheduler[WorkerJob, struct{}](workerCount, logger.Lg)
		} else {
			w.scheduler = sched
		}
	} else {
		w.scheduler = task.NewScheduler[WorkerJob, struct{}](workerCount, logger.Lg)
	}
	return w
}

func (w *Worker) Enqueue(job WorkerJob) *task.Task[WorkerJob, struct{}] {
	if w == nil || w.scheduler == nil {
		return nil
	}
	return w.scheduler.SubmitTask(context.Background(), 0, job, w.runJob)
}

// EnqueueAndWait runs a job and blocks until completion (image sync API).
func (w *Worker) EnqueueAndWait(ctx context.Context, job WorkerJob) error {
	t := w.Enqueue(job)
	if t == nil {
		return fmt.Errorf("mediagen worker unavailable")
	}
	_ = UpdateMediaGenerateJob(w.db, job.PublicID, map[string]any{
		"execution_task_id": t.ID,
		"status":            JobStatusQueued,
		"queued_at":         time.Now(),
	})
	done := make(chan error, 1)
	go func() {
		_, err := t.Wait()
		done <- err
	}()
	select {
	case <-ctx.Done():
		t.Cancel()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (w *Worker) EnqueueAsync(job WorkerJob) (string, error) {
	t := w.Enqueue(job)
	if t == nil {
		return "", fmt.Errorf("mediagen worker unavailable")
	}
	_ = UpdateMediaGenerateJob(w.db, job.PublicID, map[string]any{
		"execution_task_id": t.ID,
		"status":            JobStatusQueued,
		"queued_at":         time.Now(),
	})
	return t.ID, nil
}

// CancelTaskByID cancels a pending in-memory mediagen job.
func (w *Worker) CancelTaskByID(taskID string) bool {
	if w == nil || w.scheduler == nil || strings.TrimSpace(taskID) == "" {
		return false
	}
	return w.scheduler.CancelTaskByID(taskID)
}

// RequeueJob puts an existing durable execution task back on the in-memory queue.
func (w *Worker) RequeueJob(taskID string, job WorkerJob, priority int, submitTime time.Time) {
	if w == nil || w.scheduler == nil || strings.TrimSpace(taskID) == "" {
		return
	}
	if submitTime.IsZero() {
		submitTime = time.Now()
	}
	w.scheduler.RequeueExisting(context.Background(), taskID, priority, job, submitTime, w.runJob)
	_ = UpdateMediaGenerateJob(w.db, job.PublicID, map[string]any{
		"execution_task_id": taskID,
		"status":            JobStatusQueued,
		"queued_at":         time.Now(),
		"started_at":        gorm.Expr("NULL"),
		"finished_at":       gorm.Expr("NULL"),
		"progress":          0,
		"error_message":     "",
		"remote_url":        "",
		"storage_key":       "",
		"result_url":        "",
		"provider_task_id":  "",
	})
}

// RetryTaskByID re-enqueues a failed/canceled mediagen execution task from stored params.
func (w *Worker) RetryTaskByID(db *gorm.DB, taskID string) error {
	if w == nil || w.scheduler == nil {
		return fmt.Errorf("mediagen worker unavailable")
	}
	if db == nil {
		db = w.db
	}
	if db == nil || strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("worker or task id missing")
	}
	row, err := task.GetExecutionTaskByTaskID(db, taskID)
	if err != nil {
		return err
	}
	var job WorkerJob
	if err := json.Unmarshal([]byte(row.ParamsJSON), &job); err != nil {
		return fmt.Errorf("unmarshal params: %w", err)
	}
	if strings.TrimSpace(job.PublicID) == "" {
		return fmt.Errorf("empty media job publicId in params")
	}
	if _, err := GetMediaGenerateJobByPublicID(db, job.PublicID); err != nil {
		return fmt.Errorf("media job not found: %w", err)
	}
	if err := task.ResetExecutionTaskForRetry(db, taskID); err != nil {
		return err
	}
	w.RequeueJob(taskID, job, row.Priority, row.SubmitTime)
	return nil
}

func (w *Worker) Stop() error {
	if w == nil || w.scheduler == nil {
		return nil
	}
	return w.scheduler.Stop()
}

func (w *Worker) runJob(ctx context.Context, job WorkerJob) (struct{}, error) {
	publicID := strings.TrimSpace(job.PublicID)
	if publicID == "" {
		return struct{}{}, fmt.Errorf("empty media job id")
	}
	row, err := GetMediaGenerateJobByPublicID(w.db, publicID)
	if err != nil {
		return struct{}{}, err
	}
	execID := task.TaskIDFromContext(ctx)
	now := time.Now()
	_ = UpdateMediaGenerateJob(w.db, publicID, map[string]any{
		"status":            JobStatusRunning,
		"started_at":        now,
		"execution_task_id": firstNonEmpty(execID, row.ExecutionTaskID),
		"progress":          5,
		"error_message":     "",
	})

	var runErr error
	switch row.Kind {
	case JobKindTextToImage, JobKindImageToImage:
		runCtx, cancel := context.WithTimeout(ctx, defaultImageTimeout)
		defer cancel()
		runErr = w.svc.ExecuteImageJob(runCtx, w.db, row)
	case JobKindTextToVideo, JobKindImageToVideo:
		runCtx, cancel := context.WithTimeout(ctx, defaultVideoTimeout)
		defer cancel()
		runErr = w.svc.ExecuteVideoJob(runCtx, w.db, row)
	default:
		runErr = fmt.Errorf("unsupported media job kind: %s", row.Kind)
	}

	execTaskID := firstNonEmpty(execID, row.ExecutionTaskID)
	if runErr != nil {
		_ = UpdateMediaGenerateJob(w.db, publicID, map[string]any{
			"status":        JobStatusFailed,
			"error_message": truncateErr(runErr.Error()),
			"finished_at":   time.Now(),
			"progress":      100,
		})
		if execRow, e := task.GetExecutionTaskByTaskID(w.db, execTaskID); e == nil && execRow != nil {
			raw, _ := json.Marshal(map[string]any{"publicId": publicID, "error": runErr.Error()})
			_ = task.UpdateExecutionTask(w.db, execRow.ID, map[string]any{"result_json": string(raw)})
		}
		return struct{}{}, runErr
	}
	if execRow, e := task.GetExecutionTaskByTaskID(w.db, execTaskID); e == nil && execRow != nil {
		fresh, _ := GetMediaGenerateJobByPublicID(w.db, publicID)
		payload := map[string]any{"publicId": publicID}
		if fresh != nil {
			payload["storageKey"] = fresh.StorageKey
			payload["resultUrl"] = fresh.ResultURL
			payload["status"] = fresh.Status
		}
		raw, _ := json.Marshal(payload)
		_ = task.UpdateExecutionTask(w.db, execRow.ID, map[string]any{"result_json": string(raw)})
	}
	return struct{}{}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateErr(s string) string {
	if len(s) > 2000 {
		return s[:2000]
	}
	return s
}

package task

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestPersistentSchedulerRecoversPending(t *testing.T) {
	db := openTaskStoreTestDB(t)

	_ = SaveExecutionTaskPending(db, ExecutionTask{
		TaskID:     "task_recover",
		QueueName:  "test.queue",
		Priority:   0,
		ParamsJSON: `42`,
		SubmitTime: time.Now(),
	})

	var ran atomic.Int32
	s, err := NewPersistentScheduler(1, zap.NewNop(), PersistConfig[int, int]{
		DB:      db,
		Queue:   "test.queue",
		Handler: func(ctx context.Context, p int) (int, error) { return p * 2, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var row ExecutionTask
		err := db.First(&row, "task_id = ?", "task_recover").Error
		if err == nil && row.Status == TaskStatusSuccess.String() {
			ran.Store(1)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if ran.Load() != 1 {
		t.Fatal("recovered task did not finish")
	}
}

func TestPersistentSchedulerSubmitPersists(t *testing.T) {
	db := openTaskStoreTestDB(t)

	s, err := NewPersistentScheduler(1, zap.NewNop(), PersistConfig[int, int]{
		DB:      db,
		Queue:   "test.queue",
		Handler: func(ctx context.Context, p int) (int, error) { return p, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	tsk := s.SubmitTask(context.Background(), 0, 7, nil)
	var row ExecutionTask
	if err := db.First(&row, "task_id = ?", tsk.ID).Error; err != nil {
		t.Fatalf("pending row missing: %v", err)
	}
	if row.Status != TaskStatusPending.String() {
		t.Fatalf("status %q", row.Status)
	}
	if _, err := tsk.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&row, "task_id = ?", tsk.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != TaskStatusSuccess.String() {
		t.Fatalf("final status %q", row.Status)
	}
}

func TestSchedulerStopWithDBLeavesPendingRows(t *testing.T) {
	db := openTaskStoreTestDB(t)

	taskID := "task_stop_pending"
	if err := SaveExecutionTaskPending(db, ExecutionTask{
		TaskID:     taskID,
		QueueName:  "test.queue",
		ParamsJSON: "1",
		SubmitTime: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	s := NewScheduler[int, int](1, zap.NewNop())
	s.db = db
	s.persistQueue = "test.queue"
	s.handler = func(ctx context.Context, p int) (int, error) { return p, nil }
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}

	var row ExecutionTask
	if err := db.First(&row, "task_id = ?", taskID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != TaskStatusPending.String() {
		t.Fatalf("expected pending row after stop, got %q", row.Status)
	}
}

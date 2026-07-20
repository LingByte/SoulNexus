package task

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTaskStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&ExecutionTask{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSaveExecutionTaskPendingAndList(t *testing.T) {
	db := openTaskStoreTestDB(t)
	submit := time.Now().UTC().Truncate(time.Millisecond)

	if err := SaveExecutionTaskPending(db, ExecutionTask{
		TaskID:     "task_a",
		QueueName:  "q1",
		Priority:   5,
		ParamsJSON: `{"n":1}`,
		SubmitTime: submit,
	}); err != nil {
		t.Fatal(err)
	}
	if err := SaveExecutionTaskPending(db, ExecutionTask{
		TaskID:     "task_b",
		QueueName:  "q1",
		Priority:   1,
		ParamsJSON: `{"n":2}`,
		SubmitTime: submit.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := ListPendingExecutionTasks(db, "q1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].TaskID != "task_a" || rows[1].TaskID != "task_b" {
		t.Fatalf("pending order %#v", rows)
	}
}

func TestMarkExecutionTaskRunningAndFinished(t *testing.T) {
	db := openTaskStoreTestDB(t)

	_ = SaveExecutionTaskPending(db, ExecutionTask{
		TaskID:     "task_x",
		QueueName:  "q1",
		ParamsJSON: `{}`,
		SubmitTime: time.Now(),
	})
	if err := MarkExecutionTaskRunning(db, "task_x"); err != nil {
		t.Fatal(err)
	}
	var row ExecutionTask
	if err := db.First(&row, "task_id = ?", "task_x").Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != TaskStatusRunning.String() || row.StartedAt == nil {
		t.Fatalf("running row %#v", row)
	}
	if err := MarkExecutionTaskFinished(db, "task_x", TaskStatusSuccess, ""); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&row, "task_id = ?", "task_x").Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != TaskStatusSuccess.String() || row.FinishedAt == nil {
		t.Fatalf("finished row %#v", row)
	}
	if pending, err := ListPendingExecutionTasks(db, "q1"); err != nil || len(pending) != 0 {
		t.Fatalf("pending after finish: %v len=%d", err, len(pending))
	}
}

func TestResetRunningExecutionTasksToPending(t *testing.T) {
	db := openTaskStoreTestDB(t)

	_ = SaveExecutionTaskPending(db, ExecutionTask{
		TaskID:     "task_r",
		QueueName:  "q1",
		ParamsJSON: `{}`,
		SubmitTime: time.Now(),
	})
	_ = MarkExecutionTaskRunning(db, "task_r")
	n, err := ResetRunningExecutionTasksToPending(db, "q1")
	if err != nil || n != 1 {
		t.Fatalf("reset count=%d err=%v", n, err)
	}
	rows, err := ListPendingExecutionTasks(db, "q1")
	if err != nil || len(rows) != 1 || rows[0].TaskID != "task_r" {
		t.Fatalf("reloaded pending %#v err=%v", rows, err)
	}
}

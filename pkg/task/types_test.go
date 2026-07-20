package task

import (
	"context"
	"errors"
	"testing"
)

func TestTaskStatusString(t *testing.T) {
	cases := []struct {
		s TaskStatus
		w string
	}{
		{TaskStatusPending, "pending"},
		{TaskStatusRunning, "running"},
		{TaskStatusSuccess, "success"},
		{TaskStatusFailed, "failed"},
		{TaskStatusCanceled, "canceled"},
	}
	for _, c := range cases {
		if c.s.String() != c.w {
			t.Errorf("%v.String() = %q want %q", c.s, c.s.String(), c.w)
		}
	}
}

func TestNilTaskGuards(t *testing.T) {
	var nt *Task[int, string]
	if nt.QueueAhead() != -1 {
		t.Fatalf("QueueAhead %d", nt.QueueAhead())
	}
	if nt.QueuedTotal() != 0 || nt.RunningWorkers() != 0 || nt.UnfinishedEstimate() != 0 {
		t.Fatal("nil totals")
	}
	if _, err := nt.Wait(); err == nil || err.Error() != "nil task" {
		t.Fatalf("Wait err %v", err)
	}
	nt.Cancel()
	nt.deliver("", nil)
}

func TestTaskDeliverAndWait(t *testing.T) {
	ctx := context.Background()
	h := func(ctx context.Context, p int) (string, error) {
		return "ok", nil
	}
	tsk := newTask(ctx, 1, 42, h)
	tsk.applyQueueLabels(2, 5, 3)
	if tsk.QueueAhead() != 2 || tsk.QueuedTotal() != 5 || tsk.RunningWorkers() != 3 || tsk.UnfinishedEstimate() != 8 {
		t.Fatalf("labels ahead=%d total=%d run=%d est=%d", tsk.QueueAhead(), tsk.QueuedTotal(), tsk.RunningWorkers(), tsk.UnfinishedEstimate())
	}
	go func() {
		tsk.deliver("done", nil)
	}()
	res, err := tsk.Wait()
	if err != nil || res != "done" {
		t.Fatalf("wait got %q %v", res, err)
	}
	if tsk.Status.Load().(TaskStatus) != TaskStatusSuccess {
		t.Fatalf("status %v", tsk.Status.Load())
	}
}

func TestTaskDeliverFailureAndCanceled(t *testing.T) {
	ctx := context.Background()
	h := func(ctx context.Context, p int) (string, error) { return "", errors.New("boom") }
	tsk := newTask(ctx, 0, 0, h)
	tsk.deliver("", errors.New("boom"))
	if _, err := tsk.Wait(); err == nil {
		t.Fatal("want error")
	}
	if tsk.Status.Load().(TaskStatus) != TaskStatusFailed {
		t.Fatalf("status %v", tsk.Status.Load())
	}

	tsk2 := newTask(ctx, 0, 0, h)
	tsk2.deliver("", context.Canceled)
	if tsk2.Status.Load().(TaskStatus) != TaskStatusCanceled {
		t.Fatalf("status %v", tsk2.Status.Load())
	}
}

func TestTaskApplyDeliverNil(t *testing.T) {
	var nt *Task[int, int]
	nt.applyQueueLabels(1, 2, 3)
	nt.deliver(0, nil)
}

func TestTaskCancel(t *testing.T) {
	ctx := context.Background()
	tsk := newTask(ctx, 0, 0, func(ctx context.Context, p int) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	tsk.Cancel()
	tsk.deliver(0, context.Canceled)
	if tsk.Status.Load().(TaskStatus) != TaskStatusCanceled {
		t.Fatalf("got %v", tsk.Status.Load())
	}
}

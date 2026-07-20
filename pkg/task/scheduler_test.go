package task

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewSchedulerDefaults(t *testing.T) {
	s := NewScheduler[int, string](0, nil)
	defer s.Stop()
	if s.workerCount != 4 {
		t.Fatalf("workerCount %d", s.workerCount)
	}
}

func TestSchedulerSubmitAndWait(t *testing.T) {
	s := NewScheduler[int, string](2, zap.NewNop())
	defer s.Stop()

	tsk := s.SubmitTask(context.Background(), 5, 7, func(ctx context.Context, p int) (string, error) {
		if p != 7 {
			t.Fatalf("params %d", p)
		}
		return "x", nil
	})
	res, err := tsk.Wait()
	if err != nil || res != "x" {
		t.Fatalf("wait %q %v", res, err)
	}
	if tsk.Status.Load().(TaskStatus) != TaskStatusSuccess {
		t.Fatal(tsk.Status.Load())
	}
}

func TestSchedulerCancelPendingByID(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	defer s.Stop()

	block := make(chan struct{})
	var ran atomic.Bool
	first := s.SubmitTask(context.Background(), 10, 0, func(ctx context.Context, p int) (int, error) {
		<-block
		ran.Store(true)
		return 1, nil
	})
	second := s.SubmitTask(context.Background(), 5, 0, func(ctx context.Context, p int) (int, error) {
		return 2, nil
	})

	if !s.CancelTaskByID(second.ID) {
		t.Fatal("cancel second failed")
	}
	close(block)
	if _, err := first.Wait(); err != nil {
		t.Fatal(err)
	}
	if !ran.Load() {
		t.Fatal("first did not run")
	}
	if second.Status.Load().(TaskStatus) != TaskStatusCanceled {
		t.Fatalf("second status %v", second.Status.Load())
	}
}

func TestSchedulerStatsAndSnapshot(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	defer s.Stop()

	block := make(chan struct{})
	// Higher priority runs first; block that task so the lower-priority one stays queued.
	first := s.SubmitTask(context.Background(), 10, 0, func(ctx context.Context, p int) (int, error) {
		<-block
		return 0, nil
	})
	pending := s.SubmitTask(context.Background(), 1, 0, func(ctx context.Context, p int) (int, error) {
		return 0, nil
	})
	time.Sleep(50 * time.Millisecond)

	st := s.Stats()
	if st.ChannelLen != 0 || st.Queued < 1 || st.Running != 1 {
		t.Fatalf("stats %+v", st)
	}
	if st.Unfinished != st.Queued+int(st.Running) {
		t.Fatalf("unfinished %+v", st)
	}
	if s.GetTaskPosition(pending.ID) < 0 {
		t.Fatal("position")
	}
	snap := s.PendingSnapshot()
	if len(snap) == 0 || snap[0].TaskID != pending.ID {
		t.Fatalf("snapshot %#v pending %s", snap, pending.ID)
	}
	close(block)
	_, _ = first.Wait()
	_, _ = pending.Wait()
}

func TestSchedulerHandlerError(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	defer s.Stop()
	tsk := s.SubmitTask(context.Background(), 0, 0, func(ctx context.Context, p int) (int, error) {
		return 0, errors.New("handler err")
	})
	_, err := tsk.Wait()
	if err == nil {
		t.Fatal("want err")
	}
	if tsk.Status.Load().(TaskStatus) != TaskStatusFailed {
		t.Fatal(tsk.Status.Load())
	}
}

func TestSchedulerCanceledContextBeforeRun(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	defer s.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tsk := s.SubmitTask(ctx, 0, 0, func(ctx context.Context, p int) (int, error) {
		t.Fatal("handler should not run")
		return 0, nil
	})
	_, err := tsk.Wait()
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestSchedulerStopRejectsNewAndDoubleStop(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := s.Stop(); err == nil {
		t.Fatal("second stop should error")
	}
	tsk := s.SubmitTask(context.Background(), 0, 0, func(ctx context.Context, p int) (int, error) {
		return 1, nil
	})
	_, err := tsk.Wait()
	if err == nil || err.Error() != "scheduler stopped" {
		t.Fatalf("got %v", err)
	}
}

func TestSchedulerStopDrainsPending(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	_ = s.SubmitTask(context.Background(), 0, 0, func(ctx context.Context, p int) (int, error) {
		time.Sleep(300 * time.Millisecond)
		return 0, nil
	})
	pending := s.SubmitTask(context.Background(), 0, 0, func(ctx context.Context, p int) (int, error) {
		return 1, nil
	})
	time.Sleep(20 * time.Millisecond)
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	_, err := pending.Wait()
	if err == nil || err.Error() != "scheduler stopped" {
		t.Fatalf("pending got %v", err)
	}
}

func TestSchedulerNilStop(t *testing.T) {
	var s *Scheduler[int, int]
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerRelabelWrongTypeSkipped(t *testing.T) {
	s := NewScheduler[int, int](1, zap.NewNop())
	defer s.Stop()
	s.mu.Lock()
	s.queue.Push("not-a-task", 1, "bad-id")
	s.relabelQueuedLocked()
	s.mu.Unlock()
	if s.QueueLen() != 1 {
		t.Fatalf("len %d", s.QueueLen())
	}
	s.mu.Lock()
	s.queue.Pop()
	s.mu.Unlock()
}

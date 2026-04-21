package task

import (
	"context"
	"testing"
	"time"
)

func TestPriorityQueueOrder(t *testing.T) {
	var q PriorityQueue
	q.Push("low", 1, "a")
	q.Push("high", 10, "b")
	q.Push("mid", 5, "c")
	if x := q.Pop(); x != "high" {
		t.Fatalf("want high first, got %v", x)
	}
	if x := q.Pop(); x != "mid" {
		t.Fatalf("want mid second, got %v", x)
	}
	if q.Remove("nope") {
		t.Fatal("remove should be false")
	}
	if !q.Remove("a") {
		t.Fatal("remove a")
	}
	if q.Len() != 0 {
		t.Fatalf("len %d", q.Len())
	}
}

func TestTaskPoolAddTask(t *testing.T) {
	p := NewTaskPool[int, int](&PoolOption{WorkerCount: 2, QueueSize: 4})
	task, err := p.AddTask(context.Background(), 7, func(ctx context.Context, n int) (int, error) {
		return n * 2, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-task.Result:
		if v != 14 {
			t.Fatalf("result %d", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	if e := <-task.Err; e != nil {
		t.Fatal(e)
	}
}

func TestSchedulerStats(t *testing.T) {
	s := NewScheduler[int, int](1, nil)
	st := s.Stats()
	if st.Queued < 0 || st.Running < 0 || st.Unfinished < 0 {
		t.Fatalf("stats %v", st)
	}
}

package task

import (
	"reflect"
	"testing"
)

func TestPriorityQueuePushPopOrder(t *testing.T) {
	var pq PriorityQueue
	pq.Push("low", 1, "a")
	pq.Push("high", 10, "b")
	pq.Push("mid", 5, "c")
	if got := pq.Pop(); got != "high" {
		t.Fatalf("first pop %v", got)
	}
	if got := pq.Pop(); got != "mid" {
		t.Fatalf("second pop %v", got)
	}
	if got := pq.Pop(); got != "low" {
		t.Fatalf("third pop %v", got)
	}
	if pq.Pop() != nil || pq.Len() != 0 {
		t.Fatal("empty")
	}
}

func TestPriorityQueueRemoveAndPosition(t *testing.T) {
	var pq PriorityQueue
	pq.Push("x", 1, "id-x")
	pq.Push("y", 2, "id-y")
	if pq.GetPosition("id-x") != 1 {
		t.Fatalf("position x %d", pq.GetPosition("id-x"))
	}
	if _, ok := pq.Remove("id-x"); !ok {
		t.Fatal("remove id-x")
	}
	if _, ok := pq.Remove("missing"); ok {
		t.Fatal("remove missing")
	}
	if pq.GetPosition("id-x") != -1 {
		t.Fatal("removed still found")
	}
	if pq.Len() != 1 {
		t.Fatalf("len %d", pq.Len())
	}
}

func TestPriorityQueueSnapshotAndRelabel(t *testing.T) {
	var pq PriorityQueue
	pq.Push("a", 3, "t1")
	pq.Push("b", 7, "t2")
	snap := pq.Snapshot()
	want := []QueueItemSnapshot{{TaskID: "t2", Priority: 7}, {TaskID: "t1", Priority: 3}}
	if !reflect.DeepEqual(snap, want) {
		t.Fatalf("snapshot %#v", snap)
	}
	var positions []int
	pq.RelabelQueued(func(pos int, taskID string, task any) {
		positions = append(positions, pos)
		if taskID == "" {
			t.Fatal("empty id")
		}
	})
	if !reflect.DeepEqual(positions, []int{0, 1}) {
		t.Fatalf("positions %v", positions)
	}
}

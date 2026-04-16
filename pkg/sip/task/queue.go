package task

import (
	"sort"
	"sync"
)

type queueItem struct {
	taskID   string
	priority int
	taskPtr  any
}

// QueueItemSnapshot is one pending item in priority queue.
type QueueItemSnapshot struct {
	TaskID   string
	Priority int
}

// PriorityQueue is a mutex-protected priority list (higher priority first).
type PriorityQueue struct {
	mu    sync.Mutex
	items []*queueItem
}

// Push inserts task with priority (larger runs earlier).
func (pq *PriorityQueue) Push(task any, priority int, taskID string) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &queueItem{
		taskID:   taskID,
		priority: priority,
		taskPtr:  task,
	}
	pos := sort.Search(len(pq.items), func(i int) bool {
		return pq.items[i].priority < priority
	})
	pq.items = append(pq.items[:pos], append([]*queueItem{item}, pq.items[pos:]...)...)
}

// Pop removes and returns the highest-priority task, or nil if empty.
func (pq *PriorityQueue) Pop() any {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if len(pq.items) == 0 {
		return nil
	}
	item := pq.items[0]
	pq.items = pq.items[1:]
	return item.taskPtr
}

// Remove drops a task by id; returns true if removed.
func (pq *PriorityQueue) Remove(taskID string) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	for i, item := range pq.items {
		if item.taskID == taskID {
			pq.items = append(pq.items[:i], pq.items[i+1:]...)
			return true
		}
	}
	return false
}

// GetPosition returns queue index (0 = next to run) or -1.
func (pq *PriorityQueue) GetPosition(taskID string) int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	for i, item := range pq.items {
		if item.taskID == taskID {
			return i
		}
	}
	return -1
}

// Len returns queued task count.
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

// Snapshot returns current pending order (0 is next to dispatch).
func (pq *PriorityQueue) Snapshot() []QueueItemSnapshot {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	out := make([]QueueItemSnapshot, 0, len(pq.items))
	for _, it := range pq.items {
		out = append(out, QueueItemSnapshot{
			TaskID:   it.taskID,
			Priority: it.priority,
		})
	}
	return out
}

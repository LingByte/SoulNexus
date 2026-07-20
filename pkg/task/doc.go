// Package task provides a priority scheduler with a fixed worker pool for background job scheduling
// and similar workloads.
//
// SubmitTask pushes onto an in-memory priority queue (higher priority first). Workers block on
// sync.Cond and pop directly from that queue — there is no secondary worker channel backlog.
//
// Single-node constraint: the in-memory queue is process-local. Do not run multiple API
// replicas that share the same campaign/task workload without an external broker. See
// docs/ops-single-node.md. Optional DB durability (execution_tasks) recovers crash on one node
// only.
//
// Each Task exposes QueueAhead, QueuedTotal, RunningWorkers, and UnfinishedEstimate snapshots
// taken at enqueue/relabel time so callers can log queue position and rough system load.
//
// Use Wait() to block until a task completes; completion does not use exported channels.
package task

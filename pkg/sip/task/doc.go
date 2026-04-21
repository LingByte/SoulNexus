// Package task provides a small priority scheduler over a fixed worker pool.
//
// Use NewScheduler to submit work with priorities; Stats exposes queued, running,
// and unfinished (queued + running) counts for observability.
package task

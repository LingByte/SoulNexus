package exec

import (
	"context"
	"errors"
	"github.com/LingByte/SoulNexus/pkg/agent/plan"
	"time"
)

var (
	ErrMissingRunner   = errors.New("missing runner")
	ErrInvalidWorkflow = errors.New("invalid workflow")
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskSkipped   TaskStatus = "skipped"
)

type State struct {
	Goal string

	// Outputs by task ID.
	Outputs map[string]string

	// Arbitrary artifacts by name.
	Artifacts map[string]any

	// Shared scratchpad / notes.
	Notes string

	// Per-task feedback from evaluator for retry loops.
	Feedback map[string]string
}

type TaskResult struct {
	TaskID   string
	Status   TaskStatus
	Output   string
	Attempts int
	Feedback string
	Error    string
	Started  time.Time
	Finished time.Time
	Latency  time.Duration
}

type Result struct {
	Goal        string
	TaskResults []TaskResult
	Final       State
}

type Runner interface {
	RunTask(ctx context.Context, task plan.Task, st *State) (string, error)
}

type Evaluator interface {
	Evaluate(ctx context.Context, task plan.Task, output string, st *State) (ok bool, feedback string, err error)
}

type Options struct {
	StopOnError bool
	MaxTasks    int
	MaxAttempts int
}

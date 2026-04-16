package plan

import (
	"context"
	"errors"
	"time"
)

const (
	DecomposerLLM = "llm"
)

var (
	ErrEmptyGoal            = errors.New("empty goal")
	ErrMissingLLM           = errors.New("missing llm")
	ErrInvalidPlan          = errors.New("invalid plan")
)

type Task struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Instruction string         `json:"instruction"`
	Expected    string         `json:"expected,omitempty"`
	DependsOn   []string       `json:"depends_on,omitempty"`
	CanParallel bool           `json:"can_parallel,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
}

type Plan struct {
	Goal      string    `json:"goal"`
	Tasks     []Task    `json:"tasks"`
	CreatedAt time.Time `json:"created_at"`
	By        string    `json:"by"`
	Raw       string    `json:"-"`
}

type Request struct {
	Goal string
	MaxTasks int
	LLMModel string
	Options map[string]any
}

type Decomposer interface {
	Decompose(ctx context.Context, req Request) (*Plan, error)
}

type LLM interface {
	Query(text, model string) (string, error)
}

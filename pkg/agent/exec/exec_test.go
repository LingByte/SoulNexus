package exec

import (
	"context"
	"errors"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/agent/plan"
)

type stubRunner struct {
	out map[string]string
	err map[string]error
}

func (r *stubRunner) RunTask(ctx context.Context, task plan.Task, st *State) (string, error) {
	_ = ctx
	if e := r.err[task.ID]; e != nil {
		return "", e
	}
	v := r.out[task.ID]
	return v, nil
}

func TestTopoOrder(t *testing.T) {
	tasks := []plan.Task{
		{ID: "a"},
		{ID: "b", DependsOn: []string{"a"}},
		{ID: "c", DependsOn: []string{"b"}},
	}
	ord, err := topoOrder(tasks)
	if err != nil {
		t.Fatalf("topoOrder err: %v", err)
	}
	if ord[0].ID != "a" || ord[1].ID != "b" || ord[2].ID != "c" {
		t.Fatalf("unexpected order")
	}
}

func TestExecutor_StopOnError(t *testing.T) {
	e := &Executor{Runner: &stubRunner{out: map[string]string{"a": "ok"}, err: map[string]error{"b": errors.New("x")}}, Opts: Options{StopOnError: true, MaxAttempts: 1}}
	p := &plan.Plan{Goal: "g", Tasks: []plan.Task{{ID: "a"}, {ID: "b", DependsOn: []string{"a"}}}}
	res, err := e.Run(context.Background(), p)
	if err == nil {
		t.Fatalf("expected err")
	}
	if res == nil || len(res.TaskResults) != 2 {
		t.Fatalf("expected 2 task results")
	}
}

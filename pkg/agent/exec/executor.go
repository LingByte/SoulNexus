package exec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/agent/plan"
)

type Executor struct {
	Runner    Runner
	Evaluator Evaluator
	Opts      Options
}

func (e *Executor) Run(ctx context.Context, p *plan.Plan) (*Result, error) {
	if e == nil || e.Runner == nil {
		return nil, ErrMissingRunner
	}
	if p == nil {
		return nil, ErrInvalidWorkflow
	}

	opts := e.Opts
	if opts.MaxTasks <= 0 {
		opts.MaxTasks = 64
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 2
	}
	if len(p.Tasks) > opts.MaxTasks {
		return nil, fmt.Errorf("too many tasks: %d > %d", len(p.Tasks), opts.MaxTasks)
	}

	ordered, err := topoOrder(p.Tasks)
	if err != nil {
		return nil, err
	}

	st := State{Goal: p.Goal, Outputs: map[string]string{}, Artifacts: map[string]any{}, Feedback: map[string]string{}}
	res := &Result{Goal: p.Goal, Final: st}

	for _, t := range ordered {
		tr := TaskResult{TaskID: t.ID, Status: TaskRunning, Started: time.Now()}
		attempts := 0
		var out string
		var runErr error
		var feedback string
		for attempts < opts.MaxAttempts {
			attempts++
			tr.Attempts = attempts
			out, runErr = e.Runner.RunTask(ctx, t, &st)
			if runErr != nil {
				break
			}
			ok := true
			var evalErr error
			if e.Evaluator != nil {
				ok, feedback, evalErr = e.Evaluator.Evaluate(ctx, t, out, &st)
				if evalErr != nil {
					runErr = evalErr
					break
				}
			}
			if ok {
				feedback = ""
				break
			}
			// Not OK: store feedback and retry.
			if strings.TrimSpace(feedback) == "" {
				feedback = "output did not meet expected"
			}
			st.Feedback[t.ID] = feedback
			tr.Feedback = feedback
		}

		tr.Finished = time.Now()
		tr.Latency = tr.Finished.Sub(tr.Started)
		if runErr != nil {
			tr.Status = TaskFailed
			tr.Error = runErr.Error()
			res.TaskResults = append(res.TaskResults, tr)
			res.Final = st
			if opts.StopOnError {
				return res, runErr
			}
			continue
		}
		tr.Status = TaskSucceeded
		tr.Output = out
		st.Outputs[t.ID] = out
		// Clear feedback on success.
		delete(st.Feedback, t.ID)
		res.TaskResults = append(res.TaskResults, tr)
		res.Final = st
	}

	return res, nil
}

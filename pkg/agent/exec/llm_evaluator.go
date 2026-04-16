package exec

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/agent/plan"
	"github.com/LingByte/SoulNexus/pkg/llm"
)

type LLMTaskEvaluator struct {
	LLM   llm.LLMHandler
	Model string
}

type evalOut struct {
	OK       bool   `json:"ok"`
	Feedback string `json:"feedback"`
}

func (e *LLMTaskEvaluator) Evaluate(ctx context.Context, task plan.Task, output string, st *State) (bool, string, error) {
	_ = ctx
	if strings.TrimSpace(task.Expected) == "" {
		return true, "", nil
	}
	if e == nil || e.LLM == nil {
		return false, "missing evaluator llm", ErrMissingRunner
	}

	model := strings.TrimSpace(e.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}

	prompt := buildEvalPrompt(st.Goal, task, output)
	raw, err := e.LLM.Query(prompt, model)
	if err != nil {
		return false, "", err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, "empty evaluation", errors.New("empty evaluation")
	}

	// Try to parse JSON.
	j := extractJSON(raw)
	if strings.TrimSpace(j) == "" {
		j = raw
	}
	var out evalOut
	if err := json.Unmarshal([]byte(j), &out); err != nil {
		// Fallback heuristic: must contain "ok" or "pass".
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "ok") || strings.Contains(lower, "pass") || strings.Contains(raw, "通过") {
			return true, "", nil
		}
		return false, "evaluator parse failed", nil
	}
	return out.OK, strings.TrimSpace(out.Feedback), nil
}

func buildEvalPrompt(goal string, task plan.Task, output string) string {
	b := strings.Builder{}
	b.WriteString("你是任务验收器。判断任务输出是否满足 expected。\n")
	b.WriteString("只输出JSON，不要解释。schema: {\"ok\":bool,\"feedback\":string}\n\n")
	b.WriteString("Goal: ")
	b.WriteString(strings.TrimSpace(goal))
	b.WriteString("\n")
	b.WriteString("Task: ")
	b.WriteString(strings.TrimSpace(task.Title))
	b.WriteString("\n")
	b.WriteString("Expected: ")
	b.WriteString(strings.TrimSpace(task.Expected))
	b.WriteString("\n\n")
	b.WriteString("Output:\n")
	b.WriteString(strings.TrimSpace(output))
	b.WriteString("\n")
	return b.String()
}

// Local JSON extractor for evaluator outputs.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	l := strings.Index(s, "{")
	r := strings.LastIndex(s, "}")
	if l >= 0 && r > l {
		return strings.TrimSpace(s[l : r+1])
	}
	return ""
}

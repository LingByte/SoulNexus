package exec

import (
	"context"
	"github.com/LingByte/SoulNexus/pkg/agent/plan"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"strings"
)

type LLMTaskRunner struct {
	LLM   llm.LLMHandler
	Model string
}

func (r *LLMTaskRunner) RunTask(ctx context.Context, task plan.Task, st *State) (string, error) {
	_ = ctx
	if r == nil || r.LLM == nil {
		return "", ErrMissingRunner
	}

	model := strings.TrimSpace(r.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}

	// Minimal execution prompt. We provide the global goal, task instruction, and prior outputs.
	// The runner returns plain text output.
	b := strings.Builder{}
	b.WriteString("你是一个任务执行器。根据目标和当前任务指令完成任务。\n")
	b.WriteString("约束：如果任务需要依赖上游任务输出，请仅使用提供的上游输出；不要编造不存在的信息。\n\n")
	b.WriteString("总目标: ")
	b.WriteString(strings.TrimSpace(st.Goal))
	b.WriteString("\n")
	b.WriteString("当前任务: ")
	b.WriteString(strings.TrimSpace(task.Title))
	b.WriteString("\n")
	b.WriteString("任务指令: ")
	b.WriteString(strings.TrimSpace(task.Instruction))
	b.WriteString("\n\n")
	if len(task.DependsOn) > 0 {
		b.WriteString("上游输出：\n")
		for _, dep := range task.DependsOn {
			if dep == "" {
				continue
			}
			b.WriteString("[")
			b.WriteString(dep)
			b.WriteString("] ")
			b.WriteString(strings.TrimSpace(st.Outputs[dep]))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if st != nil && st.Feedback != nil {
		if fb := strings.TrimSpace(st.Feedback[task.ID]); fb != "" {
			b.WriteString("验收反馈（上一轮未通过原因）：\n")
			b.WriteString(fb)
			b.WriteString("\n\n")
		}
	}
	b.WriteString("请输出该任务的结果：\n")

	out, err := r.LLM.Query(b.String(), model)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

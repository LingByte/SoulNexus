package plan

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type LLMDecomposer struct {
	LLM      LLM
	Model    string
	MaxTasks int
}

func (d *LLMDecomposer) Decompose(ctx context.Context, req Request) (*Plan, error) {
	_ = ctx
	goal := strings.TrimSpace(req.Goal)
	if goal == "" {
		return nil, ErrEmptyGoal
	}
	if d == nil || d.LLM == nil {
		return nil, ErrMissingLLM
	}

	maxTasks := req.MaxTasks
	if maxTasks <= 0 {
		maxTasks = d.MaxTasks
	}
	if maxTasks <= 0 {
		maxTasks = 6
	}

	model := strings.TrimSpace(req.LLMModel)
	if model == "" {
		model = strings.TrimSpace(d.Model)
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	prompt := buildPrompt(goal, maxTasks)
	out, err := d.LLM.Query(prompt, model)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, errors.New("llm returned empty plan")
	}

	jsonText := ExtractJSON(out)
	if strings.TrimSpace(jsonText) == "" {
		jsonText = out
	}

	var p Plan
	if err := json.Unmarshal([]byte(jsonText), &p); err != nil {
		return nil, err
	}
	p.Goal = strings.TrimSpace(p.Goal)
	if p.Goal == "" {
		p.Goal = goal
	}
	p.CreatedAt = time.Now()
	p.By = DecomposerLLM
	p.Raw = out

	if err := Validate(&p, maxTasks); err != nil {
		return nil, err
	}
	return &p, nil
}

func buildPrompt(goal string, maxTasks int) string {
	return "你是任务拆分器。请根据目标生成任务列表。\n" +
		"要求：\n" +
		"- 简单任务不要拆分，但仍然要输出 1 个 task。\n" +
		"- 复杂任务可以拆分为多个 task，总数不超过 " + itoa(maxTasks) + "。\n" +
		"- task 要可执行、描述清晰，按依赖顺序列出。\n" +
		"- 为每个 task 给出 expected（验收标准/期望输出特征），用于执行后自检。\n" +
		"- 只输出 JSON，不要解释，不要 markdown。\n" +
		"JSON schema: {\"goal\":string,\"tasks\":[{\"id\":string,\"title\":string,\"instruction\":string,\"expected\":string,\"depends_on\":[string],\"can_parallel\":bool,\"input\":object}]}\n" +
		"目标: " + goal + "\n"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		d := n % 10
		buf = append(buf, byte('0'+d))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

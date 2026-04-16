package plan

import (
	"context"
	"testing"
)

type fakeLLM struct{ resp string }

func (f *fakeLLM) Query(text, model string) (string, error) { return f.resp, nil }

func TestLLMDecomposer(t *testing.T) {
	llm := &fakeLLM{resp: `{"goal":"g","tasks":[{"id":"task_1","title":"t","instruction":"do","input":{}}]}`}
	d := &LLMDecomposer{LLM: llm, Model: "fake", MaxTasks: 3}
	p, err := d.Decompose(context.Background(), Request{Goal: "g", MaxTasks: 3, LLMModel: "fake"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(p.Tasks) != 1 {
		t.Fatalf("expected 1 task")
	}
}

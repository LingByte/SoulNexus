package llm

import (
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestMergeStreamToolCallDeltas(t *testing.T) {
	idx0 := 0
	accum := mergeStreamToolCallDeltas(nil, []openai.ToolCall{
		{Index: &idx0, ID: "call_1", Type: openai.ToolTypeFunction, Function: openai.FunctionCall{Name: "good"}},
	})
	accum = mergeStreamToolCallDeltas(accum, []openai.ToolCall{
		{Index: &idx0, Function: openai.FunctionCall{Name: "bye", Arguments: `{"reason":"bye"}`}},
	})
	if len(accum) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(accum))
	}
	if accum[0].ID != "call_1" {
		t.Fatalf("id: %s", accum[0].ID)
	}
	if accum[0].Function.Name != "goodbye" {
		t.Fatalf("name: %s", accum[0].Function.Name)
	}
	if accum[0].Function.Arguments != `{"reason":"bye"}` {
		t.Fatalf("args: %s", accum[0].Function.Arguments)
	}
}

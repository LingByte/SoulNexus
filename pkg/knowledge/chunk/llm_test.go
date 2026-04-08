package chunk

import (
	"context"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/stretchr/testify/assert"
)

type fakeLLM struct {
	resp string
	err  error
}

func (f *fakeLLM) Query(text, model string) (string, error) {
	_ = text
	_ = model
	return f.resp, f.err
}

func (f *fakeLLM) Provider() string { return "fake" }
func (f *fakeLLM) QueryWithOptions(text string, options llm.QueryOptions) (string, error) {
	_ = text
	_ = options
	return f.resp, f.err
}
func (f *fakeLLM) QueryStream(text string, options llm.QueryOptions, callback func(segment string, isComplete bool) error) (string, error) {
	_ = text
	_ = options
	if callback != nil {
		_ = callback(f.resp, false)
		_ = callback("", true)
	}
	return f.resp, f.err
}
func (f *fakeLLM) RegisterFunctionTool(name, description string, parameters interface{}, callback llm.FunctionToolCallback) {
	_, _, _, _ = name, description, parameters, callback
}
func (f *fakeLLM) RegisterFunctionToolDefinition(def *llm.FunctionToolDefinition) { _ = def }
func (f *fakeLLM) GetFunctionTools() []interface{}                                 { return nil }
func (f *fakeLLM) ListFunctionTools() []string                                     { return nil }
func (f *fakeLLM) GetLastUsage() (llm.Usage, bool)                                 { return llm.Usage{}, false }
func (f *fakeLLM) ResetMessages()                                                   {}
func (f *fakeLLM) SetSystemPrompt(systemPrompt string)                              { _ = systemPrompt }
func (f *fakeLLM) GetMessages() []llm.Message                                       { return nil }
func (f *fakeLLM) Interrupt()                                                       {}
func (f *fakeLLM) Hangup()                                                          {}

var _ llm.LLMProvider = (*fakeLLM)(nil)

func TestLLMChunker_Chunk_PureJSON(t *testing.T) {
	c := &LLMChunker{LLM: &fakeLLM{resp: `[{"title":"A","text":"hello"},{"title":"B","text":"world"}]`}}
	chunks, err := c.Chunk(context.Background(), "input", &ChunkOptions{MaxChars: 100})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(chunks))
	assert.Equal(t, "A", chunks[0].Title)
	assert.Equal(t, "hello", chunks[0].Text)
}

func TestLLMChunker_Chunk_FencedJSON(t *testing.T) {
	c := &LLMChunker{LLM: &fakeLLM{resp: "```json\n[{\"title\":\"A\",\"text\":\"hello\"}]\n```"}}
	chunks, err := c.Chunk(context.Background(), "input", &ChunkOptions{MaxChars: 100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, "hello", chunks[0].Text)
}

func TestLLMChunker_Chunk_NoiseAroundJSON(t *testing.T) {
	c := &LLMChunker{LLM: &fakeLLM{resp: "Sure, here you go:\n[{\"title\":\"A\",\"text\":\"hello\"}]\nThanks"}}
	chunks, err := c.Chunk(context.Background(), "input", &ChunkOptions{MaxChars: 100})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, "A", chunks[0].Title)
}

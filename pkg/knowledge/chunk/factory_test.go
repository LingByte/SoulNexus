package chunk

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/stretchr/testify/assert"
)

type fakeChunkLLM struct{}

func (f *fakeChunkLLM) Query(text, model string) (string, error) {
	return `[{"title":"t","text":"x"}]`, nil
}
func (f *fakeChunkLLM) Provider() string { return "fake" }
func (f *fakeChunkLLM) QueryWithOptions(text string, options *llm.QueryOptions) (*llm.QueryResponse, error) {
	_ = text
	_ = options
	return &llm.QueryResponse{Choices: []llm.QueryChoice{{Content: `[{"title":"t","text":"x"}]`}}}, nil
}
func (f *fakeChunkLLM) QueryStream(text string, options *llm.QueryOptions, callback func(segment string, isComplete bool) error) (*llm.QueryResponse, error) {
	_ = text
	_ = options
	if callback != nil {
		_ = callback(`[{"title":"t","text":"x"}]`, false)
		_ = callback("", true)
	}
	return &llm.QueryResponse{Choices: []llm.QueryChoice{{Content: `[{"title":"t","text":"x"}]`}}}, nil
}
func (f *fakeChunkLLM) RegisterFunctionTool(name, description string, parameters interface{}, callback llm.FunctionToolCallback) {
	_, _, _, _ = name, description, parameters, callback
}
func (f *fakeChunkLLM) RegisterFunctionToolDefinition(def *llm.FunctionToolDefinition) { _ = def }
func (f *fakeChunkLLM) GetFunctionTools() []interface{}                                { return nil }
func (f *fakeChunkLLM) ListFunctionTools() []string                                    { return nil }
func (f *fakeChunkLLM) Interrupt()                                                     {}
func (f *fakeChunkLLM) Hangup()                                                        {}

func TestFactory_DefaultRule(t *testing.T) {
	c, err := New("", nil)
	assert.NoError(t, err)
	assert.Equal(t, "rule", c.Provider())
}

func TestFactory_Rule(t *testing.T) {
	c, err := New("rule", nil)
	assert.NoError(t, err)
	assert.Equal(t, "rule", c.Provider())
}

func TestFactory_LLM_MissingLLM(t *testing.T) {
	_, err := New("llm", &FactoryOptions{})
	assert.Error(t, err)
}

func TestFactory_LLM_OK(t *testing.T) {
	c, err := New("llm", &FactoryOptions{LLM: &fakeChunkLLM{}, Model: "m"})
	assert.NoError(t, err)
	assert.Equal(t, "llm", c.Provider())
}

func TestFactory_Unsupported(t *testing.T) {
	_, err := New("nope", nil)
	assert.ErrorIs(t, err, ErrUnsupportedChunkerType)
}

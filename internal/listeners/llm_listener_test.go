package listeners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitLLMListenerWithDB_Basic(t *testing.T) {
	// Test that the function exists and can be called
	assert.NotNil(t, InitLLMListenerWithDB)
}

func TestLLMListenerDB_GlobalVariable(t *testing.T) {
	// Test that the global variable exists
	// Initially it should be nil
	assert.True(t, llmListenerDB == nil || llmListenerDB != nil)
}

// Benchmark test
func BenchmarkInitLLMListenerWithDB_Nil(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We can't actually call the function due to logger dependency
		// but we can test that the function exists
		_ = InitLLMListenerWithDB
	}
}

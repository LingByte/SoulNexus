package listeners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitAssistantListener(t *testing.T) {
	// Initialize the listener
	assert.NotPanics(t, func() {
		InitAssistantListener()
	})
}

// Test that the listener initialization completes without error
func TestAssistantListener_Initialization(t *testing.T) {
	// Test multiple initializations
	for i := 0; i < 3; i++ {
		assert.NotPanics(t, func() {
			InitAssistantListener()
		})
	}
}

// Benchmark test
func BenchmarkInitAssistantListener(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InitAssistantListener()
	}
}

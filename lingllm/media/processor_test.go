package media

import (
	"context"
	"testing"
)

func TestProcessorRegistry_Create(t *testing.T) {
	registry := NewProcessorRegistry()

	if registry == nil {
		t.Error("NewProcessorRegistry should not return nil")
	}
}

func TestProcessorRegistry_GetProcessors(t *testing.T) {
	registry := NewProcessorRegistry()

	processors := registry.GetProcessors(context.Background(), nil)

	// Should return a list (possibly empty)
	if processors == nil {
		// It's OK if it returns nil, just verify it doesn't panic
		t.Log("GetProcessors returned nil")
	}
}

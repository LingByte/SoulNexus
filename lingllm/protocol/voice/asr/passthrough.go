package asr

import (
	"context"
)

// PassthroughComponent passes data through unchanged (useful for testing/logging).
type PassthroughComponent struct {
	name string
}

// NewPassthroughComponent creates a passthrough component.
func NewPassthroughComponent(name string) *PassthroughComponent {
	if name == "" {
		name = "passthrough"
	}
	return &PassthroughComponent{name: name}
}

// Name returns the component name.
func (p *PassthroughComponent) Name() string {
	return p.name
}

// Process passes data through unchanged.
func (p *PassthroughComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	return data, true, nil
}

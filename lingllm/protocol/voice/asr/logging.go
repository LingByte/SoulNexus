package asr

import (
	"context"
	"fmt"
)

// LoggingComponent logs data passing through (useful for debugging).
type LoggingComponent struct {
	name   string
	logger func(string) // Logging callback
}

// NewLoggingComponent creates a logging component.
func NewLoggingComponent(name string, logger func(string)) *LoggingComponent {
	if name == "" {
		name = "logging"
	}
	return &LoggingComponent{
		name:   name,
		logger: logger,
	}
}

// Name returns the component name.
func (l *LoggingComponent) Name() string {
	return l.name
}

// Process logs data and passes it through.
func (l *LoggingComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	if l.logger != nil {
		l.logger(fmt.Sprintf("[%s] Processing data: %T", l.name, data))
	}
	return data, true, nil
}

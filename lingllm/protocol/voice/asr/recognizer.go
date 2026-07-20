package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// RecognizerComponent is the terminal input-stage that feeds PCM into an ASR
// engine. Recognition results are delivered via OnTranscript / OnError callbacks
// configured before Start; the session wires those into ProcessOutput and events.
type RecognizerComponent struct {
	engine Engine

	onTranscript func(text string, isFinal bool)
	onError      func(err error, fatal bool)

	startOnce sync.Once
	startErr  error
	closed    atomic.Bool
}

// NewRecognizerComponent creates a recognizer pipeline stage.
func NewRecognizerComponent(engine Engine) (*RecognizerComponent, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	return &RecognizerComponent{engine: engine}, nil
}

// Name returns the component identifier.
func (c *RecognizerComponent) Name() string { return "recognizer" }

// SetOnTranscript registers the transcript sink.
func (c *RecognizerComponent) SetOnTranscript(fn func(text string, isFinal bool)) {
	c.onTranscript = fn
}

// SetOnError registers the error sink.
func (c *RecognizerComponent) SetOnError(fn func(err error, fatal bool)) {
	c.onError = fn
}

// Start connects the underlying engine. Safe to call multiple times.
func (c *RecognizerComponent) Start() error {
	if c == nil {
		return ErrNilEngine
	}
	c.startOnce.Do(func() {
		c.engine.OnResult(func(text string, isFinal bool) {
			if c.onTranscript != nil {
				c.onTranscript(text, isFinal)
			}
		})
		c.engine.OnError(func(err error, fatal bool) {
			if c.onError != nil {
				c.onError(err, fatal)
			}
		})
		c.startErr = c.engine.Start()
	})
	return c.startErr
}

// Process feeds one PCM frame into the recognizer. Audio does not pass to
// downstream input stages (this is the terminal uplink stage).
func (c *RecognizerComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	if c == nil {
		return nil, false, ErrNilEngine
	}
	if c.closed.Load() {
		return nil, false, ErrPipelineClosed
	}
	if err := c.Start(); err != nil {
		return nil, false, err
	}

	pcm, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if len(pcm) == 0 {
		return nil, true, nil
	}

	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}

	if err := c.engine.SendPCM(pcm, false); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

// Close stops the recognizer engine.
func (c *RecognizerComponent) Close() error {
	if c == nil || c.closed.Swap(true) {
		return nil
	}
	_ = c.engine.SendPCM(nil, true)
	c.engine.Stop()
	return nil
}

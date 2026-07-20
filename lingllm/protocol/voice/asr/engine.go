package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/lingllm/recognizer"
)

// Engine is the minimal ASR contract the voice pipeline consumes.
// It abstracts over recognizer.Recognizer and other vendor backends.
type Engine interface {
	Start() error
	Stop()
	SendPCM(pcm []byte, end bool) error
	OnResult(callback func(text string, isFinal bool))
	OnError(callback func(err error, fatal bool))
}

// RecognizerEngine adapts recognizer.Recognizer to Engine.
type RecognizerEngine struct {
	r *recognizer.Recognizer
}

// NewRecognizerEngine wraps a recognizer.Recognizer.
func NewRecognizerEngine(r *recognizer.Recognizer) *RecognizerEngine {
	return &RecognizerEngine{r: r}
}

func (e *RecognizerEngine) Start() error {
	if e == nil || e.r == nil {
		return ErrNilEngine
	}
	return e.r.Start()
}

func (e *RecognizerEngine) Stop() {
	if e == nil || e.r == nil {
		return
	}
	e.r.Stop()
}

func (e *RecognizerEngine) SendPCM(pcm []byte, end bool) error {
	if e == nil || e.r == nil {
		return ErrNilEngine
	}
	return e.r.SendAudioFrame(pcm, end)
}

func (e *RecognizerEngine) OnResult(callback func(text string, isFinal bool)) {
	if e == nil || e.r == nil {
		return
	}
	e.r.OnResult(func(res *recognizer.Result) {
		if callback == nil || res == nil {
			return
		}
		if res.Error != nil {
			return
		}
		callback(res.Text, res.IsFinal)
	})
}

func (e *RecognizerEngine) OnError(callback func(err error, fatal bool)) {
	if e == nil || e.r == nil {
		return
	}
	e.r.OnError(func(err error) {
		if callback != nil && err != nil {
			callback(err, false)
		}
	})
}

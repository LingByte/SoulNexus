package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Result struct {
	Text      string    `json:"text"`
	IsFinal   bool      `json:"is_final"`
	Timestamp time.Time `json:"timestamp"`
	Error     error     `json:"error,omitempty"`
}

// ResultCallback defines the callback interface for handling recognition results
type ResultCallback func(*Result)

// TimeoutConfig holds timeout settings
type TimeoutConfig struct {
	Send time.Duration
	Read time.Duration
}

// onResultFunc is the result callback type
type onResultFunc func(*Result)

// onErrorFunc is the error callback type
type onErrorFunc func(error)

type Recognizer struct {
	client *Client
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	// Audio buffer and configuration
	pendingAudio     []byte
	targetBufferSize int
	audioConfig      AudioConfig
	timeoutConfig    TimeoutConfig

	// Callback functions
	onResult onResultFunc
	onError  onErrorFunc

	// State management
	isEndFrameSent bool
}

func NewRecognizer(config *Config) *Recognizer {
	// Default buffer 100ms of audio data
	bufferDurationMs := config.Buffer.SegmentDurationMs
	if bufferDurationMs == 0 {
		bufferDurationMs = 100
	}

	// Calculate buffer size
	bufferSize := config.Audio.Rate * config.Audio.Bits / 8 * config.Audio.Channel * bufferDurationMs / 1000

	return &Recognizer{
		client:           NewClient(config),
		pendingAudio:     make([]byte, 0, bufferSize),
		targetBufferSize: bufferSize,
		audioConfig: AudioConfig{
			Rate:    config.Audio.Rate,
			Bits:    config.Audio.Bits,
			Channel: config.Audio.Channel,
		},
		timeoutConfig: TimeoutConfig{
			Send: 10 * time.Second,
			Read: 30 * time.Second,
		},
	}
}

// OnResult registers the callback for recognition results
func (r *Recognizer) OnResult(callback onResultFunc) {
	r.onResult = callback
}

// OnError registers the callback for error handling
func (r *Recognizer) OnError(callback onErrorFunc) {
	r.onError = callback
}

func (r *Recognizer) Start() error {
	r.ctx, r.cancel = context.WithCancel(context.Background())

	// Set client error callback to forward underlying errors to recognizer
	r.client.SetErrorCallback(func(err error) {
		if r.onError != nil {
			r.onError(err)
		}
	})

	r.client.SetTimeouts(r.timeoutConfig.Send, r.timeoutConfig.Read)
	if err := r.client.Connect(r.ctx); err != nil {
		return err
	}

	go r.receiveResults()

	return nil
}

func (r *Recognizer) SendAudioFrame(frame []byte, end bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If end frame has been sent, discard all subsequent frames
	if r.isEndFrameSent {
		return nil
	}

	// If it's an end frame, send all buffered data immediately
	if end {
		if len(r.pendingAudio) > 0 {
			if err := r.flushPendingAudioLocked(); err != nil {
				return err
			}
		}

		r.isEndFrameSent = true
		return r.client.SendAudioFrame(&AudioFrame{IsEnd: true})
	}

	r.pendingAudio = append(r.pendingAudio, frame...)
	if len(r.pendingAudio) >= r.targetBufferSize {
		return r.flushPendingAudioLocked()
	}

	return nil
}

// flushPendingAudioLocked sends the current buffer content
func (r *Recognizer) flushPendingAudioLocked() error {
	if len(r.pendingAudio) == 0 {
		return nil
	}

	toSend := make([]byte, len(r.pendingAudio))
	copy(toSend, r.pendingAudio)
	r.pendingAudio = r.pendingAudio[:0]

	return r.client.SendAudioFrame(&AudioFrame{Data: toSend})
}

func (r *Recognizer) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.client.Close()
}

// receiveResults handles response reading and conversion
func (r *Recognizer) receiveResults() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			resp, err := r.client.ReceiveResult()
			if errors.Is(err, ErrClientClosed) {
				return
			}

			result := r.convertResponseToResult(resp)

			if result.Error != nil && r.onError != nil {
				r.onError(result.Error)
			}

			if r.onResult != nil {
				r.onResult(result)
			}
		}
	}
}

// convertResponseToResult converts ASR response to Result
func (r *Recognizer) convertResponseToResult(resp *Response) *Result {
	result := &Result{
		IsFinal:   resp.IsLastPackage,
		Timestamp: time.Now(),
	}

	if resp.Code != 0 {
		result.Error = fmt.Errorf("asr error code: %d, msg: %v", resp.Code, resp.PayloadMsg)
	}

	if resp.PayloadMsg != nil && resp.PayloadMsg.Result.Text != "" {
		result.Text = resp.PayloadMsg.Result.Text
	}

	if resp.Err != nil {
		result.Error = resp.Err
	}

	return result
}

func (r *Recognizer) GetTraceID() string {
	if r != nil && r.client != nil {
		return r.client.GetTraceID()
	}
	return ""
}

package recognizer

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

type Recognizer struct {
	client *Client
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	// Audio buffer
	audioBuffer   []byte
	bufferSize    int // Target buffer size (bytes)
	sampleRate    int
	bitsPerSample int
	channels      int

	// Callback functions
	resultCallback ResultCallback
	errorCallback  func(error)

	// Control parameters
	sendTimeout time.Duration
	readTimeout time.Duration

	// Status flags
	hasSentEndFrame bool
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
		client:        NewClient(config),
		audioBuffer:   make([]byte, 0, bufferSize),
		bufferSize:    bufferSize,
		sampleRate:    config.Audio.Rate,
		bitsPerSample: config.Audio.Bits,
		channels:      config.Audio.Channel,
		sendTimeout:   10 * time.Second,
		readTimeout:   30 * time.Second,
	}
}

// SetResultCallback sets the callback function for handling recognition results
func (r *Recognizer) SetResultCallback(callback ResultCallback) {
	r.resultCallback = callback
}

// SetErrorCallback sets the callback function for handling errors
func (r *Recognizer) SetErrorCallback(callback func(error)) {
	r.errorCallback = callback
}

func (r *Recognizer) Start() error {
	r.ctx, r.cancel = context.WithCancel(context.Background())

	// Set client error callback to forward underlying errors to recognizer
	r.client.SetErrorCallback(func(err error) {
		if r.errorCallback != nil {
			r.errorCallback(err)
		}
	})

	r.client.SetTimeouts(r.sendTimeout, r.readTimeout)
	if err := r.client.Connect(r.ctx); err != nil {
		return err
	}

	go r.resultReceiver()

	return nil
}

func (r *Recognizer) SendAudioFrame(frame []byte, end bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If end frame has been sent, discard all subsequent frames
	if r.hasSentEndFrame {
		return nil
	}

	// If it's an end frame, send all buffered data immediately
	if end {
		if len(r.audioBuffer) > 0 {
			if err := r.sendAudioDataLocked(r.audioBuffer); err != nil {
				return err
			}
			r.audioBuffer = r.audioBuffer[:0]
		}

		audioPacket := &AudioFrame{
			Data:  nil,
			IsEnd: true,
		}
		r.hasSentEndFrame = true
		return r.client.SendAudioFrame(audioPacket)
	}

	r.audioBuffer = append(r.audioBuffer, frame...)
	if len(r.audioBuffer) >= r.bufferSize {
		return r.flushBufferLocked()
	}

	return nil
}

// flushBufferLocked sends the current buffer content
func (r *Recognizer) flushBufferLocked() error {
	if len(r.audioBuffer) == 0 {
		return nil
	}

	toSend := make([]byte, len(r.audioBuffer))
	copy(toSend, r.audioBuffer)
	r.audioBuffer = r.audioBuffer[:0]

	return r.sendAudioDataLocked(toSend)
}

// sendAudioDataLocked sends audio data to client
func (r *Recognizer) sendAudioDataLocked(data []byte) error {
	audioPacket := &AudioFrame{
		Data:  data,
		IsEnd: false,
	}
	return r.client.SendAudioFrame(audioPacket)
}

func (r *Recognizer) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.client.Close()
}

// resultReceiver handles response reading and conversion
func (r *Recognizer) resultReceiver() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			resp, err := r.client.ReceiveResult()
			if errors.Is(err, ErrClientClosed) {
				return
			}

			if resp.Code != 0 && r.errorCallback != nil {
				resp.Err = fmt.Errorf("code: %d , msg: %v", resp.Code, resp.PayloadMsg)
				r.errorCallback(resp.Err)
			}

			result := &Result{
				Text:      "",
				IsFinal:   resp.IsLastPackage,
				Timestamp: time.Now(),
				Error:     resp.Err,
			}

			if resp.PayloadMsg != nil && resp.PayloadMsg.Result.Text != "" {
				result.Text = resp.PayloadMsg.Result.Text
			}

			if r.resultCallback != nil {
				r.resultCallback(result)
			}
		}
	}
}

func (r *Recognizer) GetTraceID() string {
	if r != nil && r.client != nil {
		return r.client.GetTraceID()
	}
	return ""
}

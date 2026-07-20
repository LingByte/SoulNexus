package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var ErrClientClosed = errors.New("asr client closed")

type Client struct {
	config  *Config
	seq     int
	traceID string

	conn     *websocket.Conn
	mu       sync.Mutex
	isClosed bool

	// Channel management
	audioQueue   chan *AudioFrame
	resultQueue  chan *Response
	stopSignal   chan struct{}
	writeStopSig chan struct{}
	readStopSig  chan struct{}

	// Timeout configuration
	sendTimeout time.Duration
	recvTimeout time.Duration

	// Error handler
	errorHandler func(error)
}

type AudioFrame struct {
	IsEnd bool
	Data  []byte
}

func NewClient(config *Config) *Client {
	return &Client{
		seq:          1,
		config:       config,
		audioQueue:   make(chan *AudioFrame, 100),
		resultQueue:  make(chan *Response, 100),
		stopSignal:   make(chan struct{}),
		writeStopSig: make(chan struct{}, 1),
		readStopSig:  make(chan struct{}, 1),
		sendTimeout:  10 * time.Second,
		recvTimeout:  30 * time.Second,
	}
}

// SetTimeouts sets the timeouts for send and receive operations
func (c *Client) SetTimeouts(sendTimeout, recvTimeout time.Duration) {
	c.sendTimeout = sendTimeout
	c.recvTimeout = recvTimeout
}

// SetErrorCallback sets the error handler function
func (c *Client) SetErrorCallback(handler func(error)) {
	c.errorHandler = handler
}

// isNormalCloseError checks if the error is a normal WebSocket close error
func (c *Client) isNormalCloseError(err error) bool {
	// Check if it's a WebSocket normal close error
	var closeError *websocket.CloseError
	if errors.As(err, &closeError) {
		switch closeError.Code {
		case websocket.CloseNormalClosure,
			websocket.CloseGoingAway,
			websocket.CloseNoStatusReceived:
			return true
		}
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	// Check if it's a connection closed error
	if err.Error() == "connection is closed" {
		return true
	}

	return false
}

// logError logs error with operation and traceID
func (c *Client) logError(err error, operation string) {
	logrus.WithFields(logrus.Fields{
		"error":     err.Error(),
		"operation": operation,
		"traceID":   c.traceID,
	}).Error("connection error occurred")
}

// Connect establishes WebSocket connection and authenticates
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return errors.New("client is closed")
	}

	if c.config.URL == "" {
		return errors.New("url is empty")
	}

	// Connect and authenticate
	header := BuildAuthHeader(c.config.Auth)
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, c.config.URL, header)
	if err != nil {
		if ctx.Err() == context.Canceled || strings.Contains(err.Error(), "operation was canceled") {
			return nil
		}
		return fmt.Errorf("dial websocket err: %w", err)
	}
	c.traceID = resp.Header.Get("X-Tt-Logid")
	c.conn = conn

	// Send initial full client request
	if err := c.sendInitialRequest(); err != nil {
		_ = conn.Close()
		return fmt.Errorf("send initial request err: %w", err)
	}

	go c.handleWriteLoop()
	go c.handleReadLoop()

	return nil
}

// sendInitialRequest sends the initial authentication request
func (c *Client) sendInitialRequest() error {
	initReq := NewFullClientRequest(c.config)
	c.seq++
	err := c.conn.WriteMessage(websocket.BinaryMessage, initReq)
	if err != nil {
		return err
	}

	_, respData, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}
	respStruct := ParseResponse(respData)

	if respStruct.Code != 0 {
		return fmt.Errorf("initialization error: code: %d, msg: %v", respStruct.Code, respStruct.PayloadMsg)
	}

	return nil
}

func (c *Client) handleWriteLoop() {
	defer func() {
		logrus.WithField("traceID", c.traceID).Info("asr client: write loop exited")
		c.writeStopSig <- struct{}{}
	}()

	for {
		select {
		case <-c.stopSignal:
			return
		case frame, ok := <-c.audioQueue:
			if !ok {
				return
			}

			seq := c.seq
			if !frame.IsEnd {
				c.seq++
			} else {
				seq = -seq
				logrus.WithFields(logrus.Fields{
					"seq":     seq,
					"traceID": c.traceID,
				}).Info("sending final audio frame")
			}

			message := NewAudioOnlyRequest(seq, frame.Data)
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.sendTimeout))
			if err := c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				if !c.isNormalCloseError(err) && c.errorHandler != nil {
					c.logError(err, "write")
					c.errorHandler(err)
				}
				return
			}
		}
	}
}

// handleReadLoop processes incoming responses
func (c *Client) handleReadLoop() {
	defer func() {
		logrus.WithField("traceID", c.traceID).Info("asr client: read loop exited")
		c.readStopSig <- struct{}{}
	}()

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.recvTimeout))
		_, msgData, err := c.conn.ReadMessage()
		if err != nil {
			if !c.isNormalCloseError(err) && c.errorHandler != nil {
				c.logError(err, "read")
				c.errorHandler(err)
			}
			return
		}

		resp := ParseResponse(msgData)
		logrus.WithFields(logrus.Fields{
			"code":    resp.Code,
			"event":   resp.Event,
			"isFinal": resp.IsLastPackage,
			"traceID": c.traceID,
		}).Debug("asr response received")

		// Send result to upper layer
		select {
		case <-c.stopSignal:
			return
		case c.resultQueue <- resp:
		default:
			logrus.WithField("traceID", c.traceID).Warn("result queue full, dropping response")
		}

		// If it's the last frame, exit loop
		if resp.IsLastPackage {
			logrus.WithField("traceID", c.traceID).Info("asr client: received final response")
			return
		}
	}
}

func (c *Client) ReceiveResult() (*Response, error) {
	select {
	case resp := <-c.resultQueue:
		return resp, nil
	case <-c.stopSignal:
		return nil, ErrClientClosed
	}
}

func (c *Client) SendAudioFrame(frame *AudioFrame) error {
	select {
	case c.audioQueue <- frame:
		return nil
	case <-c.stopSignal:
		return ErrClientClosed
	}
}

// IsClosed returns true if the client is closed
func (c *Client) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isClosed
}

// GetTraceID returns the trace ID from the connection
func (c *Client) GetTraceID() string {
	return c.traceID
}

// Close gracefully closes the connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true
	close(c.stopSignal)

	// Wait for loops to stop with timeout
	timeout := time.After(1 * time.Second)

	select {
	case <-c.writeStopSig:
	case <-timeout:
	}

	select {
	case <-c.readStopSig:
	case <-timeout:
	}

	// Clean up resources
	if c.conn != nil {
		_ = c.conn.Close()
	}
	close(c.audioQueue)
	close(c.resultQueue)
}

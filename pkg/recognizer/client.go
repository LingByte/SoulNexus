package recognizer

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
	traceId string

	conn     *websocket.Conn
	mu       sync.Mutex
	isClosed bool

	audioChan  chan *AudioFrame
	resultChan chan *Response
	closeChan  chan struct{}

	// Loop stop signal channels
	writeLoopStopChan chan struct{}
	readLoopStopChan  chan struct{}

	// Control parameters
	writeTimeout time.Duration
	readTimeout  time.Duration

	// Error callback
	errorCallback func(error)
}

type AudioFrame struct {
	IsEnd bool
	Data  []byte
}

func NewClient(config *Config) *Client {
	return &Client{
		seq:               1,
		config:            config,
		audioChan:         make(chan *AudioFrame, 100),
		resultChan:        make(chan *Response, 100),
		closeChan:         make(chan struct{}),
		writeLoopStopChan: make(chan struct{}, 1),
		readLoopStopChan:  make(chan struct{}, 1),
		writeTimeout:      10 * time.Second,
		readTimeout:       30 * time.Second,
	}
}

// SetTimeouts sets the timeouts for write and read operations
func (c *Client) SetTimeouts(writeTimeout, readTimeout time.Duration) {
	c.writeTimeout = writeTimeout
	c.readTimeout = readTimeout
}

// SetErrorCallback sets the error callback function
func (c *Client) SetErrorCallback(callback func(error)) {
	c.errorCallback = callback
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

// logError logs error with operation and traceId
func (c *Client) logError(err error, operation string) {
	logrus.WithFields(logrus.Fields{
		"error":     err.Error(),
		"operation": operation,
		"traceId":   c.traceId,
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
	header := NewAuthHeader(c.config.Auth)
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, c.config.URL, header)
	if err != nil {
		if ctx.Err() == context.Canceled || strings.Contains(err.Error(), "operation was canceled") {
			return nil
		}
		return fmt.Errorf("dial websocket err: %w", err)
	}
	c.traceId = resp.Header.Get("X-Tt-Logid")
	c.conn = conn

	// Send initial full client request
	if err := c.sendFullClientRequest(); err != nil {
		_ = conn.Close()
		return fmt.Errorf("send full client request err: %w", err)
	}

	go c.readLoop()
	go c.writeLoop()

	return nil
}

// sendFullClientRequest sends the initial request (internal method)
func (c *Client) sendFullClientRequest() error {
	fullClientRequest := NewFullClientRequest(c.config)
	c.seq++
	err := c.conn.WriteMessage(websocket.BinaryMessage, fullClientRequest)
	if err != nil {
		return err
	}

	_, resp, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}
	respStruct := ParseResponse(resp)

	if respStruct.Code != 0 {
		return fmt.Errorf("initialization error: code: %d, msg: %v", respStruct.Code, respStruct.PayloadMsg)
	}

	return nil
}

func (c *Client) writeLoop() {
	defer func() {
		logrus.WithField("traceId", c.traceId).Info("asr client: writeLoop exited")
		c.writeLoopStopChan <- struct{}{}
	}()
	var err error

	for {
		select {
		case <-c.closeChan:
			return
		case frame, ok := <-c.audioChan:
			if !ok {
				return
			}
			var seq int
			seq = c.seq
			if !frame.IsEnd {
				c.seq++
			} else {
				seq = -seq
				logrus.WithFields(logrus.Fields{
					"seq":     seq,
					"traceId": c.traceId,
				}).Info("sending final audio frame")
			}

			message := NewAudioOnlyRequest(seq, frame.Data)
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err = c.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
				// Only notify callback for non-normal close errors
				if !c.isNormalCloseError(err) && c.errorCallback != nil {
					c.logError(err, "writeLoop")
					c.errorCallback(err)
				}
				return
			}
		}
	}
}

// readLoop handles response reading
func (c *Client) readLoop() {
	defer func() {
		logrus.WithField("traceId", c.traceId).Info("asr client: readLoop exited")
		c.readLoopStopChan <- struct{}{}
	}()

	var err error
	var msg []byte

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		if _, msg, err = c.conn.ReadMessage(); err != nil {
			// Only notify callback for non-normal close errors
			if !c.isNormalCloseError(err) && c.errorCallback != nil {
				c.logError(err, "readLoop")
				c.errorCallback(err)
			}
			return
		}

		resp := ParseResponse(msg)
		logrus.WithFields(logrus.Fields{
			"code":            resp.Code,
			"event":           resp.Event,
			"isLastPackage":   resp.IsLastPackage,
			"payloadSequence": resp.PayloadSequence,
			"payloadSize":     resp.PayloadSize,
			"payloadMsg":      resp.PayloadMsg,
			"err":             resp.Err,
			"traceId":         c.traceId,
		}).Info("asr client: response received")

		// Send result to upper layer
		select {
		case <-c.closeChan:
			return
		case c.resultChan <- resp:
		default:
			logrus.WithField("traceId", c.traceId).Warn("resultChan is full, dropping response")
		}

		// If it's the last frame, exit loop
		if resp.IsLastPackage {
			logrus.WithField("traceId", c.traceId).Info("asr client: received last package")
			return
		}
	}
}

func (c *Client) ReceiveResult() (*Response, error) {
	var (
		err  error
		resp *Response
	)
	select {
	case resp = <-c.resultChan:
	case <-c.closeChan:
		err = ErrClientClosed
	}
	return resp, err
}

func (c *Client) SendAudioFrame(frame *AudioFrame) error {
	var err error
	select {
	case c.audioChan <- frame:
	case <-c.closeChan:
		err = ErrClientClosed
	}
	return err
}

// IsClosed returns true if the client is closed
func (c *Client) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isClosed
}

// GetTraceID returns the trace ID from the connection
func (c *Client) GetTraceID() string {
	return c.traceId
}

// Close closes the connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true

	// Close main close channel to notify all loops to stop
	close(c.closeChan)

	// Wait for loop stop signals, set timeout to prevent blocking
	timeout := time.After(1 * time.Second)

	// Wait for writeLoop to stop
	select {
	case <-c.writeLoopStopChan:
	case <-timeout:
	}

	// Wait for readLoop to stop
	select {
	case <-c.readLoopStopChan:
	case <-timeout:
	}

	// Clean up resources
	if c.conn != nil {
		_ = c.conn.Close()
	}
	close(c.audioChan)
	close(c.resultChan)
}

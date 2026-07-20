package gateway

// Package gateway dials the dialog-plane WebSocket and bridges JSON events/commands
// to protocol/voice/dialog.Session.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/dialog"
	"github.com/gorilla/websocket"
)

// ClientConfig configures a per-call dialog WebSocket client.
type ClientConfig struct {
	URL              string
	CallID           string
	HandshakeHeaders http.Header
	DialTimeout      time.Duration

	OnHangup   func(reason string)
	OnASRFinal func(text string)
	OnTTSStart func(utteranceID, text string)
	OnTurn     func(dialog.TurnEvent)

	ReconnectAttempts       int
	ReconnectInitialBackoff time.Duration
	HoldTextFirst           string
	HoldTextRetry           string
	HoldTextGiveUp          string
}

// Client streams dialog events to a dialog app and executes commands on a Session.
type Client struct {
	cfg    ClientConfig
	conn   *websocket.Conn
	writeM sync.Mutex
	sess   *dialog.Session

	started atomic.Bool
	closed  atomic.Bool

	runCtx    context.Context
	runCancel context.CancelFunc
}

// NewClient validates cfg. The WebSocket is opened by Start.
func NewClient(cfg ClientConfig) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("gateway: empty URL")
	}
	if strings.TrimSpace(cfg.CallID) == "" {
		return nil, fmt.Errorf("gateway: empty CallID")
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	return &Client{cfg: cfg}, nil
}

// Bind attaches the voice session that receives commands from the dialog plane.
func (c *Client) Bind(sess *dialog.Session) {
	if c == nil {
		return
	}
	c.sess = sess
}

// Start dials the dialog WebSocket and starts the command read loop.
func (c *Client) Start(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("gateway: nil client")
	}
	if !c.started.CompareAndSwap(false, true) {
		return fmt.Errorf("gateway: already started")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.dialAndSwap(ctx); err != nil {
		return err
	}
	c.runCtx, c.runCancel = context.WithCancel(context.Background())
	go c.readLoop()
	return nil
}

// SendEvent forwards a voice-plane event to the dialog WebSocket.
func (c *Client) SendEvent(ev dialog.Event) error {
	if ev.CallID == "" {
		ev.CallID = c.cfg.CallID
	}
	switch ev.Type {
	case dialog.EvASRFinal:
		if c.cfg.OnASRFinal != nil {
			c.cfg.OnASRFinal(ev.Text)
		}
	case dialog.EvTTSStarted:
		if c.cfg.OnTTSStart != nil {
			c.cfg.OnTTSStart(ev.UtteranceID, "")
		}
	}
	return c.sendEvent(ev)
}

// PushDTMF forwards a DTMF digit to the dialog app.
func (c *Client) PushDTMF(digit string, end bool) {
	if c == nil || !c.started.Load() {
		return
	}
	_ = c.sendEvent(dialog.Event{Type: dialog.EvDTMF, CallID: c.cfg.CallID, Digit: digit, End: end})
}

// ForwardTransferRequest notifies the dialog app of a transfer request.
func (c *Client) ForwardTransferRequest(target string) {
	if c == nil || !c.started.Load() || c.closed.Load() {
		return
	}
	_ = c.sendEvent(dialog.Event{Type: dialog.EvTransferRequest, CallID: c.cfg.CallID, Target: target})
}

// Close sends call.ended and closes the WebSocket. Idempotent.
func (c *Client) Close(reason string) {
	if c == nil || c.closed.Swap(true) {
		return
	}
	_ = c.sendEvent(dialog.Event{Type: dialog.EvCallEnded, CallID: c.cfg.CallID, Reason: reason})
	if c.runCancel != nil {
		c.runCancel()
	}
	if c.conn != nil {
		_ = c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
			time.Now().Add(500*time.Millisecond),
		)
		_ = c.conn.Close()
	}
}

func (c *Client) sendEvent(ev dialog.Event) error {
	if c == nil || c.conn == nil || c.closed.Load() {
		return nil
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	c.writeM.Lock()
	defer c.writeM.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *Client) readLoop() {
	for {
		if c.runCtx != nil && c.runCtx.Err() != nil {
			return
		}
		if !c.runOnce() {
			if c.cfg.OnHangup != nil {
				c.cfg.OnHangup("dialog-ws-closed")
			}
			return
		}
	}
}

func (c *Client) runOnce() bool {
	for {
		if c.closed.Load() {
			return false
		}
		if c.conn == nil {
			return false
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if c.closed.Load() {
				return false
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return false
			}
			if c.reconnectWithHold() {
				continue
			}
			return false
		}
		var cmd dialog.Command
		if err := json.Unmarshal(data, &cmd); err != nil {
			continue
		}
		if cmd.CallID != "" && cmd.CallID != c.cfg.CallID {
			continue
		}
		c.dispatch(cmd)
	}
}

func (c *Client) reconnectWithHold() bool {
	maxN := c.cfg.ReconnectAttempts
	if maxN <= 0 || c.sess == nil {
		return false
	}
	c.sess.HandleCommand(dialog.Command{Type: dialog.CmdTTSInterrupt})

	speakHold := func(text string) {
		if text == "" {
			return
		}
		c.sess.HandleCommand(dialog.Command{
			Type:        dialog.CmdTTSSpeak,
			CallID:      c.cfg.CallID,
			Text:        text,
			UtteranceID: "reconnect-hold",
		})
	}
	speakHold(c.cfg.HoldTextFirst)

	backoff := c.cfg.ReconnectInitialBackoff
	if backoff <= 0 {
		backoff = time.Second
	}
	const maxBackoff = 30 * time.Second

	for attempt := 1; attempt <= maxN; attempt++ {
		deadline := time.Now().Add(backoff)
		for time.Now().Before(deadline) {
			if c.closed.Load() || (c.runCtx != nil && c.runCtx.Err() != nil) {
				return false
			}
			time.Sleep(250 * time.Millisecond)
		}
		if err := c.dialAndSwap(context.Background()); err != nil {
			if attempt < maxN {
				speakHold(c.cfg.HoldTextRetry)
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		return true
	}
	speakHold(c.cfg.HoldTextGiveUp)
	return false
}

func (c *Client) dialAndSwap(ctx context.Context) error {
	dialURL, err := appendCallIDQuery(c.cfg.URL, c.cfg.CallID)
	if err != nil {
		return err
	}
	timeout := c.cfg.DialTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, resp, err := websocket.DefaultDialer.DialContext(dctx, dialURL, c.cfg.HandshakeHeaders)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return fmt.Errorf("dial dialog ws %s (http=%d): %w", RedactDialogDialURL(dialURL), code, err)
	}
	c.writeM.Lock()
	old := c.conn
	c.conn = conn
	c.writeM.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (c *Client) dispatch(cmd dialog.Command) {
	if c.sess == nil {
		return
	}
	switch cmd.Type {
	case dialog.CmdTTSSpeak, dialog.CmdTTSStream, dialog.CmdTTSStreamEnd,
		dialog.CmdTTSInterrupt, dialog.CmdHangup:
		c.sess.HandleCommand(cmd)
	default:
		// ignore unknown commands
	}
}

// DrainBody is a helper for tests.
func DrainBody(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

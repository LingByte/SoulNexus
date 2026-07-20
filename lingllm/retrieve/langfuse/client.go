package langfuse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Config configures the Langfuse batch ingestion client.
type Config struct {
	Enabled   bool
	BaseURL   string
	PublicKey string
	SecretKey string
	HTTPClient *http.Client
}

// Client sends trace/span events to Langfuse ingestion API.
type Client struct {
	baseURL    string
	authHeader string
	http       *http.Client
}

// NewClient builds a Langfuse client. Returns nil when disabled or misconfigured.
func NewClient(cfg Config) (*Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	pub := strings.TrimSpace(cfg.PublicKey)
	sec := strings.TrimSpace(cfg.SecretKey)
	if pub == "" || sec == "" {
		return nil, fmt.Errorf("langfuse: public and secret keys required")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://cloud.langfuse.com"
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	token := base64.StdEncoding.EncodeToString([]byte(pub + ":" + sec))
	return &Client{
		baseURL:    base,
		authHeader: "Basic " + token,
		http:       hc,
	}, nil
}

type ingestionBatch struct {
	Batch []ingestionEvent `json:"batch"`
}

type ingestionEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Body      any       `json:"body"`
}

type traceBody struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	Input    any            `json:"input,omitempty"`
	Output   any            `json:"output,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type spanBody struct {
	ID        string         `json:"id"`
	TraceID   string         `json:"traceId"`
	Name      string         `json:"name,omitempty"`
	StartTime time.Time      `json:"startTime"`
	EndTime   time.Time      `json:"endTime,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Level     string         `json:"level,omitempty"`
	StatusMsg string         `json:"statusMessage,omitempty"`
}

// Ingest sends a batch of events. No-op when batch is empty.
func (c *Client) Ingest(ctx context.Context, events []ingestionEvent) error {
	if c == nil || len(events) == 0 {
		return nil
	}
	payload := ingestionBatch{Batch: events}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/public/ingestion", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("langfuse ingestion: status=%d body=%s", resp.StatusCode, string(raw))
	}
	return nil
}

func newEventID() string {
	return uuid.NewString()
}

func newTraceEvent(traceID, name string, input, metadata map[string]any) ingestionEvent {
	return ingestionEvent{
		ID:        newEventID(),
		Type:      "trace-create",
		Timestamp: time.Now().UTC(),
		Body: traceBody{
			ID:       traceID,
			Name:     name,
			Input:    input,
			Metadata: metadata,
		},
	}
}

func newSpanEvent(traceID, name string, start, end time.Time, metadata map[string]any, err error) ingestionEvent {
	body := spanBody{
		ID:        newEventID(),
		TraceID:   traceID,
		Name:      name,
		StartTime: start.UTC(),
		EndTime:   end.UTC(),
		Metadata:  metadata,
	}
	if err != nil {
		body.Level = "ERROR"
		body.StatusMsg = err.Error()
	}
	return ingestionEvent{
		ID:        newEventID(),
		Type:      "span-create",
		Timestamp: time.Now().UTC(),
		Body:      body,
	}
}

func traceUpdateEvent(traceID string, output map[string]any, err error) ingestionEvent {
	body := traceBody{
		ID:     traceID,
		Output: output,
	}
	if err != nil {
		if body.Metadata == nil {
			body.Metadata = map[string]any{}
		}
		body.Metadata["error"] = err.Error()
	}
	return ingestionEvent{
		ID:        newEventID(),
		Type:      "trace-create",
		Timestamp: time.Now().UTC(),
		Body:      body,
	}
}

// AsyncClient buffers events and flushes in background.
type AsyncClient struct {
	inner *Client
	mu    sync.Mutex
	buf   []ingestionEvent
}

// NewAsync wraps a Client with async enqueue + flush on close.
func NewAsync(c *Client) *AsyncClient {
	if c == nil {
		return nil
	}
	return &AsyncClient{inner: c}
}

func (a *AsyncClient) Enqueue(events ...ingestionEvent) {
	if a == nil || len(events) == 0 {
		return
	}
	a.mu.Lock()
	a.buf = append(a.buf, events...)
	a.mu.Unlock()
}

func (a *AsyncClient) Flush(ctx context.Context) error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	batch := a.buf
	a.buf = nil
	a.mu.Unlock()
	return a.inner.Ingest(ctx, batch)
}

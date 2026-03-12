package metrics

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Span 链路追踪的跨度
type Span struct {
	ID         string                 `json:"id"`
	TraceID    string                 `json:"trace_id"`
	ParentID   string                 `json:"parent_id"`
	Name       string                 `json:"name"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Duration   time.Duration          `json:"duration"`
	Tags       map[string]string      `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	Events     []Event                `json:"events"`
	Status     SpanStatus             `json:"status"`
	Error      error                  `json:"error,omitempty"`
	Children   []*Span                `json:"children,omitempty"`
	mu         sync.RWMutex
}

// Event 链路事件
type Event struct {
	Time       time.Time              `json:"time"`
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// SpanStatus 跨度状态
type SpanStatus int

const (
	SpanStatusUnset SpanStatus = iota
	SpanStatusOK
	SpanStatusError
)

// Tracer 链路追踪器
type Tracer struct {
	spans    map[string]*Span
	mu       sync.RWMutex
	maxSpans int
}

// NewTracer 创建新的追踪器
func NewTracer(maxSpans int) *Tracer {
	return &Tracer{
		spans:    make(map[string]*Span),
		maxSpans: maxSpans,
	}
}

// StartSpan 开始一个新的跨度
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	span := &Span{
		ID:         generateSpanID(),
		TraceID:    getTraceIDFromContext(ctx),
		Name:       name,
		StartTime:  time.Now(),
		Tags:       make(map[string]string),
		Attributes: make(map[string]interface{}),
		Events:     make([]Event, 0),
		Status:     SpanStatusUnset,
		Children:   make([]*Span, 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(span)
	}

	// 如果没有TraceID，生成一个新的
	if span.TraceID == "" {
		span.TraceID = generateTraceID()
	}

	// 获取父跨度ID
	if parentSpan := getSpanFromContext(ctx); parentSpan != nil {
		span.ParentID = parentSpan.ID
		parentSpan.mu.Lock()
		parentSpan.Children = append(parentSpan.Children, span)
		parentSpan.mu.Unlock()
	}

	// 存储跨度
	t.mu.Lock()
	if len(t.spans) >= t.maxSpans {
		// 清理最旧的跨度
		t.cleanupOldSpans()
	}
	t.spans[span.ID] = span
	t.mu.Unlock()

	// 将跨度添加到上下文
	newCtx := context.WithValue(ctx, spanContextKey{}, span)
	return newCtx, span
}

// EndSpan 结束跨度
func (t *Tracer) EndSpan(span *Span, err error) {
	if span == nil {
		return
	}

	span.mu.Lock()
	defer span.mu.Unlock()

	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)

	if err != nil {
		span.Status = SpanStatusError
		span.Error = err
	} else {
		span.Status = SpanStatusOK
	}
}

// AddEvent 添加事件到跨度
func (s *Span) AddEvent(name string, attrs map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := Event{
		Time:       time.Now(),
		Name:       name,
		Attributes: attrs,
	}
	s.Events = append(s.Events, event)
}

// SetTag 设置标签
func (s *Span) SetTag(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tags[key] = value
}

// SetAttribute 设置属性
func (s *Span) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes[key] = value
}

// GetSpans 获取所有跨度
func (t *Tracer) GetSpans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*Span, 0, len(t.spans))
	for _, span := range t.spans {
		spans = append(spans, span)
	}
	return spans
}

// GetSpan 获取指定ID的跨度
func (t *Tracer) GetSpan(id string) *Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.spans[id]
}

// GetTraceSpans 获取指定TraceID的所有跨度
func (t *Tracer) GetTraceSpans(traceID string) []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var spans []*Span
	for _, span := range t.spans {
		if span.TraceID == traceID {
			spans = append(spans, span)
		}
	}
	return spans
}

// cleanupOldSpans 清理最旧的跨度
func (t *Tracer) cleanupOldSpans() {
	// 按开始时间排序，删除最旧的
	spans := make([]*Span, 0, len(t.spans))
	for _, span := range t.spans {
		spans = append(spans, span)
	}

	// 按开始时间排序
	for i := 0; i < len(spans)-1; i++ {
		for j := i + 1; j < len(spans); j++ {
			if spans[i].StartTime.After(spans[j].StartTime) {
				spans[i], spans[j] = spans[j], spans[i]
			}
		}
	}

	// 删除最旧的跨度，保留一半
	keepCount := len(spans) / 2
	for i := 0; i < keepCount; i++ {
		delete(t.spans, spans[i].ID)
	}
}

// SpanOption 跨度选项
type SpanOption func(*Span)

// WithParent 设置父跨度
func WithParent(parent *Span) SpanOption {
	return func(s *Span) {
		if parent != nil {
			s.ParentID = parent.ID
		}
	}
}

// WithTags 设置标签
func WithTags(tags map[string]string) SpanOption {
	return func(s *Span) {
		for k, v := range tags {
			s.Tags[k] = v
		}
	}
}

// WithAttributes 设置属性
func WithAttributes(attrs map[string]interface{}) SpanOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// 上下文键
type spanContextKey struct{}

// getSpanFromContext 从上下文获取跨度
func getSpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey{}).(*Span); ok {
		return span
	}
	return nil
}

// getTraceIDFromContext 从上下文获取TraceID
func getTraceIDFromContext(ctx context.Context) string {
	if span := getSpanFromContext(ctx); span != nil {
		return span.TraceID
	}
	return ""
}

var (
	spanIDCounter  int64
	traceIDCounter int64
)

// generateSpanID 生成跨度ID
func generateSpanID() string {
	id := atomic.AddInt64(&spanIDCounter, 1)
	return fmt.Sprintf("span_%d_%d", time.Now().UnixNano(), id)
}

// generateTraceID 生成追踪ID
func generateTraceID() string {
	id := atomic.AddInt64(&traceIDCounter, 1)
	return fmt.Sprintf("trace_%d_%d", time.Now().UnixNano(), id)
}

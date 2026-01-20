package errhandler

import (
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// ErrorType 错误类型
type ErrorType int

const (
	// ErrorTypeFatal 致命错误（需要断开连接）
	ErrorTypeFatal ErrorType = iota
	// ErrorTypeRecoverable 可恢复错误（可以重试）
	ErrorTypeRecoverable
	// ErrorTypeTransient 临时错误（短暂故障，会自动恢复）
	ErrorTypeTransient
)

// Error 统一错误结构
type Error struct {
	Type    ErrorType
	Service string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Service, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Service, e.Message)
}

// Handler 错误处理器
type Handler struct {
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewHandler 创建错误处理器
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// IsFatal 判断是否是致命错误
func (h *Handler) IsFatal(err error) bool {
	if err == nil {
		return false
	}

	// 检查是否是我们的统一错误类型
	if e, ok := err.(*Error); ok {
		return e.Type == ErrorTypeFatal
	}

	// 检查错误消息中的关键词
	errMsg := strings.ToLower(err.Error())
	fatalKeywords := []string{
		"quota exceeded",
		"quota exhausted",
		"pkg exhausted",
		"allowance has been exhausted",
		"insufficient quota",
		"quota limit",
		"unauthorized",
		"authentication failed",
		"invalid credentials",
		"api key invalid",
		"api key expired",
		"account suspended",
		"account disabled",
	}

	for _, keyword := range fatalKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// Classify 分类错误
func (h *Handler) Classify(err error, service string) *Error {
	if err == nil {
		return nil
	}

	// 如果已经是统一错误类型，直接返回
	if e, ok := err.(*Error); ok {
		return e
	}

	// 分类错误
	errType := ErrorTypeRecoverable
	if h.IsFatal(err) {
		errType = ErrorTypeFatal
	} else if h.isTransient(err) {
		errType = ErrorTypeTransient
	}

	return &Error{
		Type:    errType,
		Service: service,
		Message: err.Error(),
		Err:     err,
	}
}

// isTransient 判断是否是临时错误
func (h *Handler) isTransient(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	transientKeywords := []string{
		"timeout",
		"connection reset",
		"connection refused",
		"network",
		"temporary",
		"retry",
	}

	for _, keyword := range transientKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// IsRateLimitError 判断是否是限流/并发超限错误
func (h *Handler) IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	rateLimitKeywords := []string{
		"并发超限",
		"concurrent",
		"rate limit",
		"too many requests",
		"429",
		"4006", // 腾讯云ASR并发超限错误码
	}

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// HandleError 处理错误
func (h *Handler) HandleError(err error, service string) error {
	if err == nil {
		return nil
	}

	classified := h.Classify(err, service)

	h.mu.Lock()
	defer h.mu.Unlock()

	switch classified.Type {
	case ErrorTypeFatal:
		h.logger.Error("致命错误",
			zap.String("service", service),
			zap.Error(err),
			zap.String("message", classified.Message),
		)
	case ErrorTypeRecoverable:
		h.logger.Warn("可恢复错误",
			zap.String("service", service),
			zap.Error(err),
			zap.String("message", classified.Message),
		)
	case ErrorTypeTransient:
		h.logger.Debug("临时错误",
			zap.String("service", service),
			zap.Error(err),
			zap.String("message", classified.Message),
		)
	}

	return classified
}

// NewFatalError 创建致命错误
func NewFatalError(service, message string, err error) *Error {
	return &Error{
		Type:    ErrorTypeFatal,
		Service: service,
		Message: message,
		Err:     err,
	}
}

// NewRecoverableError 创建可恢复错误
func NewRecoverableError(service, message string, err error) *Error {
	return &Error{
		Type:    ErrorTypeRecoverable,
		Service: service,
		Message: message,
		Err:     err,
	}
}

// NewTransientError 创建临时错误
func NewTransientError(service, message string, err error) *Error {
	return &Error{
		Type:    ErrorTypeTransient,
		Service: service,
		Message: message,
		Err:     err,
	}
}

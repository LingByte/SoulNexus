package llm

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

var requestCounter uint64

const (
	LLM_OPENAI    = "llm.openai"
	LLM_ANTHROPIC = "llm.anthropic"
	LLM_COZE      = "llm.coze"
	LLM_OLLAMA    = "llm.ollama"
	LLM_LMSTUDIO  = "llm.lmstudio"
	LLM_ALIBABA   = "llm.alibaba"
)

const (
	OutputFormatText       = "text"
	OutputFormatJSON       = "json"
	OutputFormatJSONObject = "json_object"
	OutputFormatJSONSchema = "json_schema"
	OutputFormatXML        = "xml"
	OutputFormatHTML       = "html"
	OutputFormatSQL        = "sql"
)

const (
	SignalLLMRequestStart = "llm.request.start"
	SignalLLMRequestEnd   = "llm.request.end"
	SignalLLMRequestError = "llm.request.error"
	SignalSessionCreated  = "session.created"
	SignalSessionUpdated  = "session.updated"
	SignalSessionDeleted  = "session.deleted"
	SignalMessageCreated  = "message.created"
	SignalMessageUpdated  = "message.updated"
	SignalMessageDeleted  = "message.deleted"
)

// LLMProvider common provider type
type LLMProvider string

// ToString toString for llm
func (lp LLMProvider) ToString() string {
	return string(lp)
}

type LLMOptions struct {
	Provider        string
	ApiKey          string
	BaseURL         string
	SystemPrompt    string
	FewShotExamples []FewShotExample
	Logger          *zap.Logger // Logger is optional; used for async memory summarization warnings and diagnostics.
}

type FewShotExample struct {
	User      string
	Assistant string
}

type QueryOptions struct {
	Model                string
	N                    int
	MaxTokens            int
	Temperature          float32
	TopP                 float32
	LogitBias            map[string]int
	FilterEmoji          bool
	EnableJSONOutput     bool
	OutputFormat         string
	EmotionalTone        bool   // EmotionalTone, when true, appends a short instruction so replies read warmer and more human (still factual).
	EnableQueryExpansion bool   // EnableQueryExpansion enables automatic query expansion using LLM
	ExpansionMaxTerms    int    // ExpansionMaxTerms maximum number of expansion terms
	ExpansionSeparator   string // ExpansionSeparator separator for expanded terms

	// EnableQueryRewrite rewrites the user message with a stateless LLM call before expansion/main query.
	EnableQueryRewrite bool
	// QueryRewriteModel overrides the model for the rewrite call only (empty = use Model, then handler default).
	QueryRewriteModel string
	// QueryRewriteInstruction is appended to the rewrite prompt as extra constraints.
	QueryRewriteInstruction string
	// EnableSelfQueryJSONOutput requests strict JSON object replies (response_format json_object on OpenaiHandler).
	// SelfQueryExtractor sets this by default; other handlers may ignore it and still return parseable text.
	EnableSelfQueryJSONOutput bool
	// Messages allows callers to pass short-term conversation history explicitly.
	// Handlers append current user text as the final user turn when needed.
	Messages []ChatMessage

	// 新增字段用于数据库记录
	SessionID   string // 会话ID
	UserID      string // 用户ID
	RequestType string // 请求类型: query, query_stream, rewrite, expand
}

type ChatMessage struct {
	Role    string
	Content string
}

type TokenUsage struct {
	PromptTokens            int
	CompletionTokens        int
	TotalTokens             int
	PromptTokensDetails     *PromptTokensDetails
	CompletionTokensDetails *CompletionTokensDetails
}

type CompletionTokensDetails struct {
	AudioTokens              int
	ReasoningTokens          int
	AcceptedPredictionTokens int
	RejectedPredictionTokens int
}

type PromptTokensDetails struct {
	AudioTokens  int
	CachedTokens int
}

type QueryChoice struct {
	Index        int
	Content      string
	FinishReason string
}

type QueryResponse struct {
	Provider     string          `json:"provider"`
	Model        string          `json:"model"`
	Choices      []QueryChoice   `json:"choices"`
	Usage        *TokenUsage     `json:"usage"`
	Expansion    *QueryExpansion `json:"expansion,omitempty"`
	Rewrite      *QueryRewrite   `json:"rewrite,omitempty"`
	FinishReason string          `json:"finish_reason,omitempty"`

	// 新增字段用于数据库记录
	RequestID   string `json:"request_id"`
	SessionID   string `json:"session_id"`
	UserID      string `json:"user_id"`
	RequestedAt int64  `json:"requested_at"`
	CompletedAt int64  `json:"completed_at"`
	LatencyMs   int64  `json:"latency_ms"`
}

// QueryRewrite records the optional LLM rewrite step applied before expansion / main completion.
type QueryRewrite struct {
	Original  string
	Rewritten string
}

// QueryExpansion contains the results of query expansion
type QueryExpansion struct {
	Original string
	Expanded string
	Terms    []string
	Debug    map[string]any
}

type LLMDetails struct {
	RequestID               string
	Provider                string
	BaseURL                 string
	Model                   string
	Input                   string
	SystemPrompt            string
	N                       int
	MaxTokens               int
	EstimatedMaxOutputChars int
	FilterEmoji             bool
	RequestedOutputFormat   string
	AppliedResponseFormat   string
	ResponseFormatApplied   bool
	ResponseID              string
	Object                  string
	Created                 int64
	SystemFingerprint       string
	PromptFilterResultsJSON string
	ServiceTierJSON         string
	ChoicesCount            int
	Choices                 []QueryChoice
	Usage                   *TokenUsage
	UsageRawJSON            string
	ChoicesRawJSON          string
	RawResponseJSON         string
}

// LLMHandler common llm hanlder interface
type LLMHandler interface {

	// Query base query
	Query(text, model string) (string, error)

	// QueryWithOptions query by options
	QueryWithOptions(text string, options *QueryOptions) (*QueryResponse, error)

	// QueryStream query stream
	QueryStream(text string, options *QueryOptions, callback func(segment string, isComplete bool) error) (*QueryResponse, error)

	// Provider llm provider
	Provider() string

	// RegisterFunctionTool registers a function tool.
	RegisterFunctionTool(name, description string, parameters interface{}, callback FunctionToolCallback)

	// RegisterFunctionToolDefinition registers a function tool via definition.
	RegisterFunctionToolDefinition(def *FunctionToolDefinition)

	// GetFunctionTools returns all available function tools.
	GetFunctionTools() []interface{}

	// ListFunctionTools lists all registered tool names.
	ListFunctionTools() []string

	// Interrupt llm speak interrupt
	Interrupt()

	// Hangup cleans up resources.
	Hangup()
}

func GenerateLingRequestID() string {
	host, _ := os.Hostname()
	c := atomic.AddUint64(&requestCounter, 1)
	return fmt.Sprintf("ling-%s-%d-%d-%d", host, os.Getpid(), time.Now().UnixNano(), c)
}

// LLMRequestStartData LLM请求开始信号数据
type LLMRequestStartData struct {
	RequestID   string `json:"request_id"`
	SessionID   string `json:"session_id"`
	UserID      string `json:"user_id"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	RequestType string `json:"request_type"`
	Input       string `json:"input"`
	RequestedAt int64  `json:"requested_at"`
}

// LLMRequestEndData LLM请求结束信号数据
type LLMRequestEndData struct {
	RequestID    string `json:"request_id"`
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	RequestType  string `json:"request_type"`
	Success      bool   `json:"success"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	LatencyMs    int64  `json:"latency_ms"`
	Output       string `json:"output"`
	RequestedAt  int64  `json:"requested_at"`
	CompletedAt  int64  `json:"completed_at"`
}

// LLMRequestErrorData LLM请求错误信号数据
type LLMRequestErrorData struct {
	RequestID    string `json:"request_id"`
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	RequestType  string `json:"request_type"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	LatencyMs    int64  `json:"latency_ms"`
	RequestedAt  int64  `json:"requested_at"`
	CompletedAt  int64  `json:"completed_at"`
}

// SessionCreatedData 会话创建信号数据
type SessionCreatedData struct {
	SessionID    string `json:"session_id"`
	UserID       string `json:"user_id"`
	Title        string `json:"title"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
	CreatedAt    int64  `json:"created_at"`
}

// MessageCreatedData 消息创建信号数据
type MessageCreatedData struct {
	MessageID  string `json:"message_id"`
	SessionID  string `json:"session_id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`
	Model      string `json:"model"`
	Provider   string `json:"provider"`
	RequestID  string `json:"request_id"`
	CreatedAt  int64  `json:"created_at"`
}

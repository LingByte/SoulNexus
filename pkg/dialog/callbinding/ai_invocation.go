package callbinding

import (
	"strconv"
	"strings"
	"sync"
)

const (
	AIComponentLLM = "llm"
	AIComponentASR = "asr"
	AIComponentTTS = "tts"
	AIComponentNLU = "nlu"

	AIStatusOK    = "ok"
	AIStatusError = "error"
)

// AIInvocationRecord is one AI invocation audit row before persistence.
type AIInvocationRecord struct {
	TenantID uint
	UserID   uint

	Component string
	Provider  string
	Model     string
	Status    string
	CallID    string
	Source    string

	LatencyMs    int64
	FirstTokenMs int64

	PromptTokens     int
	CompletionTokens int
	TotalTokens      int

	InputChars  int
	OutputChars int

	AudioMs    int64
	AudioBytes int64

	IntentName string
	Confidence float64

	ErrorMsg string
	Meta     map[string]any

	RequestText  string
	ResponseText string
}

var (
	aiInvocationRecorderMu sync.RWMutex
	aiInvocationRecorder   func(AIInvocationRecord)
)

// SetAIInvocationRecorder wires async persistence (registered from internal/models at startup).
func SetAIInvocationRecorder(fn func(AIInvocationRecord)) {
	aiInvocationRecorderMu.Lock()
	defer aiInvocationRecorderMu.Unlock()
	aiInvocationRecorder = fn
}

// RecordAIInvocation enriches call metadata and forwards to the registered recorder.
func RecordAIInvocation(e AIInvocationRecord) {
	aiInvocationRecorderMu.RLock()
	recorder := aiInvocationRecorder
	aiInvocationRecorderMu.RUnlock()
	if recorder == nil {
		return
	}
	entry := e
	enrichAIInvocationRecord(&entry)
	go recorder(entry)
}

func enrichAIInvocationRecord(e *AIInvocationRecord) {
	if e == nil {
		return
	}
	callID := strings.TrimSpace(e.CallID)
	if callID == "" {
		return
	}
	if e.TenantID == 0 {
		e.TenantID = GetTenantID(callID)
	}
	ResolveAIInvocationActor(e)
}

// ResolveAIInvocationActor fills user/credential metadata from call bindings.
func ResolveAIInvocationActor(e *AIInvocationRecord) {
	if e == nil {
		return
	}
	callID := strings.TrimSpace(e.CallID)
	if callID == "" {
		return
	}
	if e.UserID == 0 {
		e.UserID = GetUserID(callID)
	}
	if credID := GetCredentialID(callID); credID > 0 {
		if e.Meta == nil {
			e.Meta = map[string]any{}
		}
		if _, ok := e.Meta["credential_id"]; !ok {
			e.Meta["credential_id"] = strconv.FormatUint(uint64(credID), 10)
		}
	}
}

package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

const (
	AIComponentLLM = callbinding.AIComponentLLM
	AIComponentASR = callbinding.AIComponentASR
	AIComponentTTS = callbinding.AIComponentTTS
	AIComponentNLU = callbinding.AIComponentNLU

	AIStatusOK    = callbinding.AIStatusOK
	AIStatusError = callbinding.AIStatusError
)

const maxAIInvocationBodyRunes = 128 << 10 // 128 KiB per request/response field

// AIInvocationRecord is one AI invocation audit row before persistence.
type AIInvocationRecord = callbinding.AIInvocationRecord

// SetAIInvocationDB wires async persistence (call once at startup).
func SetAIInvocationDB(gdb *gorm.DB) {
	callbinding.SetAIInvocationRecorder(func(e callbinding.AIInvocationRecord) {
		persistAIInvocation(gdb, e)
	})
}

// RecordAIInvocation persists one entry asynchronously; no-op when DB is unset.
func RecordAIInvocation(e AIInvocationRecord) {
	callbinding.RecordAIInvocation(e)
}

// AIInvocationLog records one LLM / ASR / TTS / NLU call.
type AIInvocationLog struct {
	ID        uint   `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TenantID  uint   `gorm:"index" json:"tenant_id,string"`
	UserID    uint   `gorm:"index" json:"user_id,string"`
	Component string `gorm:"size:16;index" json:"component"`
	Provider  string `gorm:"size:64;index" json:"provider"`
	Model     string `gorm:"size:128" json:"model"`
	Status    string `gorm:"size:16;index" json:"status"`
	CallID    string `gorm:"size:128;index" json:"call_id"`
	Source    string `gorm:"size:64;index" json:"source"`

	LatencyMs    int64 `json:"latency_ms"`
	FirstTokenMs int64 `json:"first_token_ms"`

	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	InputChars  int `json:"input_chars"`
	OutputChars int `json:"output_chars"`

	AudioMs    int64 `json:"audio_ms"`
	AudioBytes int64 `json:"audio_bytes"`

	IntentName string  `gorm:"size:128" json:"intent_name"`
	Confidence float64 `json:"confidence"`

	RequestText  string `gorm:"type:longtext" json:"request_text,omitempty"`
	ResponseText string `gorm:"type:longtext" json:"response_text,omitempty"`

	ErrorMsg  string    `gorm:"type:text" json:"error_msg"`
	MetaJSON  string    `gorm:"type:json" json:"meta_json"`
	CreatedAt time.Time `json:"created_at"`
}

// AIInvocationLogTenantView is returned to tenant users (no request/response bodies).
type AIInvocationLogTenantView struct {
	ID        uint   `json:"id,string"`
	TenantID  uint   `json:"tenant_id,string"`
	UserID    uint   `json:"user_id,string"`
	Component string `json:"component"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Status    string `json:"status"`
	CallID    string `json:"call_id"`
	Source    string `json:"source"`

	LatencyMs    int64 `json:"latency_ms"`
	FirstTokenMs int64 `json:"first_token_ms"`

	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	InputChars  int `json:"input_chars"`
	OutputChars int `json:"output_chars"`

	AudioMs    int64 `json:"audio_ms"`
	AudioBytes int64 `json:"audio_bytes"`

	IntentName string  `json:"intent_name"`
	Confidence float64 `json:"confidence"`

	ErrorMsg  string    `json:"error_msg"`
	MetaJSON  string    `json:"meta_json"`
	CreatedAt time.Time `json:"created_at"`
}

// ToTenantView strips prompt/response bodies for tenant-facing APIs.
func ToTenantView(row AIInvocationLog) AIInvocationLogTenantView {
	return AIInvocationLogTenantView{
		ID: row.ID, TenantID: row.TenantID, UserID: row.UserID,
		Component: row.Component, Provider: row.Provider, Model: row.Model,
		Status: row.Status, CallID: row.CallID, Source: row.Source,
		LatencyMs: row.LatencyMs, FirstTokenMs: row.FirstTokenMs,
		PromptTokens: row.PromptTokens, CompletionTokens: row.CompletionTokens, TotalTokens: row.TotalTokens,
		InputChars: row.InputChars, OutputChars: row.OutputChars,
		AudioMs: row.AudioMs, AudioBytes: row.AudioBytes,
		IntentName: row.IntentName, Confidence: row.Confidence,
		ErrorMsg: row.ErrorMsg, MetaJSON: row.MetaJSON, CreatedAt: row.CreatedAt,
	}
}

func (AIInvocationLog) TableName() string {
	return constants.AI_INVOCATION_LOGS_TABLE_NAME
}

// AIInvocationLogFilter scopes list queries.
type AIInvocationLogFilter struct {
	TenantID  uint
	Component string
	Status    string
	Provider  string
	CallID    string
	Source    string
}

// ListAIInvocationLogsPage returns paginated logs newest first.
func ListAIInvocationLogsPage(db *gorm.DB, page, size int, f AIInvocationLogFilter) ([]AIInvocationLog, int64, error) {
	if db == nil {
		return nil, 0, nil
	}
	q := db.Model(&AIInvocationLog{})
	if f.TenantID > 0 {
		q = q.Where("tenant_id = ?", f.TenantID)
	}
	if s := f.Component; s != "" && s != "all" {
		q = q.Where("component = ?", s)
	}
	if s := f.Status; s != "" && s != "all" {
		q = q.Where("status = ?", s)
	}
	if s := f.Provider; s != "" {
		q = q.Where("provider = ?", s)
	}
	if s := f.CallID; s != "" {
		q = q.Where("call_id = ?", s)
	}
	if s := strings.TrimSpace(f.Source); s != "" {
		// Match exact or stream suffix (e.g. js_template → js_template_stream).
		q = q.Where("source = ? OR source = ?", s, s+"_stream")
	}
	return utils.FindPageQuery[AIInvocationLog](q, page, size, utils.MaxPageSize200, func(q *gorm.DB) *gorm.DB {
		return q.Order("created_at DESC")
	})
}

func persistAIInvocation(gdb *gorm.DB, e callbinding.AIInvocationRecord) {
	if gdb == nil || e.Component == "" {
		return
	}
	if e.Status == "" {
		if strings.TrimSpace(e.ErrorMsg) != "" {
			e.Status = AIStatusError
		} else {
			e.Status = AIStatusOK
		}
	}
	if e.TenantID == 0 && strings.TrimSpace(e.CallID) != "" {
		e.TenantID = resolveAIInvocationTenantFromCallID(gdb, e.CallID)
	}
	callbinding.ResolveAIInvocationActor(&e)
	metaJSON := "{}"
	if len(e.Meta) > 0 {
		if b, err := json.Marshal(e.Meta); err == nil && len(b) > 0 && string(b) != "null" {
			metaJSON = string(b)
		}
	}
	row := AIInvocationLog{
		ID:               utils.NextSnowflakeUint(),
		TenantID:         e.TenantID,
		UserID:           e.UserID,
		Component:        strings.TrimSpace(e.Component),
		Provider:         truncateAIInvocationField(e.Provider, 64),
		Model:            truncateAIInvocationField(e.Model, 128),
		Status:           truncateAIInvocationField(e.Status, 16),
		CallID:           truncateAIInvocationField(e.CallID, 128),
		Source:           truncateAIInvocationField(e.Source, 64),
		LatencyMs:        e.LatencyMs,
		FirstTokenMs:     e.FirstTokenMs,
		PromptTokens:     e.PromptTokens,
		CompletionTokens: e.CompletionTokens,
		TotalTokens:      e.TotalTokens,
		InputChars:       e.InputChars,
		OutputChars:      e.OutputChars,
		AudioMs:          e.AudioMs,
		AudioBytes:       e.AudioBytes,
		IntentName:       truncateAIInvocationField(e.IntentName, 128),
		Confidence:       e.Confidence,
		RequestText:      truncateAIInvocationBody(e.RequestText),
		ResponseText:     truncateAIInvocationBody(e.ResponseText),
		ErrorMsg:         e.ErrorMsg,
		MetaJSON:         metaJSON,
		CreatedAt:        time.Now(),
	}
	_ = gdb.Create(&row).Error
}

func resolveAIInvocationTenantFromCallID(gdb *gorm.DB, callID string) uint {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return 0
	}
	return callbinding.GetTenantID(callID)
}

func truncateAIInvocationField(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

func truncateAIInvocationBody(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxAIInvocationBodyRunes {
		return s
	}
	return string(runes[:maxAIInvocationBodyRunes]) + "\n…[truncated]"
}

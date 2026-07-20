package callbinding

import (
	"fmt"
	"strings"
	"sync"
)

// SpeakerAttrVisibility controls who may see an attribute value.
const (
	SpeakerVisibilityLLM      = "llm"
	SpeakerVisibilityInternal = "internal"
	SpeakerVisibilityTool     = "tool"
)

// SpeakerAttribute is one key/value on a logical speaker subject.
type SpeakerAttribute struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Visibility string `json:"visibility"` // llm | internal | tool
}

// SpeakerCredentialRef is a tool-runtime credential handle (never for LLM prompts).
type SpeakerCredentialRef struct {
	Provider  string   `json:"provider"` // cloudsteps | crm | ...
	SecretRef string   `json:"-"`        // raw secret / token; omitted from JSON logs
	Scopes    []string `json:"scopes,omitempty"`
	HasSecret bool     `json:"hasSecret"`
}

// SpeakerContext is the per-call resolved speaker identity for LLM + tools.
type SpeakerContext struct {
	SubjectID   uint                   `json:"subjectId,omitempty"`
	ProfileID   uint                   `json:"profileId,omitempty"`
	FeatureID   string                 `json:"featureId,omitempty"`
	DisplayName string                 `json:"displayName,omitempty"`
	Score       float64                `json:"score,omitempty"`
	Threshold   float64                `json:"threshold,omitempty"`
	Verified    bool                   `json:"verified"`
	Confidence  string                 `json:"confidence,omitempty"`
	Attributes  []SpeakerAttribute     `json:"attributes,omitempty"`
	Credentials []SpeakerCredentialRef `json:"credentials,omitempty"`
}

var callSpeakers sync.Map // callID -> SpeakerContext

const speakerHintMarker = "【本通说话人】"

// SetSpeakerContext stores the resolved speaker for this call leg.
func SetSpeakerContext(callID string, ctx SpeakerContext) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callSpeakers.Store(callID, ctx)
}

// GetSpeakerContext returns the bound speaker context (ok=false if unset).
func GetSpeakerContext(callID string) (SpeakerContext, bool) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return SpeakerContext{}, false
	}
	v, ok := callSpeakers.Load(callID)
	if !ok {
		return SpeakerContext{}, false
	}
	ctx, ok := v.(SpeakerContext)
	return ctx, ok
}

// ClearSpeakerContext removes call-scoped speaker binding.
func ClearSpeakerContext(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callSpeakers.Delete(callID)
}

// LLMAttributes returns attributes with visibility=llm (and empty→llm).
func (c SpeakerContext) LLMAttributes() []SpeakerAttribute {
	out := make([]SpeakerAttribute, 0, len(c.Attributes))
	for _, a := range c.Attributes {
		vis := strings.TrimSpace(a.Visibility)
		if vis == "" || vis == SpeakerVisibilityLLM {
			out = append(out, a)
		}
	}
	return out
}

// CredentialFor returns the first credential matching provider (case-insensitive).
func (c SpeakerContext) CredentialFor(provider string) (SpeakerCredentialRef, bool) {
	want := strings.ToLower(strings.TrimSpace(provider))
	if want == "" {
		return SpeakerCredentialRef{}, false
	}
	for _, ref := range c.Credentials {
		if strings.ToLower(strings.TrimSpace(ref.Provider)) == want && strings.TrimSpace(ref.SecretRef) != "" {
			return ref, true
		}
	}
	return SpeakerCredentialRef{}, false
}

// FormatPromptCard builds the LLM-safe speaker card (no secrets).
func FormatPromptCard(ctx SpeakerContext) string {
	if strings.TrimSpace(ctx.DisplayName) == "" && ctx.FeatureID == "" && ctx.SubjectID == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(speakerHintMarker)
	if ctx.DisplayName != "" {
		b.WriteString("\n姓名：")
		b.WriteString(ctx.DisplayName)
	}
	if ctx.FeatureID != "" {
		b.WriteString("\nfeatureId：")
		b.WriteString(ctx.FeatureID)
	}
	if ctx.SubjectID > 0 {
		b.WriteString(fmt.Sprintf("\nsubjectId：%d", ctx.SubjectID))
	}
	status := "未确认"
	if ctx.Verified {
		status = "已确认"
	}
	if ctx.Score > 0 {
		b.WriteString(fmt.Sprintf("\n置信度：%.4f（%s）", ctx.Score, status))
	} else {
		b.WriteString("\n状态：")
		b.WriteString(status)
	}
	for _, a := range ctx.LLMAttributes() {
		b.WriteString("\n")
		b.WriteString(a.Key)
		b.WriteString("：")
		b.WriteString(a.Value)
	}
	providers := make([]string, 0)
	for _, c := range ctx.Credentials {
		if c.HasSecret || strings.TrimSpace(c.SecretRef) != "" {
			providers = append(providers, c.Provider)
		}
	}
	if len(providers) > 0 {
		b.WriteString("\n可用工具凭证：")
		b.WriteString(strings.Join(providers, "、"))
	}
	b.WriteString("\n规则：涉及该说话人本人业务时，优先按其身份调用工具；未命中或低置信度时先口头确认身份，禁止臆造；禁止索要或复述任何 token/密钥。")
	return b.String()
}

// ApplySpeakerHint merges the speaker card into a system prompt (idempotent).
func ApplySpeakerHint(base, card string) string {
	base = StripSpeakerHint(base)
	card = strings.TrimSpace(card)
	if card == "" {
		return base
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return card
	}
	return base + "\n\n" + card
}

// StripSpeakerHint removes an embedded speaker card block.
func StripSpeakerHint(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, speakerHintMarker); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// PromptCardForCall returns the formatted card for a bound call (empty if unset).
func PromptCardForCall(callID string) string {
	ctx, ok := GetSpeakerContext(callID)
	if !ok {
		return ""
	}
	return FormatPromptCard(ctx)
}

// EnrichSystemPrompt appends the speaker card for this call.
func EnrichSystemPrompt(base, callID string) string {
	return ApplySpeakerHint(base, PromptCardForCall(callID))
}

// EnrichMCPArgs injects speaker credentials into MCP tool arguments when missing.
func EnrichMCPArgs(callID, toolName string, args map[string]interface{}) map[string]interface{} {
	if args == nil {
		args = map[string]interface{}{}
	}
	ctx, ok := GetSpeakerContext(callID)
	if !ok {
		return args
	}
	name := strings.ToLower(strings.TrimSpace(toolName))
	if strings.HasPrefix(name, "cloudsteps_") {
		if s, ok := args["token"].(string); ok && strings.TrimSpace(s) != "" {
			return args
		}
		if ref, ok := ctx.CredentialFor("cloudsteps"); ok {
			args["token"] = ref.SecretRef
		}
		return args
	}
	if _, has := args["token"]; !has {
		for _, ref := range ctx.Credentials {
			p := strings.ToLower(strings.TrimSpace(ref.Provider))
			if p != "" && strings.HasPrefix(name, p+"_") && strings.TrimSpace(ref.SecretRef) != "" {
				args["token"] = ref.SecretRef
				break
			}
		}
	}
	return args
}

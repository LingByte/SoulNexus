package providers

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"go.uber.org/zap"
)

// SpeakerToolHooks wires optional voiceprint speaker tools without importing stages/speaker
// (avoids import cycles with internal/models).
type SpeakerToolHooks struct {
	GetContextName        string
	GetContextDescription string
	GetContextParams      json.RawMessage
	GetContext            func(callID string) string

	IdentifyName        string
	IdentifyDescription string
	IdentifyParams      json.RawMessage
	Identify            func(ctx context.Context, callID string, tenantID, assistantID uint, args map[string]interface{}, lg *zap.Logger) string
}

var (
	speakerHooksMu sync.RWMutex
	speakerHooks   *SpeakerToolHooks
)

// SetSpeakerToolHooks installs speaker LLM tool callbacks (call from app wiring).
func SetSpeakerToolHooks(h *SpeakerToolHooks) {
	speakerHooksMu.Lock()
	speakerHooks = h
	speakerHooksMu.Unlock()
}

func getSpeakerToolHooks() *SpeakerToolHooks {
	speakerHooksMu.RLock()
	defer speakerHooksMu.RUnlock()
	return speakerHooks
}

// RegisterLLMTools registers optional knowledge + speaker tools on a ChatLLM session.
func RegisterLLMTools(provider ChatLLM, callID string, lg *zap.Logger) {
	registerKnowledgeTool(provider, callID, lg)
	registerSpeakerTools(provider, callID, lg)
}

func registerSpeakerTools(provider ChatLLM, callID string, lg *zap.Logger) {
	h := getSpeakerToolHooks()
	if provider == nil || strings.TrimSpace(callID) == "" || h == nil {
		return
	}
	if h.GetContext != nil && h.GetContextName != "" {
		params := h.GetContextParams
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","additionalProperties":false}`)
		}
		provider.RegisterFunctionTool(
			h.GetContextName,
			h.GetContextDescription,
			params,
			func(_ map[string]interface{}, _ interface{}) (string, error) {
				if lg != nil {
					lg.Info("voice: get_speaker_context invoked", zap.String("call_id", callID))
				}
				return h.GetContext(callID), nil
			},
		)
	}
	if h.Identify != nil && h.IdentifyName != "" {
		params := h.IdentifyParams
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","additionalProperties":true}`)
		}
		provider.RegisterFunctionTool(
			h.IdentifyName,
			h.IdentifyDescription,
			params,
			func(args map[string]interface{}, _ interface{}) (string, error) {
				if lg != nil {
					lg.Info("voice: identify_speaker invoked", zap.String("call_id", callID))
				}
				return h.Identify(
					context.Background(),
					callID,
					callbinding.GetTenantID(callID),
					callbinding.GetAssistantID(callID),
					args,
					lg,
				), nil
			},
		)
	}
}

func registerKnowledgeTool(provider ChatLLM, callID string, lg *zap.Logger) {
	if provider == nil || strings.TrimSpace(callID) == "" {
		return
	}
	b := stageknow.ResolveBinding(callID)
	if !b.Enabled {
		return
	}
	if lg != nil {
		lg.Info("voice: knowledge search tool registered",
			zap.String("call_id", callID),
			zap.Uint("namespace_id", b.NamespaceID),
			zap.String("collection", b.Collection),
		)
	}
	provider.RegisterFunctionTool(
		stageknow.SearchToolName,
		stageknow.SearchToolDescription,
		stageknow.SearchToolParams,
		func(args map[string]interface{}, _ interface{}) (string, error) {
			if lg != nil {
				lg.Info("voice: search_knowledge_base tool invoked",
					zap.String("call_id", callID),
					zap.Any("args", args),
				)
			}
			return stageknow.ExecuteSearchTool(context.Background(), callID, args, lg), nil
		},
	)
}

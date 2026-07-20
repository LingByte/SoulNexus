package providers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/lingllm/realtime"
	"go.uber.org/zap"
)

// RealtimeSpeakerTools returns identify/get_speaker_context FC definitions when hooks are installed.
func RealtimeSpeakerTools() []realtime.Tool {
	h := getSpeakerToolHooks()
	if h == nil {
		return nil
	}
	var out []realtime.Tool
	if h.GetContext != nil && strings.TrimSpace(h.GetContextName) != "" {
		params := h.GetContextParams
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","additionalProperties":false}`)
		}
		out = append(out, realtime.Tool{
			Name:        h.GetContextName,
			Description: h.GetContextDescription,
			Parameters:  params,
		})
	}
	if h.Identify != nil && strings.TrimSpace(h.IdentifyName) != "" {
		params := h.IdentifyParams
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","additionalProperties":true}`)
		}
		out = append(out, realtime.Tool{
			Name:        h.IdentifyName,
			Description: h.IdentifyDescription,
			Parameters:  params,
		})
	}
	return out
}

// InvokeRealtimeSpeakerTool runs a speaker FC by name. ok=false when not a speaker tool.
func InvokeRealtimeSpeakerTool(callID, name string, args map[string]any, lg *zap.Logger) (out string, ok bool) {
	h := getSpeakerToolHooks()
	if h == nil || strings.TrimSpace(callID) == "" {
		return "", false
	}
	name = strings.TrimSpace(name)
	switch name {
	case h.GetContextName:
		if h.GetContext == nil {
			return "", false
		}
		if lg != nil {
			lg.Info("voice (realtime): get_speaker_context invoked", zap.String("call_id", callID))
		}
		return h.GetContext(callID), true
	case h.IdentifyName:
		if h.Identify == nil {
			return "", false
		}
		if lg != nil {
			lg.Info("voice (realtime): identify_speaker invoked", zap.String("call_id", callID))
		}
		raw := make(map[string]interface{}, len(args))
		for k, v := range args {
			raw[k] = v
		}
		return h.Identify(
			context.Background(),
			callID,
			callbinding.GetTenantID(callID),
			callbinding.GetAssistantID(callID),
			raw,
			lg,
		), true
	default:
		return "", false
	}
}

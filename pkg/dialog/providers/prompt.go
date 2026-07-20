package providers

import (
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	dialogaudio "github.com/LingByte/SoulNexus/pkg/dialog/audio"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

var (
	systemPromptMu       sync.Mutex
	systemPromptByCallID = map[string]string{}
)

// SetCallSystemPrompt stores a per-call system prompt override consumed once by PipelineSystemPrompt.
func SetCallSystemPrompt(callID, prompt string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	systemPromptMu.Lock()
	systemPromptByCallID[callID] = strings.TrimSpace(prompt)
	systemPromptMu.Unlock()
}

// PopCallSystemPrompt returns and clears a per-call system prompt override.
func PopCallSystemPrompt(callID string) string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ""
	}
	systemPromptMu.Lock()
	defer systemPromptMu.Unlock()
	v := strings.TrimSpace(systemPromptByCallID[callID])
	delete(systemPromptByCallID, callID)
	return v
}

// PipelineSystemPrompt merges tenant instructions with per-call override and tool rules.
func PipelineSystemPrompt(env tenantcfg.VoiceEnv, callID string) string {
	core := strings.TrimSpace(env.LLMInstructions)
	if per := PopCallSystemPrompt(callID); per != "" {
		if core != "" {
			core = core + "\n\n" + per
		} else {
			core = per
		}
	}
	out := AugmentToolRules(core, stageknow.ResolveBinding(callID).Enabled)
	out = callbinding.EnrichSystemPrompt(out, callID)
	return dialogaudio.ApplyNoiseHint(out, dialogaudio.GlobalCallNoise.HintForCall(callID))
}

// AugmentToolRules appends builtin tool / knowledge rules after operatorCore.
func AugmentToolRules(operatorCore string, includeKnowledge bool) string {
	rule := toolPromptRule(includeKnowledge)
	if includeKnowledge {
		rule = mergeInstructions(rule, stageknow.SearchPromptHint())
	}
	user := strings.TrimSpace(operatorCore)
	if user == "" {
		return rule
	}
	return user + "\n\n" + rule
}

func toolPromptRule(includeKnowledge bool) string {
	tools := "后台工具（勿向用户宣读）："
	if includeKnowledge {
		tools += "search_knowledge_base 用于检索企业知识库；"
	}
	tools += "get_current_time、is_business_hours、calculate。"
	return "问时间请调用 get_current_time，不要编造。" + "\n" + tools
}

func mergeInstructions(base, hint string) string {
	base = strings.TrimSpace(base)
	hint = strings.TrimSpace(hint)
	switch {
	case base == "":
		return hint
	case hint == "":
		return base
	default:
		return base + "\n\n" + hint
	}
}

// AugmentTextDialogPrompt appends accuracy-first IM guidance (not oral short-reply voice style).
func AugmentTextDialogPrompt(systemPrompt string, _ tenantcfg.VoiceEnv) string {
	core := strings.TrimSpace(systemPrompt)
	suffix := "【文本通道】当前是文字客服/IM 对话（非实时语音）。" +
		"请严格遵循上方助手人设与业务规则；回答准确、完整，可分点说明。" +
		"不要使用「口语极短、不超过几十字」的语音话术约束。" +
		"需要调用工具或检索知识库时，先完成再给出最终答复。"
	if core == "" {
		return suffix
	}
	return core + "\n\n" + suffix
}

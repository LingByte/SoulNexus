// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package nlu wires tenant ONNX NLU into the voice dialog path (ASR → NLU → LLM/TTS).
package nlu

import (
	"fmt"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/intentonnx"
	pkgnlu "github.com/LingByte/SoulNexus/pkg/nlu"
	"go.uber.org/zap"
)

// Binding holds the per-call NLU profile (paths already resolved).
type Binding struct {
	Enabled bool
	Paths   pkgnlu.ProfilePaths
	ModelID uint
}

var bindingCache sync.Map // callID → Binding

// PrepareCallNLUBinding caches NLU paths for one call from VoiceEnv.
func PrepareCallNLUBinding(callID string, env tenantcfg.VoiceEnv, lg *zap.Logger) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	if !pkgnlu.DeployEnabled() || env.NluModelID == 0 || strings.TrimSpace(env.NluIntentsPath) == "" {
		ClearBinding(callID)
		if lg != nil {
			lg.Info("nlu: call binding skipped (WS/WebRTC share this path)",
				zap.String("call_id", callID),
				zap.Bool("nlu_deploy_enabled", pkgnlu.DeployEnabled()),
				zap.Uint("nlu_model_id", env.NluModelID),
				zap.Bool("has_intents_path", strings.TrimSpace(env.NluIntentsPath) != ""),
			)
		}
		return
	}
	paths := pkgnlu.ProfilePaths{
		ModelPath:      strings.TrimSpace(env.NluModelPath),
		TokenizerPath:  strings.TrimSpace(env.NluTokenizerPath),
		IntentsPath:    strings.TrimSpace(env.NluIntentsPath),
		PrototypesPath: strings.TrimSpace(env.NluPrototypesPath),
		MinConfidence:  env.NluMinConfidence,
	}
	if paths.MinConfidence <= 0 {
		paths.MinConfidence = pkgnlu.Get().MinConfidence
	}
	// Voice path: only high-confidence intents skip LLM (avoid false canned replies).
	const voiceMinConfidenceFloor = 0.85
	if paths.MinConfidence < voiceMinConfidenceFloor {
		paths.MinConfidence = voiceMinConfidenceFloor
	}
	b := Binding{Enabled: true, Paths: paths, ModelID: env.NluModelID}
	bindingCache.Store(callID, b)
	if lg != nil {
		lg.Info("nlu: call binding ready",
			zap.String("call_id", callID),
			zap.Uint("nlu_model_id", env.NluModelID),
			zap.Float64("min_confidence", paths.MinConfidence),
		)
	}
}

// ClearBinding drops the per-call NLU cache.
func ClearBinding(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	bindingCache.Delete(callID)
}

// ResolveBinding returns the cached binding (Enabled=false when missing).
func ResolveBinding(callID string) Binding {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return Binding{}
	}
	if v, ok := bindingCache.Load(callID); ok {
		if b, ok := v.(Binding); ok {
			return b
		}
	}
	return Binding{}
}

// TurnResult is the outcome of ProcessTurn.
type TurnResult struct {
	// SkipLLM means Reply is the full user-facing answer (intent canned reply).
	SkipLLM bool
	Reply   string
	// EnrichedUserText is the text to send to the LLM (original + NLU context block).
	EnrichedUserText string
	IntentName       string
	Confidence       float64
	Channel          string
}

// ProcessTurn runs NLU after ASR. When unbound or on error, returns EnrichedUserText=userText.
func ProcessTurn(callID, userText string, lg *zap.Logger) TurnResult {
	userText = strings.TrimSpace(userText)
	out := TurnResult{EnrichedUserText: userText}
	if userText == "" {
		return out
	}
	b := ResolveBinding(callID)
	if !b.Enabled {
		return out
	}
	route, err := pkgnlu.ParseProfile(b.Paths, userText)
	if err != nil {
		if lg != nil {
			lg.Warn("nlu: parse failed, falling back to LLM",
				zap.String("call_id", callID),
				zap.Error(err),
			)
		}
		return out
	}
	if route == nil {
		return out
	}
	out.IntentName = strings.TrimSpace(route.Prediction.IntentName)
	out.Confidence = route.Prediction.Confidence
	latencySource := "voice_nlu"
	recordNLUInvocation(callID, b.ModelID, userText, route, nil)

	switch route.Channel {
	case intentonnx.AnswerChannelIntent:
		reply := strings.TrimSpace(route.Reply)
		if reply == "" {
			out.Channel = "llm"
			out.EnrichedUserText = enrichForLLM(userText, route)
			return out
		}
		out.SkipLLM = true
		out.Reply = reply
		out.Channel = "intent"
		out.EnrichedUserText = userText
		if lg != nil {
			lg.Info("nlu: intent reply (skip LLM)",
				zap.String("call_id", callID),
				zap.String("intent", out.IntentName),
				zap.Float64("confidence", out.Confidence),
				zap.String("source", latencySource),
			)
		}
		return out
	default:
		out.Channel = "llm"
		out.EnrichedUserText = enrichForLLM(userText, route)
		if lg != nil {
			lg.Info("nlu: low/uncertain → LLM with context",
				zap.String("call_id", callID),
				zap.String("intent", out.IntentName),
				zap.Float64("confidence", out.Confidence),
			)
		}
		return out
	}
}

func enrichForLLM(userText string, route *intentonnx.RouteOutput) string {
	if route == nil {
		return userText
	}
	intent := strings.TrimSpace(route.Prediction.IntentName)
	score := route.Prediction.Confidence
	var b strings.Builder
	b.WriteString(userText)
	b.WriteString("\n\n【系统·NLU】")
	if intent != "" {
		b.WriteString(fmt.Sprintf("初步意图=%s，置信度=%.3f。", intent, score))
	} else {
		b.WriteString(fmt.Sprintf("未命中高置信意图（置信度=%.3f）。", score))
	}
	b.WriteString("请在业务意图与知识库范围内作答；不要编造未配置的意图。若意图明确，优先给出对应业务答复。")
	return b.String()
}

// IntentCatalogHint lists configured intent names for realtime system instructions.
func IntentCatalogHint(callID string) string {
	b := ResolveBinding(callID)
	if !b.Enabled || strings.TrimSpace(b.Paths.IntentsPath) == "" {
		return ""
	}
	cfg, err := intentonnx.LoadIntentConfig(b.Paths.IntentsPath)
	if err != nil || cfg == nil || len(cfg.Intents) == 0 {
		return ""
	}
	names := make([]string, 0, len(cfg.Intents))
	for _, ent := range cfg.Intents {
		n := strings.TrimSpace(ent.Name)
		if n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "【NLU·业务意图】本会话已绑定意图模型。用户说法应优先归入以下意图之一：" +
		strings.Join(names, "、") +
		"。请只在这些意图与知识库范围内作答；不确定时先澄清再答。"
}

func recordNLUInvocation(callID string, modelID uint, text string, route *intentonnx.RouteOutput, err error) {
	entry := callbinding.AIInvocationRecord{
		Component:   callbinding.AIComponentNLU,
		Provider:    "onnx",
		Model:       fmt.Sprintf("nlu_model_%d", modelID),
		CallID:      callID,
		Source:      "voice_nlu",
		RequestText: text,
		Status:      callbinding.AIStatusOK,
	}
	if route != nil {
		entry.ResponseText = fmt.Sprintf("intent=%s conf=%.3f channel=%v",
			route.Prediction.IntentName, route.Prediction.Confidence, route.Channel)
		entry.Meta = map[string]any{
			"intent":     route.Prediction.IntentName,
			"confidence": route.Prediction.Confidence,
			"channel":    fmt.Sprintf("%v", route.Channel),
		}
	}
	if err != nil {
		entry.Status = callbinding.AIStatusError
		entry.ErrorMsg = err.Error()
	}
	callbinding.RecordAIInvocation(entry)
}

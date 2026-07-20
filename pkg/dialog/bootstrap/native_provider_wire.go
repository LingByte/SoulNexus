package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	dialogaudio "github.com/LingByte/SoulNexus/pkg/dialog/audio"
	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	stageknow "github.com/LingByte/SoulNexus/pkg/dialog/stages/knowledge"
	stagenlu "github.com/LingByte/SoulNexus/pkg/dialog/stages/nlu"
	stagespeaker "github.com/LingByte/SoulNexus/pkg/dialog/stages/speaker"
	"github.com/LingByte/SoulNexus/pkg/dialog/stages/rewrite"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/turn"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceattach"
	voiceMetrics "github.com/LingByte/SoulNexus/pkg/voice/metrics"
	"go.uber.org/zap"
)

var errLLMNilProvider = errors.New("native cascaded LLM: nil provider")

type nativeCascadedLLM struct {
	provider   providers.ChatLLM
	model      string
	callID     string
	basePrompt string // without dynamic 【系统·声学环境】 block
	rewriter   rewrite.Config
	lg         *zap.Logger
}

func (s *nativeCascadedLLM) log() *zap.Logger {
	if s != nil && s.lg != nil {
		return s.lg
	}
	return zap.NewNop()
}

func (s *nativeCascadedLLM) refreshNoisePrompt() {
	if s == nil || s.provider == nil {
		return
	}
	base := callbinding.EnrichSystemPrompt(s.basePrompt, s.callID)
	base = strings.TrimSpace(base) + "\n\n" + providers.WallClockPromptHint(time.Now())
	hint := dialogaudio.GlobalCallNoise.HintForCall(s.callID)
	s.provider.SetSystemPrompt(dialogaudio.ApplyNoiseHint(base, hint))
}

func (s *nativeCascadedLLM) StreamReply(
	ctx context.Context,
	userText string,
	onDelta func(text string, isComplete bool) error,
) (string, error) {
	if s == nil || s.provider == nil {
		return "", errLLMNilProvider
	}
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return "", nil
	}
	lg := s.log()
	// Cascaded has no session.update — refresh system prompt each turn from live SNR class.
	s.refreshNoisePrompt()

	rawASR := userText
	lg.Info("cascaded: turn start",
		zap.String("call_id", s.callID),
		zap.String("turn_preview", truncateRunes(rawASR, 60)),
	)

	type kbOut struct{ block string }
	kbCh := make(chan kbOut, 1)
	forcedKB := strings.TrimSpace(cascaded.ForcedKnowledgeBlock(ctx))
	if forcedKB != "" {
		kbCh <- kbOut{block: forcedKB}
	} else {
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					lg.Error("cascaded: knowledge search panic",
						zap.String("call_id", s.callID),
						zap.Any("recover", rec),
					)
					select {
					case kbCh <- kbOut{}:
					default:
					}
				}
			}()
			block := ""
			if s.callID != "" {
				block = stageknow.SearchBlockForQuery(context.Background(), s.callID, rawASR, lg)
			}
			kbCh <- kbOut{block: block}
		}()
	}

	// Rewrite can call a remote LLM with no provider timeout — hard-cap it.
	rewritten := rawASR
	if s.rewriter.UseRewriter {
		type rwOut struct{ text string }
		rwCh := make(chan rwOut, 1)
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					rwCh <- rwOut{text: rawASR}
				}
			}()
			rwCh <- rwOut{text: rewrite.Rewrite(ctx, s.provider, s.model, s.callID, s.rewriter, rawASR)}
		}()
		select {
		case out := <-rwCh:
			rewritten = out.text
		case <-time.After(4 * time.Second):
			lg.Warn("cascaded: rewrite timed out, using ASR text",
				zap.String("call_id", s.callID),
			)
			rewritten = rawASR
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	userText = rewritten

	nluTurn := stagenlu.ProcessTurn(s.callID, userText, lg)

	kbWait := stageknow.SearchTimeout() + 800*time.Millisecond
	var kbBlock string
	select {
	case out := <-kbCh:
		kbBlock = out.block
	case <-time.After(kbWait):
		lg.Warn("cascaded: knowledge wait timed out, continuing without KB",
			zap.String("call_id", s.callID),
			zap.Duration("wait", kbWait),
		)
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// Speculative ASR partials must not play canned intent replies — the
	// utterance often still grows ("我需要" → "我需要预约课程").
	speculative := cascaded.IsSpeculativeLLM(ctx)
	if nluTurn.SkipLLM && strings.TrimSpace(nluTurn.Reply) != "" && !speculative {
		if err := onDelta(nluTurn.Reply, false); err != nil {
			return nluTurn.Reply, err
		}
		_ = onDelta("", true)
		return nluTurn.Reply, nil
	}
	if speculative && nluTurn.SkipLLM {
		lg.Info("cascaded: speculative defer NLU canned reply → LLM",
			zap.String("call_id", s.callID),
			zap.String("intent", nluTurn.IntentName),
			zap.Float64("confidence", nluTurn.Confidence),
		)
	}
	userText = nluTurn.EnrichedUserText
	if kbBlock != "" {
		before := userText
		userText = strings.TrimSpace(userText) + "\n\n" + kbBlock + "\n\n" + stageknow.QuotePromptAddon()
		lg.Info("cascaded: knowledge context injected before LLM",
			zap.String("call_id", s.callID),
			zap.Int("text_len_before", len([]rune(before))),
			zap.Int("text_len_after", len([]rune(userText))),
		)
	}

	invCallID := strings.TrimSpace(s.callID)
	invSource := ""
	if invCallID != "" {
		invSource = callbinding.GetAISource(invCallID)
	}
	// Prefer QueryStream so TTS can start on the first tokens. Non-stream
	// QueryWithOptions (tool chain) only when catalog/MCP tools are bound —
	// builtin transfer/knowledge/speaker are covered by ASR intent + KB inject.
	toolNames := s.provider.ListFunctionTools()
	useTools := providers.NeedsNonStreamToolRound(toolNames, userText)
	if cascaded.PreferTools(ctx) && len(toolNames) > 0 {
		useTools = true
	}
	lg.Info("cascaded: calling LLM",
		zap.String("call_id", s.callID),
		zap.String("ai_source", invSource),
		zap.Bool("has_tools", len(toolNames) > 0),
		zap.Bool("use_tools", useTools),
		zap.Bool("has_kb", kbBlock != ""),
		zap.Strings("tools", toolNames),
	)
	if useTools {
		temp := float32(0.7)
		maxRounds := cascaded.MaxToolRounds(ctx)
		if maxRounds <= 0 && cascaded.PreferTools(ctx) {
			maxRounds = cascaded.DefaultTextToolRounds
		}
		reply, err := s.provider.QueryWithOptions(userText, providers.LLMQueryOptions{
			Model:            s.model,
			Temperature:      &temp,
			Context:          ctx,
			InvocationCallID: invCallID,
			InvocationSource: invSource,
			MaxToolRounds:    maxRounds,
		})
		if err != nil {
			lg.Warn("cascaded: tool LLM failed, falling back to stream",
				zap.String("call_id", s.callID),
				zap.Error(err),
			)
		} else {
			if err := onDelta(reply, false); err != nil {
				return reply, err
			}
			_ = onDelta("", true)
			return reply, nil
		}
	}
	options := providers.LLMQueryOptions{
		Model:            s.model,
		Stream:           true,
		Context:          ctx,
		InvocationCallID: invCallID,
		InvocationSource: invSource,
		// Stream path cannot execute tool rounds — never advertise tools here.
		DisableTools: true,
	}
	cancelled := false
	guarded := func(segment string, isComplete bool) error {
		if cancelled {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			cancelled = true
			return ctx.Err()
		default:
		}
		return onDelta(segment, isComplete)
	}
	return s.provider.QueryStream(userText, options, guarded)
}

func (s *nativeCascadedLLM) SeedHistory(msgs []providers.LLMMessage) {
	if s == nil || s.provider == nil {
		return
	}
	s.provider.SeedMessages(msgs)
}

func (s *nativeCascadedLLM) AppendSystemAppendix(appendix string) {
	if s == nil || s.provider == nil {
		return
	}
	appendix = strings.TrimSpace(appendix)
	if appendix == "" {
		return
	}
	s.basePrompt = strings.TrimSpace(s.basePrompt) + "\n\n" + appendix
	s.refreshNoisePrompt()
}

func (s *nativeCascadedLLM) RegisterFunctionTool(
	name, description string,
	parameters interface{},
	callback providers.LLMFunctionToolCallback,
) {
	if s == nil || s.provider == nil || callback == nil {
		return
	}
	s.provider.RegisterFunctionTool(name, description, parameters, callback)
}

func (s *nativeCascadedLLM) LastToolTrace() []providers.LLMToolCall {
	if s == nil || s.provider == nil {
		return nil
	}
	return s.provider.LastToolTrace()
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if max <= 0 || len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func buildNativeCascadedLLM(ctx context.Context, env voiceattach.VoiceEnv, callID string, lg *zap.Logger) (cascaded.LLMService, error) {
	if strings.TrimSpace(env.LLMProvider) == "" {
		return nil, fmt.Errorf("native cascaded LLM: env.LLMProvider unset")
	}
	if strings.TrimSpace(env.LLMAPIKey) == "" && !strings.EqualFold(env.LLMProvider, "ollama") {
		return nil, fmt.Errorf("native cascaded LLM: missing API key for provider %q", env.LLMProvider)
	}
	model := env.LLMModel
	if model == "" {
		model = "gpt-4o"
	}
	systemPrompt := providers.PipelineSystemPrompt(tenantcfg.VoiceEnv(env), callID)
	if cascaded.IsTextDialog(ctx) {
		systemPrompt = providers.AugmentTextDialogPrompt(systemPrompt, tenantcfg.VoiceEnv(env))
	}
	basePrompt := dialogaudio.StripNoiseHint(systemPrompt)
	provider, err := providers.NewChatLLM(ctx, env.LLMProvider, env.LLMAPIKey, llmAPIURLForProvider(env), systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("native cascaded LLM: provider init: %w", err)
	}
	stageknow.PrepareCallKnowledgeBinding(
		callID,
		tenantcfg.VoiceEnv(env),
		env.TenantID,
		stageknow.SearchConfigFromVoiceEnv(tenantcfg.VoiceEnv(env)),
		lg,
	)
	stagenlu.PrepareCallNLUBinding(callID, tenantcfg.VoiceEnv(env), lg)
	speakerHint := stagespeaker.PrepareCallSpeakerBinding(callID, env.TenantID, env.AssistantID, lg)
	registerVoiceLLMTools(ctx, provider, env, callID, lg)
	if hint := providers.CatalogToolsUsageHint(provider.ListFunctionTools()); hint != "" {
		basePrompt = strings.TrimSpace(basePrompt) + "\n\n" + hint
	}
	if speakerHint != "" {
		basePrompt = strings.TrimSpace(basePrompt) + "\n\n" + speakerHint
	}
	provider.SetSystemPrompt(dialogaudio.ApplyNoiseHint(basePrompt, dialogaudio.GlobalCallNoise.HintForCall(callID)))
	if lg != nil {
		lg.Info("native cascaded LLM tools ready",
			zap.String("call_id", callID),
			zap.Uint("assistant_id", env.AssistantID),
			zap.Bool("speaker_hint", speakerHint != ""),
			zap.Strings("tools", provider.ListFunctionTools()),
		)
	}
	rewriter := rewrite.ParseConfig(env.QueryRewriterRaw)
	if len(env.QueryRewriterRaw) == 0 {
		rewriter = rewrite.ParseConfig(env.AgentConfigRaw)
	}
	llm := &nativeCascadedLLM{
		provider:   provider,
		model:      model,
		callID:     callID,
		basePrompt: basePrompt,
		rewriter:   rewriter,
		lg:         lg,
	}
	llm.refreshNoisePrompt()
	go warmLLMConnection(llmAPIURLForProvider(env), env.LLMAPIKey, lg)
	return llm, nil
}

// warmLLMConnection establishes TLS/HTTP keep-alive to the LLM endpoint so
// the first real turn avoids a cold dial. Does not touch chat history.
func warmLLMConnection(baseURL, apiKey string, lg *zap.Logger) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	probe := baseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probe, nil)
	if err != nil {
		return
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if lg != nil {
			lg.Debug("native cascaded LLM: warm probe failed", zap.Error(err))
		}
		return
	}
	_ = resp.Body.Close()
	if lg != nil {
		lg.Debug("native cascaded LLM: warm probe done", zap.Int("status", resp.StatusCode))
	}
}

func buildNativeTurnPersister(env voiceattach.VoiceEnv, callID string, lg *zap.Logger) cascaded.TurnPersister {
	asrProv := "qcloud_asr"
	if env.ASRModelType != "" {
		asrProv = env.ASRModelType
	}
	ttsProv := strings.TrimSpace(env.TTSProvider)
	if ttsProv == "" {
		ttsProv = "qcloud_tts"
	}
	llmModel := strings.TrimSpace(env.LLMModel)
	if llmModel == "" {
		llmModel = "gpt-4o"
	}
	return cascaded.TurnPersisterFunc(func(ctx context.Context, rec cascaded.TurnRecord) {
		hasContent := strings.TrimSpace(rec.UserText) != "" || strings.TrimSpace(rec.AIText) != ""
		if hasContent {
			dt := turn.Turn{
				ASRText:     rec.UserText,
				LLMText:     rec.AIText,
				ASRProvider: asrProv,
				TTSProvider: ttsProv,
				LLMModel:    llmModel,
				Trigger:     "final",
				LLMFirstMs:  rec.LLMFirstMs,
				LLMWallMs:   rec.LLMWallMs,
				PipelineMs:  rec.PipelineMs,
				At:          rec.CompletedAt,
			}
			bg := context.Background()
			go providers.RecordTurn(bg, callID, dt)
			voiceMetrics.ObserveLLMFirstByte(rec.LLMFirstMs)
			voiceMetrics.ObserveTTSFirstByte(rec.TTSFirstByteMs)
			e2e := rec.E2EFirstByteMs
			if e2e <= 0 {
				e2e = rec.TTSFirstByteMs
			}
			if e2e <= 0 {
				e2e = rec.LLMFirstMs
			}
			voiceMetrics.ObserveE2EFirstByte(e2e)
		}
	})
}

func llmAPIURLForProvider(env voiceattach.VoiceEnv) string {
	return strings.TrimSpace(env.LLMBaseURL)
}


package outbound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
)

const llmRouteSystemPromptLegacy = `You route a voice outbound script. Output only JSON {"next_id":"<id>"} matching one allowed id exactly.`

const llmRouteSystemPromptCompact = `You route a voice outbound script. Output only JSON {"i":N} where N is a non-negative integer index from the numbered list in the user message. No prose, no markdown.`

// ErrListenRouteNoLLM is returned when a listen step has transitions but CHECK_LLM_* is not configured.
var ErrListenRouteNoLLM = errors.New("listen transitions require CHECK_LLM_* configuration")

// ErrListenRouteLLMFail is returned when the routing LLM call fails or returns an invalid next_id.
var ErrListenRouteLLMFail = errors.New("listen LLM routing failed")

// resolveListenBranches picks next_id after a successful listen when the step defines transitions.
// Keyword matching is not used for listen; disambiguation is LLM-only. Empty transitions returns ("", nil).
// When multiple branch targets exist, LLM is required; on failure returns a non-nil error (caller apologizes and ends).
func resolveListenBranches(ctx context.Context, leg EstablishedLeg, step HybridStep, userText string) (picked string, err error) {
	if len(step.Transitions) == 0 {
		return "", nil
	}
	if !listenRouteLLMEnabled() {
		return "", ErrListenRouteNoLLM
	}
	allowed := collectAllowedNextIDs(step)
	if len(allowed) <= 1 {
		return "", nil
	}
	llmChoice, err := pickNextWithLLM(ctx, leg, step, userText, allowed)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrListenRouteLLMFail, err)
	}
	llmChoice = strings.TrimSpace(llmChoice)
	if llmChoice == "" || !allowed[llmChoice] {
		return "", fmt.Errorf("%w: invalid next_id %q", ErrListenRouteLLMFail, llmChoice)
	}
	return llmChoice, nil
}

func listenRouteLLMEnabled() bool {
	if isTruthyEnv(utils.GetEnv(constants.EnvCHECKLLMRouteDisabled)) {
		return false
	}
	prov := strings.ToLower(strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMProvider)))
	if prov != "" && prov != "openai" {
		return false
	}
	if strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMAPIKey)) == "" {
		return false
	}
	if strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMBaseURL)) == "" {
		return false
	}
	if strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMModel)) == "" {
		return false
	}
	return true
}

func isTruthyEnv(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func collectAllowedNextIDs(step HybridStep) map[string]bool {
	out := make(map[string]bool)
	for _, tr := range step.Transitions {
		if id := strings.TrimSpace(tr.NextID); id != "" {
			out[id] = true
		}
	}
	if d := strings.TrimSpace(step.NextID); d != "" {
		out[d] = true
	}
	return out
}

func sortedAllowedIDs(allowed map[string]bool) []string {
	ids := make([]string, 0, len(allowed))
	for id := range allowed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func routeLegacyJSON() bool {
	return isTruthyEnv(utils.GetEnv(constants.EnvCHECKLLMRouteLegacyJSON))
}

func routeMaxCompletionTokens() int {
	def := 32
	if s := strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMRouteMaxTokens)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 8 && n <= 128 {
			return n
		}
	}
	return def
}

func pickNextWithLLM(ctx context.Context, leg EstablishedLeg, step HybridStep, userText string, allowed map[string]bool) (string, error) {
	apiKey := strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMAPIKey))
	baseURL := strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMBaseURL))
	model := strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMModel))
	ms := 12000
	if s := strings.TrimSpace(utils.GetEnv(constants.EnvCHECKLLMRouteTimeoutMS)); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 2000 && n <= 120000 {
			ms = n
		}
	}
	routeCtx, cancel := context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
	defer cancel()

	sorted := sortedAllowedIDs(allowed)
	legacy := routeLegacyJSON()
	sys := llmRouteSystemPromptCompact
	if legacy {
		sys = llmRouteSystemPromptLegacy
	}
	prov, err := llm.NewLLMProvider(routeCtx, llm.ProviderOpenAI, apiKey, baseURL, sys)
	if err != nil {
		return "", err
	}

	var userPayload string
	if legacy {
		userPayload = buildListenRouteUserPrompt(step, userText, allowed)
	} else {
		userPayload = buildCompactListenRouteUserPrompt(step, userText, sorted)
	}
	opts := llm.QueryOptions{
		Model:            model,
		Temperature:      0,
		MaxTokens:        routeMaxCompletionTokens(),
		EnableJSONOutput: true,
	}
	resp, err := prov.QueryWithOptions(userPayload, &opts)
	if err != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("script listen LLM route failed",
				zap.String("call_id", leg.CallID),
				zap.String("step_id", step.ID),
				zap.Error(err))
		}
		return "", err
	}
	reply := ""
	if resp != nil && len(resp.Choices) > 0 {
		reply = resp.Choices[0].Content
	}
	nextID, perr := parseRouteLLMReply(reply, sorted, legacy)
	if perr != nil {
		if logger.Lg != nil {
			logger.Lg.Warn("script listen LLM route parse failed",
				zap.String("call_id", leg.CallID),
				zap.String("step_id", step.ID),
				zap.String("raw", truncateForLog(reply, 400)),
				zap.Error(perr))
		}
		return "", perr
	}
	return strings.TrimSpace(nextID), nil
}

func buildCompactListenRouteUserPrompt(step HybridStep, userText string, sorted []string) string {
	var b strings.Builder
	b.WriteString("Branches (index:next_id):\n")
	def := strings.TrimSpace(step.NextID)
	defIdx := -1
	for i, id := range sorted {
		fmt.Fprintf(&b, "%d:%s\n", i, id)
		if def != "" && id == def {
			defIdx = i
		}
	}
	if defIdx >= 0 {
		fmt.Fprintf(&b, "\nIf vague or off-topic, use index %d (default).\n", defIdx)
	}
	b.WriteString("\nSemantics:\n")
	byTarget := map[string][]string{}
	for _, tr := range step.Transitions {
		tid := strings.TrimSpace(tr.NextID)
		if tid == "" {
			continue
		}
		if d := strings.TrimSpace(tr.Description); d != "" {
			byTarget[tid] = append(byTarget[tid], d)
		}
	}
	for _, id := range sorted {
		if hints, ok := byTarget[id]; ok && len(hints) > 0 {
			fmt.Fprintf(&b, "- %s: %s\n", id, strings.Join(hints, " | "))
		} else {
			fmt.Fprintf(&b, "- %s: infer from user intent\n", id)
		}
	}
	ut := strings.TrimSpace(userText)
	if ut == "" {
		ut = "(empty utterance)"
	}
	fmt.Fprintf(&b, "\nUser utterance:\n\"\"\"\n%s\n\"\"\"\n", ut)
	b.WriteString("\nReturn only: {\"i\":N}\n")
	return b.String()
}

func buildListenRouteUserPrompt(step HybridStep, userText string, allowed map[string]bool) string {
	var b strings.Builder
	b.WriteString("ALLOWED next_id values (you MUST output exactly one of these strings):\n")
	ids := make([]string, 0, len(allowed))
	for id := range allowed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	def := strings.TrimSpace(step.NextID)
	if def != "" {
		fmt.Fprintf(&b, "\nDEFAULT next_id when user is vague, off-topic, or non-committal: %s\n", def)
	}
	b.WriteString("\nBranch semantics (use these to decide which next_id fits the utterance):\n")
	byTarget := map[string][]string{}
	for _, tr := range step.Transitions {
		tid := strings.TrimSpace(tr.NextID)
		if tid == "" {
			continue
		}
		if d := strings.TrimSpace(tr.Description); d != "" {
			byTarget[tid] = append(byTarget[tid], d)
		}
	}
	for _, id := range ids {
		if hints, ok := byTarget[id]; ok && len(hints) > 0 {
			fmt.Fprintf(&b, "- %s: %s\n", id, strings.Join(hints, " | "))
		} else {
			fmt.Fprintf(&b, "- %s: (infer from context; pick only if clearly matches user intent)\n", id)
		}
	}
	ut := strings.TrimSpace(userText)
	if ut == "" {
		ut = "(empty utterance)"
	}
	fmt.Fprintf(&b, "\nUser utterance:\n\"\"\"\n%s\n\"\"\"\n", ut)
	b.WriteString("\nReturn JSON only: {\"next_id\":\"...\"}\n")
	return b.String()
}

func parseRouteLLMReply(raw string, sorted []string, legacy bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty LLM reply")
	}
	raw = stripMarkdownCodeFence(raw)
	obj := extractJSONObject(raw)
	if obj == "" {
		return "", fmt.Errorf("no json object in reply")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(obj), &m); err != nil {
		return "", err
	}
	if legacy {
		v, ok := m["next_id"]
		if !ok {
			return "", fmt.Errorf("missing next_id")
		}
		s, _ := v.(string)
		if strings.TrimSpace(s) == "" {
			return "", fmt.Errorf("empty next_id")
		}
		return strings.TrimSpace(s), nil
	}
	if v, ok := m["i"]; ok {
		var idx int
		switch n := v.(type) {
		case float64:
			idx = int(n)
		default:
			return "", fmt.Errorf("invalid i type %T", v)
		}
		if idx >= 0 && idx < len(sorted) {
			return sorted[idx], nil
		}
		return "", fmt.Errorf("index %d out of range [0,%d)", idx, len(sorted))
	}
	if v, ok := m["next_id"]; ok {
		s, _ := v.(string)
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s), nil
		}
	}
	return "", fmt.Errorf("missing i or next_id")
}

// parseRouteLLMJSON is kept for tests; prefer parseRouteLLMReply in production paths.
func parseRouteLLMJSON(raw string) (string, error) {
	return parseRouteLLMReply(raw, nil, true)
}

func stripMarkdownCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToLower(s), "json") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			s = strings.TrimSpace(s[nl+1:])
		}
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func extractJSONObject(s string) string {
	i := strings.Index(s, "{")
	j := strings.LastIndex(s, "}")
	if i < 0 || j <= i {
		return ""
	}
	return s[i : j+1]
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

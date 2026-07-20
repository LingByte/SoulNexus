package speaker

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
)

func TestFormatPromptCard_noSecrets(t *testing.T) {
	card := callbinding.FormatPromptCard(mustCtx())
	if !strings.Contains(card, "张老师") {
		t.Fatalf("missing name: %s", card)
	}
	if !strings.Contains(card, "role：teacher") {
		t.Fatalf("missing llm attr: %s", card)
	}
	if strings.Contains(card, "secret-token") {
		t.Fatalf("secret leaked into prompt: %s", card)
	}
	if !strings.Contains(card, "cloudsteps") {
		t.Fatalf("expected credential provider name only: %s", card)
	}
}

func TestApplySpeakerHint_idempotent(t *testing.T) {
	base := "你是客服"
	card := callbinding.FormatPromptCard(mustCtx())
	once := callbinding.ApplySpeakerHint(base, card)
	twice := callbinding.ApplySpeakerHint(once, card)
	if strings.Count(twice, "【本通说话人】") != 1 {
		t.Fatalf("expected one speaker block, got: %s", twice)
	}
}

func TestEnrichMCPArgs_cloudsteps(t *testing.T) {
	callID := "test-speaker-cred-1"
	callbinding.SetSpeakerContext(callID, mustCtx())
	defer callbinding.ClearSpeakerContext(callID)

	args := callbinding.EnrichMCPArgs(callID, "cloudsteps_book_lesson", map[string]interface{}{})
	tok, _ := args["token"].(string)
	if tok != "secret-token" {
		t.Fatalf("expected injected token, got %#v", args)
	}
	args2 := callbinding.EnrichMCPArgs(callID, "cloudsteps_book_lesson", map[string]interface{}{"token": "keep-me"})
	if args2["token"] != "keep-me" {
		t.Fatalf("should not override explicit token")
	}
}

func mustCtx() callbinding.SpeakerContext {
	return callbinding.SpeakerContext{
		SubjectID:   1,
		DisplayName: "张老师",
		FeatureID:   "feat-1",
		Verified:    true,
		Score:       0.91,
		Attributes: []callbinding.SpeakerAttribute{
			{Key: "role", Value: "teacher", Visibility: callbinding.SpeakerVisibilityLLM},
			{Key: "internal_note", Value: "hide", Visibility: callbinding.SpeakerVisibilityInternal},
		},
		Credentials: []callbinding.SpeakerCredentialRef{
			{Provider: "cloudsteps", SecretRef: "secret-token", HasSecret: true},
		},
	}
}

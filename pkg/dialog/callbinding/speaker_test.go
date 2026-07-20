package callbinding_test

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
)

func TestSpeakerPromptAndCreds(t *testing.T) {
	ctx := callbinding.SpeakerContext{
		SubjectID:   1,
		DisplayName: "张老师",
		FeatureID:   "feat-1",
		Verified:    true,
		Score:       0.91,
		Attributes: []callbinding.SpeakerAttribute{
			{Key: "role", Value: "teacher", Visibility: callbinding.SpeakerVisibilityLLM},
			{Key: "secretish", Value: "nope", Visibility: callbinding.SpeakerVisibilityInternal},
		},
		Credentials: []callbinding.SpeakerCredentialRef{
			{Provider: "cloudsteps", SecretRef: "secret-token", HasSecret: true},
		},
	}
	card := callbinding.FormatPromptCard(ctx)
	if !strings.Contains(card, "张老师") || !strings.Contains(card, "role：teacher") {
		t.Fatalf("card incomplete: %s", card)
	}
	if strings.Contains(card, "secret-token") || strings.Contains(card, "nope") {
		t.Fatalf("leaked non-llm data: %s", card)
	}
	callID := "cb-speaker-test"
	callbinding.SetSpeakerContext(callID, ctx)
	defer callbinding.ClearSpeakerContext(callID)
	args := callbinding.EnrichMCPArgs(callID, "cloudsteps_book_lesson", nil)
	if args["token"] != "secret-token" {
		t.Fatalf("token not injected: %#v", args)
	}
}

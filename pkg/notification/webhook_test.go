package notification

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
)

func TestValidWebhookEvent(t *testing.T) {
	t.Parallel()
	for _, e := range constants.AllWebhookEvents {
		if !ValidWebhookEvent(e) {
			t.Errorf("ValidWebhookEvent(%q) = false, want true", e)
		}
	}
	if ValidWebhookEvent("not.a.event") {
		t.Error("ValidWebhookEvent(invalid) = true, want false")
	}
}

func TestDispatchWebhook_nilDB(t *testing.T) {
	t.Parallel()
	// Should not panic when db is nil.
	DispatchWebhook(nil, nil, 1, constants.WebhookEventCallStarted, "call-1", "a", "b", "inbound", nil)
}

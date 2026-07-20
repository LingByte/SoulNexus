package response

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
)

func TestNewPlatformAdminResponse(t *testing.T) {
	got := NewPlatformAdminResponse(models.PlatformAdmin{Email: "a@b.c", DisplayName: "A", Status: "active"})
	if got.Email != "a@b.c" || got.DisplayName != "A" {
		t.Fatalf("%+v", got)
	}
}

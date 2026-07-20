package response

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
)

func TestNewTenantResponse(t *testing.T) {
	got := NewTenantResponse(models.Tenant{Name: "T", Slug: "t", Status: "active"})
	if got.Name != "T" || got.Slug != "t" {
		t.Fatalf("%+v", got)
	}
}

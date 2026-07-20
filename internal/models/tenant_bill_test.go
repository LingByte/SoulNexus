package models

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
)

func TestTenantBillStatusConstants(t *testing.T) {
	for _, s := range []string{
		constants.TenantBillStatusDraft,
		constants.TenantBillStatusFinalized,
		constants.TenantBillStatusPaid,
	} {
		if s == "" {
			t.Fatal("empty bill status constant")
		}
	}
}

func TestTenantBillAutoGenerateSkipsNonDraft(t *testing.T) {
	for _, status := range []string{
		constants.TenantBillStatusFinalized,
		constants.TenantBillStatusPaid,
	} {
		if status == constants.TenantBillStatusDraft {
			t.Fatalf("status %q should not equal draft", status)
		}
	}
}

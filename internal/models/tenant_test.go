package models

import "testing"

func TestBaseSlugFromCompanyName(t *testing.T) {
	if got := BaseSlugFromCompanyName("  Acme Corp! "); got != "acme-corp" {
		t.Fatalf("%q", got)
	}
	if got := BaseSlugFromCompanyName(""); got != "tenant" {
		t.Fatalf("empty %q", got)
	}
}

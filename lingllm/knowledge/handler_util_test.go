package knowledge

import "testing"

func TestSanitizeNamespace(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"my-kb", "my_kb"},
		{"  Tenant/ABC  ", "tenant_abc"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeNamespace(tt.in)
		if got != tt.want {
			t.Fatalf("sanitizeNamespace(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

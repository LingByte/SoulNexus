package common

import "testing"

func TestValidateTenantConfiguredURL_AllowsLoopback(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1:3920/sse",
		"http://localhost:3920/sse",
		"http://192.168.1.10/mcp/sse",
	} {
		if err := ValidateTenantConfiguredURL(u); err != nil {
			t.Fatalf("%q: %v", u, err)
		}
	}
}

func TestValidateTenantConfiguredURL_BlocksMetadata(t *testing.T) {
	if err := ValidateTenantConfiguredURL("http://metadata.google.internal/computeMetadata/v1/"); err == nil {
		t.Fatal("expected metadata host to be blocked")
	}
}

func TestValidateTenantConfiguredURL_BlocksBadScheme(t *testing.T) {
	if err := ValidateTenantConfiguredURL("file:///etc/passwd"); err == nil {
		t.Fatal("expected file scheme to be blocked")
	}
}

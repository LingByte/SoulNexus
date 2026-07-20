package providers

import (
	"fmt"
	"testing"
)

func TestLoopbackAlternateSSEURL(t *testing.T) {
	alt, ok := loopbackAlternateSSEURL("http://127.0.0.1:3920/sse")
	if !ok || alt != "http://localhost:3920/sse" {
		t.Fatalf("got %q ok=%v", alt, ok)
	}
	back, ok := loopbackAlternateSSEURL(alt)
	if !ok || back != "http://127.0.0.1:3920/sse" {
		t.Fatalf("got %q ok=%v", back, ok)
	}
}

func TestIsMCPSSEEndpointWaitError(t *testing.T) {
	if !isMCPSSEEndpointWaitError(fmt.Errorf("mcp sse start: context cancelled while waiting for endpoint")) {
		t.Fatal("expected match")
	}
}

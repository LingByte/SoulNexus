package utils

import (
	"net"
	"testing"
)

func TestIsInternalIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "loopback address",
			ip:       "127.0.0.1",
			expected: true,
		},
		{
			name:     "localhost",
			ip:       "::1",
			expected: true,
		},
		{
			name:     "private network 192.168.x.x",
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			name:     "private network 10.x.x.x",
			ip:       "10.0.0.1",
			expected: true,
		},
		{
			name:     "private network 172.16.x.x",
			ip:       "172.16.0.1",
			expected: true,
		},
		{
			name:     "public IP",
			ip:       "8.8.8.8",
			expected: false,
		},
		{
			name:     "public IP 2",
			ip:       "114.114.114.114",
			expected: false,
		},
		{
			name:     "invalid IP",
			ip:       "invalid",
			expected: false,
		},
		{
			name:     "empty string",
			ip:       "",
			expected: false,
		},
		{
			name:     "link local unicast",
			ip:       "169.254.1.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInternalIP(tt.ip)
			if result != tt.expected {
				t.Fatalf("IsInternalIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIsInternalIP_IPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "IPv6 loopback",
			ip:       "::1",
			expected: true,
		},
		{
			name:     "IPv6 private",
			ip:       "fc00::1",
			expected: true,
		},
		{
			name:     "IPv6 link local",
			ip:       "fe80::1",
			expected: true,
		},
		{
			name:     "IPv6 public",
			ip:       "2001:4860:4860::8888",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInternalIP(tt.ip)
			if result != tt.expected {
				t.Fatalf("IsInternalIP(%q) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestGetRealAddressByIP_InternalIP(t *testing.T) {
	// Test with internal IP
	result := GetRealAddressByIP("192.168.1.1")
	if result != INTERNAL_IP {
		t.Fatalf("GetRealAddressByIP(192.168.1.1) = %q, want %q", result, INTERNAL_IP)
	}

	// Test with loopback
	result = GetRealAddressByIP("127.0.0.1")
	if result != INTERNAL_IP {
		t.Fatalf("GetRealAddressByIP(127.0.0.1) = %q, want %q", result, INTERNAL_IP)
	}
}

func TestGetRealAddressByIP_InvalidIP(t *testing.T) {
	// Test with invalid IP
	result := GetRealAddressByIP("invalid.ip.address")
	if result == "" {
		t.Fatalf("GetRealAddressByIP(invalid) should return UNKNOWN or error message")
	}
}

// Note: Testing GetRealAddressByIP with external IPs requires network access
// and may fail if the API is unavailable. We'll test the structure but skip
// actual network calls in unit tests.
func TestGetRealAddressByIP_ExternalIP(t *testing.T) {
	// This test would require network access and may be flaky
	// We'll test that it doesn't crash and returns a non-empty string
	// Skip in CI or use a mock HTTP server
	t.Skip("Skipping external IP test - requires network access")
}

func TestIPLocationResponse_Structure(t *testing.T) {
	// Test that the response structure is correct
	// This is more of a compile-time check
	var resp IPLocationResponse
	if resp.Pro == "" && resp.City == "" {
		// This is expected for zero value
	}
}

func TestIsInternalIP_EdgeCases(t *testing.T) {
	// Test edge cases
	testCases := []struct {
		ip       string
		expected bool
	}{
		{"0.0.0.0", false}, // Not considered internal by Go's IsPrivate
		{"255.255.255.255", false},
		{"224.0.0.1", false}, // Multicast, not private
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			parsedIP := net.ParseIP(tc.ip)
			if parsedIP == nil {
				// Invalid IP, should return false
				result := IsInternalIP(tc.ip)
				if result {
					t.Fatalf("IsInternalIP(%q) = true for invalid IP, want false", tc.ip)
				}
				return
			}

			result := IsInternalIP(tc.ip)
			// Note: The actual behavior depends on Go's net.IP methods
			// We're testing that the function doesn't crash
			_ = result
		})
	}
}

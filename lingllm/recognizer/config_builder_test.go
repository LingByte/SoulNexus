package recognizer

import (
	"testing"
)

func TestConfigReaderString(t *testing.T) {
	config := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	reader := NewConfigReader(config)

	// Test single key
	result := reader.String("key1")
	if result != "value1" {
		t.Errorf("String(key1) = %s, want value1", result)
	}

	// Test with default
	result = reader.String("nonexistent", "default")
	if result != "default" {
		t.Errorf("String(nonexistent, default) = %s, want default", result)
	}
}

func TestConfigReaderInt(t *testing.T) {
	config := map[string]interface{}{
		"port":  8080,
		"count": int64(42),
		"ratio": 3.14,
	}

	reader := NewConfigReader(config)

	// Test int
	result := reader.Int("port")
	if result != 8080 {
		t.Errorf("Int(port) = %d, want 8080", result)
	}

	// Test int64
	result = reader.Int("count")
	if result != 42 {
		t.Errorf("Int(count) = %d, want 42", result)
	}

	// Test with default
	result = reader.Int("nonexistent", 100)
	if result != 100 {
		t.Errorf("Int(nonexistent, 100) = %d, want 100", result)
	}
}

func TestGetVendor(t *testing.T) {
	vendor := GetVendor("tencent")
	if vendor != VendorQCloud {
		t.Errorf("GetVendor(tencent) = %v, want VendorQCloud", vendor)
	}

	vendor = GetVendor("google")
	if vendor != Vendor("google") {
		t.Errorf("GetVendor(google) = %v, want google", vendor)
	}
}

func TestNewTranscriberConfigFromMap(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		config   map[string]interface{}
		wantErr  bool
	}{
		{
			name:     "unsupported provider",
			provider: "unsupported",
			config:   map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "google provider",
			provider: "google",
			config: map[string]interface{}{
				"encoding": "LINEAR16",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTranscriberConfigFromMap(tt.provider, tt.config, "en-US")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTranscriberConfigFromMap error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

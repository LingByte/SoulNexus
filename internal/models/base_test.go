package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProviderConfig_Value(t *testing.T) {
	tests := []struct {
		name    string
		config  ProviderConfig
		wantNil bool
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantNil: true,
			wantErr: false,
		},
		{
			name:    "empty config",
			config:  ProviderConfig{},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "valid config",
			config: ProviderConfig{
				"provider": "test",
				"apiKey":   "key123",
			},
			wantNil: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := tt.config.Value()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.wantNil {
				assert.Nil(t, value)
			} else {
				assert.NotNil(t, value)
				// Verify it's valid JSON
				bytes, ok := value.([]byte)
				assert.True(t, ok)
				var result ProviderConfig
				err := json.Unmarshal(bytes, &result)
				assert.NoError(t, err)
				assert.Equal(t, tt.config["provider"], result["provider"])
			}
		})
	}
}

func TestProviderConfig_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
		wantNil bool
	}{
		{
			name:    "nil value",
			value:   nil,
			wantErr: false,
			wantNil: false, // Scan returns empty map, not nil
		},
		{
			name:    "empty bytes",
			value:   []byte{},
			wantErr: false,
			wantNil: false, // Scan returns empty map, not nil
		},
		{
			name:    "valid JSON bytes",
			value:   []byte(`{"provider":"test","apiKey":"key123"}`),
			wantErr: false,
			wantNil: false,
		},
		{
			name:    "invalid type",
			value:   "not bytes",
			wantErr: true,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config ProviderConfig
			err := config.Scan(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.wantNil {
				assert.Nil(t, config)
			} else {
				assert.NotNil(t, config)
			}
		})
	}
}

func TestGroupPermission_Value(t *testing.T) {
	gp := GroupPermission{
		Permissions: []string{"read", "write"},
	}

	value, err := gp.Value()
	assert.NoError(t, err)
	assert.NotNil(t, value)

	// Verify it's valid JSON
	bytes, ok := value.([]byte)
	assert.True(t, ok)
	var result GroupPermission
	err = json.Unmarshal(bytes, &result)
	assert.NoError(t, err)
	assert.Equal(t, gp.Permissions, result.Permissions)
}

func TestGroupPermission_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid JSON bytes",
			value:   []byte(`{"Permissions":["read","write"]}`),
			wantErr: false,
		},
		{
			name:    "invalid type",
			value:   "not bytes",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gp GroupPermission
			err := gp.Scan(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.wantErr {
					assert.Equal(t, []string{"read", "write"}, gp.Permissions)
				}
			}
		})
	}
}

func TestUserCredential_GetASRProvider(t *testing.T) {
	tests := []struct {
		name   string
		config ProviderConfig
		want   string
	}{
		{
			name:   "nil config",
			config: nil,
			want:   "",
		},
		{
			name:   "empty config",
			config: ProviderConfig{},
			want:   "",
		},
		{
			name: "valid provider",
			config: ProviderConfig{
				"provider": "qiniu",
			},
			want: "qiniu",
		},
		{
			name: "provider not string",
			config: ProviderConfig{
				"provider": 123,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &UserCredential{
				AsrConfig: tt.config,
			}
			assert.Equal(t, tt.want, uc.GetASRProvider())
		})
	}
}

func TestUserCredential_GetASRConfig(t *testing.T) {
	uc := &UserCredential{
		AsrConfig: ProviderConfig{
			"provider": "qiniu",
			"apiKey":   "key123",
		},
	}

	assert.Equal(t, "qiniu", uc.GetASRConfig("provider"))
	assert.Equal(t, "key123", uc.GetASRConfig("apiKey"))
	assert.Nil(t, uc.GetASRConfig("nonexistent"))

	uc.AsrConfig = nil
	assert.Nil(t, uc.GetASRConfig("provider"))
}

func TestUserCredential_GetASRConfigString(t *testing.T) {
	uc := &UserCredential{
		AsrConfig: ProviderConfig{
			"provider": "qiniu",
			"apiKey":   "key123",
		},
	}

	assert.Equal(t, "qiniu", uc.GetASRConfigString("provider"))
	assert.Equal(t, "key123", uc.GetASRConfigString("apiKey"))
	assert.Equal(t, "", uc.GetASRConfigString("nonexistent"))

	uc.AsrConfig = ProviderConfig{
		"number": 123,
	}
	assert.Equal(t, "", uc.GetASRConfigString("number")) // Not a string
}

func TestUserCredential_GetTTSProvider(t *testing.T) {
	uc := &UserCredential{
		TtsConfig: ProviderConfig{
			"provider": "qiniu",
		},
	}

	assert.Equal(t, "qiniu", uc.GetTTSProvider())

	uc.TtsConfig = nil
	assert.Equal(t, "", uc.GetTTSProvider())
}

func TestUserCredential_GetTTSConfig(t *testing.T) {
	uc := &UserCredential{
		TtsConfig: ProviderConfig{
			"provider": "qiniu",
			"apiKey":   "key123",
		},
	}

	assert.Equal(t, "qiniu", uc.GetTTSConfig("provider"))
	assert.Equal(t, "key123", uc.GetTTSConfig("apiKey"))
	assert.Nil(t, uc.GetTTSConfig("nonexistent"))
}

func TestUserCredential_GetTTSConfigString(t *testing.T) {
	uc := &UserCredential{
		TtsConfig: ProviderConfig{
			"provider": "qiniu",
		},
	}

	assert.Equal(t, "qiniu", uc.GetTTSConfigString("provider"))
	assert.Equal(t, "", uc.GetTTSConfigString("nonexistent"))
}

// Note: CloneConfig related methods are not implemented in UserCredential
// These tests are commented out until the feature is implemented
// func TestUserCredential_GetCloneProvider(t *testing.T) {
// 	uc := &UserCredential{
// 		CloneConfig: ProviderConfig{
// 			"provider": "xunfei",
// 		},
// 	}
//
// 	assert.Equal(t, "xunfei", uc.GetCloneProvider())
//
// 	uc.CloneConfig = nil
// 	assert.Equal(t, "", uc.GetCloneProvider())
// }
//
// func TestUserCredential_GetCloneConfig(t *testing.T) {
// 	uc := &UserCredential{
// 		CloneConfig: ProviderConfig{
// 			"provider": "xunfei",
// 			"apiKey":   "key123",
// 		},
// 	}
//
// 	assert.Equal(t, "xunfei", uc.GetCloneConfig("provider"))
// 	assert.Equal(t, "key123", uc.GetCloneConfig("apiKey"))
// 	assert.Nil(t, uc.GetCloneConfig("nonexistent"))
// }
//
// func TestUserCredential_GetCloneConfigString(t *testing.T) {
// 	uc := &UserCredential{
// 		CloneConfig: ProviderConfig{
// 			"provider": "xunfei",
// 		},
// 	}
//
// 	assert.Equal(t, "xunfei", uc.GetCloneConfigString("provider"))
// 	assert.Equal(t, "", uc.GetCloneConfigString("nonexistent"))
// }

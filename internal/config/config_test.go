package config

import (
	"os"
	"testing"
	"time"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
)

// ===== Load =====

func TestLoad_DefaultValues(t *testing.T) {
	err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if GlobalConfig == nil {
		t.Fatal("GlobalConfig should not be nil")
	}

	// Just verify Load doesn't panic and sets GlobalConfig
	if GlobalConfig.Server.Addr == "" {
		t.Error("Server.Addr should not be empty")
	}
	if GlobalConfig.Database.Driver == "" {
		t.Error("Database.Driver should not be empty")
	}
}

// ===== GetStringOrDefault =====

func TestGetStringOrDefault(t *testing.T) {
	os.Unsetenv("TEST_KEY")
	got := common.GetStringOrDefault("TEST_KEY", "default")
	if got != "default" {
		t.Errorf("got %q, want 'default'", got)
	}

	os.Setenv("TEST_KEY", "value")
	defer os.Unsetenv("TEST_KEY")
	got = common.GetStringOrDefault("TEST_KEY", "default")
	if got != "value" {
		t.Errorf("got %q, want 'value'", got)
	}
}

// ===== GetIntOrDefault =====

func TestGetIntOrDefault(t *testing.T) {
	os.Unsetenv("TEST_INT")
	got := common.GetIntOrDefault("TEST_INT", 42)
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}

	os.Setenv("TEST_INT", "100")
	defer os.Unsetenv("TEST_INT")
	got = common.GetIntOrDefault("TEST_INT", 42)
	if got != 100 {
		t.Errorf("got %d, want 100", got)
	}

	os.Setenv("TEST_INT", "invalid")
	got = common.GetIntOrDefault("TEST_INT", 42)
	if got != 42 {
		t.Errorf("got %d, want 42 for invalid", got)
	}
}

// ===== GetBoolOrDefault =====

func TestGetBoolOrDefault(t *testing.T) {
	os.Unsetenv("TEST_BOOL")
	got := common.GetBoolOrDefault("TEST_BOOL", true)
	if !got {
		t.Error("got false, want true")
	}

	os.Setenv("TEST_BOOL", "true")
	defer os.Unsetenv("TEST_BOOL")
	got = common.GetBoolOrDefault("TEST_BOOL", false)
	if !got {
		t.Error("got false, want true")
	}

	os.Setenv("TEST_BOOL", "false")
	got = common.GetBoolOrDefault("TEST_BOOL", true)
	if got {
		t.Error("got true, want false")
	}
}

// ===== ParseDuration =====

func TestParseDuration(t *testing.T) {
	got := common.ParseDuration("30s", time.Minute)
	if got != 30*time.Second {
		t.Errorf("got %v, want 30s", got)
	}

	got = common.ParseDuration("invalid", time.Minute)
	if got != time.Minute {
		t.Errorf("got %v, want 1m for invalid", got)
	}
}

// ===== loadMiddlewareConfig =====

func TestLoadMiddlewareConfig_Production(t *testing.T) {
	os.Setenv("MODE", pkgconst.ENV_PROD)
	defer os.Unsetenv("MODE")

	config := loadMiddlewareConfig()
	if config.RateLimit.GlobalRPS != 2000 {
		t.Errorf("GlobalRPS = %d, want 2000", config.RateLimit.GlobalRPS)
	}
}

func TestLoadMiddlewareConfig_Development(t *testing.T) {
	os.Setenv("MODE", pkgconst.ENV_DEV)
	defer os.Unsetenv("MODE")

	config := loadMiddlewareConfig()
	if config.RateLimit.GlobalRPS != 10000 {
		t.Errorf("GlobalRPS = %d, want 10000", config.RateLimit.GlobalRPS)
	}
}

// ===== Config Struct =====

func TestConfig_Struct(t *testing.T) {
	config := &Config{
		MachineID: 1,
		Server: ServerConfig{
			Name: "test",
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./test.db",
		},
	}

	if config.MachineID != 1 {
		t.Errorf("MachineID = %d", config.MachineID)
	}
	if config.Server.Name != "test" {
		t.Errorf("Server.Name = %q", config.Server.Name)
	}
}

// ===== MiddlewareConfig =====

func TestMiddlewareConfig_RateLimit(t *testing.T) {
	config := MiddlewareConfig{
		RateLimit: RateLimiterConfig{
			GlobalRPS:   1000,
			GlobalBurst: 2000,
			UserRPS:     100,
			UserBurst:   200,
			IPRPS:       50,
			IPBurst:     100,
		},
	}

	if config.RateLimit.GlobalRPS != 1000 {
		t.Errorf("GlobalRPS = %d", config.RateLimit.GlobalRPS)
	}
}

func TestMiddlewareConfig_Timeout(t *testing.T) {
	config := MiddlewareConfig{
		Timeout: TimeoutConfig{
			DefaultTimeout: 30 * time.Second,
		},
	}

	if config.Timeout.DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultTimeout = %v", config.Timeout.DefaultTimeout)
	}
}

func TestMiddlewareConfig_CircuitBreaker(t *testing.T) {
	config := MiddlewareConfig{
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold:      5,
			SuccessThreshold:      3,
			MaxConcurrentRequests: 100,
		},
	}

	if config.CircuitBreaker.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d", config.CircuitBreaker.FailureThreshold)
	}
}

package security

import (
	"os"
	"testing"

	"github.com/LingByte/SoulNexus/internal/config"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
)

func TestCheckSessionSecretProdRequiresStrongSecret(t *testing.T) {
	prev := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = prev })
	config.GlobalConfig = &config.Config{}
	config.GlobalConfig.Server.Mode = pkgconst.ENV_PROD

	t.Setenv("SESSION_SECRET", "")
	checks := checkSessionSecret()
	if len(checks) != 1 || checks[0].Level != LevelError {
		t.Fatalf("empty secret in prod: %+v", checks)
	}

	t.Setenv("SESSION_SECRET", "change-me-soulnexus-dev-secret-32b")
	checks = checkSessionSecret()
	if len(checks) != 1 || checks[0].Level != LevelError {
		t.Fatalf("weak secret in prod: %+v", checks)
	}

	t.Setenv("SESSION_SECRET", "unit-test-session-secret-value-32b!!")
	checks = checkSessionSecret()
	if len(checks) != 1 || checks[0].Level != LevelOK {
		t.Fatalf("strong secret in prod: %+v", checks)
	}
}

func TestCheckPlatformAdminPasswordProd(t *testing.T) {
	prev := config.GlobalConfig
	t.Cleanup(func() { config.GlobalConfig = prev })
	config.GlobalConfig = &config.Config{}
	config.GlobalConfig.Server.Mode = pkgconst.ENV_PROD

	_ = os.Unsetenv("PLATFORM_ADMIN_PASSWORD")
	if checks := checkPlatformAdminPassword(); len(checks) != 0 {
		t.Fatalf("empty password should skip: %+v", checks)
	}

	t.Setenv("PLATFORM_ADMIN_PASSWORD", "admin123")
	checks := checkPlatformAdminPassword()
	if len(checks) != 1 || checks[0].Level != LevelError {
		t.Fatalf("weak password: %+v", checks)
	}
}

package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/gin-gonic/gin"
)

// ===== productionLikeRuntime =====

func TestProductionLikeRuntime_TestMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if productionLikeRuntime() {
		t.Error("productionLikeRuntime should return false in test mode")
	}
}

func TestProductionLikeRuntime_DebugMode(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	// Debug mode + no APP_ENV set → should be not production
	if productionLikeRuntime() {
		t.Error("productionLikeRuntime should return false in debug mode with no APP_ENV")
	}
}

func TestProductionLikeRuntime_ReleaseMode(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	if !productionLikeRuntime() {
		t.Error("productionLikeRuntime should return true in release mode")
	}
}

func TestProductionLikeRuntime_ProdEnv(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	os.Setenv(pkgconst.EnvAppEnv, "prod")
	defer os.Unsetenv(pkgconst.EnvAppEnv)

	if !productionLikeRuntime() {
		t.Error("productionLikeRuntime should return true when APP_ENV=prod even in debug mode")
	}
}

func TestProductionLikeRuntime_ProdEnvMixedCase(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	os.Setenv(pkgconst.EnvAppEnv, "PROD")
	defer os.Unsetenv(pkgconst.EnvAppEnv)

	if !productionLikeRuntime() {
		t.Error("productionLikeRuntime should be case-insensitive for APP_ENV")
	}
}

func TestProductionLikeRuntime_DevEnv(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	os.Setenv(pkgconst.EnvAppEnv, "dev")
	defer os.Unsetenv(pkgconst.EnvAppEnv)

	// ReleaseMode takes precedence even with APP_ENV=dev
	if !productionLikeRuntime() {
		t.Error("productionLikeRuntime with ReleaseMode should return true regardless of APP_ENV")
	}
}

// ===== ValidateProductionSecurityEnv =====

func TestValidateProductionSecurityEnv_TestMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Should not panic in test mode
	ValidateProductionSecurityEnv()
}

func TestValidateProductionSecurityEnv_DebugMode(t *testing.T) {
	gin.SetMode(gin.DebugMode)
	// Should not panic in debug mode
	ValidateProductionSecurityEnv()
}

// ===== PrintBannerFromFile =====

func TestPrintBannerFromFile_Existing(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	os.WriteFile(filename, []byte("TEST BANNER"), 0644)

	err := PrintBannerFromFile(filename, "TEST")
	if err != nil {
		t.Fatalf("PrintBannerFromFile failed: %v", err)
	}
}

func TestPrintBannerFromFile_Generate(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")

	err := PrintBannerFromFile(filename, "HELLO")
	if err != nil {
		t.Fatalf("PrintBannerFromFile with generation failed: %v", err)
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("banner file should be generated")
	}
}

func TestPrintBannerFromFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	os.WriteFile(filename, []byte(""), 0644)

	err := PrintBannerFromFile(filename, "TEST")
	if err != nil {
		t.Fatalf("PrintBannerFromFile with empty file should not error: %v", err)
	}
}

func TestPrintBannerFromFile_WhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	os.WriteFile(filename, []byte("   \n  \n"), 0644)

	err := PrintBannerFromFile(filename, "TEST")
	if err != nil {
		t.Fatalf("PrintBannerFromFile with whitespace-only file should not error: %v", err)
	}
}

func TestPrintBannerFromFile_ValidASCIIArt(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	art := " _____ \n|  _  |\n| | | |\n|_| |_|\n"
	os.WriteFile(filename, []byte(art), 0644)

	err := PrintBannerFromFile(filename, "TEST")
	if err != nil {
		t.Fatalf("PrintBannerFromFile with ASCII art failed: %v", err)
	}
}

// ===== LogLingLLMVersion =====

func TestLogLingLLMVersion_NoPanic(t *testing.T) {
	// Just verify it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LogLingLLMVersion panicked: %v", r)
		}
	}()
	LogLingLLMVersion()
}

// ===== ANSI Constants =====

func TestANSIConstants(t *testing.T) {
	if pkgconst.ANSIReset == "" {
		t.Error("ANSIReset should not be empty")
	}
	gradients := []string{
		pkgconst.ANSIBannerGradient1,
		pkgconst.ANSIBannerGradient2,
		pkgconst.ANSIBannerGradient3,
		pkgconst.ANSIBannerGradient4,
		pkgconst.ANSIBannerGradient5,
		pkgconst.ANSIBannerGradient6,
	}
	for i, g := range gradients {
		if g == "" {
			t.Errorf("ANSIBannerGradient%d should not be empty", i+1)
		}
	}
}

// ===== Site Default Constants =====

func TestDefaultSiteConstants(t *testing.T) {
	if pkgconst.DefaultSiteName == "" {
		t.Error("DefaultSiteName should not be empty")
	}
	if pkgconst.DefaultSiteURL == "" {
		t.Error("DefaultSiteURL should not be empty")
	}
	if pkgconst.DefaultSiteDescription == "" {
		t.Error("DefaultSiteDescription should not be empty")
	}
}

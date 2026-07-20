package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ===== getDoomFontChars =====

func TestGetDoomFontChars_NotEmpty(t *testing.T) {
	chars := getDoomFontChars()
	if len(chars) == 0 {
		t.Fatal("getDoomFontChars should return non-empty map")
	}
}

func TestGetDoomFontChars_ExpectedCharacters(t *testing.T) {
	chars := getDoomFontChars()
	expectedChars := []rune{'L', 'I', 'N', 'G', 'A', 'B', 'C', 'D', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	for _, ch := range expectedChars {
		if _, ok := chars[ch]; !ok {
			t.Errorf("missing character %c in Doom font", ch)
		}
	}
}

func TestGetDoomFontChars_AllCharactersHaveContent(t *testing.T) {
	chars := getDoomFontChars()
	for ch, art := range chars {
		if ch == ' ' {
			continue // space character is intentionally whitespace-only
		}
		if strings.TrimSpace(art) == "" {
			t.Errorf("character %c has empty art", ch)
		}
	}
}

func TestGetDoomFontChars_HasSpace(t *testing.T) {
	chars := getDoomFontChars()
	if _, ok := chars[' ']; !ok {
		t.Error("Doom font should include space character mapping")
	}
}

// ===== generateBannerWithLocalDoom =====

func TestGenerateBannerWithLocalDoom(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("TEST")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty")
	}
}

func TestGenerateBannerWithLocalDoom_SingleChar(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("A")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty")
	}
}

func TestGenerateBannerWithLocalDoom_WithSpace(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("A B")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty")
	}
}

func TestGenerateBannerWithLocalDoom_UnknownChar(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("@#$")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty for unknown chars")
	}
}

func TestGenerateBannerWithLocalDoom_EmptyString(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner != "" {
		t.Fatalf("empty input should produce empty banner, got %q", banner)
	}
}

func TestGenerateBannerWithLocalDoom_AllKnownChars(t *testing.T) {
	chars := getDoomFontChars()
	allChars := ""
	for ch := range chars {
		allChars += string(ch)
	}
	banner, err := generateBannerWithLocalDoom(allChars)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty")
	}
}

func TestGenerateBannerWithLocalDoom_LongString(t *testing.T) {
	input := strings.Repeat("TEST", 20)
	banner, err := generateBannerWithLocalDoom(input)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty for long string")
	}
	lines := strings.Split(strings.TrimRight(banner, "\n"), "\n")
	if len(lines) < 6 {
		t.Errorf("expected at least 6 lines, got %d", len(lines))
	}
}

func TestGenerateBannerWithLocalDoom_LowerCase(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty")
	}
}

func TestGenerateBannerWithLocalDoom_MixedCase(t *testing.T) {
	banner, err := generateBannerWithLocalDoom("Test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if banner == "" {
		t.Fatal("banner should not be empty")
	}
}

// ===== generateBannerFromAPI (network-dependent, skip in -short) =====

func TestGenerateBannerFromAPI_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in short mode")
	}
	banner, err := generateBannerFromAPI("LINGECHO")
	if err != nil {
		t.Logf("API call failed (expected in CI/no-network): %v", err)
	}
	if banner != "" {
		t.Log("got banner from API")
	}
}

func TestGenerateBannerFromAPI_SpecialCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in short mode")
	}
	banner, err := generateBannerFromAPI("Test & Special!")
	if err != nil {
		t.Logf("API call failed (expected in CI): %v", err)
	}
	if banner != "" {
		t.Log("got banner from API with special chars")
	}
}

// ===== GenerateBannerWithDoomFont (network-dependent, skip in -short) =====

func TestGenerateBannerWithDoomFont_MultiChar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in short mode")
	}
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	err := GenerateBannerWithDoomFont("LING", filename)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	data, _ := os.ReadFile(filename)
	if len(data) == 0 {
		t.Fatal("file should not be empty")
	}
}

func TestGenerateBannerWithDoomFont_SingleChar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in short mode")
	}
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	err := GenerateBannerWithDoomFont("X", filename)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	data, _ := os.ReadFile(filename)
	if len(data) == 0 {
		t.Fatal("file should not be empty")
	}
}

// ===== EnsureBannerFile =====

func TestEnsureBannerFile_Exists(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")
	existingContent := "existing banner"
	os.WriteFile(filename, []byte(existingContent), 0644)

	err := EnsureBannerFile(filename, "test")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	data, _ := os.ReadFile(filename)
	if string(data) != existingContent {
		t.Fatalf("file was overwritten: %q", string(data))
	}
}

func TestEnsureBannerFile_Generate(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")

	err := EnsureBannerFile(filename, "TEST")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("banner file should exist")
	}
}

func TestEnsureBannerFile_DefaultText(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")

	err := EnsureBannerFile(filename, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("banner file should exist")
	}
}

func TestEnsureBannerFile_ASCIIOnly(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "banner.txt")

	err := EnsureBannerFile(filename, "ABC")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	data, _ := os.ReadFile(filename)
	if len(data) == 0 {
		t.Fatal("banner should not be empty")
	}
	for _, b := range data {
		if b < 32 || b > 126 {
			if b != '\n' && b != '\r' {
				t.Errorf("unexpected non-ASCII byte: %d", b)
			}
		}
	}
}

package validate

import "testing"

func TestIsEmail(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co", true},
		{"invalid", false},
		{"", false},
		{"@domain.com", false},
	}
	for _, tt := range tests {
		got := IsEmail(tt.input)
		if got != tt.want {
			t.Errorf("IsEmail(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsMobile(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"13812345678", true},
		{"15912345678", true},
		{"12345678901", false},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsMobile(tt.input)
		if got != tt.want {
			t.Errorf("IsMobile(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsDomain(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"example.com", true},
		{"sub.domain.co", true},
		{"invalid", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsDomain(tt.input)
		if got != tt.want {
			t.Errorf("IsDomain(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsSlug(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"my-slug", true},
		{"test_slug", true},
		{"abc123", true},
		{"ab", false},
		{"", false},
		{"UPPER", false},
	}
	for _, tt := range tests {
		got := IsSlug(tt.input)
		if got != tt.want {
			t.Errorf("IsSlug(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"", true},
		{"  ", true},
		{"hello", false},
	}
	for _, tt := range tests {
		got := IsEmpty(tt.input)
		if got != tt.want {
			t.Errorf("IsEmpty(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestTrim(t *testing.T) {
	got := Trim("  hello  ")
	if got != "hello" {
		t.Errorf("Trim = %q, want 'hello'", got)
	}
}

func TestTrimAll(t *testing.T) {
	got := TrimAll("  hello world  ")
	if got != "helloworld" {
		t.Errorf("TrimAll = %q, want 'helloworld'", got)
	}
}

func TestTrimLower(t *testing.T) {
	got := TrimLower("  HELLO  ")
	if got != "hello" {
		t.Errorf("TrimLower = %q, want 'hello'", got)
	}
}

func TestDefaultStr(t *testing.T) {
	got := DefaultStr("", "default")
	if got != "default" {
		t.Errorf("DefaultStr('') = %q, want 'default'", got)
	}
	got = DefaultStr("value", "default")
	if got != "value" {
		t.Errorf("DefaultStr('value') = %q, want 'value'", got)
	}
}

func TestNormalizePage(t *testing.T) {
	tests := []struct {
		page, size, maxSize int
		wantPage, wantSize  int
	}{
		{0, 0, 0, 1, 20},
		{1, 10, 100, 1, 10},
		{1, 200, 100, 1, 100},
		{-1, -1, 0, 1, 20},
	}
	for _, tt := range tests {
		gotPage, gotSize := NormalizePage(tt.page, tt.size, tt.maxSize)
		if gotPage != tt.wantPage || gotSize != tt.wantSize {
			t.Errorf("NormalizePage(%d,%d,%d) = (%d,%d), want (%d,%d)",
				tt.page, tt.size, tt.maxSize, gotPage, gotSize, tt.wantPage, tt.wantSize)
		}
	}
}

func TestParseID(t *testing.T) {
	tests := []struct {
		input   string
		want    uint
		wantErr bool
	}{
		{"1", 1, false},
		{"123", 123, false},
		{"0", 0, true},
		{"", 0, true},
		{"invalid", 0, true},
		{"-1", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseID(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.want {
			t.Errorf("ParseID(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestValidPassword(t *testing.T) {
	tests := []struct {
		pw     string
		minLen int
		want   bool
	}{
		{"password123", 8, true},
		{"short", 8, false},
		{"", 0, false},
		{"12345678", 8, true},
	}
	for _, tt := range tests {
		got := ValidPassword(tt.pw, tt.minLen)
		if got != tt.want {
			t.Errorf("ValidPassword(%q,%d) = %v, want %v", tt.pw, tt.minLen, got, tt.want)
		}
	}
}

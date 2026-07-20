package common

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils/coerce"
)

// ===== RandText / RandNumberText =====

func TestRandText(t *testing.T) {
	s := RandText(10)
	if len(s) != 10 {
		t.Fatalf("want length 10, got %d", len(s))
	}
}

func TestRandNumberText(t *testing.T) {
	s := RandNumberText(8)
	if len(s) != 8 {
		t.Fatalf("want length 8, got %d", len(s))
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("want only digits, got %q", s)
		}
	}
}

// ===== SafeCall =====

func TestSafeCall_NoPanic(t *testing.T) {
	err := SafeCall(func() error {
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
}

func TestSafeCall_WithPanic(t *testing.T) {
	called := false
	err := SafeCall(func() error {
		panic("test panic")
	}, func(e error) {
		called = true
		if e.Error() != "test panic" {
			t.Fatalf("want 'test panic', got %q", e.Error())
		}
	})
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	if !called {
		t.Fatal("fail handler should be called")
	}
}

func TestSafeCall_WithPanicError(t *testing.T) {
	called := false
	SafeCall(func() error {
		panic(errors.New("panic error"))
	}, func(e error) {
		called = true
		if e.Error() != "panic error" {
			t.Fatalf("want 'panic error', got %q", e.Error())
		}
	})
	if !called {
		t.Fatal("fail handler should be called")
	}
}

func TestSafeCall_WithPanicInt(t *testing.T) {
	called := false
	SafeCall(func() error {
		panic(123)
	}, func(e error) {
		called = true
	})
	if !called {
		t.Fatal("fail handler should be called")
	}
}

func TestSafeCall_WithPanicString(t *testing.T) {
	called := false
	SafeCall(func() error {
		panic("string panic")
	}, func(e error) {
		called = true
		if e.Error() != "string panic" {
			t.Fatalf("want 'string panic', got %q", e.Error())
		}
	})
	if !called {
		t.Fatal("fail handler should be called")
	}
}

// ===== StructAsMap =====

func TestStructAsMap(t *testing.T) {
	type TestStruct struct {
		Name  string
		Value int
		Empty string
	}
	s := TestStruct{Name: "test", Value: 42}
	vals := StructAsMap(s, []string{"Name", "Value", "Empty"})
	if vals["Name"] != "test" {
		t.Fatalf("Name=%v", vals["Name"])
	}
	if vals["Value"] != 42 {
		t.Fatalf("Value=%v", vals["Value"])
	}
	if _, exists := vals["Empty"]; exists {
		t.Fatal("Empty should not be in map")
	}
}

func TestStructAsMap_Pointer(t *testing.T) {
	type TestStruct struct {
		Name string
	}
	s := &TestStruct{Name: "ptr"}
	vals := StructAsMap(s, []string{"Name"})
	if vals["Name"] != "ptr" {
		t.Fatalf("Name=%v", vals["Name"])
	}
}

func TestStructAsMap_NonStruct(t *testing.T) {
	vals := StructAsMap("not a struct", []string{"Name"})
	if len(vals) != 0 {
		t.Fatalf("expected empty map, got %v", vals)
	}
}

// ===== GenerateSecureToken =====

func TestGenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}
}

// ===== Snowflake =====

func TestSnowflake_NextID(t *testing.T) {
	s, err := NewSnowflake()
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	id1 := s.NextID()
	id2 := s.NextID()
	if id1 == 0 || id2 == 0 {
		t.Fatal("IDs should not be 0")
	}
	if id1 == id2 {
		t.Fatal("IDs should be unique")
	}
}

func TestSnowflake_Global(t *testing.T) {
	if SnowflakeUtil == nil {
		t.Fatal("SnowflakeUtil should not be nil")
	}
	id := SnowflakeUtil.NextID()
	if id == 0 {
		t.Fatal("ID should not be 0")
	}
}

// ===== WriteFile / ReadFile =====

func TestWriteFile_ReadFile(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/test.txt"
	data := []byte("hello world")

	if err := WriteFile(filename, data); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	got, err := ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("data mismatch: %q", got)
	}
}

func TestWriteFile_CreateDir(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/sub/dir/test.txt"
	data := []byte("nested")

	if err := WriteFile(filename, data); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
}

// ===== RemoveEmoji =====

func TestRemoveEmoji(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "Hello World"},
		{"Hello 😀 World", "Hello  World"},
		{"", ""},
		{"No emoji here", "No emoji here"},
	}
	for _, tt := range tests {
		got := coerce.RemoveEmoji(tt.input)
		if got != tt.want {
			t.Errorf("RemoveEmoji(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRemoveEmojiFromJSON(t *testing.T) {
	input := `{"name":"test 😀","value":123}`
	got := coerce.RemoveEmojiFromJSON(input)
	if got == "" {
		t.Fatal("result should not be empty")
	}
}

// ===== ComputeSampleByteCount =====

func TestComputeSampleByteCount(t *testing.T) {
	// 8000 Hz, 16-bit, 1 channel
	got := ComputeSampleByteCount(8000, 16, 1)
	if got != 16 {
		t.Fatalf("want 16, got %d", got)
	}

	// 16000 Hz, 16-bit, 1 channel
	got = ComputeSampleByteCount(16000, 16, 1)
	if got != 32 {
		t.Fatalf("want 32, got %d", got)
	}
}

// ===== NormalizeFramePeriod =====

func TestNormalizeFramePeriod(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"20ms", 20 * time.Millisecond},
		{"30ms", 30 * time.Millisecond},
		{"0ms", 20 * time.Millisecond},
		{"5ms", 20 * time.Millisecond},
		{"400ms", 20 * time.Millisecond},
		{"invalid", 20 * time.Millisecond},
	}
	for _, tt := range tests {
		got := NormalizeFramePeriod(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeFramePeriod(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ===== PickImageExtFromContentType =====

func TestPickImageExtFromContentType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"IMAGE/JPEG", ".jpg"},
		{"text/plain", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := PickImageExtFromContentType(tt.input)
		if got != tt.want {
			t.Errorf("PickImageExtFromContentType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ===== JSONValueFromBytes =====

func TestJSONValueFromBytes(t *testing.T) {
	tests := []struct {
		input string
		want  any
	}{
		{`"hello"`, "hello"},
		{`123`, float64(123)},
		{`{"key":"value"}`, map[string]any{"key": "value"}},
		{`null`, nil},
		{"", nil},
		{`invalid`, nil},
	}
	for _, tt := range tests {
		got := JSONValueFromBytes([]byte(tt.input))
		if tt.want == nil && got != nil {
			t.Errorf("JSONValueFromBytes(%q) = %v, want nil", tt.input, got)
		}
	}
}

// ===== MarshalStringSliceJSON =====

func TestMarshalStringSliceJSON(t *testing.T) {
	got, err := MarshalStringSliceJSON([]string{"a", "b"}, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got != `["a","b"]` {
		t.Fatalf("got %q", got)
	}
}

func TestMarshalStringSliceJSON_Empty(t *testing.T) {
	got, err := MarshalStringSliceJSON(nil, []string{"default"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if got != `["default"]` {
		t.Fatalf("got %q", got)
	}
}

// ===== MustMarshalJSON =====

func TestMustMarshalJSON(t *testing.T) {
	got := MustMarshalJSON(map[string]string{"k": "v"})
	if string(got) != `{"k":"v"}` {
		t.Fatalf("got %q", got)
	}
}

// ===== NonEmptyOr =====

func TestNonEmptyOr(t *testing.T) {
	if NonEmptyOr("", "fallback") != "fallback" {
		t.Fatal("should return fallback")
	}
	if NonEmptyOr("value", "fallback") != "value" {
		t.Fatal("should return value")
	}
}

// ===== CloneRawMessage =====

func TestCloneRawMessage(t *testing.T) {
	original := json.RawMessage(`{"key":"value"}`)
	cloned := CloneRawMessage(original)
	if string(cloned) != string(original) {
		t.Fatalf("cloned mismatch")
	}
	// Modify original
	original[0] = '['
	if string(cloned) == string(original) {
		t.Fatal("clone should be independent")
	}
}

func TestCloneRawMessage_Empty(t *testing.T) {
	cloned := CloneRawMessage(nil)
	if string(cloned) != "null" {
		t.Fatalf("expected 'null', got %q", cloned)
	}
}

// ===== ParseOptionalRFC3339 =====

func TestParseOptionalRFC3339(t *testing.T) {
	// nil pointer
	got, err := ParseOptionalRFC3339(nil)
	if err != nil || got != nil {
		t.Fatalf("nil: got=%v, err=%v", got, err)
	}

	// Empty string
	s := ""
	got, err = ParseOptionalRFC3339(&s)
	if err != nil || got != nil {
		t.Fatalf("empty: got=%v, err=%v", got, err)
	}

	// Valid date
	s = "2024-01-15T10:30:00Z"
	got, err = ParseOptionalRFC3339(&s)
	if err != nil {
		t.Fatalf("valid: err=%v", err)
	}
	if got == nil {
		t.Fatal("valid: got should not be nil")
	}

	// Invalid date
	s = "not-a-date"
	_, err = ParseOptionalRFC3339(&s)
	if err == nil {
		t.Fatal("invalid: should return error")
	}
}

// ===== DeriveTenantSlug =====

func TestDeriveTenantSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"  test  ", "test"},
		{"test_name", "test-name"},
		{"TEST", "test"},
		{"a--b", "a-b"},
		{"", ""},
	}
	for _, tt := range tests {
		got := DeriveTenantSlug(tt.input)
		if got != tt.want {
			t.Errorf("DeriveTenantSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ===== ValidTenantSlug =====

func TestValidTenantSlug(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"ab", true},
		{"my-org", true},
		{"org123", true},
		{"a", false},
		{"", false},
		{"-start", false},
		{"end-", false},
		{"UPPER", false},
		{"has space", false},
	}
	for _, tt := range tests {
		got := ValidTenantSlug(tt.input)
		if got != tt.want {
			t.Errorf("ValidTenantSlug(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ===== Context =====

func TestContext_Background(t *testing.T) {
	ctx := context.Background()
	if ctx == nil {
		t.Fatal("context should not be nil")
	}
}

func TestDedupeUint(t *testing.T) {
	got := DedupeUint([]uint{1, 0, 1, 2, 2, 3})
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("%v", got)
	}
}

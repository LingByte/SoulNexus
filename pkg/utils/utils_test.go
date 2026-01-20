package utils

import (
	"encoding/base64"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------- randRunes / RandText / RandNumberText / RandString ----------

func TestRandRunesAndFriends(t *testing.T) {
	// Basic length assertion
	if got := RandText(16); len(got) != 16 {
		t.Fatalf("RandText length = %d, want 16", len(got))
	}
	if got := RandNumberText(8); len(got) != 8 {
		t.Fatalf("RandNumberText length = %d, want 8", len(got))
	}
	if got := RandString(12); len(got) != 12 {
		t.Fatalf("RandString length = %d, want 12", len(got))
	}

	// Number string should only contain digits
	num := RandNumberText(64)
	for i, c := range num {
		if c < '0' || c > '9' {
			t.Fatalf("RandNumberText contains non-digit at %d: %q", i, c)
		}
	}

	// Cover randRunes source path (pass in different rune source)
	alpha := []rune("ab")
	got := randRunes(32, alpha)
	for _, c := range got {
		if c != 'a' && c != 'b' {
			t.Fatalf("randRunes unexpected rune: %q", c)
		}
	}
}

// ---------- SafeCall ----------

func TestSafeCall_NoPanic(t *testing.T) {
	called := false
	err := SafeCall(func() error {
		called = true
		return nil
	}, func(error) {})
	if err != nil {
		t.Fatalf("SafeCall returned error: %v", err)
	}
	if !called {
		t.Fatalf("SafeCall did not call f()")
	}
}

func TestSafeCall_PanicError(t *testing.T) {
	var handled error
	_ = SafeCall(func() error {
		panic(assertErr("boom"))
	}, func(e error) {
		handled = e
	})
	if handled == nil || handled.Error() != "boom" {
		t.Fatalf("SafeCall did not handle error panic, got: %v", handled)
	}
}

func TestSafeCall_PanicString(t *testing.T) {
	var handled error
	_ = SafeCall(func() error {
		panic("oops")
	}, func(e error) {
		handled = e
	})
	if handled == nil || handled.Error() != "oops" {
		t.Fatalf("SafeCall did not convert string panic, got: %v", handled)
	}
}

func TestSafeCall_PanicUnknownType(t *testing.T) {
	var handled error
	_ = SafeCall(func() error {
		panic(struct{ X int }{X: 1})
	}, func(e error) {
		handled = e
	})
	if handled == nil || handled.Error() != "unknown error type" {
		t.Fatalf("SafeCall unknown-type panic => %v, want 'unknown error type'", handled)
	}
}

// Used to quickly construct types that implement error
type assertErr string

func (e assertErr) Error() string { return string(e) }

// ---------- StructAsMap ----------

func TestStructAsMap(t *testing.T) {
	type inner struct {
		A int
	}
	type demo struct {
		Name    string
		Age     int
		NotePtr *string
		InPtr   *inner
		ZeroStr string
		ZeroInt int
	}

	note := "hello"
	in := &inner{A: 7}

	// Non-struct
	if m := StructAsMap(123, []string{"X"}); len(m) != 0 {
		t.Fatalf("StructAsMap(non-struct) = %#v, want empty", m)
	}

	// Struct and pointer fields
	d := demo{
		Name:    "tom",
		Age:     18,
		NotePtr: &note,
		InPtr:   in,
	}
	// Select only some fields, including zero-value fields and non-existent fields
	fields := []string{"Name", "Age", "NotePtr", "InPtr", "ZeroStr", "ZeroInt", "NoSuch"}
	got := StructAsMap(d, fields)

	// Expected: zero-value and non-existent fields do not appear; pointer fields are dereferenced
	want := map[string]any{
		"Name":    "tom",
		"Age":     18,
		"NotePtr": "hello",
		"InPtr":   inner{A: 7},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StructAsMap got %#v, want %#v", got, want)
	}

	// Pass in pointer struct
	got2 := StructAsMap(&d, []string{"Name"})
	if got2["Name"] != "tom" || len(got2) != 1 {
		t.Fatalf("StructAsMap(ptr) = %#v", got2)
	}
}

// ---------- GenerateSecureToken ----------

func TestGenerateSecureToken_URLSafeLength(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 16, 31, 32, 33, 64} {
		tok, err := GenerateSecureToken(n)
		if err != nil {
			t.Fatalf("GenerateSecureToken(%d) error: %v", n, err)
		}
		// base64.URLEncoding length check
		wantLen := base64.URLEncoding.EncodedLen(n)
		if len(tok) != wantLen {
			t.Fatalf("token len = %d, want %d (n=%d)", len(tok), wantLen, n)
		}
		// URL-safe characters (no '+' '/')
		if strings.ContainsAny(tok, "+/") {
			t.Fatalf("token contains non-URL-safe characters: %q", tok)
		}
	}
}

// ---------- Snowflake: New / NextID ----------

func withEnv(key, val string, fn func()) {
	old := os.Getenv(key)
	_ = os.Setenv(key, val)
	defer os.Setenv(key, old)
	fn()
}

func TestNewSnowflake_OK_DefaultMachineID(t *testing.T) {
	// Invalid values will fallback to 1 (in getMachineID), which is within valid range
	withEnv("MACHINE_ID", "not-an-int", func() {
		sf, err := NewSnowflake()
		if err != nil {
			t.Fatalf("NewSnowflake with fallback id error: %v", err)
		}
		if sf.machineID != 1 {
			t.Fatalf("fallback machineID = %d, want 1", sf.machineID)
		}
	})
}

func TestNewSnowflake_ErrOutOfRange(t *testing.T) {
	// <0
	withEnv("MACHINE_ID", "-1", func() {
		if _, err := NewSnowflake(); err == nil {
			t.Fatalf("NewSnowflake expected error for id=-1")
		}
	})
	// > max
	tooBig := int64(maxMachineID) + 1
	withEnv("MACHINE_ID", os.Getenv("MACHINE_ID"), func() {
		_ = os.Setenv("MACHINE_ID", intToString(tooBig))
		if _, err := NewSnowflake(); err == nil {
			t.Fatalf("NewSnowflake expected error for id>max")
		}
	})
}

func intToString(v int64) string {
	// Avoid introducing strconv again; but this file can use strconv, so keep simple and use directly
	// Leaving the tool function here is fine
	return strconvItoa(v)
}

func strconvItoa(v int64) string {
	// Local implementation of a simple itoa (supports negative numbers) to avoid extra dependency
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [32]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func TestSnowflake_NextID_Monotonic(t *testing.T) {
	sf, err := NewSnowflake()
	if err != nil {
		t.Fatalf("NewSnowflake error: %v", err)
	}

	const N = 2000
	ids := make([]int64, N)
	for i := 0; i < N; i++ {
		ids[i] = sf.NextID()
		if ids[i] == 0 {
			t.Fatalf("NextID returned 0 unexpectedly")
		}
		if i > 0 && ids[i] <= ids[i-1] {
			t.Fatalf("IDs not strictly increasing: %d <= %d at %d", ids[i], ids[i-1], i)
		}
	}
}

func TestSnowflake_NextID_SameMicroSequenceAndRollover(t *testing.T) {
	sf, err := NewSnowflake()
	if err != nil {
		t.Fatalf("NewSnowflake error: %v", err)
	}

	// Simulate same microsecond pushing sequence to max, then NextID triggers rollover and "wait for next microsecond"
	sf.mu.Lock()
	now := currentMicro()
	sf.lastStamp = now
	sf.sequence = maxSequence
	sf.mu.Unlock()

	start := time.Now()
	id := sf.NextID()
	if id == 0 {
		t.Fatalf("NextID returned 0 on rollover")
	}
	// Due to rollover logic waiting for next microsecond, time taken should be >= 1 microsecond (usually much greater than 1µs in most environments)
	if time.Since(start) <= 0 {
		t.Fatalf("expected rollover wait to advance time")
	}
}

func TestSnowflake_NextID_ClockRollback(t *testing.T) {
	sf, err := NewSnowflake()
	if err != nil {
		t.Fatalf("NewSnowflake error: %v", err)
	}

	// Construct lastStamp > now scenario
	sf.mu.Lock()
	sf.lastStamp = currentMicro() + 10_000 // future
	sf.mu.Unlock()

	if got := sf.NextID(); got != 0 {
		t.Fatalf("clock rollback protection expected 0, got %d", got)
	}
}

// ---------- Concurrency smoke (optional, to ensure lock paths are covered more thoroughly) ----------

func TestSnowflake_Concurrent(t *testing.T) {
	sf, err := NewSnowflake()
	if err != nil {
		t.Fatalf("NewSnowflake error: %v", err)
	}

	const goroutines = 16
	const perG = 512
	var wg sync.WaitGroup
	out := make(chan int64, goroutines*perG)

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				out <- sf.NextID()
			}
		}()
	}
	wg.Wait()
	close(out)
	first := true
	for id := range out {
		if id == 0 {
			t.Fatalf("concurrent NextID produced 0")
		}
		if first {
			_ = id
			first = false
			continue
		}
		// In concurrent situations, strict monotonic order is not guaranteed, but all values should be positive and non-zero.
		if id < 0 {
			t.Fatalf("concurrent NextID produced negative id: %d", id)
		}
	}
}

// ---------- WriteFile / ReadFile ----------

func TestWriteFileAndReadFile(t *testing.T) {
	// Create temporary directory and filename
	tmpDir := os.TempDir()
	testFile := tmpDir + "/test_write_file.txt"
	testContent := []byte("Hello, World!")

	// Test writing to file
	err := WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Test reading file
	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	// Verify content
	if string(content) != string(testContent) {
		t.Fatalf("ReadFile content = %s, want %s", string(content), string(testContent))
	}

	// Clean up test file
	_ = os.Remove(testFile)
}

func TestWriteFileWithNestedPath(t *testing.T) {
	// Create nested directory path
	tmpDir := os.TempDir()
	nestedDir := tmpDir + "/test_nested_dir"
	testFile := nestedDir + "/nested/test_write_file.txt"
	testContent := []byte("Nested directory content")

	// Test writing to nested path file
	err := WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile with nested path error: %v", err)
	}

	// Test reading file
	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	// Verify content
	if string(content) != string(testContent) {
		t.Fatalf("ReadFile content = %s, want %s", string(content), string(testContent))
	}

	// Clean up test file and directory
	_ = os.RemoveAll(nestedDir)
}

func TestReadFileNotExists(t *testing.T) {
	// Attempt to read non-existent file
	_, err := ReadFile("/path/does/not/exist.txt")
	if err == nil {
		t.Fatalf("ReadFile expected error for non-existent file")
	}
}

// ---------- getMachineID ----------

func TestGetMachineID(t *testing.T) {
	// Save original environment variable
	originalVal := os.Getenv("MACHINE_ID")
	defer func() {
		// Restore original environment variable
		if originalVal != "" {
			_ = os.Setenv("MACHINE_ID", originalVal)
		} else {
			_ = os.Unsetenv("MACHINE_ID")
		}
	}()

	// Test default value (environment variable not set)
	_ = os.Unsetenv("MACHINE_ID")
	id := getMachineID()
	if id != 1 {
		t.Fatalf("getMachineID() = %d, want 1 (default)", id)
	}

	// Test valid environment variable
	_ = os.Setenv("MACHINE_ID", "42")
	id = getMachineID()
	if id != 42 {
		t.Fatalf("getMachineID() = %d, want 42", id)
	}

	// Test invalid environment variable (non-numeric)
	_ = os.Setenv("MACHINE_ID", "invalid")
	id = getMachineID()
	if id != 1 {
		t.Fatalf("getMachineID() = %d, want 1 (fallback for invalid value)", id)
	}
}

// ---------- randRunes ----------

func TestRandRunes(t *testing.T) {
	// Test using letter source
	result := randRunes(10, letterRunes)
	if len(result) != 10 {
		t.Fatalf("randRunes length = %d, want 10", len(result))
	}

	// Verify result only contains letters and digits
	for _, r := range result {
		if (r < '0' || r > '9') && (r < 'a' || r > 'z') {
			t.Fatalf("randRunes contains invalid character: %c", r)
		}
	}

	// Test using number source
	result = randRunes(5, numberRunes)
	if len(result) != 5 {
		t.Fatalf("randRunes length = %d, want 5", len(result))
	}

	// Verify result only contains digits
	for _, r := range result {
		if r < '0' || r > '9' {
			t.Fatalf("randRunes contains non-digit character: %c", r)
		}
	}

	// Test length of 0
	result = randRunes(0, letterRunes)
	if result != "" {
		t.Fatalf("randRunes(0) = %q, want empty string", result)
	}
}

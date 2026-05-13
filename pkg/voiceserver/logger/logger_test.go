package logger

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func resetLogger() {
	// 重置单例状态，让每个测试独立
	once = sync.Once{}
	Lg = nil
	cfg = nil
	currentDate = ""
}

// 测试：无效日志级别应该返回错误
func TestInitInvalidLevel(t *testing.T) {
	resetLogger()

	config := &LogConfig{
		Level:    "invalid-level",
		Filename: "",
	}

	err := Init(config, "prod")
	assert.Error(t, err, "expected error for invalid log level")
}

// 测试：开发模式控制台输出（不检查颜色，只检查内容）
func TestInitDevModeConsoleSplit(t *testing.T) {
	resetLogger()

	config := &LogConfig{
		Level:    "debug",
		Filename: "",
	}

	// 捕获 stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Init(config, "dev")
	assert.NoError(t, err)

	Info("test-dev-log")
	Sync()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// 只检查是否包含日志内容，不检查颜色
	assert.True(t, strings.Contains(output, "test-dev-log"),
		"stdout should contain dev log message")
}

// 测试：文件输出 + caller 信息是否正常
func TestAddCallerEnabled(t *testing.T) {
	resetLogger()

	tmpDir := t.TempDir()
	logFile := tmpDir + "/caller.log"

	config := &LogConfig{
		Level:    "debug",
		Filename: logFile,
	}

	err := Init(config, "prod")
	assert.NoError(t, err)

	Info("test-caller-enabled")
	Sync()

	// 等待文件写入
	time.Sleep(100 * time.Millisecond)

	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	logStr := string(content)
	assert.True(t, strings.Contains(logStr, "test-caller-enabled"),
		"log should contain caller message")
	assert.True(t, strings.Contains(logStr, "logger_test.go"),
		"log should contain caller file name")
}

// 基础功能测试
func TestLoggerBasicFunctions(t *testing.T) {
	resetLogger()

	config := &LogConfig{
		Level:    "debug",
		Filename: "",
	}
	err := Init(config, "dev")
	assert.NoError(t, err)

	// 所有级别都能调用不崩溃
	Debug("debug-test")
	Info("info-test")
	Warn("warn-test")
	Error("error-test")

	Sync()
	assert.True(t, true)
}

// 测试脱敏功能
func TestMaskString(t *testing.T) {
	assert.Equal(t, "****", MaskString("123"))
	assert.Equal(t, "12****78", MaskString("12345678"))
}

func TestMaskEmail(t *testing.T) {
	assert.Equal(t, "ab****@qq.com", MaskEmail("ab12345@qq.com"))
}

func TestMaskPhone(t *testing.T) {
	assert.Equal(t, "138****5678", MaskPhone("13812345678"))
}

// 测试上下文日志
func TestContextFields(t *testing.T) {
	resetLogger()

	config := &LogConfig{Level: "info", Filename: ""}
	_ = Init(config, "prod")

	ctx := context.WithValue(context.Background(), TraceIDKey, "trace-123")
	InfoCtx(ctx, "context-test")

	assert.True(t, true)
}

// initTestLogger drops in a no-op zap logger so the package's wrapper functions
// have a Lg to forward to. We bypass Init() entirely to avoid file I/O.
func initTestLogger(t *testing.T) {
	t.Helper()
	once = sync.Once{}
	Lg = zap.New(zapcore.NewNopCore())
	cfg = &LogConfig{Level: "debug", Filename: ""}
}

// ---------- Wrapper smoke tests (Lg present) ------------------------------

func TestWrappers_AllLevelsAndFormatters(t *testing.T) {
	initTestLogger(t)

	// non-format variants
	Info("info msg", zap.String("k", "v"))
	Warn("warn msg")
	Error("error msg")
	Debug("debug msg")

	// format variants
	Infof("infof %d", 1)
	Warnf("warnf %s", "x")
	Errorf("errorf %v", errors.New("boom"))
	Debugf("debugf %d", 7)

	// field-map variants
	InfoWithFields("info-fields", map[string]interface{}{"a": 1})
	WarnWithFields("warn-fields", map[string]interface{}{"a": 1})
	ErrorWithFields("error-fields", map[string]interface{}{"a": 1})
	DebugWithFields("debug-fields", map[string]interface{}{"a": 1})

	// Sync should be safe
	Sync()
}

func TestContextWrappers(t *testing.T) {
	initTestLogger(t)

	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, UserIDKey, "user-1")

	InfoCtx(ctx, "info-ctx")
	WarnCtx(ctx, "warn-ctx")
	ErrorCtx(ctx, "error-ctx")
	DebugCtx(ctx, "debug-ctx")
	InfofCtx(ctx, "infof-ctx %d", 42)
	ErrorfCtx(ctx, "errorf-ctx %s", "x")

	// nil ctx must be tolerated
	InfoCtx(nil, "info-nil-ctx")
	if got := appendContextFields(nil); len(got) != 0 {
		t.Errorf("appendContextFields(nil) returned %v", got)
	}
}

// ---------- Wrappers must be no-ops when Lg is nil ------------------------

func TestWrappers_NilLogger_AreNoOps(t *testing.T) {
	once = sync.Once{}
	Lg = nil

	// These four explicitly check for nil Lg.
	Info("x")
	Warn("x")
	Error("x")
	Debug("x")
	Sync()

	// Restore for any subsequent tests.
	initTestLogger(t)
}

// ---------- Field helpers --------------------------------------------------

func TestWithField_AndWithFields_AndWithError(t *testing.T) {
	f := WithField("k", 42)
	if f.key != "k" {
		t.Errorf("WithField key = %q, want %q", f.key, "k")
	}

	zfs := WithFields(map[string]interface{}{"a": 1, "b": "two"})
	if len(zfs) != 2 {
		t.Errorf("WithFields produced %d entries, want 2", len(zfs))
	}

	zf := WithError(errors.New("oops"))
	if zf.Key != "error" {
		t.Errorf("WithError key = %q, want %q", zf.Key, "error")
	}
}

// ---------- GetDailyLogFilename -------------------------------------------

func TestGetDailyLogFilename_Format(t *testing.T) {
	got := GetDailyLogFilename("/var/log/app.log")
	if !strings.HasPrefix(got, "/var/log/app-") || !strings.HasSuffix(got, ".log") {
		t.Errorf("GetDailyLogFilename produced %q", got)
	}
	// extension preserved
	if filepath.Ext(got) != ".log" {
		t.Errorf("extension lost: %q", got)
	}
	// no extension input
	if got := GetDailyLogFilename("plain"); got == "plain" || !strings.HasPrefix(got, "plain-") {
		t.Errorf("no-ext input → %q", got)
	}
}

// ---------- Masking helpers -----------------------------------------------

func TestMaskString_Branches(t *testing.T) {
	if got := MaskString(""); got != "****" {
		t.Errorf("empty → %q", got)
	}
	if got := MaskString("ab"); got != "****" {
		t.Errorf("len 2 → %q", got)
	}
	if got := MaskString("abcd"); got != "****" {
		t.Errorf("len 4 (boundary, ≤4 → mask all) → %q", got)
	}
	if got := MaskString("abcdef"); got != "ab****ef" {
		t.Errorf("len 6 → %q", got)
	}
}

func TestMaskEmail_Branches(t *testing.T) {
	if got := MaskEmail("ab@x.com"); got != "****@x.com" {
		t.Errorf("short user → %q", got)
	}
	if got := MaskEmail("alice@example.com"); got != "al****@example.com" {
		t.Errorf("long user → %q", got)
	}
	if got := MaskEmail("not-an-email"); got != "no****il" {
		// falls back to MaskString for malformed addresses
		t.Errorf("malformed → %q", got)
	}
}

func TestMaskPhone_Branches(t *testing.T) {
	if got := MaskPhone("12345"); got != "****" {
		t.Errorf("short → %q", got)
	}
	if got := MaskPhone("13800138000"); got != "138****8000" {
		t.Errorf("11-digit → %q", got)
	}
}

func TestMaskField_NonSensitivePassthrough(t *testing.T) {
	if got := MaskField("name", "Alice"); got != "Alice" {
		t.Errorf("non-sensitive should pass through, got %v", got)
	}
}

func TestMaskField_SensitiveBranches(t *testing.T) {
	cases := map[string]string{
		"password":      "secret123",
		"token":         "abcdef",
		"email":         "alice@example.com",
		"phone":         "13800138000",
		"refresh_token": "xyz",
	}
	for k, v := range cases {
		out := MaskField(k, v)
		if out == v {
			t.Errorf("sensitive key %q was not masked: %v", k, out)
		}
	}
}

func TestMaskFields_All(t *testing.T) {
	in := map[string]interface{}{
		"name":     "Alice",
		"password": "secret",
		"phone":    "13800138000",
	}
	out := MaskFields(in)
	if out["name"] != "Alice" {
		t.Errorf("non-sensitive name should remain")
	}
	if out["password"] == "secret" {
		t.Errorf("password not masked: %v", out["password"])
	}
}

func TestAddRemoveSensitiveField(t *testing.T) {
	const k = "vs_test_custom_secret"
	defer RemoveSensitiveField(k)

	if SensitiveFields[k] {
		t.Fatalf("key already registered")
	}
	AddSensitiveField(strings.ToUpper(k))
	if !SensitiveFields[k] {
		t.Errorf("AddSensitiveField did not lower-case key")
	}
	RemoveSensitiveField(strings.ToUpper(k))
	if SensitiveFields[k] {
		t.Errorf("RemoveSensitiveField did not lower-case key")
	}
}

func TestMaskValue_JSONRedaction(t *testing.T) {
	in := `{"name":"alice","password":"hunter2","other":"keep","token":"abcdef"}`
	out := MaskValue(in)
	if strings.Contains(out, "hunter2") {
		t.Errorf("password leaked: %s", out)
	}
	if !strings.Contains(out, `"name":"alice"`) {
		t.Errorf("non-sensitive name lost: %s", out)
	}
}

func TestInfoWithMaskedFields_AndError(t *testing.T) {
	initTestLogger(t)
	fields := map[string]interface{}{"password": "x"}
	InfoWithMaskedFields("masked-info", fields)
	ErrorWithMaskedFields("masked-error", fields)
}

// ---------- getLogWriter (daily and non-daily branches) -------------------

func TestGetLogWriter_DailyAndNonDailyBranches(t *testing.T) {
	tmp := t.TempDir()
	w1 := getLogWriter(filepath.Join(tmp, "a.log"), 1, 1, 1, false)
	if w1 == nil {
		t.Fatal("non-daily writer nil")
	}
	w2 := getLogWriter(filepath.Join(tmp, "b.log"), 1, 1, 1, true)
	if w2 == nil {
		t.Fatal("daily writer nil")
	}
	// Exercise Write+Sync on the daily wrapper.
	if _, err := w2.Write([]byte("hello\n")); err != nil {
		t.Errorf("daily Write: %v", err)
	}
	_ = w2.Sync()
}

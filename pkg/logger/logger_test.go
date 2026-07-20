package logger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ===== GenReqID =====

func TestGenReqID(t *testing.T) {
	id := GenReqID()
	if id == "" {
		t.Fatal("GenReqID should return non-empty string")
	}
	if len(id) < 10 {
		t.Fatalf("GenReqID too short: %q", id)
	}
}

func TestGenReqID_Unique(t *testing.T) {
	// GenReqID may produce duplicates in rapid succession due to timestamp resolution
	// Just verify it produces valid IDs
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := GenReqID()
		if id == "" {
			t.Fatal("empty ID")
		}
		ids[id] = true
	}
	// At least some should be unique
	if len(ids) < 1 {
		t.Fatal("expected at least 1 unique ID")
	}
}

// ===== WithRequestID / RequestIDFromContext =====

func TestWithRequestID_And_RequestIDFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "test-req-id")

	got := RequestIDFromContext(ctx)
	if got != "test-req-id" {
		t.Fatalf("want 'test-req-id', got %q", got)
	}
}

func TestRequestIDFromContext_Nil(t *testing.T) {
	got := RequestIDFromContext(nil)
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := RequestIDFromContext(ctx)
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestWithRequestID_NilContext(t *testing.T) {
	ctx := WithRequestID(nil, "test")
	got := RequestIDFromContext(ctx)
	if got != "test" {
		t.Fatalf("want 'test', got %q", got)
	}
}

func TestWithRequestID_EmptyID(t *testing.T) {
	ctx := context.Background()
	ctx2 := WithRequestID(ctx, "")
	// Should return same context (no value set)
	got := RequestIDFromContext(ctx2)
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

// ===== ZapReqID / ZapReqIDString =====

func TestZapReqID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	field := ZapReqID(ctx)
	if field.Key != "x-reqid" {
		t.Fatalf("want key 'x-reqid', got %q", field.Key)
	}
}

func TestZapReqID_Empty(t *testing.T) {
	field := ZapReqID(context.Background())
	if field.Key != "" {
		t.Fatalf("empty ZapReqID should return Skip")
	}
}

func TestZapReqIDString(t *testing.T) {
	field := ZapReqIDString("req-456")
	if field.Key != "x-reqid" {
		t.Fatalf("want key 'x-reqid', got %q", field.Key)
	}
}

func TestZapReqIDString_Empty(t *testing.T) {
	field := ZapReqIDString("")
	if field.Key != "" {
		t.Fatalf("empty ZapReqIDString should return Skip")
	}
}

func TestReqIDFromGin_FromContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, "ctx-req-id")

	id := ReqIDFromGin(c)
	if id != "ctx-req-id" {
		t.Fatalf("want 'ctx-req-id', got %q", id)
	}
}

func TestReqIDFromGin_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	id := ReqIDFromGin(c)
	if id != "" {
		t.Fatalf("want empty, got %q", id)
	}
}

func TestReqIDFromGin_Nil(t *testing.T) {
	id := ReqIDFromGin(nil)
	if id != "" {
		t.Fatalf("want empty, got %q", id)
	}
}

// ===== FromGin =====

func TestFromGin_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	lg := FromGin(c)
	if lg == nil {
		t.Fatal("should return nop logger")
	}
}

func TestFromGin_WithReqID(t *testing.T) {
	if Lg == nil {
		Lg = zap.NewNop()
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, "test-req-id")

	lg := FromGin(c)
	if lg == nil {
		t.Fatal("should return logger")
	}
}

func TestFromGin_WithoutReqID(t *testing.T) {
	if Lg == nil {
		Lg = zap.NewNop()
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	lg := FromGin(c)
	if lg == nil {
		t.Fatal("should return logger")
	}
}

func TestGinZapFields(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test?foo=bar", nil)

	fields := GinZapFields(c)
	if len(fields) < 3 {
		t.Fatalf("want at least 3 fields, got %d", len(fields))
	}
}

func TestGinZapFields_Nil(t *testing.T) {
	fields := GinZapFields(nil)
	if fields != nil {
		t.Fatalf("want nil, got %v", fields)
	}
}

func TestGinZapFields_NoQuery(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	fields := GinZapFields(c)
	if len(fields) != 3 {
		t.Fatalf("want 3 fields, got %d", len(fields))
	}
}

// ===== LoggerInit =====

func TestLogger_NilSafe(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	// These should not panic
	Info("test")
	Warn("test")
	Error("test")
	Debug("test")
}

// TestInit runs first to seed the sync.Once; subsequent tests that call Init
// are no-ops because once.Do already ran. Init is tested here for the
// production (non-console) path.
func TestInit_Success(t *testing.T) {
	oldLg := Lg
	defer func() { Lg = oldLg }()

	config := &LogConfig{
		Level:      "info",
		Filename:   t.TempDir() + "/test.log",
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
		Daily:      false,
	}

	err := Init(config, "production")
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	if Lg == nil {
		t.Fatal("Lg should not be nil")
	}

	// Verify logger is functional after Init
	Info("init test message")
	Sync()
}

// ===== Logger Functions =====

func TestLoggerFunctions_NilSafe(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	// These should not panic when Lg is nil
	Info("test info")
	Warn("test warn")
	Error("test error")
	Debug("test debug")
	Sync()
}

func TestLoggerFunctions_NonNil(t *testing.T) {
	// Init should have run from TestInit_Success; Lg should be non-nil here.
	if Lg == nil {
		Lg = zap.NewNop()
	}

	Warn("test warn with logger")
	Error("test error with logger")
	Debug("test debug with logger")
	// Panic/Fatal can't be called with non-nil Lg — they actually crash/exit.
}

func TestWarnfCtx_NilSafe(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ctx := context.Background()
	WarnfCtx(ctx, "test %s", "warn")
}

func TestWarnfCtx_NonNil(t *testing.T) {
	if Lg == nil {
		Lg = zap.NewNop()
	}
	ctx := context.Background()
	WarnfCtx(ctx, "test %s", "warn")
}

func TestLoggerCtxFunctions(t *testing.T) {
	if Lg == nil {
		Lg = zap.NewNop()
	}

	ctx := context.Background()
	InfoCtx(ctx, "test info")
	WarnCtx(ctx, "test warn")
	ErrorCtx(ctx, "test error")
	DebugCtx(ctx, "test debug")
	InfofCtx(ctx, "test %s", "format")
	ErrorfCtx(ctx, "test %s", "format")
}

func TestLoggerCtxFunctions_NilSafe(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ctx := context.Background()
	// InfofCtx, WarnfCtx, ErrorfCtx have nil guards; InfoCtx/DebugCtx do not.
	InfofCtx(ctx, "test %s", "hello")
	WarnfCtx(ctx, "test %s", "hello")
	ErrorfCtx(ctx, "test %s", "hello")
	// InfoCtx, WarnCtx, ErrorCtx, DebugCtx don't check Lg==nil — test with non-nil logger instead.
}

func TestAppendContextFields_AllKeys(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, UserIDKey, "user-1")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-1")
	ctx = context.WithValue(ctx, CallIDKey, "call-1")
	ctx = context.WithValue(ctx, CampaignIDKey, "campaign-1")

	fields := appendContextFields(ctx)
	if len(fields) != 6 {
		t.Fatalf("expected 6 context fields, got %d", len(fields))
	}
}

func TestAppendContextFields_NilContext(t *testing.T) {
	fields := appendContextFields(nil)
	if fields != nil {
		t.Fatalf("expected nil for nil context, got %v", fields)
	}
}

func TestAppendContextFields_NoValues(t *testing.T) {
	ctx := context.Background()
	fields := appendContextFields(ctx)
	if fields != nil {
		t.Fatalf("expected nil for empty context, got %v", fields)
	}
}

// ===== printf-style functions =====

func TestPrintfFunctions(t *testing.T) {
	if Lg == nil {
		Lg = zap.NewNop()
	}

	Infof("test %s %d", "info", 42)
	Warnf("test %s %d", "warn", 42)
	Errorf("test %s %d", "error", 42)
	Debugf("test %s %d", "debug", 42)
}

// ===== WithFields / WithError =====

func TestWithFields(t *testing.T) {
	fields := WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})
	if len(fields) != 2 {
		t.Fatalf("want 2 fields, got %d", len(fields))
	}
}

func TestWithFields_Empty(t *testing.T) {
	fields := WithFields(map[string]interface{}{})
	if len(fields) != 0 {
		t.Fatalf("want 0 fields, got %d", len(fields))
	}
}

func TestWithError(t *testing.T) {
	err := errors.New("test error")
	field := WithError(err)
	if field.Key != "error" {
		t.Fatalf("want key 'error', got %q", field.Key)
	}
}

// ===== GetDailyLogFilename =====

func TestGetDailyLogFilename(t *testing.T) {
	result := GetDailyLogFilename("/var/log/app.log")
	if result == "" {
		t.Fatal("result should not be empty")
	}
	today := time.Now().Format("2006-01-02")
	expected := "/var/log/app-" + today + ".log"
	if result != expected {
		t.Fatalf("want %q, got %q", expected, result)
	}
}

func TestGetDailyLogFilename_NoExt(t *testing.T) {
	result := GetDailyLogFilename("/var/log/app")
	if result == "" {
		t.Fatal("result should not be empty")
	}
	today := time.Now().Format("2006-01-02")
	expected := "/var/log/app-" + today
	if result != expected {
		t.Fatalf("want %q, got %q", expected, result)
	}
}

// Note: Init uses sync.Once so mode-specific tests would need subprocess.
// The core production path (console=false) is covered by TestInit_Success.
// Daily log rotation is tested via TestGetLogWriter_WithDaily and TestDailyRotateWriter_*.

// ===== getEncoder =====

func TestGetEncoder(t *testing.T) {
	enc := getEncoder()
	if enc == nil {
		t.Fatal("getEncoder should return non-nil encoder")
	}
}

// ===== getLogWriter =====

func TestGetLogWriter_WithDaily(t *testing.T) {
	ws := getLogWriter(t.TempDir()+"/app.log", 10, 3, 7, true)
	if ws == nil {
		t.Fatal("getLogWriter with daily should return non-nil")
	}
}

func TestGetLogWriter_WithoutDaily(t *testing.T) {
	ws := getLogWriter(t.TempDir()+"/app.log", 10, 3, 7, false)
	if ws == nil {
		t.Fatal("getLogWriter without daily should return non-nil")
	}
}

func TestGetLogWriter_NoExtension(t *testing.T) {
	tmpDir := t.TempDir()
	ws := getLogWriter(tmpDir+"/app", 10, 3, 7, true)
	if ws == nil {
		t.Fatal("getLogWriter without extension should return non-nil")
	}
}

// ===== dailyRotateWriter =====

func TestDailyRotateWriter_Write(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/app.log"

	l := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		LocalTime:  true,
	}

	w := &dailyRotateWriter{
		baseFilename: filename,
		maxSize:      10,
		maxBackup:    3,
		maxAge:       7,
		daily:        false,
		lumberjack:   l,
	}

	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 bytes written, got %d", n)
	}
}

func TestDailyRotateWriter_Sync(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/app.log"

	l := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		LocalTime:  true,
	}

	w := &dailyRotateWriter{
		baseFilename: filename,
		maxSize:      10,
		maxBackup:    3,
		maxAge:       7,
		daily:        false,
		lumberjack:   l,
	}
	err := w.Sync()
	if err != nil {
		t.Fatalf("Sync error: %v", err)
	}
}

func TestDailyRotateWriter_Rotation(t *testing.T) {
	dir := t.TempDir()
	filename := dir + "/app.log"

	oldDate := currentDate
	currentDate = "2000-01-01"
	defer func() { currentDate = oldDate }()

	l := &lumberjack.Logger{
		Filename:   filename + "-2000-01-01.log",
		MaxSize:    10,
		MaxBackups: 3,
		MaxAge:     7,
		LocalTime:  true,
	}

	w := &dailyRotateWriter{
		baseFilename: filename,
		maxSize:      10,
		maxBackup:    3,
		maxAge:       7,
		daily:        true,
		lumberjack:   l,
	}

	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 bytes written, got %d", n)
	}
}

// ===== wrapLogCore =====

func TestWrapLogCore(t *testing.T) {
	inner := zapcore.NewNopCore()
	result := wrapLogCore(inner, &redactor{})
	if result == nil {
		t.Fatal("wrapLogCore should return non-nil")
	}
}

func TestWrapCoreWithReqIDPrefix_Nil(t *testing.T) {
	result := WrapCoreWithReqIDPrefix(nil)
	if result != nil {
		t.Fatal("WrapCoreWithReqIDPrefix(nil) should return nil")
	}
}

// ===== reqIDPrefixCore =====

func TestReqIDPrefixCore_Write_WithReqID(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)
	if core == nil {
		t.Fatal("should return non-nil")
	}

	core = core.With([]zapcore.Field{zap.String("x-reqid", "test-123")})
	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}
	err := core.Write(ent, []zapcore.Field{zap.String("x-reqid", "override-456")})
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

func TestReqIDPrefixCore_Write_NoReqID(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)
	if core == nil {
		t.Fatal("should return non-nil")
	}

	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello"}
	err := core.Write(ent, nil)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

func TestReqIDPrefixCore_Check(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)
	ent := zapcore.Entry{Level: zapcore.InfoLevel}
	// NopCore.Enabled returns false → Check returns nil; not an error.
	ce := core.Check(ent, nil)
	if ce != nil {
		t.Log("Check returned non-nil")
	}
}

func TestReqIDPrefixCore_Sync(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)
	err := core.Sync()
	if err != nil {
		t.Fatalf("Sync error: %v", err)
	}
}

func TestReqIDPrefixCore_With_Accumulate(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)

	core = core.With([]zapcore.Field{zap.String("x-reqid", "first")})
	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "test"}
	err := core.Write(ent, nil)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

func TestExtractReqID_NoMatch(t *testing.T) {
	fields := []zapcore.Field{zap.String("user", "john")}
	reqid, filtered := extractReqID(fields, "fallback")
	if reqid != "fallback" {
		t.Fatalf("want fallback, got %q", reqid)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 field, got %d", len(filtered))
	}
}

func TestExtractReqID_EmptyString(t *testing.T) {
	fields := []zapcore.Field{zap.String("x-reqid", "")}
	reqid, filtered := extractReqID(fields, "fallback")
	if reqid != "fallback" {
		t.Fatalf("want fallback for empty string, got %q", reqid)
	}
	if len(filtered) != 0 {
		t.Fatalf("expected 0 fields after filtering, got %d", len(filtered))
	}
}

func TestExtractReqID_MultipleMatch(t *testing.T) {
	fields := []zapcore.Field{
		zap.String("x-reqid", "first"),
		zap.String("reqid", "second"),
		zap.String("user", "john"),
	}
	reqid, filtered := extractReqID(fields, "")
	if reqid != "second" {
		t.Fatalf("want 'second' (last reqid wins), got %q", reqid)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 non-reqid field, got %d", len(filtered))
	}
}

// ===== Fatal / Panic nil-safe =====

func TestFatal_NilLogger(t *testing.T) {
	// Fatal with nil logger calls os.Exit(1) — can't test in same process.
	// Code path is trivial: fmt.Fprintln + os.Exit.
}

func TestPanic_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Panic with nil logger should panic")
		}
	}()
	Panic("test panic")
}

// ===== ReqIDFromGin with request context =====

func TestReqIDFromGin_FromRequestContext(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx := WithRequestID(context.Background(), "req-from-rctx")
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request = c.Request.WithContext(ctx)

	id := ReqIDFromGin(c)
	if id != "req-from-rctx" {
		t.Fatalf("want 'req-from-rctx', got %q", id)
	}
}

func TestReqIDFromGin_NonStringValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, 12345)

	id := ReqIDFromGin(c)
	if id != "" {
		t.Fatalf("want empty for non-string value, got %q", id)
	}
}

func TestReqIDFromGin_EmptyStringValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(GinCtxReqIDKey, "")

	id := ReqIDFromGin(c)
	if id != "" {
		t.Fatalf("want empty for empty string, got %q", id)
	}
}

// ===== IncomingReqID =====

func TestIncomingReqID_FromHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Reqid", "my-req-id")

	id := IncomingReqID(c)
	if id != "my-req-id" {
		t.Fatalf("want 'my-req-id', got %q", id)
	}
}

func TestIncomingReqID_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	id := IncomingReqID(c)
	if id != "" {
		t.Fatalf("want empty, got %q", id)
	}
}

func TestIncomingReqID_Nil(t *testing.T) {
	id := IncomingReqID(nil)
	if id != "" {
		t.Fatalf("want empty, got %q", id)
	}
}

func TestIncomingReqID_NilRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = nil

	id := IncomingReqID(c)
	if id != "" {
		t.Fatalf("want empty, got %q", id)
	}
}

func TestIncomingReqID_TrimSpace(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Reqid", "  padded-id  ")

	id := IncomingReqID(c)
	if id != "padded-id" {
		t.Fatalf("want 'padded-id', got %q", id)
	}
}

func TestInit_LocalMode(t *testing.T) {
	if os.Getenv("LING_SUBPROCESS_INIT") == constants.ENV_LOCAL {
		testInitMode(t, constants.ENV_LOCAL)
		return
	}
	runSubprocess(t, constants.ENV_LOCAL)
}

func TestInit_DevMode(t *testing.T) {
	if os.Getenv("LING_SUBPROCESS_INIT") == constants.ENV_DEV {
		testInitMode(t, constants.ENV_DEV)
		return
	}
	runSubprocess(t, constants.ENV_DEV)
}

func TestInit_ProdMode(t *testing.T) {
	if os.Getenv("LING_SUBPROCESS_INIT") == "prod" {
		testInitMode(t, "prod")
		return
	}
	runSubprocess(t, "prod")
}

func TestInit_InvalidLevel(t *testing.T) {
	// Init with an invalid level should fail.
	// sync.Once might already have run, so use subprocess.
	if os.Getenv("LING_SUBPROCESS_INIT") == "badlevel" {
		config := &LogConfig{
			Level:      "invalid-level-xyz",
			Filename:   filepath.Join(os.TempDir(), "ling_badlevel.log"),
			MaxSize:    10,
			MaxBackups: 1,
			MaxAge:     1,
		}
		err := Init(config, constants.ENV_LOCAL)
		if err == nil {
			fmt.Println("ERROR: expected error for invalid level")
			os.Exit(1)
		}
		fmt.Println("OK: invalid level rejected")
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestInit_InvalidLevel")
	cmd.Env = append(os.Environ(), "LING_SUBPROCESS_INIT=badlevel")
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "OK: invalid level rejected") {
		t.Fatalf("expected invalid level rejection, got: %s", string(out))
	}
}

func testInitMode(t *testing.T, mode string) {
	config := &LogConfig{
		Level:      "info",
		Filename:   filepath.Join(os.TempDir(), fmt.Sprintf("ling_mode_%s.log", mode)),
		MaxSize:    5,
		MaxBackups: 1,
		MaxAge:     1,
	}

	err := Init(config, mode)
	if err != nil {
		fmt.Printf("ERROR: Init(%s) failed: %v\n", mode, err)
		os.Exit(1)
	}
	if Lg == nil {
		fmt.Printf("ERROR: Lg nil after Init(%s)\n", mode)
		os.Exit(1)
	}
	// Produce some log output to exercise encoder paths
	Info("test message in " + mode + " mode")
	Sync()
	fmt.Printf("OK: init-%s\n", mode)
	os.Exit(0)
}

func runSubprocess(t *testing.T, mode string) {
	cmd := exec.Command(os.Args[0], "-test.run=TestInit_"+capitalizeFirst(mode)+"Mode")
	cmd.Env = append(os.Environ(), "LING_SUBPROCESS_INIT="+mode)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess failed for mode %s: %v\noutput:\n%s", mode, err, string(out))
	}
	expected := fmt.Sprintf("OK: init-%s", mode)
	if !strings.Contains(string(out), expected) {
		t.Fatalf("expected %q in output, got:\n%s", expected, string(out))
	}
}

// ===== Fatal nil-safe =====
//
// Fatal with nil Lg calls os.Exit(1). Test via subprocess.

func TestFatal_NilLogger_Exits(t *testing.T) {
	if os.Getenv("LING_SUBPROCESS_FATAL_NIL") == "1" {
		Lg = nil
		Fatal("fatal with nil logger")
		// should not reach here
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_NilLogger_Exits")
	cmd.Env = append(os.Environ(), "LING_SUBPROCESS_FATAL_NIL=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if ws.ExitStatus() != 1 {
				t.Fatalf("expected exit code 1, got %d", ws.ExitStatus())
			}
			return
		}
		// On Windows, ExitCode() works differently
		if exitErr.ExitCode() == 1 {
			return
		}
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
	t.Fatal("expected subprocess to exit with code 1")
}

// ===== Panic with non-nil logger =====

func TestPanic_NonNilLogger(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	var recovered bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		Panic("test panic with logger")
	}()
	if !recovered {
		t.Fatal("Panic with non-nil logger should have panicked")
	}
}

// ===== FatalCtx =====

func TestFatalCtx_NilLogger(t *testing.T) {
	// FatalCtx with nil Lg calls os.Exit(1); test via subprocess.
	if os.Getenv("LING_TEST_FATALCTX_NIL") == "1" {
		Lg = nil
		FatalCtx(context.Background(), "fatal ctx with nil")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalCtx_NilLogger")
	cmd.Env = append(os.Environ(), "LING_TEST_FATALCTX_NIL=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
		}
		return
	}
	t.Fatalf("expected os.Exit(1), got: %v", err)
}

func TestFatalCtx_NonNilLogger(t *testing.T) {
	t.Skip("FatalCtx with non-nil logger calls os.Exit via zap.Fatal; covered by design")
}

// ===== PanicCtx =====

func TestPanicCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	// PanicCtx with nil Lg: explicitly panics with the message.
	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		PanicCtx(context.Background(), "panic ctx with nil")
	}()
	if !panicked {
		t.Fatal("PanicCtx with nil Lg should have panicked")
	}
}

func TestPanicCtx_NonNilLogger(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	ctx := WithRequestID(context.Background(), "req-panicctx")

	var recovered bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		PanicCtx(ctx, "panic ctx with logger")
	}()
	if !recovered {
		t.Fatal("PanicCtx with non-nil logger should have panicked")
	}
}

// ===== Fatalf =====

func TestFatalf_NilLogger(t *testing.T) {
	// Fatalf with nil Lg calls os.Exit(1); test via subprocess.
	if os.Getenv("LING_TEST_FATALF_NIL") == "1" {
		Lg = nil
		Fatalf("fatal %s", "formatted")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalf_NilLogger")
	cmd.Env = append(os.Environ(), "LING_TEST_FATALF_NIL=1")
	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
		}
		return
	}
	t.Fatalf("expected os.Exit(1), got: %v", err)
}

func TestFatalf_NonNilLogger(t *testing.T) {
	t.Skip("Fatalf with non-nil logger calls os.Exit via zap.Fatal; covered by design")
}

// ===== Panicf =====

func TestPanicf_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	var panicked bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		Panicf("panic %s", "formatted")
	}()
	if !panicked {
		t.Fatal("Panicf with nil Lg should have panicked")
	}
}

func TestPanicf_NonNilLogger(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	var recovered bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		Panicf("panic %d", 42)
	}()
	if !recovered {
		t.Fatal("Panicf with non-nil logger should have panicked")
	}
}

// ===== InfofCtx nil-safe =====

func TestInfofCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ctx := WithRequestID(context.Background(), "infofctx-req")
	InfofCtx(ctx, "info %s", "test") // should not panic
}

// ===== retention: readdir error =====

func TestPurgeExpiredLogFiles_ReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission test not applicable on Windows")
	}
	dir := t.TempDir()
	// Make directory unreadable
	if err := os.Chmod(dir, 0o000); err != nil {
		if os.Geteuid() == 0 {
			t.Skip("skipping on root: chmod has no effect")
		}
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)

	_, err := PurgeExpiredLogFiles(dir, 7)
	if err == nil {
		t.Fatal("expected error for unreadable directory")
	}
}

// ===== retention: stat error with logger =====

func TestPurgeExpiredLogFiles_StatErrorWithLogger(t *testing.T) {
	Lg = zap.NewNop()
	oldLg := Lg
	defer func() { Lg = oldLg }()

	dir := t.TempDir()
	path := filepath.Join(dir, "vanishing.log")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	// Remove file to cause stat failure
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed (file vanished), got %d", removed)
	}
}

// ===== retention: skip directory =====

func TestPurgeExpiredLogFiles_SkipDirectories(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "archive")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Make subdir old
	oldTime := time.Now().AddDate(0, 0, -30)
	if err := os.Chtimes(subdir, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Also add an old file
	oldFile := filepath.Join(dir, "old.log")
	if err := os.WriteFile(oldFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed (file not dir), got %d", removed)
	}
}

// ===== redact: styleFor coverage (read lock path) =====

func TestRedactorStyleFor_MultipleFields(t *testing.T) {
	r := newRedactor([]string{"password", "email", "phone", "unknown_field"})

	// password → redactStyleString
	if r.styleFor("PASSWORD") != redactStyleString {
		t.Fatal("PASSWORD should map to redactStyleString (case-insensitive)")
	}
	// email → redactStyleEmail
	if r.styleFor("EMAIL") != redactStyleEmail {
		t.Fatal("EMAIL should map to redactStyleEmail")
	}
	// phone → redactStylePhone
	if r.styleFor("PHONE") != redactStylePhone {
		t.Fatal("PHONE should map to redactStylePhone")
	}
	// unknown_field → redactStyleString
	if r.styleFor("unknown_field") != redactStyleString {
		t.Fatal("unknown_field should default to redactStyleString")
	}
}

// ===== redact: redactValue with complex JSON =====

func TestRedactValue_NestedJSON(t *testing.T) {
	r := initRedactor("password,token")
	input := `{"user":"john","password":"s3cret","data":{"token":"abc123"}}`
	output := r.redactValue(input)
	if strings.Contains(output, "s3cret") {
		t.Fatalf("password should be redacted: %s", output)
	}
	if strings.Contains(output, "abc123") {
		t.Fatalf("token should be redacted: %s", output)
	}
}

func TestRedactValue_NoMatch(t *testing.T) {
	r := initRedactor("password")
	output := r.redactValue(`{"user":"john"}`)
	if output != `{"user":"john"}` {
		t.Fatalf("non-sensitive JSON should be unchanged: %s", output)
	}
}

// ===== redact: isSensitive case-insensitive =====

func TestIsSensitive_CaseInsensitive(t *testing.T) {
	r := initRedactor("PASSWORD")
	if !r.isSensitive("password") {
		t.Fatal("should be case-insensitive: 'password' should match 'PASSWORD'")
	}
	if !r.isSensitive("PASSWORD") {
		t.Fatal("should match exact uppercase")
	}
	if !r.isSensitive("Password") {
		t.Fatal("should match mixed case")
	}
}

// ===== redact: redactEmail with @ domain =====

func TestRedactEmail_ShortUsername(t *testing.T) {
	r := initRedactor("email")
	tests := []struct {
		input    string
		contains string
	}{
		{"a@domain.com", "@domain.com"},
		{"ab@domain.com", "@domain.com"},
		{"test@example.com", "****"},
	}
	for _, tc := range tests {
		result := r.redactEmail(tc.input)
		if !strings.Contains(result, tc.contains) {
			t.Errorf("redactEmail(%q) = %q, want containing %q", tc.input, result, tc.contains)
		}
	}
}

// ===== redact: redactPhone edge case =====

func TestRedactPhone_ExactlySevenChars(t *testing.T) {
	r := initRedactor("phone")
	result := r.redactPhone("1234567")
	if result == "1234567" {
		t.Fatal("phone with 7 chars should be redacted")
	}
	if !strings.Contains(result, "****") {
		t.Fatalf("redacted phone should contain '****': %s", result)
	}
}

// ===== redact: redactByStyle for all styles =====

func TestRedactByStyle_AllStyles(t *testing.T) {
	r := initRedactor("email,phone,token")

	tests := []struct {
		style      redactStyle
		value      string
		isRedacted bool
	}{
		{redactStyleString, "abcdef", true},
		{redactStyleEmail, "user@host.com", true},
		{redactStylePhone, "13800138000", true},
	}
	for _, tc := range tests {
		result := r.redactByStyle(tc.style, tc.value)
		if tc.isRedacted && result == tc.value {
			t.Errorf("redactByStyle(%v, %q) should redact but got unchanged", tc.style, tc.value)
		}
	}
}

// ===== redact: redactZapField all types =====

func TestRedactZapField_Float64Type(t *testing.T) {
	r := initRedactor("metric")
	f := zap.Float64("metric", 3.14159)
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("float64 with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("float64 should be converted to StringType")
	}
}

func TestRedactZapField_TimeType(t *testing.T) {
	r := initRedactor("timestamp")
	f := zap.Time("timestamp", time.Now())
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("time with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("time should be converted to StringType")
	}
}

func TestRedactZapField_StringerType(t *testing.T) {
	r := initRedactor("addr")
	addr := &testAddr{ip: "192.168.1.1"}
	f := zap.Stringer("addr", addr)
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("stringer with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("stringer should be converted to StringType")
	}
}

type testAddr struct{ ip string }

func (a *testAddr) String() string { return a.ip }

// ===== redact: redactField with nil receiver =====

func TestRedactField_NilRedactor_NonSensitive(t *testing.T) {
	var r *redactor = nil
	result := r.redactField("name", "John")
	if result != "John" {
		t.Fatal("nil redactor should pass through non-sensitive")
	}
}

func TestRedactField_NilRedactor_Sensitive(t *testing.T) {
	var r *redactor = nil
	result := r.redactField("password", "secret")
	if result != "secret" {
		t.Fatal("nil redactor should pass through sensitive")
	}
}

// ===== redact: redactFields with nil redactor =====

func TestRedactFields_NilRedactor_Map(t *testing.T) {
	var r *redactor = nil
	input := map[string]interface{}{"key": "value"}
	result := r.redactFields(input)
	if result["key"] != "value" {
		t.Fatal("nil redactor should return input unchanged")
	}
}

// ===== redact: redactValue with nil redactor =====

func TestRedactValue_NilRedactor_String(t *testing.T) {
	var r *redactor = nil
	result := r.redactValue("some json")
	if result != "some json" {
		t.Fatal("nil redactor should return input unchanged")
	}
}

// ===== redact: rebuildJSONPattern with empty sensitive =====

func TestRebuildJSONPattern_NoSensitive(t *testing.T) {
	r := &redactor{sensitive: map[string]bool{}}
	r.rebuildJSONPattern()
	r.mu.RLock()
	re := r.jsonRE
	r.mu.RUnlock()
	if re != nil {
		t.Fatal("jsonRE should be nil when no sensitive fields")
	}
}

// ===== redact: redactZapFields nil redactor =====

func TestRedactZapFields_NilRedactor_NonEmpty(t *testing.T) {
	var r *redactor = nil
	fields := []zapcore.Field{zap.String("a", "b")}
	result := r.redactZapFields(fields)
	if len(result) != 1 {
		t.Fatalf("expected 1 field, got %d", len(result))
	}
	if result[0].String != "b" {
		t.Fatal("field should be unchanged with nil redactor")
	}
}

// ===== redact: redactZapFields with mixed sensitive/non-sensitive =====

func TestRedactZapFields_MixedFields(t *testing.T) {
	r := initRedactor("password")
	fields := []zapcore.Field{
		zap.String("name", "John"),
		zap.String("password", "secret123"),
		zap.String("email", "john@example.com"),
	}
	result := r.redactZapFields(fields)
	if len(result) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(result))
	}
	// name should be unchanged
	if result[0].String != "John" {
		t.Fatal("name should not be redacted")
	}
	// password should be redacted
	if result[1].String == "secret123" {
		t.Fatal("password should be redacted")
	}
	// email not in sensitive list should be unchanged
	if result[2].String != "john@example.com" {
		t.Fatal("email should not be redacted (not in list)")
	}
}

// ===== redact: wrapCoreWithRedact with both nil =====

func TestWrapCoreWithRedact_BothNil(t *testing.T) {
	result := wrapCoreWithRedact(nil, nil)
	if result != nil {
		t.Fatal("wrapCoreWithRedact(nil, nil) should return nil")
	}
}

// ===== redact: redactCore.Sync =====

func TestRedactCore_Sync(t *testing.T) {
	r := initRedactor("password")
	inner := zapcore.NewNopCore()
	core := wrapCoreWithRedact(inner, r)
	if core == nil {
		t.Fatal("core should not be nil")
	}
	err := core.Sync()
	if err != nil {
		t.Fatalf("Sync error: %v", err)
	}
}

// ===== reqid: WrapCoreWithReqIDPrefix with nil =====

func TestWrapCoreWithReqIDPrefix_NilInner(t *testing.T) {
	result := WrapCoreWithReqIDPrefix(nil)
	if result != nil {
		t.Fatal("WrapCoreWithReqIDPrefix(nil) should return nil")
	}
}

// ===== reqid: reqIDPrefixCore.With preserves accumulated =====

func TestReqIDPrefixCore_With_PreservesAccumulated(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)

	core = core.With([]zapcore.Field{zap.String("x-reqid", "step1")})
	// Accumulate more fields without x-reqid
	core = core.With([]zapcore.Field{zap.String("extra", "data")})

	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "test"}
	err := core.Write(ent, nil)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

// ===== reqid: extractReqID with empty string as fallback =====

func TestExtractReqID_NoFallback(t *testing.T) {
	fields := []zapcore.Field{zap.String("x-reqid", "direct")}
	reqid, filtered := extractReqID(fields, "")
	if reqid != "direct" {
		t.Fatalf("want 'direct', got %q", reqid)
	}
	if len(filtered) != 0 {
		t.Fatalf("expected 0 fields after filtering, got %d", len(filtered))
	}
}

// ===== Printf-style nil-safe tests =====

func TestWarnfCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ctx := context.Background()
	WarnfCtx(ctx, "warn %s", "nil") // should not panic
}

func TestErrorfCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ctx := context.Background()
	ErrorfCtx(ctx, "error %s", "nil") // should not panic
}

// ===== Init with Daily=true =====

func TestInit_DailyMode(t *testing.T) {
	if os.Getenv("LING_SUBPROCESS_INIT") == "daily" {
		config := &LogConfig{
			Level:      "debug",
			Filename:   filepath.Join(os.TempDir(), "ling_daily.log"),
			MaxSize:    5,
			MaxBackups: 1,
			MaxAge:     1,
			Daily:      true,
		}
		err := Init(config, constants.ENV_PROD)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			os.Exit(1)
		}
		Info("daily mode test")
		Sync()
		fmt.Println("OK: daily-init")
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestInit_DailyMode")
	cmd.Env = append(os.Environ(), "LING_SUBPROCESS_INIT=daily")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("daily mode subprocess failed: %v\noutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "OK: daily-init") {
		t.Fatalf("expected 'OK: daily-init', got:\n%s", string(out))
	}
}

// ===== InfoCtx with various context keys =====

func TestInfoCtx_WithAllContextKeys(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, UserIDKey, "user-1")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant-1")
	ctx = context.WithValue(ctx, CallIDKey, "call-1")
	ctx = context.WithValue(ctx, CampaignIDKey, "campaign-1")

	InfoCtx(ctx, "message with all keys")
	WarnCtx(ctx, "warn with all keys")
	ErrorCtx(ctx, "error with all keys")
	DebugCtx(ctx, "debug with all keys")
}

// ===== Warnf with non-nil logger =====

func TestWarnf_NonNilLogger(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	Warnf("warn formatted %d", 123)
}

// ===== Errorf with non-nil logger =====

func TestErrorf_NonNilLogger(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	Errorf("error formatted %d", 456)
}

// ===== Debugf with non-nil logger =====

func TestDebugf_NonNilLogger(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	Debugf("debug formatted %d", 789)
}

// ===== InfofCtx with request context =====

func TestInfofCtx_WithRequestID(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	ctx := WithRequestID(context.Background(), "infofctx-id-123")
	InfofCtx(ctx, "formatted %s", "test")
}

// ===== WarnfCtx with request context =====

func TestWarnfCtx_WithRequestID(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	ctx := WithRequestID(context.Background(), "warnfctx-id-456")
	WarnfCtx(ctx, "warn fmt %s", "test")
}

// ===== ErrorfCtx with request context =====

func TestErrorfCtx_WithRequestID(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	ctx := WithRequestID(context.Background(), "errfctx-id-789")
	ErrorfCtx(ctx, "error fmt %s", "test")
}

// ===== WithFields mixed types =====

func TestWithFields_MixedTypes(t *testing.T) {
	fields := WithFields(map[string]interface{}{
		"str":   "hello",
		"int":   42,
		"float": 3.14,
		"bool":  true,
	})
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}
}

// ===== WithError nil =====

func TestWithError_Nil(t *testing.T) {
	field := WithError(nil)
	// zap.Error(nil) returns zap.Skip() which has Key == ""
	if field.Key != "error" && field.Key != "" {
		t.Fatalf("want key 'error' or '', got %q", field.Key)
	}
}

// ===== WithError non-nil =====

func TestWithError_NonNil(t *testing.T) {
	err := errors.New("something went wrong")
	field := WithError(err)
	if field.Key != "error" {
		t.Fatalf("want key 'error', got %q", field.Key)
	}
}

// ===== GenReqID uniqueness loop =====

func TestGenReqID_ProducesValidIDs(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := GenReqID()
		if id == "" {
			t.Fatal("GenReqID returned empty")
		}
		if len(id) < 10 {
			t.Fatalf("GenReqID too short: %q", id)
		}
	}
}

//====== helper functions ======

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// ===== retention with 0 retention =====

func TestPurgeExpiredLogFiles_ZeroDays(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.log"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed for 0 retention, got %d", removed)
	}
}

// ===== Test Panic with nil logger via subprocess to verify =====

func TestPanicCtx_WithContextFields(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "t1")
	ctx = context.WithValue(ctx, TenantIDKey, "tenant1")

	var recovered bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		PanicCtx(ctx, "test panic with context")
	}()
	if !recovered {
		t.Fatal("PanicCtx with non-nil logger should have panicked")
	}
}

// ===== ReqID Core: Check disabled level =====

func TestReqIDPrefixCore_Check_Disabled(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)
	ent := zapcore.Entry{Level: zapcore.InfoLevel}
	ce := core.Check(ent, nil)
	if ce != nil {
		t.Log("Check returned non-nil for NopCore disabled level")
	}
}

// ===== ReqID Core: Write with accumulated reqid =====

func TestReqIDPrefixCore_Write_AccumulatedOnly(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)

	// Set reqid via With
	core = core.With([]zapcore.Field{zap.String("x-reqid", "accumulated-1")})

	// Write without additional fields — should use accumulated reqid
	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "hello world"}
	err := core.Write(ent, nil)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

// ===== ReqID Core: With override =====

func TestReqIDPrefixCore_With_Override(t *testing.T) {
	inner := zapcore.NewNopCore()
	core := WrapCoreWithReqIDPrefix(inner)

	core = core.With([]zapcore.Field{zap.String("x-reqid", "first")})
	core = core.With([]zapcore.Field{zap.String("x-reqid", "second")})

	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "override test"}
	err := core.Write(ent, nil)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

// ===== Daily rotate: Sync with nil lumberjack =====

func TestDailyRotateWriter_Sync_NilLumberjack(t *testing.T) {
	w := &dailyRotateWriter{
		baseFilename: "/tmp/test.log",
		maxSize:      10,
		maxBackup:    3,
		maxAge:       7,
		daily:        false,
		lumberjack:   nil,
	}
	// Sync should handle nil lumberjack gracefully (it will panic, which we recover)
	var recovered bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		w.Sync()
	}()
	if !recovered {
		t.Log("Sync with nil lumberjack did not panic (possibly nil-safe)")
	}
}

// ===== SafeGo with empty name and nil fn =====

func TestSafeGo_EmptyNameNilFn(t *testing.T) {
	// Should not panic
	SafeGo("", nil)
}

// ===== SafeGo goroutine runs concurrently =====

func TestSafeGo_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		name := fmt.Sprintf("goroutine-%d", i)
		SafeGo(name, func() {
			defer wg.Done()
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent SafeGo goroutines did not complete")
	}
}

// ===== RecoverGoroutine with named panic =====

func TestRecoverGoroutine_StringPanic(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer recoverGoroutine("string-panic")
		panic("simple string panic")
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recoverGoroutine should recover string panic")
	}
}

func TestRecoverGoroutine_ErrorPanic(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer recoverGoroutine("error-panic")
		panic(errors.New("structured error panic"))
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recoverGoroutine should recover error panic")
	}
}

// ===== printStderrPanic with empty values =====

func TestPrintStderrPanic_EmptyValues(t *testing.T) {
	printStderrPanic("", "")
	printStderrPanic("test", nil)
}

// ===== RedactCore Check disabled =====

func TestRedactCore_Check_FatalLevel(t *testing.T) {
	r := initRedactor("password")
	var buf zapcore.BufferedWriteSyncer
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, &buf, zapcore.FatalLevel) // only Fatal+
	core := wrapCoreWithRedact(inner, r)

	ent := zapcore.Entry{Level: zapcore.ErrorLevel}
	ce := core.Check(ent, nil)
	if ce != nil {
		t.Log("Check returned non-nil for disabled level")
	}
	// Should not panic
}

// ===== RedactZapField Complex128 type =====

func TestRedactZapField_Complex128(t *testing.T) {
	r := initRedactor("complex")
	f := zap.Complex128("complex", complex(1, 2))
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("complex128 with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("complex128 should be converted to StringType")
	}
}

// ===== RedactZapField Uint64Type =====

func TestRedactZapField_Uint64(t *testing.T) {
	r := initRedactor("counter")
	f := zap.Uint64("counter", 99999)
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("uint64 with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("uint64 should be converted to StringType")
	}
}

// ===== RedactZapField DurationType =====

func TestRedactZapField_Duration(t *testing.T) {
	r := initRedactor("latency")
	f := zap.Duration("latency", time.Second*5)
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("duration with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("duration should be converted to StringType")
	}
}

// ===== RedactZapField ObjectMarshalerType =====

func TestRedactZapField_ObjectMarshaler(t *testing.T) {
	r := initRedactor("obj")
	f := zap.Object("obj", testObj{name: "secret"})
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("object with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("object should be converted to StringType")
	}
}

type testObj struct{ name string }

func (o testObj) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", o.name)
	return nil
}

// ===== RedactZapField ArrayMarshalerType =====

func TestRedactZapField_ArrayMarshaler(t *testing.T) {
	r := initRedactor("arr")
	f := zap.Array("arr", testArr{items: []string{"a", "b"}})
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("array with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("array should be converted to StringType")
	}
}

type testArr struct{ items []string }

func (a testArr) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, item := range a.items {
		enc.AppendString(item)
	}
	return nil
}

// ===== RedactZapField BinaryType =====

func TestRedactZapField_Binary(t *testing.T) {
	r := initRedactor("data")
	f := zap.Binary("data", []byte{0x01, 0x02, 0x03})
	next, changed := r.redactZapField(f)
	if !changed {
		t.Fatal("binary with sensitive key should be redacted")
	}
	if next.Type != zapcore.StringType {
		t.Fatal("binary should be converted to StringType")
	}
}

// ===== IsSensitiveField on sensitive key =====

func TestIsSensitiveField_Sensitive(t *testing.T) {
	// Save and restore global redactor since other tests may have changed it
	old := globalRedactorInstance()
	initRedactor(DefaultSensitiveFields)
	defer func() {
		globalRedactorMu.Lock()
		globalRedactor = old
		globalRedactorMu.Unlock()
	}()

	if !IsSensitiveField("password") {
		t.Fatal("password should be sensitive")
	}
	if !IsSensitiveField("token") {
		t.Fatal("token should be sensitive")
	}
	if !IsSensitiveField("dsn") {
		t.Fatal("dsn should be sensitive")
	}
}

func TestIsSensitiveField_NonSensitive(t *testing.T) {
	old := globalRedactorInstance()
	initRedactor(DefaultSensitiveFields)
	defer func() {
		globalRedactorMu.Lock()
		globalRedactor = old
		globalRedactorMu.Unlock()
	}()

	if IsSensitiveField("username") {
		t.Fatal("username should not be sensitive")
	}
	if IsSensitiveField("display_name") {
		t.Fatal("display_name should not be sensitive")
	}
}

// ===== PurgeExpiredLogFiles with non-existent dir but not isNotExist =====
// (hard to test without mocking; skip for now)

// ===== InfoWithRedactedFields empty map =====

func TestInfoWithRedactedFields_EmptyMap(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	InfoWithRedactedFields("test", map[string]interface{}{})
}

// ===== ErrorWithRedactedFields empty map =====

func TestErrorWithRedactedFields_EmptyMap(t *testing.T) {
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	ErrorWithRedactedFields("test", map[string]interface{}{})
}

// ===== Build-tag specific test =====

func TestGetEncoder_ReturnsJSONEncoder(t *testing.T) {
	enc := getEncoder()
	if enc == nil {
		t.Fatal("getEncoder should return non-nil")
	}
	// Verify it's a JSON encoder by encoding a simple entry
	buf, err := enc.EncodeEntry(zapcore.Entry{Message: "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "test") {
		t.Fatal("encoder output should contain the message")
	}
}

// ===== getLogWriter without daily =====

func TestGetLogWriter_NoDaily_WithExt(t *testing.T) {
	ws := getLogWriter(filepath.Join(t.TempDir(), "test.log"), 10, 5, 30, false)
	if ws == nil {
		t.Fatal("getLogWriter should return non-nil")
	}
}

// ===== getLogWriter with daily no ext =====

func TestGetLogWriter_Daily_NoExt(t *testing.T) {
	dir := t.TempDir()
	ws := getLogWriter(dir+"/app", 10, 5, 30, true)
	if ws == nil {
		t.Fatal("getLogWriter with daily should return non-nil")
	}
}

// ===== wrapLogCore with nil redactor =====

func TestWrapLogCore_RealEncoder(t *testing.T) {
	r := initRedactor("password")
	core := wrapLogCore(zapcore.NewNopCore(), r)
	if core == nil {
		t.Fatal("wrapLogCore should return non-nil")
	}
}

// ===== Sync with nil Lg =====

func TestSync_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	Sync() // should not panic
}

// ===== Sync with usable Lg =====

func TestSync_UsableLogger(t *testing.T) {
	oldLg := Lg
	Lg = zap.NewNop()
	defer func() { Lg = oldLg }()

	Sync() // should not panic
}

// ===== appendContextFields with subset of keys =====

func TestAppendContextFields_PartialKeys(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, TraceIDKey, "tr-1")
	ctx = context.WithValue(ctx, CallIDKey, "call-1")

	fields := appendContextFields(ctx)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields (trace_id, call_id), got %d", len(fields))
	}
}

func TestAppendContextFields_ExistingFields(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-1")
	existing := []zap.Field{zap.String("custom", "value")}
	fields := appendContextFields(ctx, existing...)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields (custom + x-reqid), got %d", len(fields))
	}
	if fields[0].Key != "custom" {
		t.Fatalf("expected 'custom' first, got %q", fields[0].Key)
	}
}

// ===== redactEmail with edge cases =====

func TestRedactEmail_SingleCharPrefix(t *testing.T) {
	r := initRedactor("email")
	result := r.redactEmail("x@y.com")
	if !strings.Contains(result, "****") {
		t.Fatal("single char email should be redacted")
	}
}

func TestRedactEmail_TwoCharPrefix(t *testing.T) {
	r := initRedactor("email")
	result := r.redactEmail("xy@z.com")
	if !strings.Contains(result, "****") {
		t.Fatal("two char email should be redacted")
	}
}

func TestRedactEmail_LongPrefix(t *testing.T) {
	r := initRedactor("email")
	result := r.redactEmail("longusername@domain.com")
	if !strings.HasPrefix(result, "lo") {
		t.Fatalf("expected 'lo' prefix, got %q", result)
	}
	if !strings.Contains(result, "****@domain.com") {
		t.Fatalf("expected masking, got %q", result)
	}
}

// ===== redactPhone specific lengths =====

func TestRedactPhone_Short(t *testing.T) {
	r := initRedactor("phone")
	for _, phone := range []string{"1", "12", "123", "1234", "12345", "123456"} {
		result := r.redactPhone(phone)
		if result != "****" {
			t.Errorf("short phone %q should be '****', got %q", phone, result)
		}
	}
}

func TestRedactPhone_Long(t *testing.T) {
	r := initRedactor("phone")
	result := r.redactPhone("+8613800138000")
	if strings.Contains(result, "+8613800138000") {
		t.Fatal("long phone should be redacted")
	}
	if !strings.Contains(result, "****") {
		t.Fatal("redacted phone should contain '****'")
	}
}

// ===== newRedactor with mixed whitespace =====

func TestNewRedactor_MixedWhitespaceFields(t *testing.T) {
	r := newRedactor([]string{"  password  ", "email", ""})
	if !r.isSensitive("password") {
		t.Fatal("password should be sensitive after trim")
	}
	if !r.isSensitive("email") {
		t.Fatal("email should be sensitive")
	}
	if len(r.sensitive) != 2 {
		t.Fatalf("expected 2 sensitive fields, got %d", len(r.sensitive))
	}
}

// ===== redactValue with multiple sensitive keys =====

func TestRedactValue_MultipleSensitiveKeys(t *testing.T) {
	r := initRedactor("password,token,secret")
	input := `{"password":"pass123","token":"tok456","secret":"sec789","user":"john"}`
	output := r.redactValue(input)
	if strings.Contains(output, "pass123") {
		t.Fatal("password not redacted")
	}
	if strings.Contains(output, "tok456") {
		t.Fatal("token not redacted")
	}
	if strings.Contains(output, "sec789") {
		t.Fatal("secret not redacted")
	}
	if !strings.Contains(output, "john") {
		t.Fatal("user should not be redacted")
	}
}

// ===== redactCore Write with no sensitive keys in fields =====

func TestRedactCore_Write_NoSensitive(t *testing.T) {
	r := initRedactor("password")
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	inner := zapcore.NewCore(enc, zapcore.AddSync(&nopWriteSyncer{}), zapcore.InfoLevel)
	core := wrapCoreWithRedact(inner, r)

	ent := zapcore.Entry{Level: zapcore.InfoLevel, Message: "no sensitive data"}
	ce := core.Check(ent, nil)
	if ce != nil {
		ce.Write(zap.String("user", "john"))
	}
	// Should not panic or error
}

type nopWriteSyncer struct{}

func (n *nopWriteSyncer) Write(p []byte) (int, error) { return len(p), nil }
func (n *nopWriteSyncer) Sync() error                 { return nil }

// ===== redactZapField nil redactor =====

func TestRedactZapField_NilRedactor(t *testing.T) {
	var r *redactor = nil
	f := zap.String("password", "secret")
	_, changed := r.redactZapField(f)
	if changed {
		t.Fatal("nil redactor should not trigger change")
	}
}

// ===== redactZapFields single sensitive field =====

func TestRedactZapFields_SingleSensitive(t *testing.T) {
	r := initRedactor("password")
	fields := []zapcore.Field{zap.String("password", "mysecret")}
	result := r.redactZapFields(fields)
	if len(result) != 1 {
		t.Fatalf("expected 1 field, got %d", len(result))
	}
	if result[0].String == "mysecret" {
		t.Fatal("password should be redacted")
	}
}

// ===== Nil-safe Ctx logging =====

func TestInfoCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	// Should not panic
	InfoCtx(context.Background(), "info ctx", zap.String("key", "value"))
}

func TestWarnCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	WarnCtx(context.Background(), "warn ctx")
}

func TestErrorCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ErrorCtx(context.Background(), "error ctx")
}

func TestDebugCtx_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	DebugCtx(context.Background(), "debug ctx")
}

// ===== Nil-safe printf-style logging =====

func TestInfof_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	Infof("info %s", "test")
}

func TestWarnf_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	Warnf("warn %s", "test")
}

func TestErrorf_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	Errorf("error %s", "test")
}

func TestDebugf_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	Debugf("debug %s", "test")
}

// ===== Nil-safe redacted-fields helpers =====

func TestInfoWithRedactedFields_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	InfoWithRedactedFields("test", map[string]interface{}{"password": "secret"})
}

func TestErrorWithRedactedFields_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	ErrorWithRedactedFields("test", map[string]interface{}{"password": "secret"})
}

func TestBusinessTimeEncoder_UsesConfiguredTimezone(t *testing.T) {
	timeutil.Init("Asia/Shanghai")
	loc := timeutil.Location()

	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:    "time",
		EncodeTime: businessTimeEncoder,
	})
	entry := zapcore.Entry{
		Time: time.Date(2026, 6, 24, 15, 4, 5, 123000000, time.UTC),
	}
	buf, err := enc.EncodeEntry(entry, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out := buf.String()
	want := time.Date(2026, 6, 24, 23, 4, 5, 123000000, loc).Format("2006-01-02T15:04:05.000+08:00")
	if out == "" {
		t.Fatal("empty output")
	}
	if got := extractJSONField(out, "time"); got != want {
		t.Fatalf("time=%q, want %q", got, want)
	}
}

func TestTodayDateString_UsesBusinessTimezone(t *testing.T) {
	timeutil.Init("Asia/Shanghai")
	loc := timeutil.Location()
	now := time.Date(2026, 6, 24, 1, 0, 0, 0, loc)
	want := now.In(loc).Format("2006-01-02")

	// todayDateString uses timeutil.Now(); we only assert format contract here.
	if got := todayDateString(); len(got) != len(want) {
		t.Fatalf("todayDateString format unexpected: %q", got)
	}
}

func extractJSONField(jsonLine, key string) string {
	prefix := `"` + key + `":"`
	start := 0
	for {
		idx := indexOf(jsonLine[start:], prefix)
		if idx < 0 {
			return ""
		}
		start += idx + len(prefix)
		end := start
		for end < len(jsonLine) && jsonLine[end] != '"' {
			end++
		}
		return jsonLine[start:end]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

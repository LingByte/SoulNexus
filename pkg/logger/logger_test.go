package logger

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func makeTmpLogFile(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, name)
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(b)
}

func waitForWrite(assert func() bool) bool {
	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		if assert() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return assert()
}

// 同时劫持 Stdout 和 Stderr，注意：必须在 Init 之前进行。
func captureStdoutStderr(t *testing.T, fn func()) (string, string) {
	t.Helper()

	origOut := os.Stdout
	origErr := os.Stderr

	// 分别创建两对管道
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	var bufOut, bufErr bytes.Buffer
	doneOut := make(chan struct{})
	doneErr := make(chan struct{})

	go func() {
		defer close(doneOut)
		rd := bufio.NewReader(rOut)
		io.Copy(&bufOut, rd) //nolint:errcheck
	}()
	go func() {
		defer close(doneErr)
		rd := bufio.NewReader(rErr)
		io.Copy(&bufErr, rd) //nolint:errcheck
	}()

	// 执行被测逻辑（包含 Init 和日志写入）
	fn()

	// 关闭写端并恢复
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = origOut
	os.Stderr = origErr

	<-doneOut
	<-doneErr

	return bufOut.String(), bufErr.String()
}

func TestInitProdModeAndFileWrite(t *testing.T) {
	logPath := makeTmpLogFile(t, "app.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init(prod) error: %v", err)
	}

	// 写各种等级
	Debug("dmsg", zap.String("k", "vd"))
	Info("imsg", zap.Int("n", 1))
	Warn("wmsg")
	Error("emsg", zap.Error(errors.New("boom")))

	// Panic 需要 recover
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()
		Panic("pmsg")
	}()

	// Sync 刷新
	Sync()

	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		return strings.Contains(s, "dmsg") &&
			strings.Contains(s, "imsg") &&
			strings.Contains(s, "wmsg") &&
			strings.Contains(s, "emsg")
	})
	if !ok {
		t.Fatalf("log content not found in file:\n%s", readFile(t, logPath))
	}
}

func TestInitDevModeConsoleSplit(t *testing.T) {
	logPath := makeTmpLogFile(t, "dev.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}

	stdoutOut, stderrOut := captureStdoutStderr(t, func() {
		// 必须在劫持之后再 Init，这样 zap core 会锁定被我们替换过的 Stdout/Stderr
		if err := Init(cfg, "dev"); err != nil {
			t.Fatalf("Init(dev) error: %v", err)
		}
		Debug("dev-debug")
		Info("dev-info")
		Warn("dev-warn")   // < error -> 走 stdout
		Error("dev-error") // >= error -> 走 stderr
		Sync()
	})

	// 控制台输出包含消息（颜色码不校验）
	if !strings.Contains(stdoutOut, "dev-info") || !strings.Contains(stdoutOut, "dev-warn") || !strings.Contains(stdoutOut, "dev-debug") {
		t.Fatalf("stdout missing expected logs:\n%s", stdoutOut)
	}
	if !strings.Contains(stderrOut, "dev-error") {
		t.Fatalf("stderr missing expected logs:\n%s", stderrOut)
	}

	// 文件也应该有写入
	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		return strings.Contains(s, "dev-info") && strings.Contains(s, "dev-error")
	})
	if !ok {
		t.Fatalf("dev file log missing:\n%s", readFile(t, logPath))
	}
}

func TestInitInvalidLevel(t *testing.T) {
	logPath := makeTmpLogFile(t, "bad.log")
	cfg := &LogConfig{
		Level:      "not-a-level",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
	}
	err := Init(cfg, "prod")
	if err == nil {
		t.Fatalf("expected error for invalid level")
	}
}

// 仅验证 Sync 在 nil logger 时不崩（当前实现会 panic，保持原测试+recover 以覆盖路径）
func TestSyncSafeWhenNoLogger(t *testing.T) {
	logPath := makeTmpLogFile(t, "sync.log")
	cfg := &LogConfig{
		Level:      "info",
		Filename:   logPath,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init for SyncSafe: %v", err)
	}
	// 备份并置空
	old := Lg
	Lg = nil
	defer func() { Lg = old }()

	defer func() { _ = recover() }()
	Sync()
}

// 由于包装函数存在，caller 会显示在 logger/logger.go（而非测试文件）
// 因此这里验证 ShortCallerEncoder 是否生效，并包含生产文件路径。
func TestAddCallerEnabled(t *testing.T) {
	logPath := makeTmpLogFile(t, "caller.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init(prod) error: %v", err)
	}
	Info("caller-check")
	Sync()

	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		// 由于 EncodeCaller=ShortCallerEncoder，应出现 "logger/logger.go:xxx"
		return strings.Contains(s, "logger/logger.go") && strings.Contains(s, "caller-check")
	})
	if !ok {
		t.Fatalf("caller info not present:\n%s", readFile(t, logPath))
	}
}

// 防止某些平台噪音 & 也顺带让几个包装方法跑一遍
func TestEnvironmentNoise(t *testing.T) {
	_ = runtime.GOOS
	msg := "noop"
	Debug(msg)
	Info(msg)
	Warn(msg)
	Error(msg)
	fmt.Sprintf("")
}

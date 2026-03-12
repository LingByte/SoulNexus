package logger

import (
	"strings"
	"testing"
)

func TestExtraLogFunctions(t *testing.T) {
	// 初始化一个临时日志文件
	logPath := makeTmpLogFile(t, "extra.log")
	cfg := &LogConfig{
		Level:      "debug",
		Filename:   logPath,
		MaxSize:    5,
		MaxAge:     1,
		MaxBackups: 1,
	}
	if err := Init(cfg, "prod"); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	// 调用每个额外的日志函数
	LogServerConfig("127.0.0.1:8080", "mysql", "root:123@tcp(localhost:3306)/db",
		"release", "info", "app.log", 10, 7, 3)
	LogStartupSuccess("127.0.0.1:8080")
	LogConfigLoaded("/etc/app/config.yaml")
	LogError("something went wrong")
	LogAccess("GET", "/ping", "127.0.0.1", 200, 123)
	LogDatabaseConnected("mysql", "root:123@tcp(localhost:3306)/db")
	LogTaskStarted("cron-job")

	// 刷新日志
	Sync()

	// 等待文件写入并校验内容
	ok := waitForWrite(func() bool {
		s := readFile(t, logPath)
		return strings.Contains(s, "Server configuration") &&
			strings.Contains(s, "Server started successfully") &&
			strings.Contains(s, "Configuration loaded successfully") &&
			strings.Contains(s, "something went wrong") &&
			strings.Contains(s, "HTTP access log") &&
			strings.Contains(s, "Database connected successfully") &&
			strings.Contains(s, "Background task started")
	})
	if !ok {
		t.Fatalf("expected log entries not found in:\n%s", readFile(t, logPath))
	}
}

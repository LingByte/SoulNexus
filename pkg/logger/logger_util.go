package logger

import (
	"go.uber.org/zap"
)

// LogServerConfig 记录服务器配置日志
func LogServerConfig(addr, dbDriver, dsn, mode, logLevel, logFilename string, logMaxSize, logMaxAge, logMaxBackups int) {
	zap.L().Info("Server configuration",
		zap.String("addr", addr),
		zap.String("db_driver", dbDriver),
		zap.String("dns", dsn),
		zap.String("mode", mode),
		zap.String("log_level", logLevel),
		zap.String("log_filename", logFilename),
		zap.Int("log_max_size", logMaxSize),
		zap.Int("log_max_age", logMaxAge),
		zap.Int("log_max_backups", logMaxBackups),
	)
}

// LogStartupSuccess 记录服务启动成功日志
func LogStartupSuccess(addr string) {
	zap.L().Info("Server started successfully",
		zap.String("listen_addr", addr),
		zap.String("status", "running"),
	)
}

// LogConfigLoaded 记录配置加载完成日志
func LogConfigLoaded(configPath string) {
	zap.L().Info("Configuration loaded successfully",
		zap.String("config_path", configPath),
	)
}

// LogError 记录错误日志（自动包含调用堆栈）
func LogError(msg string, fields ...zap.Field) {
	zap.L().Error(msg, fields...)
}

// LogAccess 记录 HTTP 请求访问日志
func LogAccess(method, path, clientIP string, statusCode int, latency int64) {
	zap.L().Info("HTTP access log",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("client_ip", clientIP),
		zap.Int("status_code", statusCode),
		zap.Int64("latency_ms", latency),
	)
}

// LogDatabaseConnected 记录数据库连接成功日志
func LogDatabaseConnected(driver, dsn string) {
	zap.L().Info("Database connected successfully",
		zap.String("driver", driver),
		zap.String("dsn", dsn),
	)
}

// LogTaskStarted 记录定时任务启动日志
func LogTaskStarted(taskName string) {
	zap.L().Info("Background task started",
		zap.String("task_name", taskName),
	)
}

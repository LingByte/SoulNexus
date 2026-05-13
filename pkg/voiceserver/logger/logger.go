package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Daily      bool   `mapstructure:"daily"`
}

var (
	Lg          *zap.Logger
	once        sync.Once
	cfg         *LogConfig
	currentDate string
)

// Init 初始化lg，使用 sync.Once 确保只初始化一次
func Init(config *LogConfig, mode string) (err error) {
	once.Do(func() {
		cfg = config
		currentDate = time.Now().Format("2006-01-02")

		writeSyncer := getLogWriter(cfg.Filename, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge, cfg.Daily)
		encoder := getEncoder()
		var l = new(zapcore.Level)
		err = l.UnmarshalText([]byte(cfg.Level))
		if err != nil {
			return
		}
		var core zapcore.Core
		if mode == "local" || mode == "dev" || mode == "development" {
			// 进入开发模式，日志输出到终端，启用带色彩的编码器
			consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
			consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 启用色彩编码
			consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			consoleEncoderConfig.TimeKey = "time"
			consoleEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

			// 根据不同环境设置不同的颜色方案
			if mode == "local" {
				// local 环境：全部绿色
				consoleEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[32m" + t.Format("2006-01-02 15:04:05.000") + "\x1b[0m")
				}
				consoleEncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[32m" + "[" + l.CapitalString() + "]\x1b[0m")
				}
				consoleEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[32m" + caller.TrimmedPath() + "\x1b[0m")
				}
			} else {
				// dev/development 环境：多色（保持原有颜色）
				consoleEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[90m" + t.Format("2006-01-02 15:04:05.000") + "\x1b[0m")
				}
				consoleEncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
					var levelColor = map[zapcore.Level]string{
						zapcore.DebugLevel:  "\x1b[35m", // 紫色
						zapcore.InfoLevel:   "\x1b[36m", // 青色
						zapcore.WarnLevel:   "\x1b[33m", // 黄色
						zapcore.ErrorLevel:  "\x1b[31m", // 红色
						zapcore.DPanicLevel: "\x1b[31m", // 红色
						zapcore.PanicLevel:  "\x1b[31m", // 红色
						zapcore.FatalLevel:  "\x1b[31m", // 红色
					}
					color, ok := levelColor[l]
					if !ok {
						color = "\x1b[0m" // 默认颜色
					}
					enc.AppendString(color + "[" + l.CapitalString() + "]\x1b[0m")
				}
				consoleEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[90m" + caller.TrimmedPath() + "\x1b[0m")
				}
			}
			consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

			// 为不同日志级别设置不同的颜色以增强可读性
			highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= zapcore.ErrorLevel
			})
			lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl < zapcore.ErrorLevel
			})

			core = zapcore.NewTee(
				zapcore.NewCore(encoder, writeSyncer, l),
				zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), lowPriority),
				zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stderr), highPriority),
			)
		} else {
			core = zapcore.NewCore(encoder, writeSyncer, l)
		}

		Lg = zap.New(core, zap.AddCaller()) // zap.AddCaller() 添加调用栈信息

		zap.ReplaceGlobals(Lg) // 替换zap包全局的logger

		Info("init logger success")
	})
	return
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

// dailyRotateWriter 自定义写入器，支持按日期自动轮转
type dailyRotateWriter struct {
	baseFilename string
	maxSize      int
	maxBackup    int
	maxAge       int
	daily        bool
	lumberjack   *lumberjack.Logger
	mutex        sync.Mutex
}

func (w *dailyRotateWriter) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.daily {
		today := time.Now().Format("2006-01-02")
		if today != currentDate {
			// 日期变化，需要切换文件
			currentDate = today
			ext := filepath.Ext(w.baseFilename)
			base := w.baseFilename[:len(w.baseFilename)-len(ext)]
			newFilename := base + "-" + today + ext

			// 创建新的 lumberjack logger
			newLogger := &lumberjack.Logger{
				Filename:   newFilename,
				MaxSize:    w.maxSize,
				MaxBackups: w.maxBackup,
				MaxAge:     w.maxAge,
				LocalTime:  true,
			}
			w.lumberjack = newLogger
		}
	}

	return w.lumberjack.Write(p)
}

func (w *dailyRotateWriter) Sync() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.lumberjack.Close()
}

func getLogWriter(filename string, maxSize, maxBackup, maxAge int, daily bool) zapcore.WriteSyncer {
	var actualFilename string
	if daily {
		// 按日期分割日志文件
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		dateStr := time.Now().Format("2006-01-02")
		actualFilename = base + "-" + dateStr + ext
	} else {
		actualFilename = filename
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   actualFilename,
		MaxSize:    maxSize,
		MaxBackups: maxBackup,
		MaxAge:     maxAge,
		LocalTime:  true,
	}

	if daily {
		// 返回自定义的轮转写入器
		return zapcore.AddSync(&dailyRotateWriter{
			baseFilename: filename,
			maxSize:      maxSize,
			maxBackup:    maxBackup,
			maxAge:       maxAge,
			daily:        daily,
			lumberjack:   lumberJackLogger,
		})
	}

	return zapcore.AddSync(lumberJackLogger)
}

// Info 通用 info 日志方法（跳过本包封装帧，caller 指向业务调用处）
func Info(msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// Warn 通用 warn 日志方法
func Warn(msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// Error 通用 error 日志方法
func Error(msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// Debug 通用 debug 日志方法
func Debug(msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// Fatal 通用 fatal 日志方法
func Fatal(msg string, fields ...zap.Field) {
	if Lg == nil {
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

// Panic 通用 panic 日志方法
func Panic(msg string, fields ...zap.Field) {
	if Lg == nil {
		panic(msg)
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Panic(msg, fields...)
}

// Sync 刷新缓冲区
func Sync() {
	if Lg != nil {
		_ = Lg.Sync()
	}
}

// Context keys for extracting values from context
type contextKey string

const (
	TraceIDKey   contextKey = "trace_id"
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
)

// InfoCtx 带 context 的 info 日志方法
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// WarnCtx 带 context 的 warn 日志方法
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// ErrorCtx 带 context 的 error 日志方法
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// DebugCtx 带 context 的 debug 日志方法
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// FatalCtx 带 context 的 fatal 日志方法
func FatalCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

// PanicCtx 带 context 的 panic 日志方法
func PanicCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Panic(msg, fields...)
}

// appendContextFields 从 context 中提取 trace_id, request_id 等字段
func appendContextFields(ctx context.Context, fields ...zap.Field) []zap.Field {
	if ctx == nil {
		return fields
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		fields = append(fields, zap.String(string(TraceIDKey), fmt.Sprintf("%v", traceID)))
	}
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields = append(fields, zap.String(string(RequestIDKey), fmt.Sprintf("%v", requestID)))
	}
	if userID := ctx.Value(UserIDKey); userID != nil {
		fields = append(fields, zap.String(string(UserIDKey), fmt.Sprintf("%v", userID)))
	}

	return fields
}

// GetDailyLogFilename 获取按日期分割的日志文件名
func GetDailyLogFilename(baseFilename string) string {
	ext := filepath.Ext(baseFilename)
	base := baseFilename[:len(baseFilename)-len(ext)]
	dateStr := time.Now().Format("2006-01-02")
	return base + "-" + dateStr + ext
}

// Field 结构体，用于 WithField 方法
type Field struct {
	key   string
	value interface{}
}

// WithField 创建单个字段
func WithField(key string, value interface{}) Field {
	return Field{key: key, value: value}
}

// WithFields 创建多个字段
func WithFields(fields map[string]interface{}) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return zapFields
}

// WithError 创建错误字段
func WithError(err error) zap.Field {
	return zap.Error(err)
}

// Infof 使用 fmt.Sprintf 风格的 info 日志
func Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Lg.Info(msg)
}

// Warnf 使用 fmt.Sprintf 风格的 warn 日志
func Warnf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Lg.Warn(msg)
}

// Errorf 使用 fmt.Sprintf 风格的 error 日志
func Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Lg.Error(msg)
}

// Debugf 使用 fmt.Sprintf 风格的 debug 日志
func Debugf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Lg.Debug(msg)
}

// Fatalf 使用 fmt.Sprintf 风格的 fatal 日志
func Fatalf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Lg.Fatal(msg)
}

// Panicf 使用 fmt.Sprintf 风格的 panic 日志
func Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	Lg.Panic(msg)
}

// InfoWithFields 使用 map 形式的字段记录 info 日志
func InfoWithFields(msg string, fields map[string]interface{}) {
	Lg.Info(msg, WithFields(fields)...)
}

// WarnWithFields 使用 map 形式的字段记录 warn 日志
func WarnWithFields(msg string, fields map[string]interface{}) {
	Lg.Warn(msg, WithFields(fields)...)
}

// ErrorWithFields 使用 map 形式的字段记录 error 日志
func ErrorWithFields(msg string, fields map[string]interface{}) {
	Lg.Error(msg, WithFields(fields)...)
}

// DebugWithFields 使用 map 形式的字段记录 debug 日志
func DebugWithFields(msg string, fields map[string]interface{}) {
	Lg.Debug(msg, WithFields(fields)...)
}

// InfofCtx 带 context 的 printf 风格 info 日志
func InfofCtx(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fields := appendContextFields(ctx, nil...)
	Lg.Info(msg, fields...)
}

// ErrorfCtx 带 context 的 printf 风格 error 日志
func ErrorfCtx(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fields := appendContextFields(ctx, nil...)
	Lg.Error(msg, fields...)
}

// 日志脱敏相关功能

// SensitiveFields 需要脱敏的字段列表
var SensitiveFields = map[string]bool{
	"password":      true,
	"passwd":        true,
	"pwd":           true,
	"secret":        true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"api_key":       true,
	"apikey":        true,
	"authorization": true,
	"credit_card":   true,
	"ssn":           true,
	"phone":         true,
	"email":         true,
}

// MaskString 对字符串进行脱敏处理
func MaskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

// MaskEmail 对邮箱进行脱敏
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return MaskString(email)
	}
	username := parts[0]
	domain := parts[1]
	if len(username) <= 2 {
		return "****@" + domain
	}
	return username[:2] + "****@" + domain
}

// MaskPhone 对手机号进行脱敏
func MaskPhone(phone string) string {
	if len(phone) <= 7 {
		return "****"
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// MaskField 根据字段名自动选择合适的脱敏方式
func MaskField(key string, value interface{}) interface{} {
	keyLower := strings.ToLower(key)

	// 检查是否为敏感字段
	if !SensitiveFields[keyLower] {
		return value
	}

	strValue := fmt.Sprintf("%v", value)

	// 根据字段类型选择脱敏方式
	switch keyLower {
	case "email":
		return MaskEmail(strValue)
	case "phone", "mobile", "telephone":
		return MaskPhone(strValue)
	default:
		return MaskString(strValue)
	}
}

// MaskFields 批量脱敏 map 中的敏感字段
func MaskFields(fields map[string]interface{}) map[string]interface{} {
	masked := make(map[string]interface{})
	for k, v := range fields {
		masked[k] = MaskField(k, v)
	}
	return masked
}

// InfoWithMaskedFields 使用脱敏后的字段记录 info 日志
func InfoWithMaskedFields(msg string, fields map[string]interface{}) {
	Lg.Info(msg, WithFields(MaskFields(fields))...)
}

// ErrorWithMaskedFields 使用脱敏后的字段记录 error 日志
func ErrorWithMaskedFields(msg string, fields map[string]interface{}) {
	Lg.Error(msg, WithFields(MaskFields(fields))...)
}

// AddSensitiveField 添加自定义敏感字段
func AddSensitiveField(field string) {
	SensitiveFields[strings.ToLower(field)] = true
}

// RemoveSensitiveField 移除敏感字段
func RemoveSensitiveField(field string) {
	delete(SensitiveFields, strings.ToLower(field))
}

// MaskValue 对任意值进行正则匹配脱敏（如 JSON 中的敏感字段）
func MaskValue(data string) string {
	// 匹配 JSON 中的敏感字段值
	pattern := `"(password|passwd|pwd|secret|token|api_key|authorization|credit_card|ssn)"\s*:\s*"([^"]*)"`
	re := regexp.MustCompile(pattern)

	result := re.ReplaceAllStringFunc(data, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			key := submatches[1]
			value := submatches[2]
			return fmt.Sprintf(`"%s":"%s"`, key, MaskField(key, value))
		}
		return match
	})

	return result
}

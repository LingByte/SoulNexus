package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const consoleTimeLayout = "2006-01-02 15:04:05.000"

// businessTimeEncoder writes timestamps in SYSTEM_TIMEZONE (see timeutil).
// JSON logs therefore align with console output and business-day rotation.
func businessTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.In(timeutil.Location()).Format("2006-01-02T15:04:05.000Z07:00"))
}

func formatConsoleTime(t time.Time) string {
	return t.In(timeutil.Location()).Format(consoleTimeLayout)
}

func todayDateString() string {
	return timeutil.Now().In(timeutil.Location()).Format("2006-01-02")
}

type LogConfig struct {
	Level           string `mapstructure:"level"`
	Filename        string `mapstructure:"filename"`
	MaxSize         int    `mapstructure:"max_size"`
	MaxAge          int    `mapstructure:"max_age"`        // lumberjack rotation age (days)
	RetentionDays   int    `mapstructure:"retention_days"` // purge files under logs dir (from LOG_RETENTION_DAYS)
	MaxBackups      int    `mapstructure:"max_backups"`
	Daily           bool   `mapstructure:"daily"`
	SensitiveFields string `mapstructure:"sensitive_fields"` // comma-separated; from LOG_SENSITIVE_FIELDS
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
		currentDate = todayDateString()
		redactor := initRedactor(cfg.SensitiveFields)

		writeSyncer := getLogWriter(cfg.Filename, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge, cfg.Daily)
		encoder := getEncoder()
		var l = new(zapcore.Level)
		err = l.UnmarshalText([]byte(cfg.Level))
		if err != nil {
			return
		}
		var core zapcore.Core
		if mode == constants.ENV_PROD || mode == constants.ENV_DEV || mode == constants.ENV_LOCAL {
			consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
			consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			consoleEncoderConfig.EncodeTime = businessTimeEncoder
			consoleEncoderConfig.TimeKey = "time"
			consoleEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
			if mode == constants.ENV_LOCAL {
				// local: 绿色主题，醒目便于开发调试
				consoleEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[32m" + formatConsoleTime(t) + "\x1b[0m")
				}
				consoleEncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[32m" + "[" + l.CapitalString() + "]\x1b[0m")
				}
				consoleEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[32m" + caller.TrimmedPath() + "\x1b[0m")
				}
			} else if mode == constants.ENV_DEV {
				// dev: 蓝色主题，与 local/prod 区分
				consoleEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[34m" + formatConsoleTime(t) + "\x1b[0m")
				}
				consoleEncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
					var levelColor = map[zapcore.Level]string{
						zapcore.DebugLevel:  "\x1b[34m", // 蓝色
						zapcore.InfoLevel:   "\x1b[36m", // 青色
						zapcore.WarnLevel:   "\x1b[33m", // 黄色
						zapcore.ErrorLevel:  "\x1b[31m", // 红色
						zapcore.DPanicLevel: "\x1b[31m", // 红色
						zapcore.PanicLevel:  "\x1b[31m", // 红色
						zapcore.FatalLevel:  "\x1b[31m", // 红色
					}
					color, ok := levelColor[l]
					if !ok {
						color = "\x1b[34m"
					}
					enc.AppendString(color + "[" + l.CapitalString() + "]\x1b[0m")
				}
				consoleEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[34m" + caller.TrimmedPath() + "\x1b[0m")
				}
			} else {
				// prod: 灰色主题，低调专业
				consoleEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[90m" + formatConsoleTime(t) + "\x1b[0m")
				}
				consoleEncoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
					var levelColor = map[zapcore.Level]string{
						zapcore.DebugLevel:  "\x1b[90m", // 灰色
						zapcore.InfoLevel:   "\x1b[37m", // 白色
						zapcore.WarnLevel:   "\x1b[33m", // 黄色
						zapcore.ErrorLevel:  "\x1b[31m", // 红色
						zapcore.DPanicLevel: "\x1b[31m", // 红色
						zapcore.PanicLevel:  "\x1b[31m", // 红色
						zapcore.FatalLevel:  "\x1b[31m", // 红色
					}
					color, ok := levelColor[l]
					if !ok {
						color = "\x1b[37m"
					}
					enc.AppendString(color + "[" + l.CapitalString() + "]\x1b[0m")
				}
				consoleEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString("\x1b[90m" + caller.TrimmedPath() + "\x1b[0m")
				}
			}
			consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
			highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= zapcore.ErrorLevel
			})
			lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl < zapcore.ErrorLevel
			})
			// Wrap each leaf individually rather than the Tee. Wrapping
			// the Tee bypasses per-leaf level filtering: Tee.Write
			// dispatches to all sub-cores unconditionally, so an INFO
			// entry routed through a single outer-wrapper Write call
			// would hit both the stdout-low leaf AND the stderr-high
			// leaf, printing the same line twice in a terminal that
			// merges both streams. Wrapping at the leaf preserves the
			// leaf's LevelEnabler in Check.
			core = zapcore.NewTee(
				wrapLogCore(zapcore.NewCore(encoder, writeSyncer, l), redactor),
				wrapLogCore(zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), lowPriority), redactor),
				wrapLogCore(zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stderr), highPriority), redactor),
			)
		} else {
			core = wrapLogCore(zapcore.NewCore(encoder, writeSyncer, l), redactor)
		}
		Lg = zap.New(core, zap.AddCaller())
		zap.ReplaceGlobals(Lg)
		Info("initialized logger module successful")
	})
	return
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = businessTimeEncoder
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
		today := todayDateString()
		if today != currentDate {
			currentDate = today
			ext := filepath.Ext(w.baseFilename)
			base := w.baseFilename[:len(w.baseFilename)-len(ext)]
			newFilename := base + "-" + today + ext
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
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		dateStr := todayDateString()
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
	TraceIDKey    contextKey = "trace_id"
	RequestIDKey  contextKey = "request_id"
	UserIDKey     contextKey = "user_id"
	XReqIDKey     contextKey = "x-reqid"
	TenantIDKey   contextKey = "tenant_id"
	CallIDKey     contextKey = "call_id"
	CampaignIDKey contextKey = "campaign_id"
)

// InfoCtx 带 context 的 info 日志方法
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

// WarnCtx 带 context 的 warn 日志方法
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

// ErrorCtx 带 context 的 error 日志方法
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

// DebugCtx 带 context 的 debug 日志方法
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	if Lg == nil {
		return
	}
	fields = appendContextFields(ctx, fields...)
	Lg.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

// FatalCtx 带 context 的 fatal 日志方法
func FatalCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	if Lg == nil {
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
	Lg.WithOptions(zap.AddCallerSkip(1)).Fatal(msg, fields...)
}

// PanicCtx 带 context 的 panic 日志方法
func PanicCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = appendContextFields(ctx, fields...)
	if Lg == nil {
		panic(msg)
	}
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
		fields = append(fields, zap.String(string(XReqIDKey), fmt.Sprintf("%v", requestID)))
	}
	if userID := ctx.Value(UserIDKey); userID != nil {
		fields = append(fields, zap.String(string(UserIDKey), fmt.Sprintf("%v", userID)))
	}
	if tenantID := ctx.Value(TenantIDKey); tenantID != nil {
		fields = append(fields, zap.String(string(TenantIDKey), fmt.Sprintf("%v", tenantID)))
	}
	if callID := ctx.Value(CallIDKey); callID != nil {
		fields = append(fields, zap.String(string(CallIDKey), fmt.Sprintf("%v", callID)))
	}
	if campaignID := ctx.Value(CampaignIDKey); campaignID != nil {
		fields = append(fields, zap.String(string(CampaignIDKey), fmt.Sprintf("%v", campaignID)))
	}
	return fields
}

// GetDailyLogFilename 获取按日期分割的日志文件名
func GetDailyLogFilename(baseFilename string) string {
	ext := filepath.Ext(baseFilename)
	base := baseFilename[:len(baseFilename)-len(ext)]
	dateStr := todayDateString()
	return base + "-" + dateStr + ext
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
	Info(fmt.Sprintf(format, args...))
}

// Warnf 使用 fmt.Sprintf 风格的 warn 日志
func Warnf(format string, args ...interface{}) {
	Warn(fmt.Sprintf(format, args...))
}

// Errorf 使用 fmt.Sprintf 风格的 error 日志
func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

// Debugf 使用 fmt.Sprintf 风格的 debug 日志
func Debugf(format string, args ...interface{}) {
	Debug(fmt.Sprintf(format, args...))
}

// Fatalf 使用 fmt.Sprintf 风格的 fatal 日志
func Fatalf(format string, args ...interface{}) {
	Fatal(fmt.Sprintf(format, args...))
}

// Panicf 使用 fmt.Sprintf 风格的 panic 日志
func Panicf(format string, args ...interface{}) {
	Panic(fmt.Sprintf(format, args...))
}

// InfofCtx 带 context 的 printf 风格 info 日志
func InfofCtx(ctx context.Context, format string, args ...interface{}) {
	InfoCtx(ctx, fmt.Sprintf(format, args...))
}

// WarnfCtx 带 context 的 printf 风格 warn 日志
func WarnfCtx(ctx context.Context, format string, args ...interface{}) {
	WarnCtx(ctx, fmt.Sprintf(format, args...))
}

// ErrorfCtx 带 context 的 printf 风格 error 日志
func ErrorfCtx(ctx context.Context, format string, args ...interface{}) {
	ErrorCtx(ctx, fmt.Sprintf(format, args...))
}

func wrapLogCore(inner zapcore.Core, r *redactor) zapcore.Core {
	return WrapCoreWithReqIDPrefix(wrapCoreWithRedact(inner, r))
}

// InfoWithRedactedFields logs info with sensitive fields redacted.
func InfoWithRedactedFields(msg string, fields map[string]interface{}) {
	if Lg == nil {
		return
	}
	Lg.Info(msg, WithFields(RedactFields(fields))...)
}

// ErrorWithRedactedFields logs error with sensitive fields redacted.
func ErrorWithRedactedFields(msg string, fields map[string]interface{}) {
	if Lg == nil {
		return
	}
	Lg.Error(msg, WithFields(RedactFields(fields))...)
}

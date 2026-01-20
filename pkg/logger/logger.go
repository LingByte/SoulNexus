package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogConfig struct {
	Level      string       `mapstructure:"level"`
	Filename   string       `mapstructure:"filename"`
	MaxSize    int          `mapstructure:"max_size"`
	MaxAge     int          `mapstructure:"max_age"`
	MaxBackups int          `mapstructure:"max_backups"`
	Daily      bool         `mapstructure:"daily"`
	Alert      *AlertConfig `mapstructure:"alert"`
}

var (
	Lg           *zap.Logger
	alertManager *AlertManager
)

// Init 初始化lg（兼容旧版本）
func Init(cfg *LogConfig, mode string) (err error) {
	return InitWithAlert(cfg, mode, nil)
}

// InitWithAlert 初始化logger和预警功能
func InitWithAlert(cfg *LogConfig, mode string, notifiers []NotificationService) (err error) {
	writeSyncer := getLogWriter(cfg.Filename, cfg.MaxSize, cfg.MaxBackups, cfg.MaxAge, cfg.Daily)
	encoder := getEncoder()
	var l = new(zapcore.Level)
	err = l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return
	}
	var core zapcore.Core
	if mode == "dev" || mode == "development" {
		// 进入开发模式，日志输出到终端，启用带色彩的编码器
		consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
		consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 启用色彩编码
		consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoderConfig.TimeKey = "time"
		consoleEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		// 修改时间编码器，添加颜色
		consoleEncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("\x1b[90m" + t.Format("2006-01-02 15:04:05.000") + "\x1b[0m")
		}
		// 自定义级别编码器，使用[INFO]格式并添加颜色
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
		// 修改调用者编码器，添加颜色
		consoleEncoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("\x1b[90m" + caller.TrimmedPath() + "\x1b[0m")
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
	// 复习回顾：日志默认输出到app.log，如何将err日志单独在 app.err.log 记录一份

	Lg = zap.New(core, zap.AddCaller()) // zap.AddCaller() 添加调用栈信息

	zap.ReplaceGlobals(Lg) // 替换zap包全局的logger

	// 初始化预警管理器
	if cfg.Alert != nil && cfg.Alert.Enabled && notifiers != nil {
		alertManager = NewAlertManager(cfg.Alert, notifiers...)
		Info("init logger with alert manager success")
	} else {
		Info("init logger success")
	}
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

func getLogWriter(filename string, maxSize, maxBackup, maxAge int, daily bool) zapcore.WriteSyncer {
	if daily {
		// 按日期分割日志文件
		ext := filepath.Ext(filename)
		base := filename[:len(filename)-len(ext)]
		dateStr := time.Now().Format("2006-01-02")
		filename = base + "-" + dateStr + ext
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackup,
		MaxAge:     maxAge,
		LocalTime:  true, // 使用本地时间
	}
	return zapcore.AddSync(lumberJackLogger)
}

// Info 通用 info 日志方法
func Info(msg string, fields ...zap.Field) {
	Lg.Info(msg, fields...)
}

// Warn 通用 warn 日志方法
func Warn(msg string, fields ...zap.Field) {
	Lg.Warn(msg, fields...)
	// 检查预警
	if alertManager != nil {
		alertManager.CheckAndAlert(zapcore.WarnLevel, zapcore.Entry{
			Level:   zapcore.WarnLevel,
			Time:    time.Now(),
			Message: msg,
		}, fields)
	}
}

// Error 通用 error 日志方法
func Error(msg string, fields ...zap.Field) {
	Lg.Error(msg, fields...)
	// 检查预警
	if alertManager != nil {
		alertManager.CheckAndAlert(zapcore.ErrorLevel, zapcore.Entry{
			Level:   zapcore.ErrorLevel,
			Time:    time.Now(),
			Message: msg,
		}, fields)
	}
}

// Debug 通用 debug 日志方法
func Debug(msg string, fields ...zap.Field) {
	Lg.Debug(msg, fields...)
}

// Fatal 通用 fatal 日志方法
func Fatal(msg string, fields ...zap.Field) {
	Lg.Fatal(msg, fields...)
}

// Panic 通用 panic 日志方法
func Panic(msg string, fields ...zap.Field) {
	Lg.Panic(msg, fields...)
}

// Sync 刷新缓冲区
func Sync() {
	_ = Lg.Sync()
}

// GetAlertStats 获取预警统计信息
func GetAlertStats() *AlertStats {
	if alertManager != nil {
		return alertManager.GetStats()
	}
	return nil
}

// ResetAlertStats 重置预警统计
func ResetAlertStats() {
	if alertManager != nil {
		alertManager.ResetStats()
	}
}

// IsAlertEnabled 检查预警功能是否启用
func IsAlertEnabled() bool {
	return alertManager != nil
}

// GetDailyLogFilename 获取按日期分割的日志文件名
func GetDailyLogFilename(baseFilename string) string {
	ext := filepath.Ext(baseFilename)
	base := baseFilename[:len(baseFilename)-len(ext)]
	dateStr := time.Now().Format("2006-01-02")
	return base + "-" + dateStr + ext
}

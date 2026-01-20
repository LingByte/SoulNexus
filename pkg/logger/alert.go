package logger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AlertConfig 预警配置
type AlertConfig struct {
	Enabled           bool          `mapstructure:"enabled"`            // 是否启用预警
	ErrorThreshold    int           `mapstructure:"error_threshold"`    // 错误日志阈值
	WarningThreshold  int           `mapstructure:"warning_threshold"`  // 警告日志阈值
	TimeWindow        time.Duration `mapstructure:"time_window"`        // 时间窗口
	CooldownPeriod    time.Duration `mapstructure:"cooldown_period"`    // 冷却期
	AdminEmails       []string      `mapstructure:"admin_emails"`       // 管理员邮箱列表
	AdminPhones       []string      `mapstructure:"admin_phones"`       // 管理员手机号列表
	AdminUserIDs      []uint        `mapstructure:"admin_user_ids"`     // 管理员用户ID列表
	NotificationTypes []string      `mapstructure:"notification_types"` // 通知类型: email, sms, internal
}

// AlertStats 预警统计
type AlertStats struct {
	ErrorCount   int       `json:"error_count"`
	WarningCount int       `json:"warning_count"`
	LastAlert    time.Time `json:"last_alert"`
	WindowStart  time.Time `json:"window_start"`
}

// NotificationService 通知服务接口
type NotificationService interface {
	SendAlert(ctx context.Context, alert *AlertInfo) error
}

// AlertInfo 预警信息
type AlertInfo struct {
	Type       string            `json:"type"`        // 预警类型: error, warning
	Count      int               `json:"count"`       // 日志数量
	Threshold  int               `json:"threshold"`   // 阈值
	TimeWindow time.Duration     `json:"time_window"` // 时间窗口
	StartTime  time.Time         `json:"start_time"`  // 开始时间
	EndTime    time.Time         `json:"end_time"`    // 结束时间
	SampleLogs []LogEntry        `json:"sample_logs"` // 示例日志
	Metadata   map[string]string `json:"metadata"`    // 额外元数据
}

// LogEntry 日志条目
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Caller    string                 `json:"caller"`
	Fields    map[string]interface{} `json:"fields"`
}

// AlertManager 预警管理器
type AlertManager struct {
	config    *AlertConfig
	stats     *AlertStats
	mu        sync.RWMutex
	notifiers []NotificationService
	cooldown  map[string]time.Time // 冷却期记录
}

// NewAlertManager 创建预警管理器
func NewAlertManager(config *AlertConfig, notifiers ...NotificationService) *AlertManager {
	return &AlertManager{
		config:    config,
		stats:     &AlertStats{},
		notifiers: notifiers,
		cooldown:  make(map[string]time.Time),
	}
}

// CheckAndAlert 检查并发送预警
func (am *AlertManager) CheckAndAlert(level zapcore.Level, entry zapcore.Entry, fields []zap.Field) {
	if !am.config.Enabled {
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()

	// 重置时间窗口
	if am.stats.WindowStart.IsZero() || now.Sub(am.stats.WindowStart) > am.config.TimeWindow {
		am.stats.WindowStart = now
		am.stats.ErrorCount = 0
		am.stats.WarningCount = 0
	}

	// 统计日志数量
	switch level {
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		am.stats.ErrorCount++
	case zapcore.WarnLevel:
		am.stats.WarningCount++
	default:
		return
	}

	// 检查是否需要发送预警
	var shouldAlert bool
	var alertType string
	var threshold int

	if am.stats.ErrorCount >= am.config.ErrorThreshold {
		shouldAlert = true
		alertType = "error"
		threshold = am.config.ErrorThreshold
	} else if am.stats.WarningCount >= am.config.WarningThreshold {
		shouldAlert = true
		alertType = "warning"
		threshold = am.config.WarningThreshold
	}

	if !shouldAlert {
		return
	}

	// 检查冷却期
	cooldownKey := fmt.Sprintf("%s_%d", alertType, threshold)
	if lastAlert, exists := am.cooldown[cooldownKey]; exists {
		if now.Sub(lastAlert) < am.config.CooldownPeriod {
			return
		}
	}

	// 创建预警信息
	alert := &AlertInfo{
		Type:       alertType,
		Count:      am.stats.ErrorCount + am.stats.WarningCount,
		Threshold:  threshold,
		TimeWindow: am.config.TimeWindow,
		StartTime:  am.stats.WindowStart,
		EndTime:    now,
		SampleLogs: am.collectSampleLogs(level, entry, fields),
		Metadata: map[string]string{
			"service": "feng-framework",
			"version": "1.0.0",
		},
	}

	// 发送预警
	ctx := context.Background()
	for _, notifier := range am.notifiers {
		if err := notifier.SendAlert(ctx, alert); err != nil {
			// 记录通知发送失败，但不影响主流程
			zap.L().Error("Failed to send alert notification",
				zap.String("type", alertType),
				zap.Error(err))
		}
	}

	// 更新冷却期和最后预警时间
	am.cooldown[cooldownKey] = now
	am.stats.LastAlert = now
}

// collectSampleLogs 收集示例日志
func (am *AlertManager) collectSampleLogs(level zapcore.Level, entry zapcore.Entry, fields []zap.Field) []LogEntry {
	// 将 zap.Field 转换为 map
	fieldMap := make(map[string]interface{})
	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			fieldMap[field.Key] = field.String
		case zapcore.Int64Type:
			fieldMap[field.Key] = field.Integer
		case zapcore.BoolType:
			fieldMap[field.Key] = field.Integer == 1
		case zapcore.Float64Type:
			fieldMap[field.Key] = field.Interface
		default:
			fieldMap[field.Key] = field.Interface
		}
	}

	return []LogEntry{
		{
			Level:     level.String(),
			Message:   entry.Message,
			Timestamp: entry.Time,
			Caller:    entry.Caller.String(),
			Fields:    fieldMap,
		},
	}
}

// GetStats 获取预警统计
func (am *AlertManager) GetStats() *AlertStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// 返回副本
	return &AlertStats{
		ErrorCount:   am.stats.ErrorCount,
		WarningCount: am.stats.WarningCount,
		LastAlert:    am.stats.LastAlert,
		WindowStart:  am.stats.WindowStart,
	}
}

// ResetStats 重置统计
func (am *AlertManager) ResetStats() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.stats = &AlertStats{}
	am.cooldown = make(map[string]time.Time)
}

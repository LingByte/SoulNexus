package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FlexibleInt integer type that can accept both string and number formats
type FlexibleInt int

// UnmarshalJSON implements custom JSON parsing, supporting both string and number formats
func (fi *FlexibleInt) UnmarshalJSON(data []byte) error {
	// Handle null values
	if string(data) == "null" {
		return nil
	}

	// Try parsing as string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// If it's a string, try converting to integer
		if str == "" {
			return nil
		}
		val, err := strconv.Atoi(str)
		if err != nil {
			return err
		}
		*fi = FlexibleInt(val)
		return nil
	}

	// 尝试解析为数字
	var num int
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*fi = FlexibleInt(num)
	return nil
}

// Int 转换为int指针
func (fi FlexibleInt) Int() *int {
	val := int(fi)
	return &val
}

// Device represents an IoT device
type Device struct {
	ID          string `json:"id" gorm:"primaryKey;size:64"` // MAC address as ID
	UserID      uint   `json:"userId" gorm:"index"`
	GroupID     *uint  `json:"groupId,omitempty" gorm:"index"` // 组织ID，如果设置则表示这是组织共享的设备
	MacAddress  string `json:"macAddress" gorm:"size:64;uniqueIndex"`
	DeviceName  string `json:"deviceName,omitempty" gorm:"size:128"` // 设备名称/别名
	Board       string `json:"board,omitempty" gorm:"size:128"`      // Board type
	AppVersion  string `json:"appVersion,omitempty" gorm:"size:64"`  // Application version
	AutoUpdate  int    `json:"autoUpdate" gorm:"default:1"`          // 0 = disabled, 1 = enabled
	AssistantID *uint  `json:"assistantId,omitempty" gorm:"index"`   // Assistant ID (对应 xiaozhi-esp32 的 agentId)
	Alias       string `json:"alias,omitempty" gorm:"size:128"`      // Device alias

	// 运行状态监控
	IsOnline    bool       `json:"isOnline" gorm:"default:false;index"`  // 在线状态
	LastSeen    *time.Time `json:"lastSeen,omitempty" gorm:"index"`      // 最后在线时间
	StartTime   *time.Time `json:"startTime,omitempty"`                  // 启动时间
	Uptime      int64      `json:"uptime" gorm:"default:0"`              // 运行时长(秒)
	ErrorCount  int        `json:"errorCount" gorm:"default:0"`          // 错误计数
	LastError   string     `json:"lastError,omitempty" gorm:"type:text"` // 最后错误信息
	LastErrorAt *time.Time `json:"lastErrorAt,omitempty"`                // 最后错误时间

	// 系统信息
	SystemInfo   *string `json:"systemInfo,omitempty" gorm:"type:json"`   // 系统信息JSON
	HardwareInfo *string `json:"hardwareInfo,omitempty" gorm:"type:json"` // 硬件信息JSON
	NetworkInfo  *string `json:"networkInfo,omitempty" gorm:"type:json"`  // 网络信息JSON

	// 性能状态
	CPUUsage    float64 `json:"cpuUsage"`    // CPU使用率
	MemoryUsage float64 `json:"memoryUsage"` // 内存使用率
	Temperature float64 `json:"temperature"` // 设备温度

	// 音频设备状态
	AudioStatus *string `json:"audioStatus,omitempty" gorm:"type:json"` // 音频设备状态JSON

	// 服务状态
	ServiceStatus *string `json:"serviceStatus,omitempty" gorm:"type:json"` // 服务状态JSON

	LastConnected *time.Time `json:"lastConnected,omitempty"`
	CreatedAt     time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
}

// TableName specifies the table name
func (Device) TableName() string {
	return "devices"
}

// ConversationTurn 对话轮次记录
type ConversationTurn struct {
	TurnID    int       `json:"turnId"`    // 轮次ID
	Timestamp time.Time `json:"timestamp"` // 全局时间戳
	Type      string    `json:"type"`      // 类型: "user" 或 "ai"
	Content   string    `json:"content"`   // 内容
	StartTime time.Time `json:"startTime"` // 开始时间
	EndTime   time.Time `json:"endTime"`   // 结束时间
	Duration  int64     `json:"duration"`  // 持续时间(毫秒)

	// 用户输入特有字段
	ASRStartTime *time.Time `json:"asrStartTime,omitempty"` // ASR开始时间
	ASREndTime   *time.Time `json:"asrEndTime,omitempty"`   // ASR结束时间
	ASRDuration  *int64     `json:"asrDuration,omitempty"`  // ASR处理时间(毫秒)

	// AI回复特有字段
	LLMStartTime *time.Time `json:"llmStartTime,omitempty"` // LLM开始时间
	LLMEndTime   *time.Time `json:"llmEndTime,omitempty"`   // LLM结束时间
	LLMDuration  *int64     `json:"llmDuration,omitempty"`  // LLM处理时间(毫秒)
	TTSStartTime *time.Time `json:"ttsStartTime,omitempty"` // TTS开始时间
	TTSEndTime   *time.Time `json:"ttsEndTime,omitempty"`   // TTS结束时间
	TTSDuration  *int64     `json:"ttsDuration,omitempty"`  // TTS处理时间(毫秒)

	// 延迟指标
	ResponseDelay *int64 `json:"responseDelay,omitempty"` // 从用户说话结束到AI开始回复的延迟(毫秒)
	TotalDelay    *int64 `json:"totalDelay,omitempty"`    // 从用户说话结束到AI回复完成的总延迟(毫秒)
}

// ConversationDetails 详细对话记录
type ConversationDetails struct {
	SessionID     string             `json:"sessionId"`     // 会话ID
	StartTime     time.Time          `json:"startTime"`     // 会话开始时间
	EndTime       time.Time          `json:"endTime"`       // 会话结束时间
	TotalTurns    int                `json:"totalTurns"`    // 总轮次数
	UserTurns     int                `json:"userTurns"`     // 用户发言轮次
	AITurns       int                `json:"aiTurns"`       // AI回复轮次
	Turns         []ConversationTurn `json:"turns"`         // 对话轮次列表
	Interruptions int                `json:"interruptions"` // 中断次数
}

// TimingMetrics 时间指标统计
type TimingMetrics struct {
	// 全局指标
	SessionDuration int64 `json:"sessionDuration"` // 会话总时长(毫秒)

	// ASR指标
	ASRCalls       int   `json:"asrCalls"`       // ASR调用次数
	ASRTotalTime   int64 `json:"asrTotalTime"`   // ASR总处理时间(毫秒)
	ASRAverageTime int64 `json:"asrAverageTime"` // ASR平均处理时间(毫秒)
	ASRMinTime     int64 `json:"asrMinTime"`     // ASR最短处理时间(毫秒)
	ASRMaxTime     int64 `json:"asrMaxTime"`     // ASR最长处理时间(毫秒)

	// LLM指标
	LLMCalls       int   `json:"llmCalls"`       // LLM调用次数
	LLMTotalTime   int64 `json:"llmTotalTime"`   // LLM总处理时间(毫秒)
	LLMAverageTime int64 `json:"llmAverageTime"` // LLM平均处理时间(毫秒)
	LLMMinTime     int64 `json:"llmMinTime"`     // LLM最短处理时间(毫秒)
	LLMMaxTime     int64 `json:"llmMaxTime"`     // LLM最长处理时间(毫秒)

	// TTS指标
	TTSCalls       int   `json:"ttsCalls"`       // TTS调用次数
	TTSTotalTime   int64 `json:"ttsTotalTime"`   // TTS总处理时间(毫秒)
	TTSAverageTime int64 `json:"ttsAverageTime"` // TTS平均处理时间(毫秒)
	TTSMinTime     int64 `json:"ttsMinTime"`     // TTS最短处理时间(毫秒)
	TTSMaxTime     int64 `json:"ttsMaxTime"`     // TTS最长处理时间(毫秒)

	// 响应延迟指标
	ResponseDelays       []int64 `json:"responseDelays"`       // 所有响应延迟列表(毫秒)
	AverageResponseDelay int64   `json:"averageResponseDelay"` // 平均响应延迟(毫秒)
	MinResponseDelay     int64   `json:"minResponseDelay"`     // 最短响应延迟(毫秒)
	MaxResponseDelay     int64   `json:"maxResponseDelay"`     // 最长响应延迟(毫秒)

	// 总延迟指标
	TotalDelays       []int64 `json:"totalDelays"`       // 所有总延迟列表(毫秒)
	AverageTotalDelay int64   `json:"averageTotalDelay"` // 平均总延迟(毫秒)
	MinTotalDelay     int64   `json:"minTotalDelay"`     // 最短总延迟(毫秒)
	MaxTotalDelay     int64   `json:"maxTotalDelay"`     // 最长总延迟(毫秒)
}

// GetDeviceByMacAddress gets device by MAC address
func GetDeviceByMacAddress(db *gorm.DB, macAddress string) (*Device, error) {
	var device Device
	err := db.Where("mac_address = ?", macAddress).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// CreateDevice creates a new device
func CreateDevice(db *gorm.DB, device *Device) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	if device == nil {
		return fmt.Errorf("device is nil")
	}

	// 记录即将创建的设备信息
	logger.Info("准备创建设备数据库记录",
		zap.String("deviceId", device.ID),
		zap.String("macAddress", device.MacAddress),
		zap.Uint("userId", device.UserID),
		zap.Any("assistantId", device.AssistantID),
		zap.Any("groupId", device.GroupID),
		zap.String("board", device.Board),
		zap.String("appVersion", device.AppVersion),
		zap.Any("lastSeen", device.LastSeen),
		zap.Any("lastConnected", device.LastConnected))

	// 检查必填字段
	if device.ID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}
	if device.MacAddress == "" {
		return fmt.Errorf("device MAC address cannot be empty")
	}
	if device.UserID == 0 {
		return fmt.Errorf("device user ID cannot be zero")
	}

	// 执行数据库创建操作
	result := db.Create(device)
	if result.Error != nil {
		logger.Error("数据库创建设备记录失败",
			zap.Error(result.Error),
			zap.String("deviceId", device.ID),
			zap.String("macAddress", device.MacAddress),
			zap.Int64("rowsAffected", result.RowsAffected))
		return result.Error
	}

	logger.Info("设备数据库记录创建成功",
		zap.String("deviceId", device.ID),
		zap.String("macAddress", device.MacAddress),
		zap.Int64("rowsAffected", result.RowsAffected))

	return nil
}

// UpdateDevice updates device information
func UpdateDevice(db *gorm.DB, device *Device) error {
	return db.Save(device).Error
}

// GetDeviceByID gets device by ID
func GetDeviceByID(db *gorm.DB, id string) (*Device, error) {
	var device Device
	err := db.Where("id = ?", id).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// DeleteDevice deletes a device
func DeleteDevice(db *gorm.DB, id string) error {
	return db.Delete(&Device{}, "id = ?", id).Error
}

// GetUserDevices 获取用户的设备列表（支持组织权限）
func GetUserDevices(db *gorm.DB, userID uint, assistantID *uint) ([]Device, error) {
	var devices []Device

	// 获取用户所属的组织ID列表
	var groupIDs []uint
	var groupMembers []GroupMember
	if err := db.Where("user_id = ?", userID).Find(&groupMembers).Error; err == nil {
		for _, member := range groupMembers {
			groupIDs = append(groupIDs, member.GroupID)
		}
	}
	// 获取用户创建的组织ID
	var userGroups []Group
	if err := db.Where("creator_id = ?", userID).Find(&userGroups).Error; err == nil {
		for _, group := range userGroups {
			groupIDs = append(groupIDs, group.ID)
		}
	}

	// 构建查询
	query := db.Model(&Device{})
	if assistantID != nil {
		query = query.Where("assistant_id = ?", *assistantID)
	}

	// 权限过滤：用户自己的设备 + 组织共享的设备
	if len(groupIDs) > 0 {
		query = query.Where("user_id = ? OR (group_id IS NOT NULL AND group_id IN (?))", userID, groupIDs)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Order("last_seen DESC").Find(&devices).Error
	return devices, err
}

// UpdateDeviceStatus 更新设备状态
func UpdateDeviceStatus(db *gorm.DB, macAddress string, status map[string]interface{}) error {
	return db.Model(&Device{}).Where("mac_address = ?", macAddress).Updates(status).Error
}

// UpdateDeviceOnlineStatus 更新设备在线状态
func UpdateDeviceOnlineStatus(db *gorm.DB, macAddress string, isOnline bool) error {
	now := time.Now()
	updates := map[string]interface{}{
		"is_online": isOnline,
		"last_seen": &now,
	}
	if isOnline {
		updates["start_time"] = &now
	}
	return UpdateDeviceStatus(db, macAddress, updates)
}

// DeviceReportReq represents device report request
type DeviceReportReq struct {
	Version             *FlexibleInt           `json:"version,omitempty"`
	FlashSize           *FlexibleInt           `json:"flash_size,omitempty"`
	MinimumFreeHeapSize *FlexibleInt           `json:"minimum_free_heap_size,omitempty"`
	MacAddress          string                 `json:"mac_address,omitempty"`
	UUID                string                 `json:"uuid,omitempty"`
	ChipModelName       string                 `json:"chip_model_name,omitempty"`
	ChipInfo            *ChipInfo              `json:"chip_info,omitempty"`
	Application         *Application           `json:"application,omitempty"`
	PartitionTable      []Partition            `json:"partition_table,omitempty"`
	Ota                 *OtaInfo               `json:"ota,omitempty"`
	Board               *BoardInfo             `json:"board,omitempty"`
	Device              map[string]interface{} `json:"device,omitempty"`
	Model               string                 `json:"model,omitempty"`
}

type ChipInfo struct {
	Model    *FlexibleInt `json:"model,omitempty"`
	Cores    *FlexibleInt `json:"cores,omitempty"`
	Revision *FlexibleInt `json:"revision,omitempty"`
	Features *FlexibleInt `json:"features,omitempty"`
}

type Application struct {
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	CompileTime string `json:"compile_time,omitempty"`
	IdfVersion  string `json:"idf_version,omitempty"`
	ElfSha256   string `json:"elf_sha256,omitempty"`
}

type Partition struct {
	Label   string       `json:"label,omitempty"`
	Type    *FlexibleInt `json:"type,omitempty"`
	Subtype *FlexibleInt `json:"subtype,omitempty"`
	Address *FlexibleInt `json:"address,omitempty"`
	Size    *FlexibleInt `json:"size,omitempty"`
}

type OtaInfo struct {
	Label string `json:"label,omitempty"`
}

type BoardInfo struct {
	Type    string       `json:"type,omitempty"`
	SSID    string       `json:"ssid,omitempty"`
	RSSI    *FlexibleInt `json:"rssi,omitempty"`
	Channel *FlexibleInt `json:"channel,omitempty"`
	IP      string       `json:"ip,omitempty"`
	MAC     string       `json:"mac,omitempty"`
}

// DeviceReportResp represents device report response
type DeviceReportResp struct {
	ServerTime *ServerTime `json:"server_time,omitempty"`
	Activation *Activation `json:"activation,omitempty"`
	Error      string      `json:"error,omitempty"`
	Firmware   *Firmware   `json:"firmware,omitempty"`
	Websocket  *Websocket  `json:"websocket,omitempty"`
	MQTT       *MQTT       `json:"mqtt,omitempty"`
}

type ServerTime struct {
	Timestamp      int64 `json:"timestamp"`
	TimezoneOffset int   `json:"timezone_offset"`
}

type Activation struct {
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	Challenge string `json:"challenge,omitempty"`
}

type Firmware struct {
	Version string `json:"version"`
	URL     string `json:"url,omitempty"`
}

type Websocket struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

type MQTT struct {
	Endpoint       string `json:"endpoint"`
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	PublishTopic   string `json:"publish_topic"`
	SubscribeTopic string `json:"subscribe_topic"`
}

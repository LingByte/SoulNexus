package models

import (
	"encoding/json"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// FlexibleInt 可以接受字符串或数字的整数类型
type FlexibleInt int

// UnmarshalJSON 实现自定义JSON解析，支持字符串和数字两种格式
func (fi *FlexibleInt) UnmarshalJSON(data []byte) error {
	// 处理null值
	if string(data) == "null" {
		return nil
	}

	// 尝试解析为字符串
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// 如果是字符串，尝试转换为整数
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
	ID            string     `json:"id" gorm:"primaryKey;size:64"` // MAC address as ID
	UserID        uint       `json:"userId" gorm:"index"`
	GroupID       *uint      `json:"groupId,omitempty" gorm:"index"` // 组织ID，如果设置则表示这是组织共享的设备
	MacAddress    string     `json:"macAddress" gorm:"size:64;uniqueIndex"`
	Board         string     `json:"board,omitempty" gorm:"size:128"`     // Board type
	AppVersion    string     `json:"appVersion,omitempty" gorm:"size:64"` // Application version
	AutoUpdate    int        `json:"autoUpdate" gorm:"default:1"`         // 0 = disabled, 1 = enabled
	AssistantID   *uint      `json:"assistantId,omitempty" gorm:"index"`  // Assistant ID (对应 xiaozhi-esp32 的 agentId)
	Alias         string     `json:"alias,omitempty" gorm:"size:128"`     // Device alias
	LastConnected *time.Time `json:"lastConnected,omitempty"`
	CreatedAt     time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `json:"updatedAt" gorm:"autoUpdateTime"`
}

// TableName specifies the table name
func (Device) TableName() string {
	return "devices"
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
	return db.Create(device).Error
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

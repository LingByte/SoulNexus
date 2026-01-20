package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// OverviewConfig 概览页面配置
type OverviewConfig struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	OrganizationID uint      `json:"organizationId" gorm:"uniqueIndex:idx_org_config"`
	Name           string    `json:"name" gorm:"size:200"`
	Description    string    `json:"description,omitempty" gorm:"type:text"`
	Config         JSON      `json:"config" gorm:"type:json"` // 存储完整的配置JSON
}

func (OverviewConfig) TableName() string {
	return "overview_configs"
}

// GetOverviewConfig 获取组织的概览配置
func GetOverviewConfig(db *gorm.DB, organizationID uint) (*OverviewConfig, error) {
	var config OverviewConfig
	err := db.Where("organization_id = ?", organizationID).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // 返回nil表示没有配置
	}
	return &config, err
}

// SaveOverviewConfig 保存或更新概览配置
func SaveOverviewConfig(db *gorm.DB, organizationID uint, name, description string, configData map[string]interface{}) (*OverviewConfig, error) {
	configJSON, err := json.Marshal(configData)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	var config OverviewConfig
	err = db.Where("organization_id = ?", organizationID).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = OverviewConfig{
			OrganizationID: organizationID,
			Name:           name,
			Description:    description,
			Config:         JSON(configJSON),
		}
		err = db.Create(&config).Error
	} else if err == nil {
		// 更新现有配置
		config.Name = name
		config.Description = description
		config.Config = JSON(configJSON)
		err = db.Save(&config).Error
	}

	if err != nil {
		return nil, fmt.Errorf("保存配置失败: %w", err)
	}

	return &config, nil
}

// DeleteOverviewConfig 删除概览配置
func DeleteOverviewConfig(db *gorm.DB, organizationID uint) error {
	return db.Where("organization_id = ?", organizationID).Delete(&OverviewConfig{}).Error
}

// JSON 类型用于存储JSON数据
type JSON json.RawMessage

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSON value: %v", value)
	}
	result := json.RawMessage(bytes)
	*j = JSON(result)
	return nil
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// MarshalJSON 实现 json.Marshaler 接口
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

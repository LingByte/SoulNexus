package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// OverviewConfig represents overview page configuration
type OverviewConfig struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
	OrganizationID uint      `json:"organizationId" gorm:"uniqueIndex:idx_org_config"`
	Name           string    `json:"name" gorm:"size:200"`
	Description    string    `json:"description,omitempty" gorm:"type:text"`
	Config         JSON      `json:"config" gorm:"type:json"` // Stores complete configuration JSON
}

func (OverviewConfig) TableName() string {
	return "overview_configs"
}

// GetOverviewConfig gets organization's overview configuration
func GetOverviewConfig(db *gorm.DB, organizationID uint) (*OverviewConfig, error) {
	var config OverviewConfig
	err := db.Where("organization_id = ?", organizationID).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // Return nil indicates no configuration
	}
	return &config, err
}

// SaveOverviewConfig saves or updates overview configuration
func SaveOverviewConfig(db *gorm.DB, organizationID uint, name, description string, configData map[string]interface{}) (*OverviewConfig, error) {
	configJSON, err := json.Marshal(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize config: %w", err)
	}

	var config OverviewConfig
	err = db.Where("organization_id = ?", organizationID).First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// Create new configuration
		config = OverviewConfig{
			OrganizationID: organizationID,
			Name:           name,
			Description:    description,
			Config:         JSON(configJSON),
		}
		err = db.Create(&config).Error
	} else if err == nil {
		// Update existing configuration
		config.Name = name
		config.Description = description
		config.Config = JSON(configJSON)
		err = db.Save(&config).Error
	}

	if err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return &config, nil
}

// DeleteOverviewConfig deletes overview configuration
func DeleteOverviewConfig(db *gorm.DB, organizationID uint) error {
	return db.Where("organization_id = ?", organizationID).Delete(&OverviewConfig{}).Error
}

// JSON type for storing JSON data
type JSON json.RawMessage

// Value implements driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
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

// UnmarshalJSON implements json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

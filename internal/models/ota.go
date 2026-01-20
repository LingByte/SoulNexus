package models

import (
	"time"

	"gorm.io/gorm"
)

// OTA represents firmware information
type OTA struct {
	ID           string    `json:"id" gorm:"primaryKey;size:64"`
	FirmwareName string    `json:"firmwareName" gorm:"size:128"`
	Type         string    `json:"type" gorm:"size:64;index"` // Board type (e.g., "default", "esp32")
	Version      string    `json:"version" gorm:"size:64"`
	Size         int64     `json:"size"` // File size in bytes
	Remark       string    `json:"remark,omitempty" gorm:"type:text"`
	FirmwarePath string    `json:"firmwarePath" gorm:"size:512"` // File path or URL
	Sort         int       `json:"sort" gorm:"default:0"`
	CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// TableName specifies the table name
func (OTA) TableName() string {
	return "ai_ota"
}

// GetLatestOTA gets the latest OTA firmware for a specific type
func GetLatestOTA(db *gorm.DB, otaType string) (*OTA, error) {
	var ota OTA
	err := db.Where("type = ?", otaType).
		Order("updated_at DESC").
		First(&ota).Error
	if err != nil {
		return nil, err
	}
	return &ota, nil
}

// CreateOTA creates a new OTA firmware record
func CreateOTA(db *gorm.DB, ota *OTA) error {
	return db.Create(ota).Error
}

// UpdateOTA updates OTA firmware record
func UpdateOTA(db *gorm.DB, ota *OTA) error {
	return db.Save(ota).Error
}

// DeleteOTA deletes OTA firmware record
func DeleteOTA(db *gorm.DB, id string) error {
	return db.Delete(&OTA{}, "id = ?", id).Error
}

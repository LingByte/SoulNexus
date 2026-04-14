package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserCredentialRequest struct {
	Name string `json:"name"` // 应用名称 or 用途备注

	LLMProvider string `json:"llmProvider"`
	LLMApiKey   string `json:"llmApiKey"`
	LLMApiURL   string `json:"llmApiUrl"`

	// JSON格式配置
	AsrConfig ProviderConfig `json:"asrConfig"` // ASR配置,格式: {"provider": "qiniu", "apiKey": "...", "baseUrl": "..."} 或 {"provider": "qcloud", "appId": "...", "secretId": "...", "secretKey": "..."}
	TtsConfig ProviderConfig `json:"ttsConfig"` // TTS配置
}

type CredentialStatus string

const (
	CredentialStatusActive    CredentialStatus = "active"
	CredentialStatusBanned    CredentialStatus = "banned"
	CredentialStatusSuspended CredentialStatus = "suspended"
)

// ProviderConfig 提供商的灵活配置,支持任意键值对
type ProviderConfig map[string]interface{}

// Value 实现 driver.Valuer 接口
func (pc ProviderConfig) Value() (driver.Value, error) {
	if pc == nil || len(pc) == 0 {
		return nil, nil
	}
	return json.Marshal(pc)
}

// Scan 实现 sql.Scanner 接口
func (pc *ProviderConfig) Scan(value interface{}) error {
	if value == nil {
		*pc = make(ProviderConfig)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to convert value to []byte")
	}
	if len(bytes) == 0 {
		*pc = make(ProviderConfig)
		return nil
	}
	return json.Unmarshal(bytes, pc)
}

type UserCredential struct {
	BaseModel
	UserID       uint             `gorm:"index;" json:"userId"`
	Name         string           `json:"name"`                                                      // 应用名称 or 用途备注
	APIKey       string           `gorm:"uniqueIndex:idx_api_key,length:100;not null" json:"apiKey"` // 用于认证
	APISecret    string           `gorm:"not null" json:"apiSecret"`                                 // 用于签名校验
	Status       CredentialStatus `gorm:"type:varchar(20);default:'active'" json:"status"`           // 状态: active, banned, suspended
	BannedAt     *time.Time       `gorm:"index" json:"bannedAt"`                                     // 封禁时间
	BannedReason string           `gorm:"type:text" json:"bannedReason"`                             // 封禁原因
	BannedBy     *uint            `gorm:"index" json:"bannedBy"`                                     // 封禁操作者ID
	ExpiresAt    *time.Time       `gorm:"index" json:"expiresAt"`                                    // 过期时间
	LastUsedAt   *time.Time       `gorm:"index" json:"lastUsedAt"`                                   // 最后使用时间
	UsageCount   int64            `gorm:"default:0" json:"usageCount"`                               // 使用次数
	LLMProvider  string           `json:"llmProvider"`
	LLMApiKey    string           `json:"llmApiKey"`
	LLMApiURL    string           `json:"llmApiUrl"`
	AsrConfig    ProviderConfig   `json:"asrConfig" gorm:"type:json"`
	TtsConfig    ProviderConfig   `json:"ttsConfig" gorm:"type:json"`
}

// UserCredentialResponse 用于返回给前端的凭证信息（不包含敏感信息）
type UserCredentialResponse struct {
	ID          uint      `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	UserID      uint      `json:"userId"`
	Name        string    `json:"name"`
	LLMProvider string    `json:"llmProvider"`
	// 只返回 provider 信息，不返回具体的凭证
	AsrProvider string `json:"asrProvider"`
	TtsProvider string `json:"ttsProvider"`
}

// ToResponse 将 UserCredential 转换为 UserCredentialResponse（不包含敏感信息）
func (uc *UserCredential) ToResponse() *UserCredentialResponse {
	asrProvider := ""
	if uc.AsrConfig != nil {
		if provider, ok := uc.AsrConfig["provider"].(string); ok {
			asrProvider = provider
		}
	}

	ttsProvider := ""
	if uc.TtsConfig != nil {
		if provider, ok := uc.TtsConfig["provider"].(string); ok {
			ttsProvider = provider
		}
	}

	return &UserCredentialResponse{
		ID:          uc.ID,
		CreatedAt:   uc.CreatedAt,
		UpdatedAt:   uc.UpdatedAt,
		UserID:      uc.UserID,
		Name:        uc.Name,
		LLMProvider: uc.LLMProvider,
		AsrProvider: asrProvider,
		TtsProvider: ttsProvider,
	}
}

func (uc *UserCredential) IsExpired() bool {
	if uc.ExpiresAt == nil {
		return false
	}
	return uc.ExpiresAt.Before(time.Now())
}

func (uc *UserCredential) IsAvailable() bool {
	return uc.Status == CredentialStatusActive && !uc.IsExpired()
}

func (uc *UserCredential) Ban(reason string, operatorID *uint) {
	now := time.Now()
	uc.Status = CredentialStatusBanned
	uc.BannedAt = &now
	uc.BannedReason = reason
	uc.BannedBy = operatorID
}

func (uc *UserCredential) Unban() {
	uc.Status = CredentialStatusActive
	uc.BannedAt = nil
	uc.BannedReason = ""
	uc.BannedBy = nil
}

func (uc *UserCredential) Suspend() {
	uc.Status = CredentialStatusSuspended
}

func (uc *UserCredential) Activate() {
	uc.Status = CredentialStatusActive
}

// ToResponseList 将 UserCredential 列表转换为 UserCredentialResponse 列表
func ToResponseList(credentials []*UserCredential) []*UserCredentialResponse {
	responses := make([]*UserCredentialResponse, len(credentials))
	for i, cred := range credentials {
		responses[i] = cred.ToResponse()
	}
	return responses
}

func (uc *UserCredential) TableName() string {
	return constants.USER_CREDENTIAL_TABLE_NAME
}

// GetASRProvider 从AsrConfig获取provider
func (uc *UserCredential) GetASRProvider() string {
	if uc.AsrConfig != nil {
		if provider, ok := uc.AsrConfig["provider"].(string); ok {
			return provider
		}
	}
	return ""
}

// GetASRConfig 获取ASR配置值
func (uc *UserCredential) GetASRConfig(key string) interface{} {
	if uc.AsrConfig != nil {
		return uc.AsrConfig[key]
	}
	return nil
}

// GetASRConfigString 获取ASR配置字符串值
func (uc *UserCredential) GetASRConfigString(key string) string {
	if uc.AsrConfig != nil {
		if val, ok := uc.AsrConfig[key].(string); ok {
			return val
		}
	}
	return ""
}

// GetTTSProvider 从TtsConfig获取provider
func (uc *UserCredential) GetTTSProvider() string {
	if uc.TtsConfig != nil {
		if provider, ok := uc.TtsConfig["provider"].(string); ok {
			return provider
		}
	}
	return ""
}

// GetTTSConfig 获取TTS配置值
func (uc *UserCredential) GetTTSConfig(key string) interface{} {
	if uc.TtsConfig != nil {
		return uc.TtsConfig[key]
	}
	return nil
}

// GetTTSConfigString 获取TTS配置字符串值
func (uc *UserCredential) GetTTSConfigString(key string) string {
	if uc.TtsConfig != nil {
		if val, ok := uc.TtsConfig[key].(string); ok {
			return val
		}
	}
	return ""
}

// BuildASRConfig 从请求中构建ASR配置
func (req *UserCredentialRequest) BuildASRConfig() ProviderConfig {
	// 如果已经提供了配置,直接返回
	if req.AsrConfig != nil && len(req.AsrConfig) > 0 {
		// 确保provider字段存在
		if _, ok := req.AsrConfig["provider"]; !ok {
			return nil // provider 是必需的
		}
		return req.AsrConfig
	}
	return nil
}

// BuildTTSConfig 从请求中构建TTS配置
func (req *UserCredentialRequest) BuildTTSConfig() ProviderConfig {
	// 如果已经提供了配置,直接返回
	if req.TtsConfig != nil && len(req.TtsConfig) > 0 {
		// 确保provider字段存在
		if _, ok := req.TtsConfig["provider"]; !ok {
			return nil // provider 是必需的
		}
		return req.TtsConfig
	}
	return nil
}

// CreateUserCredential 创建用户凭证
func CreateUserCredential(db *gorm.DB, userID uint, credential *UserCredentialRequest) (*UserCredential, error) {
	apiKey, err := utils.GenerateSecureToken(32)
	if err != nil {
		return nil, err
	}

	apiSecret, err := utils.GenerateSecureToken(64)
	if err != nil {
		return nil, err
	}

	// 构建新格式的配置
	asrConfig := credential.BuildASRConfig()
	ttsConfig := credential.BuildTTSConfig()

	userCred := &UserCredential{
		UserID:       userID,
		APIKey:       apiKey,
		APISecret:    apiSecret,
		Name:         credential.Name,
		Status:       CredentialStatusActive,
		LLMProvider:  credential.LLMProvider,
		LLMApiKey:    credential.LLMApiKey,
		LLMApiURL:    credential.LLMApiURL,
		AsrConfig:    asrConfig,
		TtsConfig:    ttsConfig,
		UsageCount:   0,
		LastUsedAt:   nil,
		BannedAt:     nil,
		BannedReason: "",
		BannedBy:     nil,
		ExpiresAt:    nil,
	}

	err = db.Create(userCred).Error
	if err != nil {
		return nil, err
	}

	return userCred, nil
}

// GetUserCredentials 根据用户ID获取其所有的凭证信息
func GetUserCredentials(db *gorm.DB, userID uint) ([]*UserCredential, error) {
	var credentials []*UserCredential
	err := db.Where("user_id = ?", userID).Find(&credentials).Error
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func GetUserCredentialByApiSecretAndApiKey(db *gorm.DB, apiKey, apiSecret string) (*UserCredential, error) {
	var credential UserCredential
	result := db.Where("api_key = ? AND api_secret = ?", apiKey, apiSecret).First(&credential)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	if !credential.IsAvailable() {
		return nil, nil
	}
	return &credential, nil
}

func MarkCredentialUsed(db *gorm.DB, credentialID uint) error {
	now := time.Now()
	return db.Model(&UserCredential{}).
		Where("id = ?", credentialID).
		Updates(map[string]interface{}{
			"last_used_at": now,
			"usage_count":  gorm.Expr("usage_count + 1"),
		}).Error
}

// CheckAndReserveCredits 原子性校验并预占额度（可选）。need 为需要的额度。
func CheckAndReserveCredits(db *gorm.DB, credentialID uint, need int64) (*UserCredential, error) {
	var cred UserCredential
	if need <= 0 {
		need = 1
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cred, credentialID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// CommitCredits 扣减已预占额度
func CommitCredits(db *gorm.DB, credentialID uint, used int64) error {
	if used <= 0 {
		used = 1
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var cred UserCredential
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&cred, credentialID).Error; err != nil {
			return err
		}
		return nil
	})
}

// ReleaseReservedCredits 释放预占额度（在失败或取消时）
func ReleaseReservedCredits(db *gorm.DB, credentialID uint, amount int64) error {
	if amount <= 0 {
		return nil
	}
	return db.Model(&UserCredential{}).
		Where("id = ? AND credits_hold >= ?", credentialID, amount).
		UpdateColumn("credits_hold", gorm.Expr("credits_hold - ?", amount)).Error
}

// DeleteUserCredential 删除用户凭证
func DeleteUserCredential(db *gorm.DB, userID uint, credentialID uint) error {
	result := db.Where("user_id = ? AND id = ?", userID, credentialID).Delete(&UserCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("credential not found or access denied")
	}
	return nil
}

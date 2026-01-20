package models

import (
	"time"

	"gorm.io/gorm"
)

// QuotaType 配额类型
type QuotaType string

const (
	QuotaTypeLLMTokens    QuotaType = "llm_tokens"    // LLM Token 使用量
	QuotaTypeLLMCalls     QuotaType = "llm_calls"     // LLM 调用次数
	QuotaTypeAPICalls     QuotaType = "api_calls"     // API 调用次数
	QuotaTypeCallDuration QuotaType = "call_duration" // 通话时长（秒）
	QuotaTypeCallCount    QuotaType = "call_count"    // 通话次数
	QuotaTypeASRDuration  QuotaType = "asr_duration"  // 语音识别时长（秒）
	QuotaTypeASRCount     QuotaType = "asr_count"     // 语音识别次数
	QuotaTypeTTSDuration  QuotaType = "tts_duration"  // 语音合成时长（秒）
	QuotaTypeTTSCount     QuotaType = "tts_count"     // 语音合成次数
)

// QuotaPeriod 配额周期
type QuotaPeriod string

const (
	QuotaPeriodLifetime QuotaPeriod = "lifetime" // 永久有效
	QuotaPeriodMonthly  QuotaPeriod = "monthly"  // 按月重置
	QuotaPeriodYearly   QuotaPeriod = "yearly"   // 按年重置
)

// UserQuota 用户配额配置
type UserQuota struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	UserID     uint        `json:"userId" gorm:"index:idx_user_quota_type"`
	QuotaType  QuotaType   `json:"quotaType" gorm:"size:50;index:idx_user_quota_type"`
	TotalQuota int64       `json:"totalQuota" gorm:"default:0"`              // 总配额，0表示无限制
	UsedQuota  int64       `json:"usedQuota" gorm:"default:0"`               // 已使用配额（缓存值）
	Period     QuotaPeriod `json:"period" gorm:"size:20;default:'lifetime'"` // 配额周期
	ResetAt    *time.Time  `json:"resetAt,omitempty"`                        // 配额重置时间

	// 描述信息
	Description string `json:"description,omitempty" gorm:"size:500"`
}

func (UserQuota) TableName() string {
	return "user_quotas"
}

// GroupQuota 组织配额配置
type GroupQuota struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`

	GroupID    uint        `json:"groupId" gorm:"index:idx_group_quota_type"`
	QuotaType  QuotaType   `json:"quotaType" gorm:"size:50;index:idx_group_quota_type"`
	TotalQuota int64       `json:"totalQuota" gorm:"default:0"`              // 总配额，0表示无限制
	UsedQuota  int64       `json:"usedQuota" gorm:"default:0"`               // 已使用配额（缓存值）
	Period     QuotaPeriod `json:"period" gorm:"size:20;default:'lifetime'"` // 配额周期
	ResetAt    *time.Time  `json:"resetAt,omitempty"`                        // 配额重置时间

	// 描述信息
	Description string `json:"description,omitempty" gorm:"size:500"`
}

func (GroupQuota) TableName() string {
	return "group_quotas"
}

// GetUserQuota 获取用户配额
func GetUserQuota(db *gorm.DB, userID uint, quotaType QuotaType) (*UserQuota, error) {
	var quota UserQuota
	err := db.Where("user_id = ? AND quota_type = ?", userID, quotaType).First(&quota).Error
	if err == gorm.ErrRecordNotFound {
		// 如果不存在，返回默认配额（无限制）
		return &UserQuota{
			UserID:     userID,
			QuotaType:  quotaType,
			TotalQuota: 0, // 0 表示无限制
			UsedQuota:  0,
			Period:     QuotaPeriodLifetime,
		}, nil
	}
	return &quota, err
}

// GetGroupQuota 获取组织配额
func GetGroupQuota(db *gorm.DB, groupID uint, quotaType QuotaType) (*GroupQuota, error) {
	var quota GroupQuota
	err := db.Where("group_id = ? AND quota_type = ?", groupID, quotaType).First(&quota).Error
	if err == gorm.ErrRecordNotFound {
		// 如果不存在，返回默认配额（无限制）
		return &GroupQuota{
			GroupID:    groupID,
			QuotaType:  quotaType,
			TotalQuota: 0, // 0 表示无限制
			UsedQuota:  0,
			Period:     QuotaPeriodLifetime,
		}, nil
	}
	return &quota, err
}

// GetEffectiveQuota 获取有效配额（用户配额 + 组织配额）
// 如果用户属于组织，则取两者中的较大值
// usedQuota 返回的是该用户的实际使用量（从 UsageRecord 表统计）
func GetEffectiveQuota(db *gorm.DB, userID uint, quotaType QuotaType) (totalQuota int64, usedQuota int64, err error) {
	// 获取用户配额
	userQuota, err := GetUserQuota(db, userID, quotaType)
	if err != nil {
		return 0, 0, err
	}

	totalQuota = userQuota.TotalQuota

	// 计算用户的实际使用量（从 UsageRecord 表统计）
	usedQuota = calculateUserQuotaUsage(db, userID, quotaType)

	// 检查用户是否属于组织，如果有组织配额，取较大值
	var groupMembers []GroupMember
	if err := db.Where("user_id = ?", userID).Find(&groupMembers).Error; err == nil {
		for _, member := range groupMembers {
			groupQuota, err := GetGroupQuota(db, member.GroupID, quotaType)
			if err == nil && groupQuota.TotalQuota > 0 {
				// 如果组织配额更大，使用组织配额
				if groupQuota.TotalQuota > totalQuota || totalQuota == 0 {
					totalQuota = groupQuota.TotalQuota
					// 注意：对于组织配额，usedQuota 仍然是该用户的使用量
					// 如果需要组织总使用量，需要单独调用 UpdateGroupQuotaUsage
				}
			}
		}
	}

	return totalQuota, usedQuota, nil
}

// calculateUserQuotaUsage 计算用户的实际配额使用量（从 UsageRecord 表统计）
func calculateUserQuotaUsage(db *gorm.DB, userID uint, quotaType QuotaType) int64 {
	var used int64
	switch quotaType {
	case QuotaTypeLLMTokens:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeLLM).
			Select("COALESCE(SUM(total_tokens), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeLLMCalls:
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeLLM).
			Count(&used)

	case QuotaTypeAPICalls:
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeAPI).
			Count(&used)

	case QuotaTypeCallDuration:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeCall).
			Select("COALESCE(SUM(call_duration), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeCallCount:
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeCall).
			Count(&used)

	case QuotaTypeASRDuration:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeASR).
			Select("COALESCE(SUM(audio_duration), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeASRCount:
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeASR).
			Count(&used)

	case QuotaTypeTTSDuration:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeTTS).
			Select("COALESCE(SUM(audio_duration), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeTTSCount:
		db.Model(&UsageRecord{}).
			Where("user_id = ? AND usage_type = ?", userID, UsageTypeTTS).
			Count(&used)
	}
	return used
}

// UpdateUserQuotaUsage 更新用户配额使用量（缓存到 UserQuota.UsedQuota）
func UpdateUserQuotaUsage(db *gorm.DB, userID uint, quotaType QuotaType) error {
	quota, err := GetUserQuota(db, userID, quotaType)
	if err != nil {
		return err
	}

	// 计算实际使用量
	used := calculateUserQuotaUsage(db, userID, quotaType)

	// 如果配额不存在（默认配额），创建它以便保存使用量
	if quota.ID == 0 {
		quota.UserID = userID
		quota.QuotaType = quotaType
		quota.TotalQuota = 0 // 默认无限制
		quota.Period = QuotaPeriodLifetime
		if err := db.Create(quota).Error; err != nil {
			return err
		}
	}

	// 更新配额使用量缓存
	quota.UsedQuota = used
	return db.Save(quota).Error
}

// UpdateGroupQuotaUsage 更新组织配额使用量
func UpdateGroupQuotaUsage(db *gorm.DB, groupID uint, quotaType QuotaType) error {
	quota, err := GetGroupQuota(db, groupID, quotaType)
	if err != nil {
		return err
	}

	// 获取组织所有成员
	var members []GroupMember
	if err := db.Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		return err
	}

	if len(members) == 0 {
		quota.UsedQuota = 0
		return db.Save(quota).Error
	}

	// 收集所有成员的用户ID
	var userIDs []uint
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}

	// 根据配额类型从 UsageRecord 表直接统计所有成员的使用量
	var used int64
	switch quotaType {
	case QuotaTypeLLMTokens:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeLLM).
			Select("COALESCE(SUM(total_tokens), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeLLMCalls:
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeLLM).
			Count(&used)

	case QuotaTypeAPICalls:
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeAPI).
			Count(&used)

	case QuotaTypeCallDuration:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeCall).
			Select("COALESCE(SUM(call_duration), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeCallCount:
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeCall).
			Count(&used)

	case QuotaTypeASRDuration:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeASR).
			Select("COALESCE(SUM(audio_duration), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeASRCount:
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeASR).
			Count(&used)

	case QuotaTypeTTSDuration:
		var result struct{ Total int64 }
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeTTS).
			Select("COALESCE(SUM(audio_duration), 0) as total").
			Scan(&result)
		used = result.Total

	case QuotaTypeTTSCount:
		db.Model(&UsageRecord{}).
			Where("user_id IN ? AND usage_type = ?", userIDs, UsageTypeTTS).
			Count(&used)
	}

	quota.UsedQuota = used
	return db.Save(quota).Error
}

// ResetQuotaIfNeeded 如果需要，重置配额
func ResetQuotaIfNeeded(db *gorm.DB, quota interface{}) error {
	// 这里需要根据配额周期和重置时间来判断是否需要重置
	// 简化实现：检查 ResetAt 是否已过期
	return nil
}

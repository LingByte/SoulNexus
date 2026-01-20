package models

import (
	"time"

	"gorm.io/gorm"
)

type VoiceTrainingTask struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`                              // 用户ID，确保隔离
	GroupID       *uint          `json:"group_id,omitempty" gorm:"index"`                            // 组织ID，如果设置则表示这是组织共享的音色训练任务
	TaskID        string         `json:"task_id" gorm:"uniqueIndex:idx_task_id,length:100;not null"` // 讯飞返回的任务ID
	TaskName      string         `json:"task_name" gorm:"not null"`                                  // 任务名称
	Sex           int            `json:"sex" gorm:"default:1"`                                       // 性别 1:男 2:女
	AgeGroup      int            `json:"age_group" gorm:"default:2"`                                 // 年龄段 1:儿童 2:青年 3:中年 4:中老年
	Language      string         `json:"language" gorm:"default:'zh'"`                               // 语言
	Status        int            `json:"status" gorm:"default:-1"`                                   // 训练状态 -1:训练中 0:失败 1:成功 2:排队中
	TextID        int64          `json:"text_id" gorm:"default:5001"`                                // 使用的训练文本ID
	TextSegID     int64          `json:"text_seg_id"`                                                // 使用的文本段落ID
	AudioURL      string         `json:"audio_url"`                                                  // 上传的音频文件URL
	AudioDuration float64        `json:"audio_duration"`                                             // 音频时长（秒）
	AudioSize     int64          `json:"audio_size"`                                                 // 音频文件大小（字节）
	TrainVID      string         `json:"train_vid"`                                                  // 讯飞返回的音库ID
	AssetID       string         `json:"asset_id"`                                                   // 讯飞返回的音色ID
	FailedReason  string         `json:"failed_reason"`                                              // 失败原因
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// VoiceClone 音色克隆（训练成功后的音色资源）
type VoiceClone struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"user_id" gorm:"not null;index"`                                // 用户ID
	GroupID          *uint          `json:"group_id,omitempty" gorm:"index"`                              // 组织ID，如果设置则表示这是组织共享的音色
	TrainingTaskID   uint           `json:"training_task_id" gorm:"index"`                                // 关联的训练任务ID（火山引擎可能为0）
	Provider         string         `json:"provider" gorm:"default:'xunfei';index"`                       // 平台提供商: xunfei, volcengine
	AssetID          string         `json:"asset_id" gorm:"uniqueIndex:idx_asset_id,length:100;not null"` // 平台返回的音色ID
	TrainVID         string         `json:"train_vid"`                                                    // 平台返回的音库ID（讯飞专用）
	VoiceName        string         `json:"voice_name" gorm:"not null"`                                   // 音色名称（用户自定义）
	VoiceDescription string         `json:"voice_description"`                                            // 音色描述
	IsActive         bool           `json:"is_active" gorm:"default:true"`                                // 是否激活
	UsageCount       int            `json:"usage_count" gorm:"default:0"`                                 // 使用次数
	LastUsedAt       *time.Time     `json:"last_used_at"`                                                 // 最后使用时间
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// VoiceSynthesis 语音合成记录
type VoiceSynthesis struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`        // 用户ID
	VoiceCloneID  uint           `json:"voice_clone_id" gorm:"not null;index"` // 使用的音色ID
	Text          string         `json:"text" gorm:"not null"`                 // 合成的文本
	Language      string         `json:"language" gorm:"default:'zh'"`         // 语言
	AudioURL      string         `json:"audio_url"`                            // 生成的音频URL
	AudioDuration float64        `json:"audio_duration"`                       // 音频时长
	AudioSize     int64          `json:"audio_size"`                           // 音频文件大小
	Status        string         `json:"status" gorm:"default:'success'"`      // 合成状态 success/failed
	ErrorMessage  string         `json:"error_message"`                        // 错误信息
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// VoiceTrainingText 训练文本（缓存讯飞的训练文本）
type VoiceTrainingText struct {
	ID           uint                       `json:"id" gorm:"primaryKey"`
	TextID       int64                      `json:"text_id" gorm:"uniqueIndex:idx_text_id;not null"` // 讯飞的文本ID
	TextName     string                     `json:"text_name" gorm:"not null"`                       // 文本名称
	Language     string                     `json:"language" gorm:"default:'zh'"`                    // 语言
	IsActive     bool                       `json:"is_active" gorm:"default:true"`                   // 是否可用
	CreatedAt    time.Time                  `json:"created_at"`
	UpdatedAt    time.Time                  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt             `json:"deleted_at" gorm:"index"`
	TextSegments []VoiceTrainingTextSegment `json:"text_segments" gorm:"-"` // 关联的段落（不保存到数据库）
}

// TableName 指定表名
func (VoiceTrainingText) TableName() string {
	return "voice_training_texts"
}

// VoiceTrainingTextSegment 训练文本段落
type VoiceTrainingTextSegment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	TextID    uint           `json:"text_id" gorm:"not null;index"` // 关联的训练文本ID
	SegID     string         `json:"seg_id" gorm:"not null"`        // 段落ID
	SegText   string         `json:"seg_text" gorm:"not null"`      // 段落文本
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName 指定表名
func (VoiceTrainingTextSegment) TableName() string {
	return "voice_training_text_segments"
}

// 训练状态常量
const (
	TrainingStatusInProgress = -1 // 训练中
	TrainingStatusFailed     = 0  // 失败
	TrainingStatusSuccess    = 1  // 成功
	TrainingStatusQueued     = 2  // 排队中
)

// 性别常量
const (
	SexMale   = 1 // 男性
	SexFemale = 2 // 女性
)

// 年龄段常量
const (
	AgeGroupChild   = 1 // 儿童
	AgeGroupYouth   = 2 // 青年
	AgeGroupMiddle  = 3 // 中年
	AgeGroupElderly = 4 // 中老年
)

// 语言常量
const (
	LanguageChinese  = "zh" // 中文
	LanguageEnglish  = "en" // 英文
	LanguageJapanese = "ja" // 日文
	LanguageKorean   = "ko" // 韩文
	LanguageRussian  = "ru" // 俄文
)

// 获取训练状态文本
func (t *VoiceTrainingTask) GetStatusText() string {
	switch t.Status {
	case TrainingStatusInProgress:
		return "训练中"
	case TrainingStatusFailed:
		return "失败"
	case TrainingStatusSuccess:
		return "成功"
	case TrainingStatusQueued:
		return "排队中"
	default:
		return "未知状态"
	}
}

// 获取性别文本
func (t *VoiceTrainingTask) GetSexText() string {
	switch t.Sex {
	case SexMale:
		return "男性"
	case SexFemale:
		return "女性"
	default:
		return "未知"
	}
}

// 获取年龄段文本
func (t *VoiceTrainingTask) GetAgeGroupText() string {
	switch t.AgeGroup {
	case AgeGroupChild:
		return "儿童"
	case AgeGroupYouth:
		return "青年"
	case AgeGroupMiddle:
		return "中年"
	case AgeGroupElderly:
		return "中老年"
	default:
		return "未知"
	}
}

// IsCompleted 检查训练是否完成
func (t *VoiceTrainingTask) IsCompleted() bool {
	return t.Status == TrainingStatusSuccess || t.Status == TrainingStatusFailed
}

// IsSuccess 检查训练是否成功
func (t *VoiceTrainingTask) IsSuccess() bool {
	return t.Status == TrainingStatusSuccess
}

// IncrementUsage 增加使用次数
func (v *VoiceClone) IncrementUsage() {
	v.UsageCount++
	now := time.Now()
	v.LastUsedAt = &now
}

// IsAvailable 检查音色是否可用
func (v *VoiceClone) IsAvailable() bool {
	return v.IsActive && v.AssetID != ""
}

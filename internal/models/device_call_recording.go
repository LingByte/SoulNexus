package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/gorm"
)

// CreateCallRecording 创建通话录音记录
func CreateCallRecording(db *gorm.DB, recording *CallRecording) error {
	return db.Create(recording).Error
}

// GetCallRecordingsByAssistant 获取助手的通话录音列表
func GetCallRecordingsByAssistant(db *gorm.DB, userID, assistantID uint, limit, offset int) ([]CallRecording, int64, error) {
	var recordings []CallRecording
	var total int64
	query := db.Where("user_id = ? AND assistant_id = ?", userID, assistantID)
	query.Model(&CallRecording{}).Count(&total)
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&recordings).Error
	return recordings, total, err
}

// GetCallRecordingsByDevice 获取设备的通话录音列表
func GetCallRecordingsByDevice(db *gorm.DB, userID uint, macAddress string, limit, offset int) ([]CallRecording, int64, error) {
	var recordings []CallRecording
	var total int64
	query := db.Where("user_id = ? AND mac_address = ? AND is_deleted = ?", userID, macAddress, false)
	query.Model(&CallRecording{}).Count(&total)
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&recordings).Error
	return recordings, total, err
}

// GetCallRecordingByID 根据ID获取通话录音
func GetCallRecordingByID(db *gorm.DB, userID uint, recordingID uint) (*CallRecording, error) {
	var recording CallRecording
	err := db.Where("id = ? AND user_id = ? AND is_deleted = ?", recordingID, userID, false).First(&recording).Error
	return &recording, err
}

// DeleteCallRecordingLogical 逻辑删除通话录音
func DeleteCallRecordingLogical(db *gorm.DB, recordingID, userID uint) error {
	now := time.Now()
	return db.Model(&CallRecording{}).
		Where("id = ? AND user_id = ?", recordingID, userID).
		Updates(map[string]interface{}{
			"is_deleted": true,
			"deleted_at": now,
		}).Error
}

// GetExpiredRecordings 获取已过期的录音（用于清理）
func GetExpiredRecordings(db *gorm.DB, limit int) ([]CallRecording, error) {
	var recordings []CallRecording
	now := time.Now()
	err := db.Where("expires_at IS NOT NULL AND expires_at < ? AND is_deleted = ?", now, false).
		Limit(limit).
		Find(&recordings).Error
	return recordings, err
}

// GetCallRecordingsByUser 获取用户的所有通话录音列表
func GetCallRecordingsByUser(db *gorm.DB, userID uint, limit, offset int) ([]CallRecording, int64, error) {
	var recordings []CallRecording
	var total int64
	query := db.Where("user_id = ?", userID)
	if err := query.Model(&CallRecording{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&recordings).Error
	return recordings, total, err
}

// CallRecording 通话录音表
type CallRecording struct {
	BaseModel
	UserID                  uint       `json:"userId" gorm:"index;not null"`
	AssistantID             uint       `json:"assistantId" gorm:"index;not null"`
	DeviceID                string     `json:"deviceId" gorm:"size:64;index"`                         // 设备ID (MAC地址)
	MacAddress              string     `json:"macAddress" gorm:"size:64;index"`                       // MAC地址
	SessionID               string     `json:"sessionId" gorm:"size:128;index"`                       // 会话ID
	StorageURL              string     `json:"storageUrl" gorm:"size:1024"`                           // 存储音频的URL
	AudioFormat             string     `json:"audioFormat" gorm:"size:16"`                            // 音频格式 (wav, mp3, opus)
	AudioSize               int64      `json:"audioSize" gorm:"default:0"`                            // 文件大小(字节)
	Duration                int        `json:"duration" gorm:"default:0"`                             // 录音时长(秒)
	SampleRate              int        `json:"sampleRate" gorm:"default:16000"`                       // 采样率
	Channels                int        `json:"channels" gorm:"default:1"`                             // 声道数
	CallType                string     `json:"callType" gorm:"size:32;index"`                         // 通话类型 (voice, text)
	CallStatus              string     `json:"callStatus" gorm:"size:32;index"`                       // 通话状态 (completed, interrupted, error)
	StartTime               time.Time  `json:"startTime" gorm:"index"`                                // 开始时间
	EndTime                 time.Time  `json:"endTime" gorm:"index"`                                  // 结束时间
	Summary                 string     `json:"summary" gorm:"type:text"`                              // 对话摘要
	Keywords                string     `json:"keywords" gorm:"type:text"`                             // 关键词JSON数组（存储为文本）
	AudioQuality            float64    `json:"audioQuality"`                                          // 音频质量评分 (0-1)
	NoiseLevel              float64    `json:"noiseLevel"`                                            // 噪音水平
	Tags                    string     `json:"tags" gorm:"type:text"`                                 // 标签JSON数组（存储为文本）
	Category                string     `json:"category" gorm:"size:64;index"`                         // 分类
	IsImportant             bool       `json:"isImportant" gorm:"default:false;index"`                // 是否重要
	IsArchived              bool       `json:"isArchived" gorm:"default:false;index"`                 // 是否归档
	AIAnalysis              string     `json:"aiAnalysis" gorm:"type:text"`                           // AI分析结果
	AnalysisStatus          string     `json:"analysisStatus" gorm:"size:32;default:'pending';index"` // 分析状态: pending, analyzing, completed, failed
	AnalysisError           string     `json:"analysisError" gorm:"type:text"`                        // 分析错误信息
	AnalyzedAt              *time.Time `json:"analyzedAt"`                                            // 分析完成时间
	AutoAnalyzed            bool       `json:"autoAnalyzed" gorm:"default:false"`                     // 是否自动分析
	AnalysisVersion         int        `json:"analysisVersion" gorm:"default:1"`                      // 分析版本号
	ConversationDetailsJSON string     `json:"-" gorm:"type:longtext;column:conversation_details"`    // 对话详情JSON数据
	TimingMetricsJSON       string     `json:"-" gorm:"type:longtext;column:timing_metrics"`          // 时间指标JSON数据
	Speakers                string     `json:"speakers" gorm:"type:text"`                             // 使用的发音人ID列表，格式: "101040;101019;101004"
	LLMModel                string     `json:"llmModel" gorm:"size:128"`                              // 使用的LLM模型
	TTSProvider             string     `json:"ttsProvider" gorm:"size:64"`                            // TTS提供商
	ASRProvider             string     `json:"asrProvider" gorm:"size:64"`                            // ASR提供商
}

func (CallRecording) TableName() string {
	return constants.CALL_RECORDING_TABLE_NAME
}

// SetConversationDetails 设置对话详情数据
func (cr *CallRecording) SetConversationDetails(details *ConversationDetails) error {
	if details == nil {
		cr.ConversationDetailsJSON = ""
		return nil
	}

	data, err := json.Marshal(details)
	if err != nil {
		return err
	}
	cr.ConversationDetailsJSON = string(data)
	return nil
}

// GetConversationDetails 获取对话详情数据
func (cr *CallRecording) GetConversationDetails() (*ConversationDetails, error) {
	if cr.ConversationDetailsJSON == "" {
		return nil, nil
	}

	var details ConversationDetails
	err := json.Unmarshal([]byte(cr.ConversationDetailsJSON), &details)
	if err != nil {
		return nil, err
	}
	return &details, nil
}

// SetTimingMetrics 设置时间指标数据
func (cr *CallRecording) SetTimingMetrics(metrics *TimingMetrics) error {
	if metrics == nil {
		cr.TimingMetricsJSON = ""
		return nil
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	cr.TimingMetricsJSON = string(data)
	return nil
}

// GetTimingMetrics 获取时间指标数据
func (cr *CallRecording) GetTimingMetrics() (*TimingMetrics, error) {
	if cr.TimingMetricsJSON == "" {
		return nil, nil
	}
	var metrics TimingMetrics
	err := json.Unmarshal([]byte(cr.TimingMetricsJSON), &metrics)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// AddSpeaker 添加发音人到列表
func (cr *CallRecording) AddSpeaker(speakerID string) {
	if cr.Speakers == "" {
		cr.Speakers = speakerID
	} else {
		// 检查是否已存在
		speakers := strings.Split(cr.Speakers, ";")
		for _, s := range speakers {
			if s == speakerID {
				return
			}
		}
		cr.Speakers = cr.Speakers + ";" + speakerID
	}
}

// GetSpeakers 获取发音人列表
func (cr *CallRecording) GetSpeakers() []string {
	if cr.Speakers == "" {
		return []string{}
	}
	return strings.Split(cr.Speakers, ";")
}

// HasSpeaker 检查是否使用过某个发音人
func (cr *CallRecording) HasSpeaker(speakerID string) bool {
	speakers := cr.GetSpeakers()
	for _, s := range speakers {
		if s == speakerID {
			return true
		}
	}
	return false
}

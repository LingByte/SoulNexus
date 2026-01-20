package voiceclone

import (
	"context"
	"io"
	"time"
)

// Provider 语音克隆服务提供商
type Provider string

const (
	ProviderXunfei     Provider = "xunfei"     // 讯飞星火
	ProviderVolcengine Provider = "volcengine" // 火山引擎
)

// TrainingStatus 训练状态
type TrainingStatus int

const (
	TrainingStatusQueued     TrainingStatus = 2  // 排队中
	TrainingStatusInProgress TrainingStatus = -1 // 训练中
	TrainingStatusSuccess    TrainingStatus = 1  // 成功
	TrainingStatusFailed     TrainingStatus = 0  // 失败
)

// TrainingText 训练文本
type TrainingText struct {
	TextID   int64         `json:"text_id"`
	TextName string        `json:"text_name"`
	Segments []TextSegment `json:"segments"`
}

// TextSegment 文本段落
type TextSegment struct {
	SegID   interface{} `json:"seg_id"` // 可能是字符串或数字
	SegText string      `json:"seg_text"`
}

// CreateTaskRequest 创建训练任务请求
type CreateTaskRequest struct {
	TaskName string `json:"task_name"` // 任务名称
	Sex      int    `json:"sex"`       // 性别 1:男 2:女
	AgeGroup int    `json:"age_group"` // 年龄段 1:儿童 2:青年 3:中年 4:中老年
	Language string `json:"language"`  // 语言代码，如 zh, en
}

// CreateTaskResponse 创建训练任务响应
type CreateTaskResponse struct {
	TaskID string `json:"task_id"` // 任务ID
}

// SubmitAudioRequest 提交音频请求
type SubmitAudioRequest struct {
	TaskID    string    `json:"task_id"`     // 任务ID
	TextID    int64     `json:"text_id"`     // 训练文本ID
	TextSegID int64     `json:"text_seg_id"` // 文本段落ID
	AudioFile io.Reader `json:"-"`           // 音频文件
	Language  string    `json:"language"`    // 语言代码
}

// TaskStatus 任务状态
type TaskStatus struct {
	TaskID     string         `json:"task_id"`     // 任务ID
	TaskName   string         `json:"task_name"`   // 任务名称
	Status     TrainingStatus `json:"status"`      // 训练状态
	AssetID    string         `json:"asset_id"`    // 音色ID（训练成功后返回）
	TrainVID   string         `json:"train_vid"`   // 音库ID
	FailedDesc string         `json:"failed_desc"` // 失败原因
	Progress   float64        `json:"progress"`    // 训练进度 0-100
	CreatedAt  time.Time      `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time      `json:"updated_at"`  // 更新时间
}

// SynthesizeRequest 合成请求
type SynthesizeRequest struct {
	AssetID  string `json:"asset_id"` // 音色ID
	Text     string `json:"text"`     // 要合成的文本
	Language string `json:"language"` // 语言代码
}

// SynthesizeResponse 合成响应
type SynthesizeResponse struct {
	AudioData  []byte  `json:"audio_data"`  // 音频数据
	Format     string  `json:"format"`      // 音频格式，如 pcm, wav, mp3
	SampleRate int     `json:"sample_rate"` // 采样率
	Duration   float64 `json:"duration"`    // 音频时长（秒）
}

// SynthesisHandler 流式合成处理器接口（与 synthesis 包兼容）
type SynthesisHandler interface {
	OnMessage([]byte)
	OnTimestamp(timestamp SentenceTimestamp)
}

// SentenceTimestamp 句子时间戳（兼容 synthesis 包）
type SentenceTimestamp struct {
	StartTime int64 `json:"start_time"` // 开始时间（毫秒）
	EndTime   int64 `json:"end_time"`   // 结束时间（毫秒）
}

// VoiceCloneService 语音克隆服务接口
type VoiceCloneService interface {
	// Provider 返回服务提供商名称
	Provider() Provider

	// GetTrainingTexts 获取训练文本列表
	GetTrainingTexts(ctx context.Context, textID int64) (*TrainingText, error)

	// CreateTask 创建训练任务
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error)

	// SubmitAudio 提交音频文件
	SubmitAudio(ctx context.Context, req *SubmitAudioRequest) error

	// QueryTaskStatus 查询任务状态
	QueryTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)

	// Synthesize 使用训练好的音色合成语音（批量模式，返回完整音频）
	Synthesize(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error)

	// SynthesizeStream 使用训练好的音色流式合成语音（流式模式，通过 handler 回调）
	SynthesizeStream(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error

	// SynthesizeToStorage 合成并保存到存储
	SynthesizeToStorage(ctx context.Context, req *SynthesizeRequest, storageKey string) (string, error)
}

// Config 语音克隆配置
type Config struct {
	Provider Provider               `json:"provider"` // 服务提供商
	Options  map[string]interface{} `json:"options"`  // 提供商特定配置
}

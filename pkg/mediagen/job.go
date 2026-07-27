package mediagen

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/gorm"
)

const (
	JobKindTextToImage  = "text_to_image"
	JobKindImageToImage = "image_to_image"
	JobKindTextToVideo  = "text_to_video"
	JobKindImageToVideo = "image_to_video"

	JobStatusPending   = "pending"
	JobStatusQueued    = "queued"
	JobStatusRunning   = "running"
	JobStatusSucceeded = "succeeded"
	JobStatusFailed    = "failed"
	JobStatusCanceled  = "canceled"
)

// MediaGenerateJob persists one image/video generation attempt and step metrics.
type MediaGenerateJob struct {
	common.BaseModel

	PublicID        string     `json:"publicId" gorm:"size:64;uniqueIndex;not null;comment:对外任务ID"`
	TenantID        uint       `json:"tenantId" gorm:"index;not null;comment:租户ID"`
	UserID          uint       `json:"userId" gorm:"index;default:0;comment:创建用户ID"`
	Kind            string     `json:"kind" gorm:"size:32;index;not null;comment:任务类型"`
	Status          string     `json:"status" gorm:"size:24;index;not null;default:pending;comment:业务状态"`
	Prompt          string     `json:"prompt" gorm:"type:text;comment:提示词"`
	NegativePrompt  string     `json:"negativePrompt,omitempty" gorm:"type:text;comment:负向提示词"`
	ParamsJSON      string     `json:"paramsJson,omitempty" gorm:"type:longtext;comment:生成参数JSON"`
	StepsJSON       string     `json:"stepsJson,omitempty" gorm:"type:longtext;comment:步骤指标JSON"`
	MetricsJSON     string     `json:"metricsJson,omitempty" gorm:"type:longtext;comment:汇总指标JSON"`
	ExecutionTaskID string     `json:"executionTaskId,omitempty" gorm:"size:64;index;comment:execution_tasks.task_id"`
	Provider        string     `json:"provider,omitempty" gorm:"size:32;index;comment:上游厂商"`
	ProviderModel   string     `json:"providerModel,omitempty" gorm:"size:128;comment:上游模型"`
	ProviderTaskID  string     `json:"providerTaskId,omitempty" gorm:"size:128;index;comment:上游任务ID"`
	ReferenceKey    string     `json:"referenceKey,omitempty" gorm:"size:512;comment:参考图存储key"`
	LastFrameKey    string     `json:"lastFrameKey,omitempty" gorm:"size:512;comment:尾帧存储key"`
	RemoteURL       string     `json:"remoteUrl,omitempty" gorm:"size:1024;comment:上游临时URL"`
	StorageKey      string     `json:"storageKey,omitempty" gorm:"size:512;comment:对象存储key"`
	ResultURL       string     `json:"resultUrl,omitempty" gorm:"size:1024;comment:对外访问URL"`
	ErrorMessage    string     `json:"errorMessage,omitempty" gorm:"type:text;comment:错误信息"`
	Progress        int        `json:"progress" gorm:"not null;default:0;comment:进度0-100"`
	Width           int        `json:"width,omitempty" gorm:"comment:图片宽"`
	Height          int        `json:"height,omitempty" gorm:"comment:图片高"`
	Ratio           string     `json:"ratio,omitempty" gorm:"size:16;comment:视频比例"`
	Duration        int        `json:"duration,omitempty" gorm:"comment:视频时长秒"`
	Resolution      string     `json:"resolution,omitempty" gorm:"size:16;comment:分辨率"`
	QueuedAt        *time.Time `json:"queuedAt,omitempty" gorm:"comment:入队时间"`
	StartedAt       *time.Time `json:"startedAt,omitempty" gorm:"comment:开始执行"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty" gorm:"comment:结束时间"`
}

func (MediaGenerateJob) TableName() string {
	return constants.MEDIA_GENERATE_JOB_TABLE_NAME
}

// JobStep is one recorded pipeline stage for a generation job.
type JobStep struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	StartedAt  *time.Time     `json:"startedAt,omitempty"`
	FinishedAt *time.Time     `json:"finishedAt,omitempty"`
	DurationMs int64          `json:"durationMs,omitempty"`
	Message    string         `json:"message,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
}

// JobMetrics aggregates timing across a job.
type JobMetrics struct {
	QueueWaitMs      int64 `json:"queueWaitMs,omitempty"`
	ProviderCreateMs int64 `json:"providerCreateMs,omitempty"`
	ProviderPollMs   int64 `json:"providerPollMs,omitempty"`
	DownloadMs       int64 `json:"downloadMs,omitempty"`
	TotalMs          int64 `json:"totalMs,omitempty"`
	PollAttempts     int   `json:"pollAttempts,omitempty"`
}

func CreateMediaGenerateJob(db *gorm.DB, row *MediaGenerateJob) error {
	if db == nil || row == nil {
		return gorm.ErrInvalidData
	}
	if strings.TrimSpace(row.PublicID) == "" {
		return gorm.ErrInvalidData
	}
	if strings.TrimSpace(row.Status) == "" {
		row.Status = JobStatusPending
	}
	return db.Create(row).Error
}

func GetMediaGenerateJobByPublicID(db *gorm.DB, publicID string) (*MediaGenerateJob, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row MediaGenerateJob
	err := db.Where("public_id = ?", publicID).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func GetMediaGenerateJobForTenant(db *gorm.DB, tenantID uint, publicID string) (*MediaGenerateJob, error) {
	row, err := GetMediaGenerateJobByPublicID(db, publicID)
	if err != nil {
		return nil, err
	}
	if tenantID > 0 && row.TenantID != tenantID {
		return nil, gorm.ErrRecordNotFound
	}
	return row, nil
}

func UpdateMediaGenerateJob(db *gorm.DB, publicID string, updates map[string]any) error {
	if db == nil || strings.TrimSpace(publicID) == "" || len(updates) == 0 {
		return gorm.ErrInvalidData
	}
	res := db.Model(&MediaGenerateJob{}).Where("public_id = ?", publicID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ListMediaGenerateJobsPage(db *gorm.DB, tenantID uint, kind, status string, page, pageSize int) ([]MediaGenerateJob, int64, error) {
	if db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	q := db.Model(&MediaGenerateJob{})
	if tenantID > 0 {
		q = q.Where("tenant_id = ?", tenantID)
	}
	if v := strings.TrimSpace(kind); v != "" && !strings.EqualFold(v, "all") {
		switch strings.ToLower(v) {
		case "image":
			q = q.Where("kind IN ?", []string{JobKindTextToImage, JobKindImageToImage})
		case "video":
			q = q.Where("kind IN ?", []string{JobKindTextToVideo, JobKindImageToVideo})
		default:
			q = q.Where("kind = ?", v)
		}
	}
	if v := strings.TrimSpace(status); v != "" && !strings.EqualFold(v, "all") {
		q = q.Where("status = ?", v)
	}
	return utils.FindPage[MediaGenerateJob](q, page, pageSize, "id DESC", utils.MaxPageSize200)
}

func EncodeJobSteps(steps []JobStep) string {
	if len(steps) == 0 {
		return "[]"
	}
	b, err := json.Marshal(steps)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func DecodeJobSteps(raw string) []JobStep {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var steps []JobStep
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return nil
	}
	return steps
}

func EncodeJobMetrics(m JobMetrics) string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func DecodeJobMetrics(raw string) JobMetrics {
	var m JobMetrics
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return m
	}
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

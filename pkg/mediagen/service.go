package mediagen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid"
	"gorm.io/gorm"
)

type Service struct {
	cfg      Config
	seedream *SeedreamClient
	seedance *SeedanceClient
	storage  *Storage
}

func NewService(cfg Config) *Service {
	return &Service{
		cfg:      cfg,
		seedream: NewSeedreamClient(cfg),
		seedance: NewSeedanceClient(cfg),
		storage:  NewStorage(cfg),
	}
}

type ImageGenerateResult struct {
	JobID      string `json:"jobId,omitempty"`
	URL        string `json:"url"`
	Cached     bool   `json:"cached"`
	StorageKey string `json:"storageKey,omitempty"`
	Status     string `json:"status,omitempty"`
}

type VideoTaskCreateResult struct {
	TaskID string `json:"taskId"`
	JobID  string `json:"jobId,omitempty"`
	Status string `json:"status"`
}

type VideoTaskResult struct {
	TaskID       string      `json:"taskId"`
	JobID        string      `json:"jobId,omitempty"`
	Kind         string      `json:"kind,omitempty"`
	Status       string      `json:"status"`
	Prompt       string      `json:"prompt,omitempty"`
	Style        string      `json:"style,omitempty"`
	URL          string      `json:"url,omitempty"`
	RemoteURL    string      `json:"remoteUrl,omitempty"`
	StorageKey   string      `json:"storageKey,omitempty"`
	Resolution   string      `json:"resolution,omitempty"`
	Ratio        string      `json:"ratio,omitempty"`
	Duration     int         `json:"duration,omitempty"`
	Width        int         `json:"width,omitempty"`
	Height       int         `json:"height,omitempty"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	Progress     int         `json:"progress,omitempty"`
	Steps        []JobStep   `json:"steps,omitempty"`
	Metrics      *JobMetrics `json:"metrics,omitempty"`
	CreatedAt    int64       `json:"createdAt,omitempty"`
}

type AppConfigResult struct {
	Configured           bool   `json:"configured"`
	VideoPollIntervalMs  int    `json:"videoPollIntervalMs"`
	VideoPollMaxAttempts int    `json:"videoPollMaxAttempts"`
	SeedreamModelID      string `json:"seedreamModelId"`
	SeedanceModelID      string `json:"seedanceModelId"`
}

type CreateImageJobInput struct {
	TenantID       uint
	UserID         uint
	Kind           string
	Prompt         string
	NegativePrompt string
	Width          int
	Height         int
	Style          string
	Category       string
	Reference      *PreparedReference
}

type CreateVideoJobInput struct {
	TenantID      uint
	UserID        uint
	Kind          string
	Prompt        string
	Ratio         string
	Duration      int
	Resolution    string
	GenerateAudio *bool
	Watermark     bool
	Category      string
	Motion        string
	FPS           string
	FirstFrame    *PreparedReference
	LastFrame     *PreparedReference
}

func (s *Service) Storage() *Storage {
	return s.storage
}

func (s *Service) Config() Config {
	return s.cfg
}

func (s *Service) AppConfig() AppConfigResult {
	return AppConfigResult{
		Configured:           s.cfg.Configured(),
		VideoPollIntervalMs:  s.cfg.PollIntervalMs,
		VideoPollMaxAttempts: s.cfg.PollMaxAttempts,
		SeedreamModelID:      s.cfg.SeedreamModel,
		SeedanceModelID:      s.cfg.SeedanceModel,
	}
}

func NewPublicID(prefix string) string {
	id, _ := gonanoid.Nanoid()
	if prefix == "" {
		return "mg_" + id
	}
	return prefix + "_" + id
}

func (s *Service) CreateImageJobRecord(db *gorm.DB, in CreateImageJobInput) (*MediaGenerateJob, error) {
	if !s.cfg.Configured() {
		return nil, ErrAPIKeyMissing
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return nil, ErrEmptyPrompt
	}
	kind := in.Kind
	if kind == "" {
		if in.Reference != nil {
			kind = JobKindImageToImage
		} else {
			kind = JobKindTextToImage
		}
	}
	params := map[string]any{
		"width":    in.Width,
		"height":   in.Height,
		"style":    in.Style,
		"category": in.Category,
	}
	raw, _ := json.Marshal(params)
	row := &MediaGenerateJob{
		PublicID:       NewPublicID("img"),
		TenantID:       in.TenantID,
		UserID:         in.UserID,
		Kind:           kind,
		Status:         JobStatusPending,
		Prompt:         in.Prompt,
		NegativePrompt: in.NegativePrompt,
		ParamsJSON:     string(raw),
		Provider:       "seedream",
		ProviderModel:  s.cfg.SeedreamModel,
		Width:          in.Width,
		Height:         in.Height,
	}
	if in.Reference != nil {
		row.ReferenceKey = in.Reference.ReferenceKey
	}
	if err := CreateMediaGenerateJob(db, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) CreateVideoJobRecord(db *gorm.DB, in CreateVideoJobInput) (*MediaGenerateJob, error) {
	if !s.cfg.Configured() {
		return nil, ErrAPIKeyMissing
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return nil, ErrEmptyPrompt
	}
	kind := in.Kind
	if kind == "" {
		if in.FirstFrame != nil {
			kind = JobKindImageToVideo
		} else {
			kind = JobKindTextToVideo
		}
	}
	params := map[string]any{
		"ratio":         in.Ratio,
		"duration":      in.Duration,
		"resolution":    in.Resolution,
		"generateAudio": in.GenerateAudio,
		"watermark":     in.Watermark,
		"category":      in.Category,
		"motion":        in.Motion,
		"fps":           in.FPS,
	}
	raw, _ := json.Marshal(params)
	row := &MediaGenerateJob{
		PublicID:      NewPublicID("vid"),
		TenantID:      in.TenantID,
		UserID:        in.UserID,
		Kind:          kind,
		Status:        JobStatusPending,
		Prompt:        in.Prompt,
		ParamsJSON:    string(raw),
		Provider:      "seedance",
		ProviderModel: s.cfg.SeedanceModel,
		Ratio:         in.Ratio,
		Duration:      in.Duration,
		Resolution:    in.Resolution,
	}
	if in.FirstFrame != nil {
		row.ReferenceKey = in.FirstFrame.ReferenceKey
	}
	if in.LastFrame != nil {
		row.LastFrameKey = in.LastFrame.ReferenceKey
	}
	if err := CreateMediaGenerateJob(db, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) ExecuteImageJob(ctx context.Context, db *gorm.DB, row *MediaGenerateJob) error {
	if row == nil {
		return fmt.Errorf("nil media job")
	}
	tracker := newStepTracker()
	metrics := JobMetrics{}
	started := time.Now()
	if row.QueuedAt != nil {
		metrics.QueueWaitMs = started.Sub(*row.QueuedAt).Milliseconds()
	}

	persist := func(progress int) {
		_ = UpdateMediaGenerateJob(db, row.PublicID, map[string]any{
			"steps_json":   EncodeJobSteps(tracker.snapshot()),
			"metrics_json": EncodeJobMetrics(metrics),
			"progress":     progress,
		})
	}

	step := tracker.begin("call_seedream")
	persist(20)
	var remoteURL string
	var err error
	switch row.Kind {
	case JobKindImageToImage:
		refURL, refErr := s.resolveReferenceURL(row.ReferenceKey)
		if refErr != nil {
			tracker.endFail(step, refErr, nil)
			persist(100)
			return refErr
		}
		remoteURL, err = s.seedream.GenerateImageToImage(ctx, row.Prompt, row.Width, row.Height, refURL)
	default:
		remoteURL, err = s.seedream.GenerateTextToImage(ctx, row.Prompt, row.Width, row.Height)
	}
	if err != nil {
		tracker.endFail(step, err, nil)
		persist(100)
		return err
	}
	tracker.endOK(step, "", map[string]any{"hasRemoteUrl": remoteURL != ""})
	metrics.ProviderCreateMs = tracker.steps[step].DurationMs
	persist(60)

	dl := tracker.begin("download_persist")
	key, err := s.storage.DownloadAndSave(remoteURL, s.cfg.ElementsSubdir, ".png", "image/png")
	if err != nil {
		tracker.endFail(dl, err, nil)
		persist(100)
		return err
	}
	tracker.endOK(dl, "", map[string]any{"storageKey": key})
	metrics.DownloadMs = tracker.steps[dl].DurationMs
	metrics.TotalMs = time.Since(started).Milliseconds()
	resultURL := s.storage.PublicURL(key)
	finished := time.Now()
	_ = UpdateMediaGenerateJob(db, row.PublicID, map[string]any{
		"status":       JobStatusSucceeded,
		"remote_url":   remoteURL,
		"storage_key":  key,
		"result_url":   resultURL,
		"steps_json":   EncodeJobSteps(tracker.snapshot()),
		"metrics_json": EncodeJobMetrics(metrics),
		"progress":     100,
		"finished_at":  finished,
		"error_message": "",
	})
	return nil
}

func (s *Service) ExecuteVideoJob(ctx context.Context, db *gorm.DB, row *MediaGenerateJob) error {
	if row == nil {
		return fmt.Errorf("nil media job")
	}
	tracker := newStepTracker()
	metrics := JobMetrics{}
	started := time.Now()
	if row.QueuedAt != nil {
		metrics.QueueWaitMs = started.Sub(*row.QueuedAt).Milliseconds()
	}
	params := VideoGenerateParams{
		Prompt:     row.Prompt,
		Ratio:      row.Ratio,
		Duration:   row.Duration,
		Resolution: row.Resolution,
	}
	var paramMap map[string]any
	_ = json.Unmarshal([]byte(row.ParamsJSON), &paramMap)
	if v, ok := paramMap["watermark"].(bool); ok {
		params.Watermark = v
	}
	if v, ok := paramMap["generateAudio"].(bool); ok {
		params.GenerateAudio = &v
	}

	persist := func(progress int, extra map[string]any) {
		updates := map[string]any{
			"steps_json":   EncodeJobSteps(tracker.snapshot()),
			"metrics_json": EncodeJobMetrics(metrics),
			"progress":     progress,
		}
		for k, v := range extra {
			updates[k] = v
		}
		_ = UpdateMediaGenerateJob(db, row.PublicID, updates)
	}

	createStep := tracker.begin("create_seedance_task")
	persist(10, nil)
	var providerTaskID string
	var err error
	switch row.Kind {
	case JobKindImageToVideo:
		firstURL, e1 := s.resolveReferenceURL(row.ReferenceKey)
		if e1 != nil {
			tracker.endFail(createStep, e1, nil)
			persist(100, nil)
			return e1
		}
		lastURL := ""
		if strings.TrimSpace(row.LastFrameKey) != "" {
			lastURL, err = s.resolveReferenceURL(row.LastFrameKey)
			if err != nil {
				tracker.endFail(createStep, err, nil)
				persist(100, nil)
				return err
			}
		}
		providerTaskID, err = s.seedance.CreateImageToVideoTask(ctx, params, firstURL, lastURL)
	default:
		providerTaskID, err = s.seedance.CreateTextToVideoTask(ctx, params)
	}
	if err != nil {
		tracker.endFail(createStep, err, nil)
		persist(100, nil)
		return err
	}
	tracker.endOK(createStep, "", map[string]any{"providerTaskId": providerTaskID})
	metrics.ProviderCreateMs = tracker.steps[createStep].DurationMs
	persist(25, map[string]any{"provider_task_id": providerTaskID})

	pollStep := tracker.begin("poll_seedance")
	pollStarted := time.Now()
	var remoteURL string
	var resolution, ratio string
	var duration int
	interval := time.Duration(s.cfg.PollIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 10 * time.Second
	}
	maxAttempts := s.cfg.PollMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 120
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			err := ctx.Err()
			tracker.endFail(pollStep, err, map[string]any{"attempt": attempt})
			persist(100, nil)
			return err
		}
		task, getErr := s.seedance.GetTask(ctx, providerTaskID)
		if getErr != nil {
			tracker.endFail(pollStep, getErr, map[string]any{"attempt": attempt})
			persist(100, nil)
			return getErr
		}
		status := strings.ToLower(strings.TrimSpace(task.Status))
		progress := 25 + (attempt*50)/maxAttempts
		if progress > 75 {
			progress = 75
		}
		metrics.PollAttempts = attempt
		persist(progress, map[string]any{"provider_task_id": providerTaskID})
		switch status {
		case "succeeded":
			if task.Content == nil || strings.TrimSpace(task.Content.VideoURL) == "" {
				err := fmt.Errorf("Seedance 任务成功但未返回视频 URL")
				tracker.endFail(pollStep, err, map[string]any{"attempt": attempt})
				persist(100, nil)
				return err
			}
			remoteURL = strings.TrimSpace(task.Content.VideoURL)
			resolution = task.Resolution
			ratio = task.Ratio
			duration = task.Duration
			tracker.endOK(pollStep, "", map[string]any{"attempt": attempt, "status": status})
			metrics.ProviderPollMs = time.Since(pollStarted).Milliseconds()
			goto download
		case "failed", "cancelled", "canceled", "expired":
			msg := "视频生成失败"
			if task.Error != nil && strings.TrimSpace(task.Error.Message) != "" {
				msg = task.Error.Message
			}
			err := fmt.Errorf("%s", msg)
			tracker.endFail(pollStep, err, map[string]any{"attempt": attempt, "status": status})
			persist(100, nil)
			return err
		}
		select {
		case <-ctx.Done():
			err := ctx.Err()
			tracker.endFail(pollStep, err, map[string]any{"attempt": attempt})
			persist(100, nil)
			return err
		case <-time.After(interval):
		}
	}
	err = fmt.Errorf("视频生成超时（已轮询 %d 次）", maxAttempts)
	tracker.endFail(pollStep, err, map[string]any{"attempt": maxAttempts})
	persist(100, nil)
	return err

download:
	dl := tracker.begin("download_persist")
	persist(80, map[string]any{"remote_url": remoteURL})
	key, err := s.storage.DownloadAndSave(remoteURL, s.cfg.VideosSubdir, ".mp4", "video/mp4")
	if err != nil {
		tracker.endFail(dl, err, nil)
		persist(100, nil)
		return err
	}
	tracker.endOK(dl, "", map[string]any{"storageKey": key})
	metrics.DownloadMs = tracker.steps[dl].DurationMs
	metrics.TotalMs = time.Since(started).Milliseconds()
	resultURL := s.storage.PublicURL(key)
	finished := time.Now()
	updates := map[string]any{
		"status":        JobStatusSucceeded,
		"remote_url":    remoteURL,
		"storage_key":   key,
		"result_url":    resultURL,
		"steps_json":    EncodeJobSteps(tracker.snapshot()),
		"metrics_json":  EncodeJobMetrics(metrics),
		"progress":      100,
		"finished_at":   finished,
		"error_message": "",
	}
	if resolution != "" {
		updates["resolution"] = resolution
	}
	if ratio != "" {
		updates["ratio"] = ratio
	}
	if duration > 0 {
		updates["duration"] = duration
	}
	return UpdateMediaGenerateJob(db, row.PublicID, updates)
}

func (s *Service) resolveReferenceURL(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", ErrInvalidImage
	}
	if url := s.storage.PublicURL(key); strings.HasPrefix(strings.ToLower(url), "http://") || strings.HasPrefix(strings.ToLower(url), "https://") {
		return url, nil
	}
	data, err := s.storage.ReadBytes(key)
	if err != nil {
		return "", fmt.Errorf("读取参考图失败: %w", err)
	}
	ct := detectImageContentType(data, key, "")
	return "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (s *Service) JobToImageResult(row *MediaGenerateJob) *ImageGenerateResult {
	if row == nil {
		return nil
	}
	url := row.ResultURL
	if url == "" && row.StorageKey != "" {
		url = s.storage.PublicURL(row.StorageKey)
	}
	return &ImageGenerateResult{
		JobID:      row.PublicID,
		URL:        url,
		StorageKey: row.StorageKey,
		Status:     row.Status,
	}
}

func (s *Service) JobToVideoResult(row *MediaGenerateJob) *VideoTaskResult {
	if row == nil {
		return nil
	}
	url := row.ResultURL
	if url == "" && row.StorageKey != "" {
		url = s.storage.PublicURL(row.StorageKey)
	}
	style := ""
	var paramMap map[string]any
	if err := json.Unmarshal([]byte(row.ParamsJSON), &paramMap); err == nil {
		if v, ok := paramMap["style"].(string); ok {
			style = v
		}
	}
	out := &VideoTaskResult{
		TaskID:       row.PublicID,
		JobID:        row.PublicID,
		Kind:         row.Kind,
		Status:       mapJobStatusToVideoStatus(row.Status),
		Prompt:       row.Prompt,
		Style:        style,
		URL:          url,
		RemoteURL:    row.RemoteURL,
		StorageKey:   row.StorageKey,
		Resolution:   row.Resolution,
		Ratio:        row.Ratio,
		Duration:     row.Duration,
		Width:        row.Width,
		Height:       row.Height,
		ErrorMessage: publicJobErrorMessage(row.ErrorMessage),
		Progress:     row.Progress,
		Steps:        DecodeJobSteps(row.StepsJSON),
		CreatedAt:    row.CreatedAt.UnixMilli(),
	}
	m := DecodeJobMetrics(row.MetricsJSON)
	out.Metrics = &m
	return out
}

// publicJobErrorMessage hides upstream/provider internals from client-facing payloads.
func publicJobErrorMessage(raw string) string {
	msg := strings.TrimSpace(raw)
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	sensitive := []string{
		"seedream", "seedance", "volces.com", "ark.cn-beijing",
		"context canceled", "context deadline", "api 失败", "post \"http",
	}
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			return "生成失败请联系管理员"
		}
	}
	if strings.Contains(msg, "http://") || strings.Contains(msg, "https://") || len(msg) > 120 {
		return "生成失败请联系管理员"
	}
	return msg
}

func mapJobStatusToVideoStatus(status string) string {
	switch status {
	case JobStatusPending, JobStatusQueued:
		return "queued"
	case JobStatusRunning:
		return "running"
	case JobStatusSucceeded:
		return "succeeded"
	case JobStatusFailed:
		return "failed"
	case JobStatusCanceled:
		return "cancelled"
	default:
		return status
	}
}

// WithPublicURL fills URL from storage key using the provided resolver when PublicURL is empty.
func (r *ImageGenerateResult) WithPublicURL(resolver func(key string) string) *ImageGenerateResult {
	if r == nil {
		return nil
	}
	if r.URL == "" && r.StorageKey != "" {
		r.URL = resolver(r.StorageKey)
	}
	return r
}

func (r *VideoTaskResult) WithPublicURL(resolver func(key string) string) *VideoTaskResult {
	if r == nil {
		return nil
	}
	if r.URL == "" && r.StorageKey != "" {
		r.URL = resolver(r.StorageKey)
	}
	return r
}

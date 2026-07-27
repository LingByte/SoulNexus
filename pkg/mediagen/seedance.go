package mediagen

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type SeedanceClient struct {
	cfg  Config
	http *http.Client
}

func NewSeedanceClient(cfg Config) *SeedanceClient {
	return &SeedanceClient{
		cfg: cfg,
		http: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

type VideoGenerateParams struct {
	Prompt        string
	Ratio         string
	Duration      int
	Resolution    string
	GenerateAudio *bool
	Watermark     bool
}

type seedanceCreateRequest struct {
	Model         string                 `json:"model"`
	Content       []seedanceContentItem  `json:"content"`
	GenerateAudio bool                   `json:"generate_audio"`
	Ratio         string                 `json:"ratio"`
	Duration      int                    `json:"duration"`
	Resolution    string                 `json:"resolution"`
	Watermark     bool                   `json:"watermark"`
}

type seedanceContentItem struct {
	Type     string            `json:"type"`
	Text     string            `json:"text,omitempty"`
	Role     string            `json:"role,omitempty"`
	ImageURL *seedanceImageURL `json:"image_url,omitempty"`
}

type seedanceImageURL struct {
	URL string `json:"url"`
}

type SeedanceTask struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	Status     string `json:"status"`
	Resolution string `json:"resolution"`
	Ratio      string `json:"ratio"`
	Duration   int    `json:"duration"`
	Content    *struct {
		VideoURL string `json:"video_url"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (c *SeedanceClient) CreateTextToVideoTask(ctx context.Context, params VideoGenerateParams) (string, error) {
	content := []seedanceContentItem{{Type: "text", Text: params.Prompt}}
	return c.createTask(ctx, params, content)
}

func (c *SeedanceClient) CreateImageToVideoTask(ctx context.Context, params VideoGenerateParams, firstFrameDataURI, lastFrameDataURI string) (string, error) {
	content := []seedanceContentItem{
		{Type: "text", Text: params.Prompt},
		{
			Type:     "image_url",
			Role:     "first_frame",
			ImageURL: &seedanceImageURL{URL: firstFrameDataURI},
		},
	}
	if strings.TrimSpace(lastFrameDataURI) != "" {
		content = append(content, seedanceContentItem{
			Type:     "image_url",
			Role:     "last_frame",
			ImageURL: &seedanceImageURL{URL: lastFrameDataURI},
		})
	}
	return c.createTask(ctx, params, content)
}

func (c *SeedanceClient) GetTask(ctx context.Context, taskID string) (*SeedanceTask, error) {
	if !c.cfg.Configured() {
		return nil, ErrAPIKeyMissing
	}
	url := c.cfg.SeedanceBaseURL + "/contents/generations/tasks/" + strings.TrimSpace(taskID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &UpstreamError{Message: "Seedance 查询任务失败: " + err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, upstreamHTTPError("Seedance 查询任务失败", resp.StatusCode, raw)
	}
	var task SeedanceTask
	if err := json.Unmarshal(raw, &task); err != nil {
		return nil, &UpstreamError{Message: "Seedance 任务响应解析失败"}
	}
	if task.ID == "" {
		task.ID = taskID
	}
	return &task, nil
}

func (c *SeedanceClient) createTask(ctx context.Context, params VideoGenerateParams, content []seedanceContentItem) (string, error) {
	if !c.cfg.Configured() {
		return "", ErrAPIKeyMissing
	}
	generateAudio := c.cfg.GenerateAudio
	if params.GenerateAudio != nil {
		generateAudio = *params.GenerateAudio
	}
	ratio := params.Ratio
	if ratio == "" {
		ratio = "16:9"
	}
	duration := params.Duration
	if duration < 5 {
		duration = 5
	}
	if duration > 10 {
		duration = 10
	}
	resolution := params.Resolution
	if resolution == "" {
		resolution = "1080p"
	}

	body := seedanceCreateRequest{
		Model:         c.cfg.SeedanceModel,
		Content:       content,
		GenerateAudio: generateAudio,
		Ratio:         ratio,
		Duration:      duration,
		Resolution:    resolution,
		Watermark:     params.Watermark,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	url := c.cfg.SeedanceBaseURL + "/contents/generations/tasks"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", &UpstreamError{Message: "Seedance 创建任务失败: " + err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", upstreamHTTPError("Seedance 创建任务失败", resp.StatusCode, raw)
	}
	var task SeedanceTask
	if err := json.Unmarshal(raw, &task); err != nil {
		return "", &UpstreamError{Message: "Seedance 创建任务响应解析失败"}
	}
	if strings.TrimSpace(task.ID) == "" {
		return "", ErrNoTaskID
	}
	return strings.TrimSpace(task.ID), nil
}

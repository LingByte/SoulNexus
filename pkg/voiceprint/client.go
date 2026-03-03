package voiceprint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client 声纹识别客户端
type Client struct {
	config     *Config
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient 创建新的声纹识别客户端
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			MaxIdleConnsPerHost: 5,
		},
	}

	client := &Client{
		config:     config,
		httpClient: httpClient,
		logger:     zap.L().Named("voiceprint"),
	}

	return client, nil
}

// IsEnabled 检查服务是否启用
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	if !c.config.Enabled {
		return nil, ErrServiceDisabled
	}

	url := fmt.Sprintf("%s/voiceprint/health?key=%s", c.config.BaseURL, c.config.APIKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create request failed: %v", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ErrAPIRequest(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var result HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	result.Timestamp = time.Now()

	if c.config.LogEnabled {
		c.logger.Info("Health check completed",
			zap.String("status", result.Status),
			zap.Int("total_voiceprints", result.TotalVoiceprints))
	}

	return &result, nil
}

// RegisterVoiceprint 注册声纹
func (c *Client) RegisterVoiceprint(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if !c.config.Enabled {
		return nil, ErrServiceDisabled
	}

	if err := c.validateRegisterRequest(req); err != nil {
		return nil, err
	}

	// 创建 multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加 speaker_id 字段
	if err := writer.WriteField("speaker_id", req.SpeakerID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write speaker_id failed: %v", err))
	}

	// 添加 assistant_id 字段
	if err := writer.WriteField("assistant_id", req.AssistantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write assistant_id failed: %v", err))
	}

	// 添加音频文件
	part, err := writer.CreateFormFile("file", req.SpeakerID+".wav")
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create form file failed: %v", err))
	}

	if _, err := part.Write(req.AudioData); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write audio data failed: %v", err))
	}

	writer.Close()

	// 创建请求
	url := fmt.Sprintf("%s/voiceprint/register", c.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create request failed: %v", err))
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrRegistrationFailed(req.SpeakerID, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ErrRegistrationFailed(req.SpeakerID, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	result.SpeakerID = req.SpeakerID
	result.Timestamp = time.Now()

	if c.config.LogEnabled {
		c.logger.Info("Voiceprint registered successfully",
			zap.String("speaker_id", req.SpeakerID),
			zap.String("assistant_id", req.AssistantID),
			zap.Bool("success", result.Success),
			zap.Duration("duration", time.Since(startTime)))
	}

	return &result, nil
}

// IdentifyVoiceprint 识别声纹
func (c *Client) IdentifyVoiceprint(ctx context.Context, req *IdentifyRequest) (*IdentifyResponse, error) {
	if !c.config.Enabled {
		return nil, ErrServiceDisabled
	}

	if err := c.validateIdentifyRequest(req); err != nil {
		return nil, err
	}

	// 创建 multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加候选人ID
	speakerIDs := strings.Join(req.CandidateIDs, ",")
	if err := writer.WriteField("speaker_ids", speakerIDs); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write speaker_ids failed: %v", err))
	}

	// 添加助手ID
	if err := writer.WriteField("assistant_id", req.AssistantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write assistant_id failed: %v", err))
	}

	// 添加音频文件
	part, err := writer.CreateFormFile("file", "identify.wav")
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create form file failed: %v", err))
	}

	if _, err := part.Write(req.AudioData); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write audio data failed: %v", err))
	}

	writer.Close()

	// 创建请求
	url := fmt.Sprintf("%s/voiceprint/identify", c.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create request failed: %v", err))
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// 发送请求
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrIdentificationFailed(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ErrIdentificationFailed(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var result IdentifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	result.Timestamp = time.Now()
	result.Confidence = c.getConfidenceLevel(result.Score)

	if c.config.LogEnabled {
		c.logger.Info("Voiceprint identified",
			zap.String("speaker_id", result.SpeakerID),
			zap.String("assistant_id", req.AssistantID),
			zap.Float64("score", result.Score),
			zap.String("confidence", result.Confidence),
			zap.Int("candidates", len(req.CandidateIDs)),
			zap.Duration("duration", time.Since(startTime)))
	}

	return &result, nil
}

// DeleteVoiceprint 删除声纹
func (c *Client) DeleteVoiceprint(ctx context.Context, speakerID string, assistantID ...string) (*DeleteResponse, error) {
	if !c.config.Enabled {
		return nil, ErrServiceDisabled
	}

	if speakerID == "" {
		return nil, ErrInvalidConfig("speaker_id is required")
	}

	url := fmt.Sprintf("%s/voiceprint/%s", c.config.BaseURL, speakerID)

	// 如果指定了助手ID，添加到请求体中
	var reqBody io.Reader
	if len(assistantID) > 0 && assistantID[0] != "" {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("assistant_id", assistantID[0])
		writer.Close()
		reqBody = &buf
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, reqBody)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create request failed: %v", err))
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrDeletionFailed(speakerID, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ErrDeletionFailed(speakerID, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var result DeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	result.SpeakerID = speakerID
	result.Timestamp = time.Now()

	if c.config.LogEnabled {
		assistantInfo := ""
		if len(assistantID) > 0 && assistantID[0] != "" {
			assistantInfo = fmt.Sprintf(" (assistant: %s)", assistantID[0])
		}
		c.logger.Info("Voiceprint deleted",
			zap.String("speaker_id", speakerID),
			zap.String("assistant_info", assistantInfo),
			zap.Bool("success", result.Success),
			zap.Duration("duration", time.Since(startTime)))
	}

	return &result, nil
}

// validateRegisterRequest 验证注册请求
func (c *Client) validateRegisterRequest(req *RegisterRequest) error {
	if req.SpeakerID == "" {
		return ErrInvalidConfig("speaker_id is required")
	}

	if req.AssistantID == "" {
		return ErrInvalidConfig("assistant_id is required")
	}

	if len(req.AudioData) == 0 {
		return ErrInvalidConfig("audio_data is required")
	}

	// 检查音频格式（简单检查WAV文件头）
	if len(req.AudioData) < 12 || string(req.AudioData[0:4]) != "RIFF" || string(req.AudioData[8:12]) != "WAVE" {
		return ErrInvalidAudioFormat
	}

	return nil
}

// validateIdentifyRequest 验证识别请求
func (c *Client) validateIdentifyRequest(req *IdentifyRequest) error {
	if len(req.CandidateIDs) == 0 {
		return ErrInvalidConfig("candidate_ids is required")
	}

	if req.AssistantID == "" {
		return ErrInvalidConfig("assistant_id is required")
	}

	if len(req.CandidateIDs) > c.config.MaxCandidates {
		return ErrTooManyCandidates
	}

	if len(req.AudioData) == 0 {
		return ErrInvalidConfig("audio_data is required")
	}

	// 检查音频格式
	if len(req.AudioData) < 12 || string(req.AudioData[0:4]) != "RIFF" || string(req.AudioData[8:12]) != "WAVE" {
		return ErrInvalidAudioFormat
	}

	return nil
}

// getConfidenceLevel 根据分数获取置信度等级
func (c *Client) getConfidenceLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "very_high"
	case score >= 0.6:
		return "high"
	case score >= 0.4:
		return "medium"
	case score >= 0.2:
		return "low"
	default:
		return "very_low"
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	// 清理资源
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

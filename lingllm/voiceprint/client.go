package voiceprint

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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

func resolveRequestTenantID(tenantID, agentID string) string {
	if strings.TrimSpace(tenantID) != "" {
		return strings.TrimSpace(tenantID)
	}
	return strings.TrimSpace(agentID)
}

func writeOptionalFormField(writer *multipart.Writer, key, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return writer.WriteField(key, strings.TrimSpace(value))
}

func resolveRegisterFeatureID(req *RegisterRequest) string {
	if req == nil {
		return ""
	}
	if id := strings.TrimSpace(req.FeatureID); id != "" {
		return id
	}
	return strings.TrimSpace(req.SpeakerID)
}

func resolveRegisterName(req *RegisterRequest) string {
	if req == nil {
		return ""
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		return name
	}
	if req.Metadata == nil {
		return ""
	}
	if raw, ok := req.Metadata["name"]; ok {
		if name, ok := raw.(string); ok {
			return strings.TrimSpace(name)
		}
	}
	return ""
}

func resolveRegisterProvider(req *RegisterRequest) string {
	if req == nil {
		return "http"
	}
	if provider := strings.TrimSpace(req.Provider); provider != "" {
		return provider
	}
	return "http"
}

func resolveRegisterStatus(req *RegisterRequest) string {
	if req == nil {
		return "active"
	}
	if status := strings.TrimSpace(req.Status); status != "" {
		return status
	}
	return "active"
}

func normalizeIdentifyResponse(resp *IdentifyResponse) {
	if resp == nil {
		return
	}
	if resp.FeatureID == "" {
		resp.FeatureID = strings.TrimSpace(resp.SpeakerID)
	}
	if resp.SpeakerID == "" {
		resp.SpeakerID = strings.TrimSpace(resp.FeatureID)
	}
}

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

	// 添加 feature_id 字段（与 VoiceprintProfile 对齐）
	featureID := resolveRegisterFeatureID(req)
	if err := writer.WriteField("feature_id", featureID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write feature_id failed: %v", err))
	}
	// 兼容旧微服务字段
	if err := writer.WriteField("speaker_id", featureID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write speaker_id failed: %v", err))
	}

	// 添加 tenant_id 字段
	tenantID := resolveRequestTenantID(req.TenantID, req.AgentID)
	if err := writer.WriteField("tenant_id", tenantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write tenant_id failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "assistant_id", req.AssistantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write assistant_id failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "profile_id", req.ProfileID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write profile_id failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "name", resolveRegisterName(req)); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write name failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "provider", resolveRegisterProvider(req)); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write provider failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "status", resolveRegisterStatus(req)); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write status failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "description", req.Description); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write description failed: %v", err))
	}

	// 添加音频文件
	part, err := writer.CreateFormFile("file", featureID+".wav")
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
		return nil, ErrRegistrationFailed(featureID, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ErrRegistrationFailed(featureID, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}

	result.SpeakerID = featureID
	result.Timestamp = time.Now()

	if c.config.LogEnabled {
		c.logger.Info("Voiceprint registered successfully",
			zap.String("feature_id", featureID),
			zap.String("tenant_id", tenantID),
			zap.String("assistant_id", req.AssistantID),
			zap.String("name", resolveRegisterName(req)),
			zap.String("provider", resolveRegisterProvider(req)),
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

	// 添加候选人 feature_id
	candidateIDs := strings.Join(req.CandidateIDs, ",")
	if err := writer.WriteField("feature_ids", candidateIDs); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write feature_ids failed: %v", err))
	}
	if err := writer.WriteField("speaker_ids", candidateIDs); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write speaker_ids failed: %v", err))
	}

	// 添加租户与可选助手
	tenantID := resolveRequestTenantID(req.TenantID, req.AgentID)
	if err := writer.WriteField("tenant_id", tenantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write tenant_id failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "assistant_id", req.AssistantID); err != nil {
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
	normalizeIdentifyResponse(&result)

	if c.config.LogEnabled {
		c.logger.Info("Voiceprint identified",
			zap.String("feature_id", result.FeatureID),
			zap.String("tenant_id", tenantID),
			zap.String("assistant_id", req.AssistantID),
			zap.Float64("score", result.Score),
			zap.String("confidence", result.Confidence),
			zap.Int("candidates", len(req.CandidateIDs)),
			zap.Duration("duration", time.Since(startTime)))
	}

	return &result, nil
}

// DeleteVoiceprint 删除声纹
func (c *Client) DeleteVoiceprint(ctx context.Context, speakerID string, scope ...string) (*DeleteResponse, error) {
	if !c.config.Enabled {
		return nil, ErrServiceDisabled
	}

	if speakerID == "" {
		return nil, ErrInvalidConfig("speaker_id is required")
	}

	tenantID := ""
	assistantID := ""
	if len(scope) > 0 {
		tenantID = strings.TrimSpace(scope[0])
	}
	if len(scope) > 1 {
		assistantID = strings.TrimSpace(scope[1])
	}
	if tenantID == "" {
		return nil, ErrInvalidConfig("tenant_id is required")
	}

	url := fmt.Sprintf("%s/voiceprint/%s", c.config.BaseURL, speakerID)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("tenant_id", tenantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write tenant_id failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "assistant_id", assistantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write assistant_id failed: %v", err))
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, &buf)
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
		c.logger.Info("Voiceprint deleted",
			zap.String("speaker_id", speakerID),
			zap.String("tenant_id", tenantID),
			zap.String("assistant_id", assistantID),
			zap.Bool("success", result.Success),
			zap.Duration("duration", time.Since(startTime)))
	}

	return &result, nil
}

// BindVoiceprintAssistant 绑定或解绑助手
func (c *Client) BindVoiceprintAssistant(ctx context.Context, tenantID, speakerID, assistantID string) (*DeleteResponse, error) {
	if !c.config.Enabled {
		return nil, ErrServiceDisabled
	}
	tenantID = strings.TrimSpace(tenantID)
	speakerID = strings.TrimSpace(speakerID)
	if tenantID == "" || speakerID == "" {
		return nil, ErrInvalidConfig("tenant_id and speaker_id are required")
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("tenant_id", tenantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write tenant_id failed: %v", err))
	}
	if err := writeOptionalFormField(writer, "assistant_id", assistantID); err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("write assistant_id failed: %v", err))
	}
	writer.Close()

	url := fmt.Sprintf("%s/voiceprint/%s/assistant", c.config.BaseURL, speakerID)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, &buf)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("create request failed: %v", err))
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ErrAPIRequest(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, ErrAPIRequest(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	var result DeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ErrInvalidResponse
	}
	result.SpeakerID = speakerID
	result.Timestamp = time.Now()
	return &result, nil
}

// validateRegisterRequest 验证注册请求
func (c *Client) validateRegisterRequest(req *RegisterRequest) error {
	if resolveRegisterFeatureID(req) == "" {
		return ErrInvalidConfig("feature_id is required")
	}

	if resolveRequestTenantID(req.TenantID, req.AgentID) == "" {
		return ErrInvalidConfig("tenant_id is required")
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

	if resolveRequestTenantID(req.TenantID, req.AgentID) == "" {
		return ErrInvalidConfig("tenant_id is required")
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

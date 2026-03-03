package vad

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client VAD 服务客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// DetectRequest VAD 检测请求
type DetectRequest struct {
	AudioData   string  `json:"audio_data"`   // Base64 编码的音频数据
	AudioFormat string  `json:"audio_format"` // "pcm" 或 "opus"
	SampleRate  int     `json:"sample_rate"`
	Channels    int     `json:"channels"`
	Threshold   float64 `json:"threshold,omitempty"` // VAD 阈值（可选）
}

// DetectResponse VAD 检测响应
type DetectResponse struct {
	HaveVoice  bool    `json:"have_voice"`
	VoiceStop  bool    `json:"voice_stop"`
	SpeechProb float64 `json:"speech_prob,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// NewClient 创建新的 VAD 客户端
func NewClient(baseURL string, logger *zap.Logger) *Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// SetTimeout 设置 HTTP 超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// HealthCheck 健康检查
func (c *Client) HealthCheck() error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.logger.Error("health check failed", zap.Error(err))
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	var healthResp HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	c.logger.Info("VAD service health check passed", zap.String("status", healthResp.Status))
	return nil
}

// Detect 检测音频中的语音活动
func (c *Client) Detect(
	audioData []byte,
	format string,
	sessionID string,
	threshold float64,
) (*DetectResponse, error) {
	// Base64 编码音频数据
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	req := DetectRequest{
		AudioData:   audioBase64,
		AudioFormat: format,
		SampleRate:  16000,
		Channels:    1,
		Threshold:   threshold,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("failed to marshal request", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/vad?session_id=%s", c.baseURL, sessionID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.logger.Error("failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("failed to send request", zap.Error(err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("VAD service error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("VAD service error (status %d): %s", resp.StatusCode, string(body))
	}

	var detectResp DetectResponse
	if err := json.NewDecoder(resp.Body).Decode(&detectResp); err != nil {
		c.logger.Error("failed to decode response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &detectResp, nil
}

// ResetSession 重置会话状态
func (c *Client) ResetSession(sessionID string) error {
	url := fmt.Sprintf("%s/vad/reset?session_id=%s", c.baseURL, sessionID)
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		c.logger.Error("failed to reset session", zap.Error(err))
		return fmt.Errorf("failed to reset session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reset session failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("session reset", zap.String("session_id", sessionID))
	return nil
}

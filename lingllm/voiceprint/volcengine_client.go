// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// VolcengineClient 火山引擎声纹识别客户端
type VolcengineClient struct {
	config     *VolcengineConfig
	httpClient *http.Client
	logger     *zap.Logger
}

// VolcengineConfig 火山引擎配置
type VolcengineConfig struct {
	AccessKey string        `json:"access_key"`
	SecretKey string        `json:"secret_key"`
	Region    string        `json:"region"`
	BaseURL   string        `json:"base_url"`
	Timeout   time.Duration `json:"timeout"`
}

// NewVolcengineClient 创建火山引擎客户端
func NewVolcengineClient(config *VolcengineConfig, logger *zap.Logger) (*VolcengineClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("access_key and secret_key are required")
	}

	if config.Region == "" {
		config.Region = "cn-north-1"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://rtc.volcengineapi.com"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if logger == nil {
		logger = zap.NewNop()
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

	client := &VolcengineClient{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}

	return client, nil
}

// Register 注册声纹
func (c *VolcengineClient) Register(ctx context.Context, audio, audioName, metaInfo string) (*VolcengineRegisterResult, error) {
	req := &VolcengineRegisterRequest{
		Audio:     audio,
		AudioName: audioName,
		MetaInfo:  metaInfo,
	}

	var resp VolcengineRegisterResponse
	if err := c.sendRequest(ctx, "IotVoicePrintRegister", req, &resp); err != nil {
		return nil, err
	}

	return &resp.Result, nil
}

// Query 查询声纹
func (c *VolcengineClient) Query(ctx context.Context, uuids []string, limit int, iterator string) (*VolcengineQueryResult, error) {
	req := &VolcengineQueryRequest{
		UUIDs:    uuids,
		Limit:    limit,
		Iterator: iterator,
	}

	if limit == 0 {
		req.Limit = 50 // 默认值
	}

	var resp VolcengineQueryResponse
	if err := c.sendRequest(ctx, "IotVoicePrintQuery", req, &resp); err != nil {
		return nil, err
	}

	return &resp.Result, nil
}

// Update 更新声纹
func (c *VolcengineClient) Update(ctx context.Context, uuid, audio, audioName, metaInfo string) error {
	req := &VolcengineUpdateRequest{
		UUID:      uuid,
		Audio:     audio,
		AudioName: audioName,
		MetaInfo:  metaInfo,
	}

	var resp VolcengineUpdateResponse
	if err := c.sendRequest(ctx, "IotVoicePrintUpdate", req, &resp); err != nil {
		return err
	}

	return nil
}

// Delete 删除声纹
func (c *VolcengineClient) Delete(ctx context.Context, uuid string) error {
	req := &VolcengineDeleteRequest{
		UUID: uuid,
	}

	var resp VolcengineDeleteResponse
	if err := c.sendRequest(ctx, "IotVoicePrintDelete", req, &resp); err != nil {
		return err
	}

	return nil
}

// Identify 1:N 声纹识别
func (c *VolcengineClient) Identify(ctx context.Context, audioBase64 string, uuids []string, topK int) (*VolcengineIdentifyResult, error) {
	if topK <= 0 {
		topK = len(uuids)
	}
	if topK <= 0 {
		topK = 5
	}
	req := &VolcengineIdentifyRequest{
		Audio: audioBase64,
		UUIDs: uuids,
		TopK:  topK,
	}
	var resp VolcengineIdentifyResponse
	if err := c.sendRequest(ctx, "IotVoicePrintIdentify", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// Verify 1:1 声纹验证
func (c *VolcengineClient) Verify(ctx context.Context, audioBase64, uuid string) (*VolcengineVerifyResult, error) {
	req := &VolcengineVerifyRequest{
		Audio: audioBase64,
		UUID:  uuid,
	}
	var resp VolcengineVerifyResponse
	if err := c.sendRequest(ctx, "IotVoicePrintVerify", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Result, nil
}

// sendRequest 发送请求
func (c *VolcengineClient) sendRequest(ctx context.Context, action string, reqBody interface{}, respBody interface{}) error {
	// 构建URL
	url := fmt.Sprintf("%s?Action=%s&Version=2025-08-01", c.config.BaseURL, action)

	// 序列化请求体
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if err := signVolcengineRequest(httpReq, c.config.AccessKey, c.config.SecretKey, c.config.Region, body); err != nil {
		return err
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		var errResp VolcengineError
		if err := json.Unmarshal(respData, &errResp); err == nil && errResp.Error.Code != "" {
			return fmt.Errorf("volcengine API error: code=%s, message=%s", errResp.Error.Code, errResp.Error.Message)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respData))
	}

	// 解析响应
	if err := json.Unmarshal(respData, respBody); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// HealthCheck 健康检查
func (c *VolcengineClient) HealthCheck(ctx context.Context) error {
	// 通过查询空列表来验证连接
	_, err := c.Query(ctx, []string{}, 1, "")
	return err
}

// Close 关闭客户端
func (c *VolcengineClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

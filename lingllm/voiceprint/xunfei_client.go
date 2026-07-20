// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// XunfeiClient 讯飞声纹识别客户端
type XunfeiClient struct {
	config     *XunfeiConfig
	auth       *XunfeiAuth
	httpClient *http.Client
	logger     *zap.Logger
}

// XunfeiConfig 讯飞配置
type XunfeiConfig struct {
	APIKey    string        `json:"api_key"`
	APISecret string        `json:"api_secret"`
	AppID     string        `json:"app_id"`
	BaseURL   string        `json:"base_url"`
	Timeout   time.Duration `json:"timeout"`
}

// NewXunfeiClient 创建讯飞客户端
func NewXunfeiClient(config *XunfeiConfig, logger *zap.Logger) (*XunfeiClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.APIKey == "" || config.APISecret == "" {
		return nil, fmt.Errorf("api_key and api_secret are required")
	}

	if config.AppID == "" {
		return nil, fmt.Errorf("app_id is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.xf-yun.com/v1/private/s782b4996"
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

	auth := NewXunfeiAuth(config.APIKey, config.APISecret, "api.xf-yun.com")

	client := &XunfeiClient{
		config:     config,
		auth:       auth,
		httpClient: httpClient,
		logger:     logger,
	}

	return client, nil
}

// CreateGroup 创建声纹特征库
func (c *XunfeiClient) CreateGroup(ctx context.Context, groupID, groupName, groupInfo string) (*CreateGroupResult, error) {
	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:      "createGroup",
				GroupID:   groupID,
				GroupName: groupName,
				GroupInfo: groupInfo,
				CreateGroupRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result CreateGroupResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// CreateFeature 添加音频特征
func (c *XunfeiClient) CreateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte) (*CreateFeatureResult, error) {
	resource, err := BuildXunfeiAudioResource(audioData)
	if err != nil {
		return nil, err
	}

	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:        "createFeature",
				GroupID:     groupID,
				FeatureID:   featureID,
				FeatureInfo: featureInfo,
				CreateFeatureRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
		Payload: &XunfeiPayload{
			Resource: resource,
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result CreateFeatureResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// UpdateFeature 更新音频特征
func (c *XunfeiClient) UpdateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte, cover bool) (*UpdateFeatureResult, error) {
	resource, err := BuildXunfeiAudioResource(audioData)
	if err != nil {
		return nil, err
	}

	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:        "updateFeature",
				GroupID:     groupID,
				FeatureID:   featureID,
				FeatureInfo: featureInfo,
				Cover:       cover,
				UpdateFeatureRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
		Payload: &XunfeiPayload{
			Resource: resource,
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result UpdateFeatureResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// QueryFeatureList 查询特征列表
func (c *XunfeiClient) QueryFeatureList(ctx context.Context, groupID string) (*QueryFeatureListResult, error) {
	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:    "queryFeatureList",
				GroupID: groupID,
				QueryFeatureListRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result QueryFeatureListResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// SearchScoreFea 特征比对1:1
func (c *XunfeiClient) SearchScoreFea(ctx context.Context, groupID, dstFeatureID string, audioData []byte) (*SearchScoreFeaResult, error) {
	resource, err := BuildXunfeiAudioResource(audioData)
	if err != nil {
		return nil, err
	}

	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:         "searchScoreFea",
				GroupID:      groupID,
				DstFeatureID: dstFeatureID,
				SearchScoreFeaRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
		Payload: &XunfeiPayload{
			Resource: resource,
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result SearchScoreFeaResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// SearchFea 特征比对1:N
func (c *XunfeiClient) SearchFea(ctx context.Context, groupID string, topK int, audioData []byte) (*SearchFeaResult, error) {
	resource, err := BuildXunfeiAudioResource(audioData)
	if err != nil {
		return nil, err
	}

	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:    "searchFea",
				GroupID: groupID,
				TopK:    topK,
				SearchFeaRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
		Payload: &XunfeiPayload{
			Resource: resource,
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result SearchFeaResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeleteFeature 删除指定特征
func (c *XunfeiClient) DeleteFeature(ctx context.Context, groupID, featureID string) (*DeleteFeatureResult, error) {
	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:      "deleteFeature",
				GroupID:   groupID,
				FeatureID: featureID,
				DeleteFeatureRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result DeleteFeatureResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// DeleteGroup 删除声纹特征库
func (c *XunfeiClient) DeleteGroup(ctx context.Context, groupID string) (*DeleteGroupResult, error) {
	req := &XunfeiRequest{
		Header: XunfeiHeader{
			AppID:  c.config.AppID,
			Status: 3,
		},
		Parameter: XunfeiParameter{
			S782b4996: XunfeiServiceParam{
				Func:    "deleteGroup",
				GroupID: groupID,
				DeleteGroupRes: &ResponseFormat{
					Encoding: "utf8",
					Compress: "raw",
					Format:   "json",
				},
			},
		},
	}

	respData, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var result DeleteGroupResult
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// sendRequest 发送请求
func (c *XunfeiClient) sendRequest(ctx context.Context, req *XunfeiRequest) ([]byte, error) {
	// 生成认证URL
	authURL, err := c.auth.BuildAuthURL(c.config.BaseURL, "/v1/private/s782b4996")
	if err != nil {
		return nil, fmt.Errorf("failed to build auth URL: %w", err)
	}

	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", authURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	var xunfeiResp XunfeiResponse
	if err := json.Unmarshal(respBody, &xunfeiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查错误
	if xunfeiResp.Header.Code != 0 {
		return nil, fmt.Errorf("xunfei API error: code=%d, message=%s", xunfeiResp.Header.Code, xunfeiResp.Header.Message)
	}

	// 获取响应数据
	var textResp *TextResponse
	switch {
	case xunfeiResp.Payload.CreateGroupRes != nil:
		textResp = xunfeiResp.Payload.CreateGroupRes
	case xunfeiResp.Payload.CreateFeatureRes != nil:
		textResp = xunfeiResp.Payload.CreateFeatureRes
	case xunfeiResp.Payload.UpdateFeatureRes != nil:
		textResp = xunfeiResp.Payload.UpdateFeatureRes
	case xunfeiResp.Payload.QueryFeatureListRes != nil:
		textResp = xunfeiResp.Payload.QueryFeatureListRes
	case xunfeiResp.Payload.SearchScoreFeaRes != nil:
		textResp = xunfeiResp.Payload.SearchScoreFeaRes
	case xunfeiResp.Payload.SearchFeaRes != nil:
		textResp = xunfeiResp.Payload.SearchFeaRes
	case xunfeiResp.Payload.DeleteFeatureRes != nil:
		textResp = xunfeiResp.Payload.DeleteFeatureRes
	case xunfeiResp.Payload.DeleteGroupRes != nil:
		textResp = xunfeiResp.Payload.DeleteGroupRes
	default:
		return nil, fmt.Errorf("no response data found")
	}

	if textResp == nil || textResp.Text == "" {
		return nil, fmt.Errorf("empty response text")
	}

	// Base64解码响应数据
	decodedData, err := base64.StdEncoding.DecodeString(textResp.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response text: %w", err)
	}

	return decodedData, nil
}

// Close 关闭客户端
func (c *XunfeiClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

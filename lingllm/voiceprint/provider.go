// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"context"
	"encoding/base64"
	"fmt"
)

// Provider 声纹识别提供商类型
type Provider string

const (
	ProviderHTTP       Provider = "http"
	ProviderXunfei     Provider = "xunfei"
	ProviderVolcengine Provider = "volcengine"
)

// VoiceprintProvider 声纹识别提供商接口
type VoiceprintProvider interface {
	// 特征库管理
	CreateGroup(ctx context.Context, groupID, groupName, groupInfo string) (*CreateGroupResult, error)
	DeleteGroup(ctx context.Context, groupID string) error

	// 特征管理
	CreateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte) (*CreateFeatureResult, error)
	UpdateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte, cover bool) (*UpdateFeatureResult, error)
	DeleteFeature(ctx context.Context, groupID, featureID string) error
	QueryFeatureList(ctx context.Context, groupID string) (*QueryFeatureListResult, error)

	// 特征比对
	SearchScoreFea(ctx context.Context, groupID, dstFeatureID string, audioData []byte) (*SearchScoreFeaResult, error)
	SearchFea(ctx context.Context, groupID string, topK int, audioData []byte) (*SearchFeaResult, error)

	// 健康检查
	HealthCheck(ctx context.Context) error

	// 关闭
	Close() error
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Provider Provider               `json:"provider"`
	Options  map[string]interface{} `json:"options"`
}

// Factory 声纹识别工厂
type Factory struct{}

// NewFactory 创建工厂
func NewFactory() *Factory {
	return &Factory{}
}

// CreateProvider 创建提供商
func (f *Factory) CreateProvider(config *ProviderConfig) (VoiceprintProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch config.Provider {
	case ProviderHTTP:
		return f.createHTTPProvider(config.Options)
	case ProviderXunfei:
		return f.createXunfeiProvider(config.Options)
	case ProviderVolcengine:
		return f.createVolcengineProvider(config.Options)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// createHTTPProvider 创建 HTTP 提供商
func (f *Factory) createHTTPProvider(options map[string]interface{}) (VoiceprintProvider, error) {
	// 从 options 中提取配置
	config := &Config{
		Enabled:       true,
		MaxCandidates: 10, // 设置默认值
	}

	if baseURL, ok := options["base_url"].(string); ok && baseURL != "" {
		config.BaseURL = baseURL
	} else {
		config.BaseURL = "http://localhost:8080"
	}

	if apiKey, ok := options["api_key"].(string); ok && apiKey != "" {
		config.APIKey = apiKey
	}

	client, err := NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &HTTPProviderAdapter{client: client}, nil
}

// createXunfeiProvider 创建讯飞提供商
func (f *Factory) createXunfeiProvider(options map[string]interface{}) (VoiceprintProvider, error) {
	if options == nil {
		return nil, fmt.Errorf("options is required for xunfei provider")
	}

	apiKey, ok := options["api_key"].(string)
	if !ok || apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	apiSecret, ok := options["api_secret"].(string)
	if !ok || apiSecret == "" {
		return nil, fmt.Errorf("api_secret is required")
	}

	appID, ok := options["app_id"].(string)
	if !ok || appID == "" {
		return nil, fmt.Errorf("app_id is required")
	}

	config := &XunfeiConfig{
		APIKey:    apiKey,
		APISecret: apiSecret,
		AppID:     appID,
	}

	if baseURL, ok := options["base_url"].(string); ok && baseURL != "" {
		config.BaseURL = baseURL
	}

	client, err := NewXunfeiClient(config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create xunfei client: %w", err)
	}

	return &XunfeiProviderAdapter{client: client}, nil
}

// HTTPProviderAdapter HTTP 提供商适配器
type HTTPProviderAdapter struct {
	client *Client
}

// CreateGroup 创建特征库
func (a *HTTPProviderAdapter) CreateGroup(ctx context.Context, groupID, groupName, groupInfo string) (*CreateGroupResult, error) {
	// HTTP 客户端不支持此操作，返回错误
	return nil, fmt.Errorf("HTTP provider does not support CreateGroup")
}

// DeleteGroup 删除特征库
func (a *HTTPProviderAdapter) DeleteGroup(ctx context.Context, groupID string) error {
	return fmt.Errorf("HTTP provider does not support DeleteGroup")
}

// CreateFeature 创建特征
func (a *HTTPProviderAdapter) CreateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte) (*CreateFeatureResult, error) {
	return nil, fmt.Errorf("HTTP provider does not support CreateFeature")
}

// UpdateFeature 更新特征
func (a *HTTPProviderAdapter) UpdateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte, cover bool) (*UpdateFeatureResult, error) {
	return nil, fmt.Errorf("HTTP provider does not support UpdateFeature")
}

// DeleteFeature 删除特征
func (a *HTTPProviderAdapter) DeleteFeature(ctx context.Context, groupID, featureID string) error {
	return fmt.Errorf("HTTP provider does not support DeleteFeature")
}

// QueryFeatureList 查询特征列表
func (a *HTTPProviderAdapter) QueryFeatureList(ctx context.Context, groupID string) (*QueryFeatureListResult, error) {
	return nil, fmt.Errorf("HTTP provider does not support QueryFeatureList")
}

// SearchScoreFea 1:1比对
func (a *HTTPProviderAdapter) SearchScoreFea(ctx context.Context, groupID, dstFeatureID string, audioData []byte) (*SearchScoreFeaResult, error) {
	return nil, fmt.Errorf("HTTP provider does not support SearchScoreFea")
}

// SearchFea 1:N比对
func (a *HTTPProviderAdapter) SearchFea(ctx context.Context, groupID string, topK int, audioData []byte) (*SearchFeaResult, error) {
	return nil, fmt.Errorf("HTTP provider does not support SearchFea")
}

// HealthCheck 健康检查
func (a *HTTPProviderAdapter) HealthCheck(ctx context.Context) error {
	_, err := a.client.HealthCheck(ctx)
	return err
}

// Close 关闭
func (a *HTTPProviderAdapter) Close() error {
	return nil
}

// XunfeiProviderAdapter 讯飞提供商适配器
type XunfeiProviderAdapter struct {
	client *XunfeiClient
}

// CreateGroup 创建特征库
func (a *XunfeiProviderAdapter) CreateGroup(ctx context.Context, groupID, groupName, groupInfo string) (*CreateGroupResult, error) {
	return a.client.CreateGroup(ctx, groupID, groupName, groupInfo)
}

// DeleteGroup 删除特征库
func (a *XunfeiProviderAdapter) DeleteGroup(ctx context.Context, groupID string) error {
	_, err := a.client.DeleteGroup(ctx, groupID)
	return err
}

// CreateFeature 创建特征
func (a *XunfeiProviderAdapter) CreateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte) (*CreateFeatureResult, error) {
	return a.client.CreateFeature(ctx, groupID, featureID, featureInfo, audioData)
}

// UpdateFeature 更新特征
func (a *XunfeiProviderAdapter) UpdateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte, cover bool) (*UpdateFeatureResult, error) {
	return a.client.UpdateFeature(ctx, groupID, featureID, featureInfo, audioData, cover)
}

// DeleteFeature 删除特征
func (a *XunfeiProviderAdapter) DeleteFeature(ctx context.Context, groupID, featureID string) error {
	_, err := a.client.DeleteFeature(ctx, groupID, featureID)
	return err
}

// QueryFeatureList 查询特征列表
func (a *XunfeiProviderAdapter) QueryFeatureList(ctx context.Context, groupID string) (*QueryFeatureListResult, error) {
	return a.client.QueryFeatureList(ctx, groupID)
}

// SearchScoreFea 1:1比对
func (a *XunfeiProviderAdapter) SearchScoreFea(ctx context.Context, groupID, dstFeatureID string, audioData []byte) (*SearchScoreFeaResult, error) {
	return a.client.SearchScoreFea(ctx, groupID, dstFeatureID, audioData)
}

// SearchFea 1:N比对
func (a *XunfeiProviderAdapter) SearchFea(ctx context.Context, groupID string, topK int, audioData []byte) (*SearchFeaResult, error) {
	return a.client.SearchFea(ctx, groupID, topK, audioData)
}

// HealthCheck 健康检查
func (a *XunfeiProviderAdapter) HealthCheck(ctx context.Context) error {
	// 讯飞API没有专门的健康检查端点，可以通过创建一个临时特征库来验证
	// 这里简单返回nil，实际应用可以根据需要实现
	return nil
}

// Close 关闭
func (a *XunfeiProviderAdapter) Close() error {
	return a.client.Close()
}

// createVolcengineProvider 创建火山引擎提供商
func (f *Factory) createVolcengineProvider(options map[string]interface{}) (VoiceprintProvider, error) {
	if options == nil {
		return nil, fmt.Errorf("options is required for volcengine provider")
	}

	accessKey, ok := options["access_key"].(string)
	if !ok || accessKey == "" {
		return nil, fmt.Errorf("access_key is required")
	}

	secretKey, ok := options["secret_key"].(string)
	if !ok || secretKey == "" {
		return nil, fmt.Errorf("secret_key is required")
	}

	config := &VolcengineConfig{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	if region, ok := options["region"].(string); ok && region != "" {
		config.Region = region
	}

	if baseURL, ok := options["base_url"].(string); ok && baseURL != "" {
		config.BaseURL = baseURL
	}

	client, err := NewVolcengineClient(config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create volcengine client: %w", err)
	}

	return &VolcengineProviderAdapter{client: client}, nil
}

// VolcengineProviderAdapter 火山引擎提供商适配器
type VolcengineProviderAdapter struct {
	client *VolcengineClient
}

// CreateGroup 创建特征库 - 火山引擎不支持
func (a *VolcengineProviderAdapter) CreateGroup(ctx context.Context, groupID, groupName, groupInfo string) (*CreateGroupResult, error) {
	return nil, fmt.Errorf("volcengine provider does not support CreateGroup")
}

// DeleteGroup 删除特征库 - 火山引擎不支持
func (a *VolcengineProviderAdapter) DeleteGroup(ctx context.Context, groupID string) error {
	return fmt.Errorf("volcengine provider does not support DeleteGroup")
}

// CreateFeature 创建特征 - 映射到Register
func (a *VolcengineProviderAdapter) CreateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte) (*CreateFeatureResult, error) {
	// 将音频数据转换为Base64
	audioBase64 := encodeBase64(audioData)

	result, err := a.client.Register(ctx, audioBase64, featureID, featureInfo)
	if err != nil {
		return nil, err
	}

	return &CreateFeatureResult{
		FeatureID: result.UUID,
	}, nil
}

// UpdateFeature 更新特征 - 映射到Update
func (a *VolcengineProviderAdapter) UpdateFeature(ctx context.Context, groupID, featureID, featureInfo string, audioData []byte, cover bool) (*UpdateFeatureResult, error) {
	audioBase64 := ""
	if len(audioData) > 0 {
		audioBase64 = encodeBase64(audioData)
	}

	err := a.client.Update(ctx, featureID, audioBase64, featureID, featureInfo)
	if err != nil {
		return nil, err
	}

	return &UpdateFeatureResult{
		Message: "success",
	}, nil
}

// DeleteFeature 删除特征 - 映射到Delete
func (a *VolcengineProviderAdapter) DeleteFeature(ctx context.Context, groupID, featureID string) error {
	return a.client.Delete(ctx, featureID)
}

// QueryFeatureList 查询特征列表 - 映射到Query
func (a *VolcengineProviderAdapter) QueryFeatureList(ctx context.Context, groupID string) (*QueryFeatureListResult, error) {
	result, err := a.client.Query(ctx, nil, 0, "")
	if err != nil {
		return nil, err
	}

	features := make([]FeatureItem, len(result.VoicePrints))
	for i, vp := range result.VoicePrints {
		features[i] = FeatureItem{
			FeatureID:   vp.UUID,
			FeatureInfo: vp.MetaInfo,
		}
	}

	return &QueryFeatureListResult{
		Features: features,
	}, nil
}

// SearchScoreFea 1:1比对
func (a *VolcengineProviderAdapter) SearchScoreFea(ctx context.Context, groupID, dstFeatureID string, audioData []byte) (*SearchScoreFeaResult, error) {
	result, err := a.client.Verify(ctx, encodeBase64(audioData), dstFeatureID)
	if err != nil {
		return nil, err
	}
	return &SearchScoreFeaResult{
		Score:     result.Score,
		FeatureID: result.UUID,
	}, nil
}

// SearchFea 1:N比对
func (a *VolcengineProviderAdapter) SearchFea(ctx context.Context, groupID string, topK int, audioData []byte) (*SearchFeaResult, error) {
	result, err := a.client.Identify(ctx, encodeBase64(audioData), nil, topK)
	if err != nil {
		return nil, err
	}
	items := result.Items()
	out := make([]SearchScoreFeaResult, 0, len(items))
	for _, item := range items {
		out = append(out, SearchScoreFeaResult{
			Score:     item.Score,
			FeatureID: item.UUID,
		})
	}
	return &SearchFeaResult{ScoreList: out}, nil
}

// IdentifyAudio runs 1:N against explicit candidate UUIDs.
func (a *VolcengineProviderAdapter) IdentifyAudio(ctx context.Context, uuids []string, topK int, audioData []byte) (*SearchFeaResult, error) {
	result, err := a.client.Identify(ctx, encodeBase64(audioData), uuids, topK)
	if err != nil {
		return nil, err
	}
	items := result.Items()
	out := make([]SearchScoreFeaResult, 0, len(items))
	for _, item := range items {
		out = append(out, SearchScoreFeaResult{
			Score:     item.Score,
			FeatureID: item.UUID,
		})
	}
	return &SearchFeaResult{ScoreList: out}, nil
}

// HealthCheck 健康检查
func (a *VolcengineProviderAdapter) HealthCheck(ctx context.Context) error {
	return a.client.HealthCheck(ctx)
}

// Close 关闭
func (a *VolcengineProviderAdapter) Close() error {
	return a.client.Close()
}

// encodeBase64 编码为Base64
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

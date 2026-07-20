package voiceprint

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"time"
)

// RegisterRequest 声纹注册请求（字段与 VoiceprintProfile 对齐）
type RegisterRequest struct {
	ProfileID   string                 `json:"profile_id,omitempty"`
	FeatureID   string                 `json:"feature_id" validate:"required"`
	SpeakerID   string                 `json:"speaker_id,omitempty"` // deprecated alias of feature_id
	TenantID    string                 `json:"tenant_id" validate:"required"`
	AssistantID string                 `json:"assistant_id,omitempty"`
	AgentID     string                 `json:"agent_id,omitempty"` // deprecated alias of tenant_id
	Name        string                 `json:"name,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Description string                 `json:"description,omitempty"`
	AudioData   []byte                 `json:"-"`
	AudioFormat string                 `json:"audio_format,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterResponse 声纹注册响应
type RegisterResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"msg"`
	SpeakerID string    `json:"speaker_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// IdentifyRequest 声纹识别请求
type IdentifyRequest struct {
	CandidateIDs []string `json:"candidate_ids" validate:"required,min=1"`
	TenantID     string   `json:"tenant_id" validate:"required"`
	AssistantID  string   `json:"assistant_id,omitempty"`
	AgentID      string   `json:"agent_id,omitempty"` // deprecated alias of tenant_id
	AudioData    []byte   `json:"-"`
	AudioFormat  string   `json:"audio_format,omitempty"`
	Threshold    float64  `json:"threshold,omitempty"`
	MaxResults   int      `json:"max_results,omitempty"`
}

// IdentifyResponse 声纹识别响应
type IdentifyResponse struct {
	FeatureID  string    `json:"feature_id"`
	SpeakerID  string    `json:"speaker_id"` // deprecated alias of feature_id
	Score      float64   `json:"score"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	Confidence string    `json:"confidence,omitempty"`
}

// DeleteRequest 声纹删除请求
type DeleteRequest struct {
	SpeakerID   string `json:"speaker_id" validate:"required"`
	TenantID    string `json:"tenant_id" validate:"required"`
	AssistantID string `json:"assistant_id,omitempty"`
	AgentID     string `json:"agent_id,omitempty"` // deprecated alias of tenant_id
}

// DeleteResponse 声纹删除响应
type DeleteResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"msg"`
	SpeakerID string    `json:"speaker_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status           string    `json:"status"`
	TotalVoiceprints int       `json:"total_voiceprints"`
	Timestamp        time.Time `json:"timestamp,omitempty"`
}

// SpeakerInfo 说话人信息
type SpeakerInfo struct {
	SpeakerID    string                 `json:"speaker_id"`
	RegisterTime time.Time              `json:"register_time"`
	UpdateTime   time.Time              `json:"update_time"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// IdentifyResult 识别结果（扩展版本）
type IdentifyResult struct {
	SpeakerID   string        `json:"speaker_id"`
	Score       float64       `json:"score"`
	Confidence  string        `json:"confidence"`
	Threshold   float64       `json:"threshold"`
	IsMatch     bool          `json:"is_match"`
	ProcessTime time.Duration `json:"process_time"`
	Timestamp   time.Time     `json:"timestamp"`
}

// BatchRegisterRequest 批量注册请求
type BatchRegisterRequest struct {
	Speakers []RegisterRequest `json:"speakers" validate:"required,min=1"`
}

// BatchRegisterResponse 批量注册响应
type BatchRegisterResponse struct {
	Success   int                `json:"success"`
	Failed    int                `json:"failed"`
	Total     int                `json:"total"`
	Results   []RegisterResponse `json:"results"`
	Timestamp time.Time          `json:"timestamp"`
}

// BatchIdentifyRequest 批量识别请求
type BatchIdentifyRequest struct {
	Requests []IdentifyRequest `json:"requests" validate:"required,min=1"`
}

// BatchIdentifyResponse 批量识别响应
type BatchIdentifyResponse struct {
	Success   int                `json:"success"`
	Failed    int                `json:"failed"`
	Total     int                `json:"total"`
	Results   []IdentifyResponse `json:"results"`
	Timestamp time.Time          `json:"timestamp"`
}

// Statistics 统计信息
type Statistics struct {
	TotalSpeakers        int       `json:"total_speakers"`
	TotalIdentifications int       `json:"total_identifications"`
	SuccessRate          float64   `json:"success_rate"`
	AverageScore         float64   `json:"average_score"`
	LastActivity         time.Time `json:"last_activity"`
}

// GetConfidenceLevel 根据相似度分数获取置信度等级
func (r *IdentifyResult) GetConfidenceLevel() string {
	switch {
	case r.Score >= 0.8:
		return "very_high"
	case r.Score >= 0.6:
		return "high"
	case r.Score >= 0.4:
		return "medium"
	case r.Score >= 0.2:
		return "low"
	default:
		return "very_low"
	}
}

// IsHighConfidence 判断是否为高置信度
func (r *IdentifyResult) IsHighConfidence() bool {
	return r.Score >= 0.6
}

// IsValidMatch 判断是否为有效匹配
func (r *IdentifyResult) IsValidMatch() bool {
	return r.Score >= r.Threshold
}

// ============ Xunfei Types ============

// XunfeiRequest 讯飞请求基础结构
type XunfeiRequest struct {
	Header    XunfeiHeader    `json:"header"`
	Parameter XunfeiParameter `json:"parameter"`
	Payload   *XunfeiPayload  `json:"payload,omitempty"`
}

// XunfeiHeader 讯飞请求头
type XunfeiHeader struct {
	AppID  string `json:"app_id"`
	Status int    `json:"status"` // 3 = 一次传完
}

// XunfeiParameter 讯飞参数
type XunfeiParameter struct {
	S782b4996 XunfeiServiceParam `json:"s782b4996"`
}

// XunfeiServiceParam 讯飞服务参数
type XunfeiServiceParam struct {
	Func string `json:"func"`

	// 创建特征库参数
	GroupID   string `json:"groupId,omitempty"`
	GroupName string `json:"groupName,omitempty"`
	GroupInfo string `json:"groupInfo,omitempty"`

	// 特征相关参数
	FeatureID    string `json:"featureId,omitempty"`
	FeatureInfo  string `json:"featureInfo,omitempty"`
	DstFeatureID string `json:"dstFeatureId,omitempty"`

	// 查询参数
	TopK int `json:"topK,omitempty"`

	// 更新参数
	Cover bool `json:"cover,omitempty"`

	// 响应格式配置
	CreateGroupRes      *ResponseFormat `json:"createGroupRes,omitempty"`
	CreateFeatureRes    *ResponseFormat `json:"createFeatureRes,omitempty"`
	UpdateFeatureRes    *ResponseFormat `json:"updateFeatureRes,omitempty"`
	QueryFeatureListRes *ResponseFormat `json:"queryFeatureListRes,omitempty"`
	SearchScoreFeaRes   *ResponseFormat `json:"searchScoreFeaRes,omitempty"`
	SearchFeaRes        *ResponseFormat `json:"searchFeaRes,omitempty"`
	DeleteFeatureRes    *ResponseFormat `json:"deleteFeatureRes,omitempty"`
	DeleteGroupRes      *ResponseFormat `json:"deleteGroupRes,omitempty"`
}

// ResponseFormat 响应格式配置
type ResponseFormat struct {
	Encoding string `json:"encoding"` // utf8
	Compress string `json:"compress"` // raw
	Format   string `json:"format"`   // json
}

// XunfeiPayload 讯飞请求负载
type XunfeiPayload struct {
	Resource *AudioResource `json:"resource,omitempty"`
}

// AudioResource 音频资源
type AudioResource struct {
	Encoding   string `json:"encoding"`    // lame
	SampleRate int    `json:"sample_rate"` // 16000
	Channels   int    `json:"channels"`    // 1
	BitDepth   int    `json:"bit_depth"`   // 16
	Status     int    `json:"status"`      // 3
	Audio      string `json:"audio"`       // base64编码的音频数据
}

// XunfeiResponse 讯飞响应基础结构
type XunfeiResponse struct {
	Header  XunfeiResponseHeader  `json:"header"`
	Payload XunfeiResponsePayload `json:"payload"`
}

// XunfeiResponseHeader 讯飞响应头
type XunfeiResponseHeader struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	SID     string `json:"sid"`
}

// XunfeiResponsePayload 讯飞响应负载
type XunfeiResponsePayload struct {
	CreateGroupRes      *TextResponse `json:"createGroupRes,omitempty"`
	CreateFeatureRes    *TextResponse `json:"createFeatureRes,omitempty"`
	UpdateFeatureRes    *TextResponse `json:"updateFeatureRes,omitempty"`
	QueryFeatureListRes *TextResponse `json:"queryFeatureListRes,omitempty"`
	SearchScoreFeaRes   *TextResponse `json:"searchScoreFeaRes,omitempty"`
	SearchFeaRes        *TextResponse `json:"searchFeaRes,omitempty"`
	DeleteFeatureRes    *TextResponse `json:"deleteFeatureRes,omitempty"`
	DeleteGroupRes      *TextResponse `json:"deleteGroupRes,omitempty"`
}

// TextResponse 文本响应
type TextResponse struct {
	Status string `json:"status,omitempty"`
	Text   string `json:"text"`
}

// CreateGroupResult 创建特征库结果
type CreateGroupResult struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
	GroupInfo string `json:"groupInfo"`
}

// CreateFeatureResult 创建特征结果
type CreateFeatureResult struct {
	FeatureID string `json:"featureId"`
}

// UpdateFeatureResult 更新特征结果
type UpdateFeatureResult struct {
	Message string `json:"msg"`
}

// QueryFeatureListResult 查询特征列表结果
type QueryFeatureListResult struct {
	Features []FeatureItem `json:"features"`
}

// FeatureItem 特征项
type FeatureItem struct {
	FeatureID   string `json:"featureId"`
	FeatureInfo string `json:"featureInfo"`
}

// SearchScoreFeaResult 1:1比对结果
type SearchScoreFeaResult struct {
	Score       float64 `json:"score"`
	FeatureID   string  `json:"featureId"`
	FeatureInfo string  `json:"featureInfo"`
}

// SearchFeaResult 1:N比对结果
type SearchFeaResult struct {
	ScoreList []SearchScoreFeaResult `json:"scoreList"`
}

// DeleteFeatureResult 删除特征结果
type DeleteFeatureResult struct {
	Message string `json:"msg"`
}

// DeleteGroupResult 删除特征库结果
type DeleteGroupResult struct {
	Message string `json:"msg"`
}

// UnmarshalJSON 自定义JSON解析，处理数组响应
func (q *QueryFeatureListResult) UnmarshalJSON(data []byte) error {
	var features []FeatureItem
	if err := json.Unmarshal(data, &features); err != nil {
		return err
	}
	q.Features = features
	return nil
}

// UnmarshalJSON 自定义JSON解析，处理数组响应
func (s *SearchFeaResult) UnmarshalJSON(data []byte) error {
	var scoreList []SearchScoreFeaResult
	if err := json.Unmarshal(data, &scoreList); err != nil {
		return err
	}
	s.ScoreList = scoreList
	return nil
}

// ============ Volcengine Types ============

// VolcengineRegisterRequest 火山引擎注册请求
type VolcengineRegisterRequest struct {
	Audio     string `json:"Audio"`     // Base64编码的WAV音频数据
	MetaInfo  string `json:"MetaInfo"`  // 声纹元信息，可选
	AudioName string `json:"AudioName"` // 音频样本名称
}

// VolcengineRegisterResponse 火山引擎注册响应
type VolcengineRegisterResponse struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Result           VolcengineRegisterResult   `json:"Result"`
}

// VolcengineRegisterResult 注册结果
type VolcengineRegisterResult struct {
	UUID string `json:"UUID"` // 注册成功的声纹唯一标识符
}

// VolcengineQueryRequest 火山引擎查询请求
type VolcengineQueryRequest struct {
	UUIDs    []string `json:"UUIDs,omitempty"`    // 声纹ID列表，可选
	Limit    int      `json:"Limit,omitempty"`    // 每页返回的最大结果数量，默认50
	Iterator string   `json:"Iterator,omitempty"` // 分页标识
}

// VolcengineQueryResponse 火山引擎查询响应
type VolcengineQueryResponse struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Result           VolcengineQueryResult      `json:"Result"`
}

// VolcengineQueryResult 查询结果
type VolcengineQueryResult struct {
	VoicePrints  []VolcengineVoicePrint `json:"VoicePrints,omitempty"`  // 声纹对象数组
	NextIterator string                 `json:"NextIterator,omitempty"` // 下一页的分页标识
}

// VolcengineVoicePrint 声纹对象
type VolcengineVoicePrint struct {
	UUID       string `json:"UUID,omitempty"`       // 声纹唯一标识符
	MetaInfo   string `json:"MetaInfo,omitempty"`   // 声纹元信息
	AccountID  string `json:"AccountID,omitempty"`  // 火山引擎账户ID
	AudioName  string `json:"AudioName,omitempty"`  // 音频样本名称
	CreatedAt  int64  `json:"CreatedAt,omitempty"`  // 创建时间，Unix时间戳（毫秒）
	UpdatedAt  int64  `json:"UpdatedAt,omitempty"`  // 最后更新时间，Unix时间戳（毫秒）
	VoicePrint string `json:"VoicePrint,omitempty"` // 声纹特征数据
}

// VolcengineUpdateRequest 火山引擎更新请求
type VolcengineUpdateRequest struct {
	UUID      string `json:"UUID"`                // 要更新的声纹唯一标识符
	Audio     string `json:"Audio,omitempty"`     // Base64编码的WAV音频数据，可选
	MetaInfo  string `json:"MetaInfo,omitempty"`  // 声纹元信息，可选
	AudioName string `json:"AudioName,omitempty"` // 音频样本名称，可选
}

// VolcengineUpdateResponse 火山引擎更新响应
type VolcengineUpdateResponse struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Result           interface{}                `json:"Result"`
}

// VolcengineDeleteRequest 火山引擎删除请求
type VolcengineDeleteRequest struct {
	UUID string `json:"UUID"` // 要删除的声纹唯一标识符
}

// VolcengineDeleteResponse 火山引擎删除响应
type VolcengineDeleteResponse struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Result           interface{}                `json:"Result"`
}

// VolcengineIdentifyRequest 1:N 识别请求
type VolcengineIdentifyRequest struct {
	Audio string   `json:"Audio"`
	UUIDs []string `json:"UUIDs,omitempty"`
	TopK  int      `json:"TopK,omitempty"`
}

// VolcengineIdentifyResponse 1:N 识别响应
type VolcengineIdentifyResponse struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Result           VolcengineIdentifyResult   `json:"Result"`
}

// VolcengineIdentifyResult 1:N 识别结果
type VolcengineIdentifyResult struct {
	ScoreList []VolcengineScoreItem `json:"ScoreList,omitempty"`
	Matches   []VolcengineScoreItem `json:"Matches,omitempty"`
}

// VolcengineScoreItem 比对得分项
type VolcengineScoreItem struct {
	UUID  string  `json:"UUID,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

func (r *VolcengineIdentifyResult) Items() []VolcengineScoreItem {
	if len(r.ScoreList) > 0 {
		return r.ScoreList
	}
	return r.Matches
}

// VolcengineVerifyRequest 1:1 验证请求
type VolcengineVerifyRequest struct {
	Audio string `json:"Audio"`
	UUID  string `json:"UUID"`
}

// VolcengineVerifyResponse 1:1 验证响应
type VolcengineVerifyResponse struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Result           VolcengineVerifyResult     `json:"Result"`
}

// VolcengineVerifyResult 1:1 验证结果
type VolcengineVerifyResult struct {
	UUID  string  `json:"UUID,omitempty"`
	Score float64 `json:"Score,omitempty"`
}

// VolcengineResponseMetadata 响应元数据
type VolcengineResponseMetadata struct {
	Action    string `json:"Action"`    // 接口名称
	Region    string `json:"Region"`    // 区域
	Service   string `json:"Service"`   // 服务名称
	Version   string `json:"Version"`   // 接口版本
	RequestID string `json:"RequestId"` // 请求ID
}

// VolcengineError 火山引擎错误响应
type VolcengineError struct {
	ResponseMetadata VolcengineResponseMetadata `json:"ResponseMetadata"`
	Error            VolcengineErrorDetail      `json:"Error"`
}

// VolcengineErrorDetail 错误详情
type VolcengineErrorDetail struct {
	Code    string `json:"Code"`    // 错误码
	Message string `json:"Message"` // 错误信息
}

// VolcengineVoicePrintInfo 声纹信息（用于统一接口）
type VolcengineVoicePrintInfo struct {
	UUID       string    // 声纹唯一标识符
	MetaInfo   string    // 声纹元信息
	AudioName  string    // 音频样本名称
	CreatedAt  time.Time // 创建时间
	UpdatedAt  time.Time // 最后更新时间
	VoicePrint string    // 声纹特征数据
}

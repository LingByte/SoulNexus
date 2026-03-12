package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// Assistant 表示一个自定义的 AI 助手
type Assistant struct {
	ID                   int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID               uint      `json:"userId" gorm:"index"`
	GroupID              *uint     `json:"groupId,omitempty" gorm:"index"` // 组织ID，如果设置则表示这是组织共享的助手
	Name                 string    `json:"name" gorm:"index"`
	Description          string    `json:"description"`
	Icon                 string    `json:"icon"`
	SystemPrompt         string    `json:"systemPrompt"`
	PersonaTag           string    `json:"personaTag"`
	Temperature          float32   `json:"temperature"`
	JsSourceID           string    `json:"jsSourceId" gorm:"index:idx_assistant_js_source"` // 关联的JS模板ID
	MaxTokens            int       `json:"maxTokens"`
	Language             string    `json:"language" gorm:"column:language"`                                     // 语言设置
	Speaker              string    `json:"speaker" gorm:"column:speaker"`                                       // 发音人ID
	VoiceCloneID         *int      `json:"voiceCloneId" gorm:"column:voice_clone_id"`                           // 训练音色ID（可选）
	KnowledgeBaseID      *string   `json:"knowledgeBaseId" gorm:"column:knowledge_base_id"`                     // 知识库ID（可选）
	TtsProvider          string    `json:"ttsProvider" gorm:"column:tts_provider"`                              // TTS提供商
	ApiKey               string    `json:"apiKey" gorm:"column:api_key"`                                        // API密钥
	ApiSecret            string    `json:"apiSecret" gorm:"column:api_secret"`                                  // API密钥
	LLMModel             string    `json:"llmModel" gorm:"column:llm_model"`                                    // LLM模型名称
	EnableGraphMemory    bool      `json:"enableGraphMemory" gorm:"column:enable_graph_memory;default:false"`   // 是否启用基于图数据库的长期记忆
	EnableVAD            bool      `json:"enableVAD" gorm:"column:enable_vad;default:true"`                     // 是否启用VAD（语音活动检测）用于打断TTS
	VADThreshold         float64   `json:"vadThreshold" gorm:"column:vad_threshold;default:500"`                // VAD阈值（RMS值，范围0-32768，默认500）
	VADConsecutiveFrames int       `json:"vadConsecutiveFrames" gorm:"column:vad_consecutive_frames;default:2"` // 需要连续超过阈值的帧数（默认2帧，约40ms）
	EnableJSONOutput     bool      `json:"enableJSONOutput" gorm:"column:enable_json_output;default:false"`     // 是否启用JSON格式化输出
	CreatedAt            time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt            time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// AssistantTool 表示助手自定义的Function Tool
type AssistantTool struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	AssistantID int64     `json:"assistantId" gorm:"index;not null"`     // 关联的助手ID
	Name        string    `json:"name" gorm:"size:100;not null"`         // 工具名称（唯一标识）
	Description string    `json:"description" gorm:"type:text"`          // 工具描述
	Parameters  string    `json:"parameters" gorm:"type:text"`           // JSON格式的参数定义（JSON Schema）
	Code        string    `json:"code,omitempty" gorm:"type:text"`       // 可选的代码实现（用于自定义执行逻辑：weather, calculator等）
	WebhookURL  string    `json:"webhookUrl,omitempty" gorm:"type:text"` // Webhook URL（用于自定义工具执行）
	Enabled     bool      `json:"enabled" gorm:"default:true"`           // 是否启用
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (AssistantTool) TableName() string {
	return "assistant_tools"
}

// ChatSessionLog 表示一次聊天会话记录
type ChatSessionLog struct {
	ID           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID    string `json:"sessionId" gorm:"index"` // 会话ID，用于关联同一次对话
	UserID       uint   `json:"userId" gorm:"index"`
	AssistantID  int64  `json:"assistantId" gorm:"index"` // 关联的助手ID
	ChatType     string `json:"chatType"`                 // 聊天类型：realtime(实时通话), press(按住说话), text(文本聊天)
	UserMessage  string `json:"userMessage"`              // 用户消息
	AgentMessage string `json:"agentMessage"`             // AI助手回复
	AudioURL     string `json:"audioUrl,omitempty"`       // 音频URL（如果有）
	Duration     int    `json:"duration,omitempty"`       // 通话时长（秒）

	// LLM Usage 信息
	LLMUsage string `json:"llmUsage,omitempty" gorm:"type:text"` // LLM使用信息的JSON字符串

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// LLMUsage 记录LLM调用的详细信息
type LLMUsage struct {
	// Request Information
	Model               string   `json:"model"`
	MaxTokens           *int     `json:"maxTokens,omitempty"`
	MaxCompletionTokens *int     `json:"maxCompletionTokens,omitempty"`
	Temperature         *float32 `json:"temperature,omitempty"`
	TopP                *float32 `json:"topP,omitempty"`
	FrequencyPenalty    *float32 `json:"frequencyPenalty,omitempty"`
	PresencePenalty     *float32 `json:"presencePenalty,omitempty"`
	Stop                []string `json:"stop,omitempty"`
	N                   *int     `json:"n,omitempty"`
	User                string   `json:"user,omitempty"`
	Stream              bool     `json:"stream"`
	Seed                *int     `json:"seed,omitempty"`

	// Response Information
	ResponseID       string `json:"responseId,omitempty"`
	Object           string `json:"object,omitempty"`
	Created          int64  `json:"created,omitempty"`
	FinishReason     string `json:"finishReason,omitempty"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	TotalTokens      int    `json:"totalTokens"`

	// Context Information
	SystemPrompt string `json:"systemPrompt,omitempty"`
	MessageCount int    `json:"messageCount,omitempty"`

	// Timing Information
	StartTime time.Time `json:"startTime,omitempty"` // 调用开始时间
	EndTime   time.Time `json:"endTime,omitempty"`   // 调用结束时间
	Duration  int64     `json:"duration,omitempty"`  // 调用持续时间（毫秒）

	// Tool Call Information
	HasToolCalls  bool           `json:"hasToolCalls,omitempty"`  // 是否调用了工具
	ToolCallCount int            `json:"toolCallCount,omitempty"` // 工具调用数量
	ToolCalls     []ToolCallInfo `json:"toolCalls,omitempty"`     // 工具调用详情
}

// ConvertLLMUsageInfoToLLMUsage 从 pkg/llm 的 LLMUsageInfo 转换为 models.LLMUsage
func ConvertLLMUsageInfoToLLMUsage(usageInfo interface{}) *LLMUsage {
	// 使用类型断言或反射来转换
	// 这里我们通过JSON序列化/反序列化来实现转换，因为两个结构体字段相似
	usageJSON, err := json.Marshal(usageInfo)
	if err != nil {
		return nil
	}

	var usage LLMUsage
	if err := json.Unmarshal(usageJSON, &usage); err != nil {
		return nil
	}

	return &usage
}

// 聊天类型常量
const (
	ChatTypeRealtime = "realtime" // 实时通话
	ChatTypePress    = "press"    // 按住说话
	ChatTypeText     = "text"     // 文本聊天
)

// CreateChatSessionLog 创建聊天记录
func CreateChatSessionLog(db *gorm.DB, userID uint, assistantID int64, chatType, sessionID, userMessage, agentMessage, audioURL string, duration int) (*ChatSessionLog, error) {
	return CreateChatSessionLogWithUsage(db, userID, assistantID, chatType, sessionID, userMessage, agentMessage, audioURL, duration, nil)
}

// CreateChatSessionLogWithUsage 创建聊天记录（包含LLM Usage信息）
func CreateChatSessionLogWithUsage(db *gorm.DB, userID uint, assistantID int64, chatType, sessionID, userMessage, agentMessage, audioURL string, duration int, usage *LLMUsage) (*ChatSessionLog, error) {
	// 清理消息中的 emoji，避免数据库字符集不兼容问题
	cleanedUserMessage := utils.RemoveEmoji(userMessage)
	cleanedAgentMessage := utils.RemoveEmoji(agentMessage)

	log := &ChatSessionLog{
		SessionID:    sessionID,
		UserID:       userID,
		AssistantID:  assistantID,
		ChatType:     chatType,
		UserMessage:  cleanedUserMessage,
		AgentMessage: cleanedAgentMessage,
		AudioURL:     audioURL,
		Duration:     duration,
	}

	// 如果有Usage信息，序列化为JSON并清理 emoji
	if usage != nil {
		usageJSON, err := json.Marshal(usage)
		if err != nil {
			return nil, fmt.Errorf("序列化LLM Usage失败: %w", err)
		}
		// 清理 JSON 中的 emoji
		cleanedJSON := utils.RemoveEmojiFromJSON(string(usageJSON))
		log.LLMUsage = cleanedJSON
	}

	if err := db.Create(log).Error; err != nil {
		return nil, err
	}

	return log, nil
}

// GetChatSessionLogs 获取用户的聊天记录列表
// 按 session_id 分组，返回每个 session 的最新记录作为预览，同时返回该 session 的消息数量
func GetChatSessionLogs(db *gorm.DB, userID uint, pageSize int, cursor int64) ([]ChatSessionLogSummary, error) {
	var logs []ChatSessionLogSummary

	// 按 session_id 分组，获取每个会话的最新记录，同时统计消息数量
	query := db.Table("chat_session_logs csl").
		Select(`
			csl.id, 
			csl.session_id, 
			csl.assistant_id, 
			a.name as assistant_name, 
			csl.chat_type, 
			COALESCE(SUBSTR(csl.user_message, 1, 50), SUBSTR(csl.agent_message, 1, 50)) as preview, 
			csl.created_at,
			(SELECT COUNT(*) FROM chat_session_logs WHERE session_id = csl.session_id AND user_id = csl.user_id) as message_count
		`).
		Joins("LEFT JOIN assistants a ON csl.assistant_id = a.id").
		Where("csl.user_id = ? AND csl.id IN (SELECT MAX(id) FROM chat_session_logs WHERE user_id = ? GROUP BY session_id)", userID, userID)

	if cursor > 0 {
		// 使用子查询找到 cursor 对应的 session_id，然后找到比它更早的 session
		query = query.Where("csl.id < ?", cursor)
	}

	err := query.Order("csl.id DESC").Limit(pageSize).Scan(&logs).Error
	return logs, err
}

// GetChatSessionLogDetail 获取聊天记录详情
func GetChatSessionLogDetail(db *gorm.DB, logID int64, userID uint) (*ChatSessionLogDetail, error) {
	fmt.Printf("GetChatSessionLogDetail: 查询 logID=%d, userID=%d\n", logID, userID)

	// 先检查记录是否存在
	var count int64
	err := db.Table("chat_session_logs").Where("id = ? AND user_id = ?", logID, userID).Count(&count).Error
	if err != nil {
		fmt.Printf("检查记录存在性失败: %v\n", err)
		return nil, err
	}
	fmt.Printf("找到记录数量: %d\n", count)

	if count == 0 {
		return nil, fmt.Errorf("record not found")
	}

	var log ChatSessionLog
	err = db.Table("chat_session_logs csl").
		Select("csl.*, a.name as assistant_name").
		Joins("LEFT JOIN assistants a ON csl.assistant_id = a.id").
		Where("csl.id = ? AND csl.user_id = ?", logID, userID).
		First(&log).Error

	if err != nil {
		fmt.Printf("查询详情失败: %v\n", err)
		return nil, err
	}

	// 获取助手名称
	assistantName := ""
	db.Table("assistants").Where("id = ?", log.AssistantID).Select("name").Scan(&assistantName)

	// 构建详情对象
	detail := ChatSessionLogDetail{
		ID:            log.ID,
		SessionID:     log.SessionID,
		AssistantID:   log.AssistantID,
		AssistantName: assistantName,
		ChatType:      log.ChatType,
		UserMessage:   log.UserMessage,
		AgentMessage:  log.AgentMessage,
		AudioURL:      log.AudioURL,
		Duration:      log.Duration,
		CreatedAt:     log.CreatedAt,
		UpdatedAt:     log.UpdatedAt,
	}

	// 解析LLM Usage信息
	if log.LLMUsage != "" {
		var usage LLMUsage
		if err := json.Unmarshal([]byte(log.LLMUsage), &usage); err == nil {
			detail.LLMUsage = &usage
		}
	}

	fmt.Printf("查询详情成功: %+v\n", detail)
	return &detail, nil
}

// GetChatSessionLogsBySession 获取指定会话的所有聊天记录
func GetChatSessionLogsBySession(db *gorm.DB, sessionID string, userID uint) ([]ChatSessionLog, error) {
	var logs []ChatSessionLog

	err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).
		Order("created_at ASC").
		Find(&logs).Error

	return logs, err
}

// ChatSessionLogSummary 聊天记录摘要（用于列表显示）
type ChatSessionLogSummary struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"sessionId"`
	AssistantID   int64     `json:"assistantId"`
	AssistantName string    `json:"assistantName"`
	ChatType      string    `json:"chatType"`
	Preview       string    `json:"preview"` // 预览文本（用户消息或AI回复的前50个字符）
	CreatedAt     time.Time `json:"createdAt"`
	MessageCount  int       `json:"messageCount"` // 该 session 下的消息数量
}

// ChatSessionLogDetail 聊天记录详情（用于详情页面）
type ChatSessionLogDetail struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"sessionId"`
	AssistantID   int64     `json:"assistantId"`
	AssistantName string    `json:"assistantName"`
	ChatType      string    `json:"chatType"`
	UserMessage   string    `json:"userMessage"`
	AgentMessage  string    `json:"agentMessage"`
	AudioURL      string    `json:"audioUrl,omitempty"`
	Duration      int       `json:"duration,omitempty"`
	LLMUsage      *LLMUsage `json:"llmUsage,omitempty"` // LLM使用信息
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type JSTemplate struct {
	ID                    string    `json:"id" gorm:"primaryKey"`
	JsSourceID            string    `json:"jsSourceId" gorm:"uniqueIndex:idx_js_templates_source_id;size:50"` // 唯一标识符，用于应用接入
	Name                  string    `json:"name" gorm:"index:idx_js_templates_name"`                          // template name
	Type                  string    `json:"type" gorm:"index:idx_js_templates_type"`                          // "default" 或 "custom"
	Content               string    `json:"content"`                                                          // content
	Usage                 string    `json:"usage"`                                                            // usage description
	UserID                uint      `json:"user_id" gorm:"index:idx_js_templates_user"`                       // user id
	GroupID               *uint     `json:"group_id,omitempty" gorm:"index"`                                  // 组织ID，如果设置则表示这是组织共享的JS模板
	Version               uint      `json:"version" gorm:"default:1"`                                         // 当前版本号
	Status                string    `json:"status" gorm:"size:32;default:'active'"`                           // 状态: active, draft, archived
	WebhookSecret         string    `json:"webhook_secret,omitempty" gorm:"size:128"`                         // Webhook签名密钥
	WebhookEnabled        bool      `json:"webhook_enabled" gorm:"default:false"`                             // 是否启用Webhook
	QuotaMaxExecutionTime int       `json:"quota_max_execution_time" gorm:"default:30000"`                    // 最大执行时间(ms)
	QuotaMaxMemoryMB      int       `json:"quota_max_memory_mb" gorm:"default:100"`                           // 最大内存(MB)
	QuotaMaxAPICalls      int       `json:"quota_max_api_calls" gorm:"default:1000"`                          // 最大API调用次数
	CreatedAt             time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// JSTemplateVersion 存储JS模板的历史版本
type JSTemplateVersion struct {
	ID          string      `json:"id" gorm:"primaryKey"`
	TemplateID  string      `json:"templateId" gorm:"index:idx_js_template_versions_template_id;not null"`
	Version     uint        `json:"version" gorm:"not null"`
	Name        string      `json:"name" gorm:"size:128"`
	Content     string      `json:"content" gorm:"type:text"`
	Status      string      `json:"status" gorm:"size:32"`       // draft, active, archived
	Grayscale   int         `json:"grayscale" gorm:"default:0"`  // 灰度百分比 0-100
	ChangeNote  string      `json:"changeNote" gorm:"type:text"` // 版本变更说明
	CreatedBy   uint        `json:"createdBy" gorm:"index"`
	CreatedAt   time.Time   `json:"createdAt" gorm:"autoCreateTime"`
	TemplateRef *JSTemplate `json:"-" gorm:"foreignKey:TemplateID"`
}

// TableName 指定数据库表名
func (JSTemplate) TableName() string {
	return "js_templates"
}

// TableName 指定版本表名
func (JSTemplateVersion) TableName() string {
	return "js_template_versions"
}

// CreateJSTemplate create a new template
func CreateJSTemplate(db *gorm.DB, template *JSTemplate) error {
	// 如果没有提供ID，生成一个唯一的UUID
	if template.ID == "" {
		template.ID = generateUUID()
	}

	// 如果没有提供jsSourceId，生成一个唯一的
	if template.JsSourceID == "" {
		template.JsSourceID = generateUniqueJsSourceID(db, template.UserID)
	}
	return db.Create(template).Error
}

// generateUniqueJsSourceID 生成唯一的jsSourceId，确保不重复
func generateUniqueJsSourceID(db *gorm.DB, userID uint) string {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		// 使用UUID + 时间戳 + 用户ID的组合确保唯一性
		jsSourceID := fmt.Sprintf("js_%d_%d_%s", userID, time.Now().UnixNano(), generateRandomString(8))

		// 检查是否已存在
		var count int64
		err := db.Model(&JSTemplate{}).Where("js_source_id = ?", jsSourceID).Count(&count).Error
		if err != nil {
			// 如果查询出错，使用时间戳重试
			continue
		}

		if count == 0 {
			return jsSourceID
		}

		// 如果存在重复，等待1微秒后重试
		time.Sleep(time.Nanosecond)
	}

	// 如果所有尝试都失败，使用UUID作为最后手段
	return fmt.Sprintf("js_%d_%s", userID, generateUUID())
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	// 使用crypto/rand生成安全的随机数
	for i := range b {
		// 生成0到len(charset)-1之间的随机索引
		randByte := make([]byte, 1)
		_, err := rand.Read(randByte)
		if err != nil {
			// 如果crypto/rand失败，使用时间戳作为后备（不推荐但比崩溃好）
			b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		} else {
			b[i] = charset[int(randByte[0])%len(charset)]
		}
	}
	return string(b)
}

// generateUUID 生成简单的UUID（使用安全的随机数）
func generateUUID() string {
	// 使用crypto/rand生成16字节的随机数
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// 如果crypto/rand失败，使用时间戳作为后备
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			time.Now().UnixNano(),
			time.Now().UnixNano()>>32,
			time.Now().UnixNano()>>16,
			time.Now().UnixNano()>>8,
			time.Now().UnixNano())
	}
	// 将随机字节转换为UUID格式
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// GetJSTemplateByJsSourceID get template by jsSourceId
func GetJSTemplateByJsSourceID(db *gorm.DB, jsSourceID string) (*JSTemplate, error) {
	var template JSTemplate
	err := db.Where("js_source_id = ?", jsSourceID).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetJSTemplateByID get template by id
func GetJSTemplateByID(db *gorm.DB, id string) (*JSTemplate, error) {
	var template JSTemplate
	err := db.Where("id = ?", id).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetJSTemplatesByName get template list by name
func GetJSTemplatesByName(db *gorm.DB, name string) ([]JSTemplate, error) {
	var templates []JSTemplate
	err := db.Where("name = ?", name).Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// ListJSTemplates 获取JS模板列表
func ListJSTemplates(db *gorm.DB, userID uint, offset, limit int) ([]JSTemplate, error) {
	var templates []JSTemplate
	query := db.Offset(offset).Limit(limit).Order("created_at DESC")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// ListJSTemplatesByType 根据类型获取JS模板列表
func ListJSTemplatesByType(db *gorm.DB, templateType string, userID uint, offset, limit int) ([]JSTemplate, error) {
	var templates []JSTemplate
	query := db.Where("type = ?", templateType).Offset(offset).Limit(limit).Order("created_at DESC")

	// 如果是自定义模板，还要根据用户ID过滤
	if templateType == "custom" && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// UpdateJSTemplate 更新JS模板
func UpdateJSTemplate(db *gorm.DB, id string, updates map[string]interface{}) error {
	return db.Model(&JSTemplate{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteJSTemplate 删除JS模板
func DeleteJSTemplate(db *gorm.DB, id string) error {
	return db.Where("id = ?", id).Delete(&JSTemplate{}).Error
}

// IsJSTemplateOwner 检查用户是否为模板的所有者
func IsJSTemplateOwner(db *gorm.DB, id string, userID uint) (bool, error) {
	var count int64
	err := db.Model(&JSTemplate{}).Where("id = ? AND user_id = ?", id, userID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetJSTemplatesCount 获取模板总数
func GetJSTemplatesCount(db *gorm.DB, templateType string, userID uint) (int64, error) {
	var count int64
	query := db.Model(&JSTemplate{})

	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}

	if userID > 0 && templateType == "custom" {
		query = query.Where("user_id = ?", userID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SearchJSTemplates 搜索JS模板
func SearchJSTemplates(db *gorm.DB, keyword string, userID uint, offset, limit int) ([]JSTemplate, error) {
	var templates []JSTemplate
	query := db.Offset(offset).Limit(limit).Order("created_at DESC")

	// 构建搜索条件
	if keyword != "" {
		searchTerm := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR content LIKE ?", searchTerm, searchTerm)
	}

	// 如果用户ID大于0，只搜索该用户的自定义模板
	if userID > 0 {
		query = query.Where("(type = 'custom' AND user_id = ?) OR type = 'default'", userID)
	}

	err := query.Find(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// ========== JSTemplateVersion 版本管理相关函数 ==========

// CreateJSTemplateVersion 创建JS模板版本
func CreateJSTemplateVersion(db *gorm.DB, version *JSTemplateVersion) error {
	return db.Create(version).Error
}

// GetJSTemplateVersions 获取模板的所有版本
func GetJSTemplateVersions(db *gorm.DB, templateID string, offset, limit int) ([]JSTemplateVersion, error) {
	var versions []JSTemplateVersion
	err := db.Where("template_id = ?", templateID).
		Offset(offset).Limit(limit).
		Order("version DESC").
		Find(&versions).Error
	return versions, err
}

// GetJSTemplateVersion 获取指定版本
func GetJSTemplateVersion(db *gorm.DB, templateID string, versionID string) (*JSTemplateVersion, error) {
	var version JSTemplateVersion
	err := db.Where("id = ? AND template_id = ?", versionID, templateID).First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// GetActiveJSTemplateVersion 获取当前激活的版本（考虑灰度）
func GetActiveJSTemplateVersion(db *gorm.DB, templateID string) (*JSTemplateVersion, error) {
	var version JSTemplateVersion
	// 优先获取灰度发布的版本，如果没有则获取状态为active的版本
	err := db.Where("template_id = ? AND status = ? AND grayscale > 0", templateID, "active").
		Order("grayscale DESC, version DESC").
		First(&version).Error
	if err != nil {
		// 如果没有灰度版本，获取普通active版本
		err = db.Where("template_id = ? AND status = ?", templateID, "active").
			Order("version DESC").
			First(&version).Error
		if err != nil {
			return nil, err
		}
	}
	return &version, nil
}

// UpdateJSTemplateVersion 更新模板版本
func UpdateJSTemplateVersion(db *gorm.DB, versionID string, updates map[string]interface{}) error {
	return db.Model(&JSTemplateVersion{}).Where("id = ?", versionID).Updates(updates).Error
}

// RollbackJSTemplateVersion 回滚到指定版本
func RollbackJSTemplateVersion(db *gorm.DB, templateID string, versionID string) error {
	// 获取要回滚的版本
	version, err := GetJSTemplateVersion(db, templateID, versionID)
	if err != nil {
		return err
	}

	// 获取当前模板
	template, err := GetJSTemplateByID(db, templateID)
	if err != nil {
		return err
	}

	// 保存当前版本到历史
	currentVersion := JSTemplateVersion{
		TemplateID: templateID,
		Version:    template.Version,
		Name:       template.Name,
		Content:    template.Content,
		Status:     template.Status,
		ChangeNote: fmt.Sprintf("Rollback to version %d", version.Version),
		CreatedBy:  template.UserID,
	}
	if err := CreateJSTemplateVersion(db, &currentVersion); err != nil {
		return err
	}

	// 恢复版本内容
	updates := map[string]interface{}{
		"content": version.Content,
		"version": template.Version + 1,
	}
	return UpdateJSTemplate(db, templateID, updates)
}

type TemplateManager struct {
	defaultTemplates map[string]string
	customTemplates  map[string]*JSTemplate
}

// GetAssistantByJSTemplateID 根据JS模板ID获取关联的助手
func GetAssistantByJSTemplateID(db *gorm.DB, jsTemplateID string) (*Assistant, error) {
	var assistant Assistant
	err := db.Where("js_source_id = ?", jsTemplateID).First(&assistant).Error
	if err != nil {
		return nil, err
	}
	return &assistant, nil
}

// ========== AssistantTool 相关函数 ==========

// GetAssistantTools 获取指定助手的所有工具
func GetAssistantTools(db *gorm.DB, assistantID int64) ([]AssistantTool, error) {
	var tools []AssistantTool
	err := db.Where("assistant_id = ? AND enabled = ?", assistantID, true).
		Order("created_at ASC").
		Find(&tools).Error
	return tools, err
}

// GetAssistantToolByID 根据ID获取工具
func GetAssistantToolByID(db *gorm.DB, toolID int64, assistantID int64) (*AssistantTool, error) {
	var tool AssistantTool
	err := db.Where("id = ? AND assistant_id = ?", toolID, assistantID).First(&tool).Error
	if err != nil {
		return nil, err
	}
	return &tool, nil
}

// CreateAssistantTool 创建新工具
func CreateAssistantTool(db *gorm.DB, tool *AssistantTool) error {
	// 检查助手是否存在
	var count int64
	if err := db.Model(&Assistant{}).Where("id = ?", tool.AssistantID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("assistant not found")
	}

	// 检查同一助手下是否已有同名工具
	var existingCount int64
	if err := db.Model(&AssistantTool{}).
		Where("assistant_id = ? AND name = ?", tool.AssistantID, tool.Name).
		Count(&existingCount).Error; err != nil {
		return err
	}
	if existingCount > 0 {
		return fmt.Errorf("tool with name '%s' already exists for this assistant", tool.Name)
	}

	return db.Create(tool).Error
}

// UpdateAssistantTool 更新工具
func UpdateAssistantTool(db *gorm.DB, toolID int64, assistantID int64, updates map[string]interface{}) error {
	// 如果更新名称，检查是否与其他工具重名
	if name, ok := updates["name"].(string); ok {
		var existingCount int64
		if err := db.Model(&AssistantTool{}).
			Where("assistant_id = ? AND name = ? AND id != ?", assistantID, name, toolID).
			Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount > 0 {
			return fmt.Errorf("tool with name '%s' already exists for this assistant", name)
		}
	}

	updates["updated_at"] = time.Now()
	return db.Model(&AssistantTool{}).
		Where("id = ? AND assistant_id = ?", toolID, assistantID).
		Updates(updates).Error
}

// DeleteAssistantTool 删除工具
func DeleteAssistantTool(db *gorm.DB, toolID int64, assistantID int64) error {
	return db.Where("id = ? AND assistant_id = ?", toolID, assistantID).
		Delete(&AssistantTool{}).Error
}

// IsAssistantToolOwner 检查工具是否属于指定助手
func IsAssistantToolOwner(db *gorm.DB, toolID int64, assistantID int64) (bool, error) {
	var count int64
	err := db.Model(&AssistantTool{}).
		Where("id = ? AND assistant_id = ?", toolID, assistantID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

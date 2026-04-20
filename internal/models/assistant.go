package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/rand"
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
	Speaker              string    `json:"speaker" gorm:"column:speaker"`                                       // 发音人ID
	VoiceCloneID         *int      `json:"voiceCloneId" gorm:"column:voice_clone_id"`                           // 训练音色ID（可选）
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

// ChatSession 聊天会话表
type ChatSession struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	UserID       string     `json:"user_id" gorm:"type:varchar(64);not null;index"`
	AssistantID  int64      `json:"assistant_id" gorm:"index"`
	Title        string     `json:"title" gorm:"type:varchar(255)"`
	Provider     string     `json:"provider" gorm:"type:varchar(50);not null"`
	Model        string     `json:"model" gorm:"type:varchar(100);not null"`
	SystemPrompt string     `json:"system_prompt" gorm:"type:text"`
	Status       string     `json:"status" gorm:"type:varchar(20);default:'active'"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// ChatMessage 聊天消息表
type ChatMessage struct {
	ID         string     `json:"id" gorm:"primaryKey;type:varchar(64)"`
	SessionID  string     `json:"session_id" gorm:"type:varchar(64);not null;index"`
	Role       string     `json:"role" gorm:"type:varchar(20);not null"`
	Content    string     `json:"content" gorm:"type:text;not null"`
	TokenCount int        `json:"token_count" gorm:"default:0"`
	Model      string     `json:"model" gorm:"type:varchar(100);not null"`
	Provider   string     `json:"provider" gorm:"type:varchar(50);not null"`
	RequestID  string     `json:"request_id" gorm:"type:varchar(64);index"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// LLMUsage LLM用量统计表
type LLMUsage struct {
	ID              string    `json:"id" gorm:"primaryKey;type:varchar(64)"`
	RequestID       string    `json:"request_id" gorm:"type:varchar(64);uniqueIndex;not null"`
	SessionID       string    `json:"session_id" gorm:"type:varchar(64);index"`
	Provider        string    `json:"provider" gorm:"type:varchar(50);not null;index"`
	Model           string    `json:"model" gorm:"type:varchar(100);not null;index"`
	BaseURL         string    `json:"base_url" gorm:"type:varchar(255)"`
	RequestType     string    `json:"request_type" gorm:"type:varchar(20);not null"`
	InputTokens     int       `json:"input_tokens" gorm:"default:0"`
	OutputTokens    int       `json:"output_tokens" gorm:"default:0"`
	TotalTokens     int       `json:"total_tokens" gorm:"default:0"`
	LatencyMs       int64     `json:"latency_ms" gorm:"default:0"`
	TTFTMs          int64     `json:"ttft_ms" gorm:"default:0"`
	TPS             float64   `json:"tps" gorm:"default:0"`
	QueueTimeMs     int64     `json:"queue_time_ms" gorm:"default:0"`
	RequestContent  string    `json:"request_content" gorm:"type:text"`
	ResponseContent string    `json:"response_content" gorm:"type:text"`
	UserAgent       string    `json:"user_agent" gorm:"type:varchar(500)"`
	IPAddress       string    `json:"ip_address" gorm:"type:varchar(45)"`
	StatusCode      int       `json:"status_code" gorm:"default:200"`
	Success         bool      `json:"success" gorm:"default:true"`
	ErrorCode       string    `json:"error_code" gorm:"type:varchar(50)"`
	ErrorMessage    string    `json:"error_message" gorm:"type:text"`
	RequestedAt     time.Time `json:"requested_at" gorm:"not null;index"`
	StartedAt       time.Time `json:"started_at" gorm:"index"`
	FirstTokenAt    time.Time `json:"first_token_at" gorm:"index"`
	CompletedAt     time.Time `json:"completed_at"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// ChatSessionLog 兼容旧接口返回结构（非持久化表）
type ChatSessionLog struct {
	ID           int64     `json:"id"`
	SessionID    string    `json:"sessionId"`
	UserID       uint      `json:"userId"`
	AssistantID  int64     `json:"assistantId"`
	ChatType     string    `json:"chatType"`
	UserMessage  string    `json:"userMessage"`
	AgentMessage string    `json:"agentMessage"`
	AudioURL     string    `json:"audioUrl,omitempty"`
	Duration     int       `json:"duration,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
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
	cleanedUserMessage := utils.RemoveEmoji(userMessage)
	cleanedAgentMessage := utils.RemoveEmoji(agentMessage)
	if sessionID == "" {
		sessionID = "session_" + utils.SnowflakeUtil.GenID()
	}

	provider := "unknown"
	model := "unknown"
	if usage != nil {
		if usage.Provider != "" {
			provider = usage.Provider
		}
		if usage.Model != "" {
			model = usage.Model
		}
	}

	session := ChatSession{
		ID:          sessionID,
		UserID:      fmt.Sprintf("%d", userID),
		AssistantID: assistantID,
		Title:       fmt.Sprintf("assistant_%d", assistantID),
		Provider:    provider,
		Model:       model,
		Status:      "active",
	}
	if err := db.Where("id = ?", sessionID).Assign(session).FirstOrCreate(&session).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	if cleanedUserMessage != "" {
		userMsg := ChatMessage{
			ID:        utils.SnowflakeUtil.GenID(),
			SessionID: sessionID,
			Role:      "user",
			Content:   cleanedUserMessage,
			Model:     model,
			Provider:  provider,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if usage != nil {
			userMsg.RequestID = usage.RequestID
		}
		if err := db.Create(&userMsg).Error; err != nil {
			return nil, err
		}
	}

	if cleanedAgentMessage != "" {
		agentMsg := ChatMessage{
			ID:         utils.SnowflakeUtil.GenID(),
			SessionID:  sessionID,
			Role:       "assistant",
			Content:    cleanedAgentMessage,
			Model:      model,
			Provider:   provider,
			TokenCount: 0,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if usage != nil {
			agentMsg.RequestID = usage.RequestID
			agentMsg.TokenCount = usage.OutputTokens
		}
		if err := db.Create(&agentMsg).Error; err != nil {
			return nil, err
		}
	}

	if usage != nil {
		if usage.ID == "" {
			usage.ID = utils.SnowflakeUtil.GenID()
		}
		if usage.RequestID == "" {
			usage.RequestID = "req_" + usage.ID
		}
		usage.SessionID = sessionID
		if usage.RequestedAt.IsZero() {
			usage.RequestedAt = now
		}
		if usage.StartedAt.IsZero() {
			usage.StartedAt = now
		}
		if usage.CompletedAt.IsZero() {
			usage.CompletedAt = now
		}
		if err := db.Where("request_id = ?", usage.RequestID).Assign(usage).FirstOrCreate(&LLMUsage{}).Error; err != nil {
			return nil, err
		}
	}

	return &ChatSessionLog{
		ID:           now.UnixMilli(),
		SessionID:    sessionID,
		UserID:       userID,
		AssistantID:  assistantID,
		ChatType:     chatType,
		UserMessage:  cleanedUserMessage,
		AgentMessage: cleanedAgentMessage,
		AudioURL:     audioURL,
		Duration:     duration,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// GetChatSessionLogs 获取用户的聊天记录列表
// 按 session_id 分组，返回每个 session 的最新记录作为预览，同时返回该 session 的消息数量
func GetChatSessionLogs(db *gorm.DB, userID uint, pageSize int, cursor int64) ([]ChatSessionLogSummary, error) {
	var sessions []ChatSession
	query := db.Where("user_id = ? AND status != ?", fmt.Sprintf("%d", userID), "deleted")
	if cursor > 0 {
		query = query.Where("updated_at < ?", time.UnixMilli(cursor))
	}
	if err := query.Order("updated_at DESC").Limit(pageSize).Find(&sessions).Error; err != nil {
		return nil, err
	}

	logs := make([]ChatSessionLogSummary, 0, len(sessions))
	for _, s := range sessions {
		var msgCount int64
		_ = db.Model(&ChatMessage{}).Where("session_id = ?", s.ID).Count(&msgCount).Error

		var lastMsg ChatMessage
		_ = db.Where("session_id = ?", s.ID).Order("created_at DESC").First(&lastMsg).Error

		assistantName := ""
		if s.AssistantID > 0 {
			var assistant Assistant
			if err := db.Select("name").Where("id = ?", s.AssistantID).First(&assistant).Error; err == nil {
				assistantName = assistant.Name
			}
		}

		preview := lastMsg.Content
		if len(preview) > 50 {
			preview = preview[:50]
		}
		logs = append(logs, ChatSessionLogSummary{
			ID:            s.UpdatedAt.UnixMilli(),
			SessionID:     s.ID,
			AssistantID:   s.AssistantID,
			AssistantName: assistantName,
			ChatType:      ChatTypeText,
			Preview:       preview,
			CreatedAt:     s.CreatedAt,
			MessageCount:  int(msgCount),
		})
	}
	return logs, nil
}

// GetChatSessionLogDetail 获取聊天记录详情
func GetChatSessionLogDetail(db *gorm.DB, logID int64, userID uint) (*ChatSessionLogDetail, error) {
	sessions, err := GetChatSessionLogs(db, userID, 200, 0)
	if err != nil {
		return nil, err
	}
	var target *ChatSessionLogSummary
	for i := range sessions {
		if sessions[i].ID == logID {
			target = &sessions[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("record not found")
	}

	logs, err := GetChatSessionLogsBySession(db, target.SessionID, userID)
	if err != nil || len(logs) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	last := logs[len(logs)-1]
	detail := ChatSessionLogDetail{
		ID:            target.ID,
		SessionID:     target.SessionID,
		AssistantID:   target.AssistantID,
		AssistantName: target.AssistantName,
		ChatType:      ChatTypeText,
		UserMessage:   last.UserMessage,
		AgentMessage:  last.AgentMessage,
		CreatedAt:     last.CreatedAt,
		UpdatedAt:     last.UpdatedAt,
	}

	var usage LLMUsage
	if err := db.Where("session_id = ?", target.SessionID).Order("created_at DESC").First(&usage).Error; err == nil {
		detail.LLMUsage = &usage
	}
	return &detail, nil
}

// GetChatSessionLogsBySession 获取指定会话的所有聊天记录
func GetChatSessionLogsBySession(db *gorm.DB, sessionID string, userID uint) ([]ChatSessionLog, error) {
	var session ChatSession
	if err := db.Where("id = ? AND user_id = ?", sessionID, fmt.Sprintf("%d", userID)).First(&session).Error; err != nil {
		return nil, err
	}

	var messages []ChatMessage
	if err := db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error; err != nil {
		return nil, err
	}

	var logs []ChatSessionLog
	for i := 0; i < len(messages); i++ {
		if messages[i].Role != "user" {
			continue
		}
		userMsg := messages[i]
		var agentMsg *ChatMessage
		for j := i + 1; j < len(messages); j++ {
			if messages[j].Role == "assistant" {
				agentMsg = &messages[j]
				break
			}
		}

		log := ChatSessionLog{
			ID:           userMsg.CreatedAt.UnixMilli(),
			SessionID:    sessionID,
			UserID:       userID,
			AssistantID:  session.AssistantID,
			ChatType:     ChatTypeText,
			UserMessage:  userMsg.Content,
			AgentMessage: "",
			CreatedAt:    userMsg.CreatedAt,
			UpdatedAt:    userMsg.UpdatedAt,
		}
		if agentMsg != nil {
			log.AgentMessage = agentMsg.Content
			log.UpdatedAt = agentMsg.UpdatedAt
		}
		logs = append(logs, log)
	}
	return logs, nil
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

func (ChatSession) TableName() string { return "chat_sessions" }
func (ChatMessage) TableName() string { return "chat_messages" }
func (LLMUsage) TableName() string    { return "llm_usage" }

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

package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// camelToSnake 将 camelCase 转换为 snake_case
func camelToSnake(str string) string {
	// 匹配大写字母前插入下划线
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// convertOrderBy 将 orderBy 字符串中的字段名从 camelCase 转换为 snake_case
func convertOrderBy(orderBy string) string {
	// 匹配字段名（字母数字下划线）后跟可选的 ASC/DESC（不区分大小写）
	re := regexp.MustCompile(`(?i)(\w+)(\s+(?:ASC|DESC))?`)

	return re.ReplaceAllStringFunc(orderBy, func(match string) string {
		parts := strings.Fields(match)
		if len(parts) == 0 {
			return match
		}

		fieldName := parts[0]
		// 如果已经是 snake_case，不需要转换
		if strings.Contains(fieldName, "_") {
			return match
		}

		snakeField := camelToSnake(fieldName)

		if len(parts) > 1 {
			return snakeField + " " + strings.ToUpper(parts[1])
		}
		return snakeField
	})
}

// UsageType 使用量类型
type UsageType string

const (
	UsageTypeLLM  UsageType = "llm"  // LLM调用
	UsageTypeCall UsageType = "call" // 通话
	UsageTypeASR  UsageType = "asr"  // 语音识别
	UsageTypeTTS  UsageType = "tts"  // 语音合成
	UsageTypeAPI  UsageType = "api"  // API调用
)

// UsageRecord 使用量记录
type UsageRecord struct {
	ID           uint    `json:"id" gorm:"primaryKey"`
	UserID       uint    `json:"userId" gorm:"index:idx_user_credential_time"`
	GroupID      *uint   `json:"groupId,omitempty" gorm:"index"` // 组织ID，如果设置则表示这是组织相关的使用量记录
	CredentialID uint    `json:"credentialId" gorm:"index:idx_user_credential_time"`
	AssistantID  *uint   `json:"assistantId,omitempty" gorm:"index"`
	SessionID    string  `json:"sessionId,omitempty" gorm:"index;size:200"`
	CallLogID    *uint64 `json:"callLogId,omitempty" gorm:"index"`

	// 使用量类型
	UsageType UsageType `json:"usageType" gorm:"index;size:50"`

	// LLM相关
	Model            string `json:"model,omitempty" gorm:"size:100"`
	PromptTokens     int    `json:"promptTokens" gorm:"default:0"`
	CompletionTokens int    `json:"completionTokens" gorm:"default:0"`
	TotalTokens      int    `json:"totalTokens" gorm:"default:0"`

	// 通话相关
	CallDuration int `json:"callDuration" gorm:"default:0"` // 通话时长（秒）
	CallCount    int `json:"callCount" gorm:"default:0"`    // 通话次数

	// ASR/TTS相关
	AudioDuration int   `json:"audioDuration" gorm:"default:0"` // 音频时长（秒）
	AudioSize     int64 `json:"audioSize" gorm:"default:0"`     // 音频大小（字节）

	// API调用相关
	APICallCount int `json:"apiCallCount" gorm:"default:0"` // API调用次数

	// 元数据
	Metadata    string `json:"metadata,omitempty" gorm:"type:text"` // JSON格式的额外信息
	Description string `json:"description,omitempty" gorm:"size:500"`

	// 时间信息
	UsageTime time.Time `json:"usageTime" gorm:"index:idx_user_credential_time"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (UsageRecord) TableName() string {
	return "usage_records"
}

// BillStatus 账单状态
type BillStatus string

const (
	BillStatusDraft     BillStatus = "draft"     // 草稿
	BillStatusGenerated BillStatus = "generated" // 已生成
	BillStatusExported  BillStatus = "exported"  // 已导出
	BillStatusArchived  BillStatus = "archived"  // 已归档
)

// Bill 账单
type Bill struct {
	ID           uint  `json:"id" gorm:"primaryKey"`
	UserID       uint  `json:"userId" gorm:"index"`
	GroupID      *uint `json:"groupId,omitempty" gorm:"index"` // 组织ID，如果设置则表示这是组织相关的账单
	CredentialID *uint `json:"credentialId,omitempty" gorm:"index"`

	// 账单基本信息
	BillNo string     `json:"billNo" gorm:"uniqueIndex;size:100"` // 账单编号
	Title  string     `json:"title" gorm:"size:200"`              // 账单标题
	Status BillStatus `json:"status" gorm:"index;size:50"`

	// 时间范围
	StartTime time.Time `json:"startTime" gorm:"index"`
	EndTime   time.Time `json:"endTime" gorm:"index"`

	// 使用量统计
	TotalLLMCalls         int64 `json:"totalLLMCalls" gorm:"default:0"`         // LLM调用次数
	TotalLLMTokens        int64 `json:"totalLLMTokens" gorm:"default:0"`        // LLM总Token数
	TotalPromptTokens     int64 `json:"totalPromptTokens" gorm:"default:0"`     // Prompt Token数
	TotalCompletionTokens int64 `json:"totalCompletionTokens" gorm:"default:0"` // Completion Token数

	TotalCallDuration int64 `json:"totalCallDuration" gorm:"default:0"` // 总通话时长（秒）
	TotalCallCount    int64 `json:"totalCallCount" gorm:"default:0"`    // 总通话次数

	TotalASRDuration int64 `json:"totalASRDuration" gorm:"default:0"` // ASR总时长（秒）
	TotalASRCount    int64 `json:"totalASRCount" gorm:"default:0"`    // ASR调用次数

	TotalTTSDuration int64 `json:"totalTTSDuration" gorm:"default:0"` // TTS总时长（秒）
	TotalTTSCount    int64 `json:"totalTTSCount" gorm:"default:0"`    // TTS调用次数

	TotalAPICalls int64 `json:"totalAPICalls" gorm:"default:0"` // 总API调用次数

	// 导出信息
	ExportFormat string     `json:"exportFormat,omitempty" gorm:"size:50"` // 导出格式：csv, excel
	ExportPath   string     `json:"exportPath,omitempty" gorm:"size:500"`  // 导出文件路径
	ExportedAt   *time.Time `json:"exportedAt,omitempty"`                  // 导出时间

	// 备注
	Notes string `json:"notes,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Bill) TableName() string {
	return "bills"
}

// UsageStatistics 使用量统计（用于查询结果）
type UsageStatistics struct {
	// 时间范围
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`

	// LLM统计
	LLMCalls         int64 `json:"llmCalls"`
	LLMTokens        int64 `json:"llmTokens"`
	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`

	// 通话统计
	CallDuration    int64   `json:"callDuration"`    // 总时长（秒）
	CallCount       int64   `json:"callCount"`       // 通话次数
	AvgCallDuration float64 `json:"avgCallDuration"` // 平均通话时长（秒）

	// ASR统计
	ASRDuration int64 `json:"asrDuration"` // ASR总时长（秒）
	ASRCount    int64 `json:"asrCount"`    // ASR调用次数

	// TTS统计
	TTSDuration int64 `json:"ttsDuration"` // TTS总时长（秒）
	TTSCount    int64 `json:"ttsCount"`    // TTS调用次数

	// API统计
	APICalls int64 `json:"apiCalls"` // API调用次数
}

// CreateUsageRecord 创建使用量记录
func CreateUsageRecord(db *gorm.DB, record *UsageRecord) error {
	return db.Create(record).Error
}

// GetUsageRecords 获取使用量记录列表
func GetUsageRecords(db *gorm.DB, userID uint, params map[string]interface{}) ([]UsageRecord, int64, error) {
	var records []UsageRecord
	var total int64

	query := db.Model(&UsageRecord{}).Where("user_id = ?", userID)

	// 按组织ID筛选
	if groupID, ok := params["groupId"].(uint); ok && groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	} else if groupID, ok := params["groupId"].(*uint); ok && groupID != nil && *groupID > 0 {
		query = query.Where("group_id = ?", *groupID)
	}

	// 按凭证ID筛选
	if credentialID, ok := params["credentialId"].(uint); ok && credentialID > 0 {
		query = query.Where("credential_id = ?", credentialID)
	}

	// 按助手ID筛选
	if assistantID, ok := params["assistantId"].(uint); ok && assistantID > 0 {
		query = query.Where("assistant_id = ?", assistantID)
	}

	// 按使用量类型筛选
	if usageType, ok := params["usageType"].(UsageType); ok && usageType != "" {
		query = query.Where("usage_type = ?", usageType)
	}

	// 按时间范围筛选
	if startTime, ok := params["startTime"].(time.Time); ok {
		query = query.Where("usage_time >= ?", startTime)
	}
	if endTime, ok := params["endTime"].(time.Time); ok {
		query = query.Where("usage_time <= ?", endTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	page := 1
	size := 20
	if p, ok := params["page"].(int); ok && p > 0 {
		page = p
	}
	if s, ok := params["size"].(int); ok && s > 0 {
		size = s
	}
	offset := (page - 1) * size

	// 排序
	orderBy := "usage_time DESC"
	if order, ok := params["orderBy"].(string); ok && order != "" {
		orderBy = convertOrderBy(order)
	}

	err := query.Order(orderBy).Offset(offset).Limit(size).Find(&records).Error
	return records, total, err
}

// GetUsageStatistics 获取使用量统计
func GetUsageStatistics(db *gorm.DB, userID uint, startTime, endTime time.Time, credentialID *uint, groupID *uint) (*UsageStatistics, error) {
	stats := &UsageStatistics{
		StartTime: startTime,
		EndTime:   endTime,
	}

	// 创建基础查询（每次创建新的查询，避免条件累积）
	createBaseQuery := func() *gorm.DB {
		query := db.Model(&UsageRecord{}).Where("user_id = ? AND usage_time >= ? AND usage_time <= ?",
			userID, startTime, endTime)
		if credentialID != nil && *credentialID > 0 {
			query = query.Where("credential_id = ?", *credentialID)
		}
		if groupID != nil && *groupID > 0 {
			// 组织账单：只统计该组织下助手的使用量
			var assistantIDs []uint
			if err := db.Table("assistants").Where("group_id = ?", *groupID).Pluck("id", &assistantIDs).Error; err == nil && len(assistantIDs) > 0 {
				query = query.Where("assistant_id IN (?)", assistantIDs)
			} else {
				// 如果组织下没有助手，返回空统计
				query = query.Where("1 = 0") // 永远不匹配的条件
			}
		}
		return query
	}

	// LLM统计
	var llmStats struct {
		Count            int64
		TotalTokens      int64
		PromptTokens     int64
		CompletionTokens int64
	}
	createBaseQuery().Where("usage_type = ?", UsageTypeLLM).
		Select("COUNT(*) as count, SUM(total_tokens) as total_tokens, SUM(prompt_tokens) as prompt_tokens, SUM(completion_tokens) as completion_tokens").
		Scan(&llmStats)
	stats.LLMCalls = llmStats.Count
	stats.LLMTokens = llmStats.TotalTokens
	stats.PromptTokens = llmStats.PromptTokens
	stats.CompletionTokens = llmStats.CompletionTokens

	// 通话统计
	var callStats struct {
		Count    int64
		Duration int64
	}
	createBaseQuery().Where("usage_type = ?", UsageTypeCall).
		Select("COUNT(*) as count, SUM(call_duration) as duration").
		Scan(&callStats)
	stats.CallCount = callStats.Count
	stats.CallDuration = callStats.Duration
	if callStats.Count > 0 {
		stats.AvgCallDuration = float64(callStats.Duration) / float64(callStats.Count)
	}

	// ASR统计
	var asrStats struct {
		Count    int64
		Duration int64
	}
	createBaseQuery().Where("usage_type = ?", UsageTypeASR).
		Select("COUNT(*) as count, SUM(audio_duration) as duration").
		Scan(&asrStats)
	stats.ASRCount = asrStats.Count
	stats.ASRDuration = asrStats.Duration

	// TTS统计
	var ttsStats struct {
		Count    int64
		Duration int64
	}
	createBaseQuery().Where("usage_type = ?", UsageTypeTTS).
		Select("COUNT(*) as count, SUM(audio_duration) as duration").
		Scan(&ttsStats)
	stats.TTSCount = ttsStats.Count
	stats.TTSDuration = ttsStats.Duration

	// API统计
	var apiStats struct {
		Count int64
	}
	createBaseQuery().Where("usage_type = ?", UsageTypeAPI).
		Select("SUM(api_call_count) as count").
		Scan(&apiStats)
	stats.APICalls = apiStats.Count

	return stats, nil
}

// DailyUsageData 每日使用量数据
type DailyUsageData struct {
	Date         string `json:"date"`         // 日期 (YYYY-MM-DD)
	LLMCalls     int64  `json:"llmCalls"`     // LLM调用次数
	LLMTokens    int64  `json:"llmTokens"`    // LLM Token数
	CallCount    int64  `json:"callCount"`    // 通话次数
	CallDuration int64  `json:"callDuration"` // 通话时长（秒）
	ASRCount     int64  `json:"asrCount"`     // ASR调用次数
	ASRDuration  int64  `json:"asrDuration"`  // ASR时长（秒）
	TTSCount     int64  `json:"ttsCount"`     // TTS调用次数
	TTSDuration  int64  `json:"ttsDuration"`  // TTS时长（秒）
	APICalls     int64  `json:"apiCalls"`     // API调用次数
}

// GetDailyUsageData 获取按日期分组的使用量数据
func GetDailyUsageData(db *gorm.DB, userID uint, startTime, endTime time.Time, credentialID *uint, groupID *uint) ([]DailyUsageData, error) {
	var results []DailyUsageData

	// 创建基础查询
	baseQuery := db.Model(&UsageRecord{}).
		Where("user_id = ? AND usage_time >= ? AND usage_time <= ?", userID, startTime, endTime)

	if credentialID != nil && *credentialID > 0 {
		baseQuery = baseQuery.Where("credential_id = ?", *credentialID)
	}
	if groupID != nil && *groupID > 0 {
		// 组织账单：只统计该组织下助手的使用量
		var assistantIDs []uint
		if err := db.Table("assistants").Where("group_id = ?", *groupID).Pluck("id", &assistantIDs).Error; err == nil && len(assistantIDs) > 0 {
			baseQuery = baseQuery.Where("assistant_id IN (?)", assistantIDs)
		} else {
			// 如果组织下没有助手，返回空数据
			baseQuery = baseQuery.Where("1 = 0") // 永远不匹配的条件
		}
	}

	// 使用 DATE() 函数按日期分组（MySQL）
	// 对于其他数据库，可能需要调整
	var dailyStats []struct {
		Date         string
		LLMCalls     int64
		LLMTokens    int64
		CallCount    int64
		CallDuration int64
		ASRCount     int64
		ASRDuration  int64
		TTSCount     int64
		TTSDuration  int64
		APICalls     int64
	}

	err := baseQuery.
		Select(`
			DATE(usage_time) as date,
			SUM(CASE WHEN usage_type = ? THEN 1 ELSE 0 END) as llm_calls,
			SUM(CASE WHEN usage_type = ? THEN total_tokens ELSE 0 END) as llm_tokens,
			SUM(CASE WHEN usage_type = ? THEN 1 ELSE 0 END) as call_count,
			SUM(CASE WHEN usage_type = ? THEN call_duration ELSE 0 END) as call_duration,
			SUM(CASE WHEN usage_type = ? THEN 1 ELSE 0 END) as asr_count,
			SUM(CASE WHEN usage_type = ? THEN audio_duration ELSE 0 END) as asr_duration,
			SUM(CASE WHEN usage_type = ? THEN 1 ELSE 0 END) as tts_count,
			SUM(CASE WHEN usage_type = ? THEN audio_duration ELSE 0 END) as tts_duration,
			SUM(CASE WHEN usage_type = ? THEN api_call_count ELSE 0 END) as api_calls
		`, UsageTypeLLM, UsageTypeLLM, UsageTypeCall, UsageTypeCall, UsageTypeASR, UsageTypeASR, UsageTypeTTS, UsageTypeTTS, UsageTypeAPI).
		Group("DATE(usage_time)").
		Order("DATE(usage_time) ASC").
		Scan(&dailyStats).Error

	if err != nil {
		return nil, err
	}

	// 转换为 DailyUsageData
	for _, stat := range dailyStats {
		results = append(results, DailyUsageData{
			Date:         stat.Date,
			LLMCalls:     stat.LLMCalls,
			LLMTokens:    stat.LLMTokens,
			CallCount:    stat.CallCount,
			CallDuration: stat.CallDuration,
			ASRCount:     stat.ASRCount,
			ASRDuration:  stat.ASRDuration,
			TTSCount:     stat.TTSCount,
			TTSDuration:  stat.TTSDuration,
			APICalls:     stat.APICalls,
		})
	}

	return results, nil
}

// CreateBill 创建账单
func CreateBill(db *gorm.DB, bill *Bill) error {
	return db.Create(bill).Error
}

// GetBills 获取账单列表
func GetBills(db *gorm.DB, userID uint, params map[string]interface{}) ([]Bill, int64, error) {
	var bills []Bill
	var total int64

	query := db.Model(&Bill{}).Where("user_id = ?", userID)

	// 按组织ID筛选
	if groupID, ok := params["groupId"].(uint); ok && groupID > 0 {
		query = query.Where("group_id = ?", groupID)
	} else if groupID, ok := params["groupId"].(*uint); ok && groupID != nil && *groupID > 0 {
		query = query.Where("group_id = ?", *groupID)
	}

	// 按凭证ID筛选
	if credentialID, ok := params["credentialId"].(uint); ok && credentialID > 0 {
		query = query.Where("credential_id = ?", credentialID)
	}

	// 按状态筛选
	if status, ok := params["status"].(BillStatus); ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// 按时间范围筛选
	if startTime, ok := params["startTime"].(time.Time); ok {
		query = query.Where("start_time >= ?", startTime)
	}
	if endTime, ok := params["endTime"].(time.Time); ok {
		query = query.Where("end_time <= ?", endTime)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	page := 1
	size := 20
	if p, ok := params["page"].(int); ok && p > 0 {
		page = p
	}
	if s, ok := params["size"].(int); ok && s > 0 {
		size = s
	}
	offset := (page - 1) * size

	// 排序
	orderBy := "created_at DESC"
	if order, ok := params["orderBy"].(string); ok && order != "" {
		orderBy = convertOrderBy(order)
	}

	err := query.Order(orderBy).Offset(offset).Limit(size).Find(&bills).Error
	return bills, total, err
}

// GetBill 获取单个账单
func GetBill(db *gorm.DB, userID uint, billID uint) (*Bill, error) {
	var bill Bill
	err := db.Where("user_id = ? AND id = ?", userID, billID).First(&bill).Error
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

// UpdateBill 更新账单
func UpdateBill(db *gorm.DB, bill *Bill) error {
	return db.Save(bill).Error
}

// GenerateBillNo 生成账单编号
func GenerateBillNo() string {
	randomBytes := make([]byte, 3)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)
	return "BILL-" + time.Now().Format("20060102150405") + "-" + randomStr
}

// RecordLLMUsage 记录LLM使用量
func RecordLLMUsage(db *gorm.DB, userID, credentialID uint, assistantID *uint, groupID *uint, sessionID string, model string, promptTokens, completionTokens, totalTokens int) error {
	record := &UsageRecord{
		UserID:           userID,
		GroupID:          groupID,
		CredentialID:     credentialID,
		AssistantID:      assistantID,
		SessionID:        sessionID,
		UsageType:        UsageTypeLLM,
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		UsageTime:        time.Now(),
	}
	return CreateUsageRecord(db, record)
}

// RecordCallUsage 记录通话使用量
func RecordCallUsage(db *gorm.DB, userID, credentialID uint, assistantID *uint, groupID *uint, sessionID string, callLogID *uint64, duration int) error {
	record := &UsageRecord{
		UserID:       userID,
		GroupID:      groupID,
		CredentialID: credentialID,
		AssistantID:  assistantID,
		SessionID:    sessionID,
		CallLogID:    callLogID,
		UsageType:    UsageTypeCall,
		CallDuration: duration,
		CallCount:    1,
		UsageTime:    time.Now(),
	}
	return CreateUsageRecord(db, record)
}

// RecordASRUsage 记录ASR使用量
func RecordASRUsage(db *gorm.DB, userID, credentialID uint, assistantID *uint, groupID *uint, sessionID string, duration int, audioSize int64) error {
	record := &UsageRecord{
		UserID:        userID,
		GroupID:       groupID,
		CredentialID:  credentialID,
		AssistantID:   assistantID,
		SessionID:     sessionID,
		UsageType:     UsageTypeASR,
		AudioDuration: duration,
		AudioSize:     audioSize,
		UsageTime:     time.Now(),
	}
	return CreateUsageRecord(db, record)
}

// RecordTTSUsage 记录TTS使用量
func RecordTTSUsage(db *gorm.DB, userID, credentialID uint, assistantID *uint, groupID *uint, sessionID string, duration int, audioSize int64) error {
	record := &UsageRecord{
		UserID:        userID,
		GroupID:       groupID,
		CredentialID:  credentialID,
		AssistantID:   assistantID,
		SessionID:     sessionID,
		UsageType:     UsageTypeTTS,
		AudioDuration: duration,
		AudioSize:     audioSize,
		UsageTime:     time.Now(),
	}
	return CreateUsageRecord(db, record)
}

// RecordAPIUsage 记录API使用量
func RecordAPIUsage(db *gorm.DB, userID, credentialID uint, assistantID *uint, groupID *uint, sessionID string, apiCallCount int, description string) error {
	record := &UsageRecord{
		UserID:       userID,
		GroupID:      groupID,
		CredentialID: credentialID,
		AssistantID:  assistantID,
		SessionID:    sessionID,
		UsageType:    UsageTypeAPI,
		APICallCount: apiCallCount,
		Description:  description,
		UsageTime:    time.Now(),
	}
	return CreateUsageRecord(db, record)
}

// GenerateBill 生成账单
func GenerateBill(db *gorm.DB, userID uint, credentialID *uint, groupID *uint, startTime, endTime time.Time, title string) (*Bill, error) {
	// 获取统计信息
	stats, err := GetUsageStatistics(db, userID, startTime, endTime, credentialID, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage statistics: %w", err)
	}

	// 创建账单
	bill := &Bill{
		UserID:                userID,
		GroupID:               groupID,
		CredentialID:          credentialID,
		BillNo:                GenerateBillNo(),
		Title:                 title,
		Status:                BillStatusGenerated,
		StartTime:             startTime,
		EndTime:               endTime,
		TotalLLMCalls:         stats.LLMCalls,
		TotalLLMTokens:        stats.LLMTokens,
		TotalPromptTokens:     stats.PromptTokens,
		TotalCompletionTokens: stats.CompletionTokens,
		TotalCallDuration:     stats.CallDuration,
		TotalCallCount:        stats.CallCount,
		TotalASRDuration:      stats.ASRDuration,
		TotalASRCount:         stats.ASRCount,
		TotalTTSDuration:      stats.TTSDuration,
		TotalTTSCount:         stats.TTSCount,
		TotalAPICalls:         stats.APICalls,
	}

	if err := CreateBill(db, bill); err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	return bill, nil
}

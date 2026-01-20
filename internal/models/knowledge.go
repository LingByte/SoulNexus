package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"gorm.io/gorm"
)

// Knowledge 表示一个知识库实体
type Knowledge struct {
	ID            int       `json:"id" gorm:"column:id"`
	UserID        int       `json:"user_id" gorm:"column:user_id"`
	GroupID       *uint     `json:"group_id,omitempty" gorm:"column:group_id;index"` // 组织ID，如果设置则表示这是组织共享的知识库
	KnowledgeKey  string    `json:"knowledge_key" gorm:"column:knowledge_key"`
	KnowledgeName string    `json:"knowledge_name" gorm:"column:knowledge_name"`
	Provider      string    `json:"provider" gorm:"column:provider;default:aliyun"` // 知识库提供者类型
	Config        string    `json:"config" gorm:"column:config;type:text"`          // 配置信息（JSON格式）
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdateAt      time.Time `json:"update_at" gorm:"column:update_at"`
	DeleteAt      time.Time `json:"delete_at" gorm:"column:delete_at"`
}

// KnowledgeList 包含知识库列表的包装结构
type KnowledgeList struct {
	Knowledge []Knowledge `json:"knowledge"`
}

// CreateKnowledgeRequest 创建知识库的请求结构
type CreateKnowledgeRequest struct {
	UserID        int                    `json:"user_id"`
	KnowledgeKey  string                 `json:"knowledge_key"`
	KnowledgeName string                 `json:"knowledge_name"`
	Provider      string                 `json:"provider"` // 知识库提供者类型
	Config        map[string]interface{} `json:"config"`   // 配置信息
}

// UpdateKnowledgeRequest 更新知识库的请求结构
type UpdateKnowledgeRequest struct {
	ID            int    `json:"id"`
	KnowledgeKey  string `json:"knowledge_key,omitempty"`
	KnowledgeName string `json:"knowledge_name,omitempty"`
}

// GetKnowledgeByUserRequest 根据用户ID获取知识库的请求结构
type GetKnowledgeByUserRequest struct {
	UserID int `json:"user_id"`
}

// CreateKnowledge 创建知识库
func CreateKnowledge(db *gorm.DB, userID int, knowledgeKey string, knowledgeName string, provider string, config map[string]interface{}, groupID *uint) (Knowledge, error) {
	// 1. 检查用户是否存在
	var user User
	err := db.Model(&User{}).
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Knowledge{}, errors.New("用户不存在")
		}
		return Knowledge{}, errors.Join(errors.New("创建知识库失败"), err)
	}

	// 2. 检查同一用户下知识库标识键是否已存在
	var existingKnowledge Knowledge
	err = db.Model(&Knowledge{}).
		Where("knowledge_key = ? AND user_id = ?", knowledgeKey, userID).
		First(&existingKnowledge).Error
	if err == nil {
		return Knowledge{}, errors.New("该知识库标识键已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Knowledge{}, errors.Join(errors.New("创建知识库失败"), err)
	}

	// 3. 默认provider为aliyun（兼容旧代码）
	if provider == "" {
		provider = knowledge.ProviderAliyun
	}

	// 4. 序列化配置信息
	configJSON := ""
	if config != nil {
		configBytes, err := json.Marshal(config)
		if err != nil {
			return Knowledge{}, fmt.Errorf("序列化配置失败: %w", err)
		}
		configJSON = string(configBytes)
	}

	// 5. 插入新知识库
	now := time.Now()
	knowledge := Knowledge{
		UserID:        userID,
		GroupID:       groupID,
		KnowledgeKey:  knowledgeKey,
		KnowledgeName: knowledgeName,
		Provider:      provider,
		Config:        configJSON,
		CreatedAt:     now,
		UpdateAt:      now,
		DeleteAt:      now,
	}

	err = db.Create(&knowledge).Error
	if err != nil {
		return Knowledge{}, errors.Join(errors.New("创建知识库失败"), err)
	}

	return knowledge, nil
}

// GetKnowledgeByUserID 根据用户ID查询其所有知识库，包括组织共享的知识库
func GetKnowledgeByUserID(db *gorm.DB, userID int) ([]Knowledge, error) {
	// 定义接收结果的切片（应该是切片类型，因为一个用户可能有多个知识库）
	var knowledgeList []Knowledge

	// 获取用户所在的组织ID列表
	var groupIDs []uint
	db.Model(&GroupMember{}).
		Where("user_id = ?", userID).
		Pluck("group_id", &groupIDs)

	// 查询：用户自己的知识库 OR 组织共享的知识库
	query := db.Model(&Knowledge{})
	if len(groupIDs) > 0 {
		query = query.Where("user_id = ? OR (group_id IN ? AND group_id IS NOT NULL)", userID, groupIDs)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	// 使用Gorm查询：ORDER BY created_at DESC
	err := query.Order("created_at DESC").Find(&knowledgeList).Error

	// 处理错误
	if err != nil {
		return nil, fmt.Errorf("查询知识库列表失败: %v", err)
	}

	// 返回查询结果（即使无数据，也返回空切片而非nil，方便上层处理）
	return knowledgeList, nil
}

func DeleteKnowledge(db *gorm.DB, knowledgeKey string) error {
	// 定义知识库结构体，用于 GORM 操作
	type Knowledge struct {
		ID           int    `gorm:"column:id"`
		KnowledgeKey string `gorm:"column:knowledge_key"`
	}

	// 检查知识库是否存在
	var existing Knowledge
	result := db.Where("knowledge_key = ?", knowledgeKey).First(&existing)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("知识库不存在")
		}
		return fmt.Errorf("数据库查询错误: %v", result.Error)
	}

	// 删除知识库
	result = db.Where("knowledge_key = ?", knowledgeKey).Delete(&Knowledge{})
	if result.Error != nil {
		return fmt.Errorf("删除知识库失败: %v", result.Error)
	}

	// 检查是否有记录被删除（可选）
	if result.RowsAffected == 0 {
		return fmt.Errorf("删除失败，未找到匹配的知识库")
	}

	return nil
}

// GetKnowledge 根据knowledgeKey获取知识库信息
func GetKnowledge(db *gorm.DB, knowledgeKey string) (*Knowledge, error) {
	var k Knowledge
	err := db.Where("knowledge_key = ?", knowledgeKey).First(&k).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("知识库不存在")
		}
		return nil, fmt.Errorf("查询知识库失败: %w", err)
	}
	return &k, nil
}

// GetKnowledgeBaseInfo 获取知识库中的信息（使用新的统一接口）
// 这个方法保持向后兼容，返回拼接的文本内容
func GetKnowledgeBaseInfo(db *gorm.DB, knowledgeKey string) (string, error) {
	return GetKnowledgeBaseInfoWithQuery(db, knowledgeKey, "请给我这个知识库中的信息")
}

// GetKnowledgeBaseInfoWithQuery 根据查询获取知识库中的信息
func GetKnowledgeBaseInfoWithQuery(db *gorm.DB, knowledgeKey string, query string) (string, error) {
	// 1. 从数据库获取知识库信息
	k, err := GetKnowledge(db, knowledgeKey)
	if err != nil {
		return "", err
	}

	// 2. 解析配置信息
	var config map[string]interface{}
	if k.Config != "" {
		if err := json.Unmarshal([]byte(k.Config), &config); err != nil {
			return "", fmt.Errorf("解析配置失败: %w", err)
		}
	}

	// 3. 获取知识库实例
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		return "", fmt.Errorf("创建知识库实例失败: %w", err)
	}

	// 4. 执行检索
	options := knowledge.SearchOptions{
		Query: query,
		TopK:  10, // 默认返回前10条
	}
	results, err := kb.Search(nil, knowledgeKey, options)
	if err != nil {
		return "", fmt.Errorf("检索知识库失败: %w", err)
	}

	// 5. 拼接结果（保持向后兼容）
	if len(results) == 0 {
		return "", fmt.Errorf("知识库中没有找到有效文本内容")
	}

	var messages string
	for _, result := range results {
		messages += result.Content + "\n"
	}

	return messages, nil
}

// SearchKnowledgeBase 搜索知识库并返回结构化结果
func SearchKnowledgeBase(db *gorm.DB, knowledgeKey string, query string, topK int) ([]knowledge.SearchResult, error) {
	// 1. 从数据库获取知识库信息
	k, err := GetKnowledge(db, knowledgeKey)
	if err != nil {
		return nil, err
	}

	// 2. 解析配置信息
	var config map[string]interface{}
	if k.Config != "" {
		if err := json.Unmarshal([]byte(k.Config), &config); err != nil {
			return nil, fmt.Errorf("解析配置失败: %w", err)
		}
	}

	// 3. 获取知识库实例
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("创建知识库实例失败: %w", err)
	}

	// 4. 执行检索
	options := knowledge.SearchOptions{
		Query: query,
		TopK:  topK,
	}
	return kb.Search(nil, knowledgeKey, options)
}

// GetStringOrDefault returns default value if string is empty
func GetStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// ParseKnowledgeConfig parses knowledge base config JSON string
func ParseKnowledgeConfig(configJSON string) (map[string]interface{}, error) {
	if configJSON == "" {
		return make(map[string]interface{}), nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return config, nil
}

// GetKnowledgeConfigOrDefault gets knowledge base config, uses default if empty
func GetKnowledgeConfigOrDefault(provider, configJSON string, getDefaultConfig func(string) map[string]interface{}) (map[string]interface{}, error) {
	if configJSON != "" {
		config, err := ParseKnowledgeConfig(configJSON)
		if err != nil {
			return nil, err
		}
		if len(config) > 0 {
			return config, nil
		}
	}
	// Use default config
	return getDefaultConfig(provider), nil
}

// GenerateKnowledgeKey generates knowledge base key (userID + knowledge name)
func GenerateKnowledgeKey(userID int, knowledgeName string) string {
	return fmt.Sprintf("%d%s%s", userID, knowledge.KnowledgeNameSeparator, knowledgeName)
}

// GenerateKnowledgeName generates knowledge base name (prefix with userID)
func GenerateKnowledgeName(userID int, name string) string {
	return fmt.Sprintf("%d%s%s", userID, knowledge.KnowledgeNameSeparator, name)
}

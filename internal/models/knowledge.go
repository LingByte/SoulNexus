package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"gorm.io/gorm"
)

// Knowledge represents a knowledge base entity
type Knowledge struct {
	ID            int       `json:"id" gorm:"column:id"`
	UserID        int       `json:"user_id" gorm:"column:user_id"`
	GroupID       *uint     `json:"group_id,omitempty" gorm:"column:group_id;index"` // Organization ID, if set indicates this is an organization-shared knowledge base
	KnowledgeKey  string    `json:"knowledge_key" gorm:"column:knowledge_key"`
	KnowledgeName string    `json:"knowledge_name" gorm:"column:knowledge_name"`
	IndexId       string    `json:"index_id" gorm:"column:index_id"`                // Index ID for providers like Aliyun (may differ from knowledge_key)
	Provider      string    `json:"provider" gorm:"column:provider;default:aliyun"` // Knowledge base provider type
	Config        string    `json:"config" gorm:"column:config;type:text"`          // Configuration information (JSON format)
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
	UpdateAt      time.Time `json:"update_at" gorm:"column:update_at"`
	DeleteAt      time.Time `json:"delete_at" gorm:"column:delete_at"`
}

// KnowledgeList contains knowledge base list wrapper structure
type KnowledgeList struct {
	Knowledge []Knowledge `json:"knowledge"`
}

// CreateKnowledgeRequest request structure for creating knowledge base
type CreateKnowledgeRequest struct {
	UserID        int                    `json:"user_id"`
	KnowledgeKey  string                 `json:"knowledge_key"`
	KnowledgeName string                 `json:"knowledge_name"`
	Provider      string                 `json:"provider"` // Knowledge base provider type
	Config        map[string]interface{} `json:"config"`   // Configuration information
}

// UpdateKnowledgeRequest request structure for updating knowledge base
type UpdateKnowledgeRequest struct {
	ID            int    `json:"id"`
	KnowledgeKey  string `json:"knowledge_key,omitempty"`
	KnowledgeName string `json:"knowledge_name,omitempty"`
}

// GetKnowledgeByUserRequest request structure for getting knowledge base by user ID
type GetKnowledgeByUserRequest struct {
	UserID int `json:"user_id"`
}

// CreateKnowledge creates a knowledge base
func CreateKnowledge(db *gorm.DB, userID int, knowledgeKey string, knowledgeName string, provider string, config map[string]interface{}, groupID *uint) (Knowledge, error) {
	return CreateKnowledgeWithIndexId(db, userID, knowledgeKey, knowledgeName, provider, config, groupID, knowledgeKey)
}

// CreateKnowledgeWithIndexId creates a knowledge base with explicit indexId
func CreateKnowledgeWithIndexId(db *gorm.DB, userID int, knowledgeKey string, knowledgeName string, provider string, config map[string]interface{}, groupID *uint, indexId string) (Knowledge, error) {
	// Check if user exists
	var user User
	err := db.Model(&User{}).
		Where("id = ?", userID).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Knowledge{}, errors.New("user not found")
		}
		return Knowledge{}, errors.Join(errors.New("failed to create knowledge base"), err)
	}

	// Check if knowledge base key already exists for the same user
	var existingKnowledge Knowledge
	err = db.Model(&Knowledge{}).
		Where("knowledge_key = ? AND user_id = ?", knowledgeKey, userID).
		First(&existingKnowledge).Error
	if err == nil {
		return Knowledge{}, errors.New("knowledge base key already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Knowledge{}, errors.Join(errors.New("failed to create knowledge base"), err)
	}

	// Default provider is aliyun (for backward compatibility)
	if provider == "" {
		provider = knowledge.ProviderAliyun
	}

	// Serialize configuration information
	configJSON := ""
	if config != nil {
		configBytes, err := json.Marshal(config)
		if err != nil {
			return Knowledge{}, fmt.Errorf("failed to serialize config: %w", err)
		}
		configJSON = string(configBytes)
	}

	// Insert new knowledge base
	now := time.Now()
	knowledge := Knowledge{
		UserID:        userID,
		GroupID:       groupID,
		KnowledgeKey:  knowledgeKey,
		KnowledgeName: knowledgeName,
		IndexId:       indexId,
		Provider:      provider,
		Config:        configJSON,
		CreatedAt:     now,
		UpdateAt:      now,
		DeleteAt:      now,
	}

	err = db.Create(&knowledge).Error
	if err != nil {
		return Knowledge{}, errors.Join(errors.New("failed to create knowledge base"), err)
	}

	return knowledge, nil
}

// GetKnowledgeByUserID queries all knowledge bases for a user, including organization-shared knowledge bases
func GetKnowledgeByUserID(db *gorm.DB, userID int) ([]Knowledge, error) {
	// Define slice to receive results (should be slice type since a user may have multiple knowledge bases)
	var knowledgeList []Knowledge

	// Get list of organization IDs the user belongs to
	var groupIDs []uint
	db.Model(&GroupMember{}).
		Where("user_id = ?", userID).
		Pluck("group_id", &groupIDs)

	// Query: user's own knowledge bases OR organization-shared knowledge bases
	query := db.Model(&Knowledge{})
	if len(groupIDs) > 0 {
		query = query.Where("user_id = ? OR (group_id IN ? AND group_id IS NOT NULL)", userID, groupIDs)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	// Use Gorm query: ORDER BY created_at DESC
	err := query.Order("created_at DESC").Find(&knowledgeList).Error

	// Handle errors
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge base list: %v", err)
	}

	// Return query results (even if no data, return empty slice instead of nil for easier handling by upper layer)
	return knowledgeList, nil
}

func DeleteKnowledge(db *gorm.DB, knowledgeKey string) error {
	// Define knowledge base struct for GORM operations
	type Knowledge struct {
		ID           int    `gorm:"column:id"`
		KnowledgeKey string `gorm:"column:knowledge_key"`
		IndexId      string `gorm:"column:index_id"`
	}

	// Check if knowledge base exists (try both knowledge_key and index_id)
	var existing Knowledge
	result := db.Where("knowledge_key = ?", knowledgeKey).First(&existing)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Try to find by index_id
			log.Printf("DEBUG: Exact key not found for deletion: %s, trying IndexId search", knowledgeKey)
			result = db.Where("index_id = ?", knowledgeKey).First(&existing)
			if result.Error != nil {
				if errors.Is(result.Error, gorm.ErrRecordNotFound) {
					return fmt.Errorf("knowledge base not found")
				}
				return fmt.Errorf("database query error: %v", result.Error)
			}
			log.Printf("DEBUG: Found knowledge base by IndexId for deletion: %s", existing.IndexId)
		} else {
			return fmt.Errorf("database query error: %v", result.Error)
		}
	}

	// Delete knowledge base using the found ID
	result = db.Where("id = ?", existing.ID).Delete(&Knowledge{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete knowledge base: %v", result.Error)
	}

	// Check if any records were deleted (optional)
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete failed, no matching knowledge base found")
	}

	log.Printf("DEBUG: Successfully deleted knowledge base - ID: %d, KnowledgeKey: %s, IndexId: %s",
		existing.ID, existing.KnowledgeKey, existing.IndexId)

	return nil
}

// GetKnowledge gets knowledge base information by knowledgeKey or indexId
func GetKnowledge(db *gorm.DB, knowledgeKey string) (*Knowledge, error) {
	var k Knowledge

	// First try to find by knowledge_key
	err := db.Where("knowledge_key = ?", knowledgeKey).First(&k).Error
	if err == nil {
		return &k, nil
	}

	// If not found by knowledge_key, try to find by index_id
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("DEBUG: Exact key not found: %s, trying IndexId search", knowledgeKey)
		err = db.Where("index_id = ?", knowledgeKey).First(&k).Error
		if err == nil {
			log.Printf("DEBUG: Found knowledge base by IndexId: %s", k.IndexId)
			return &k, nil
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("knowledge base not found")
		}
	}

	return nil, fmt.Errorf("failed to query knowledge base: %w", err)
}

// GetKnowledgeBaseInfo gets information from knowledge base (using new unified interface)
// This method maintains backward compatibility, returning concatenated text content
func GetKnowledgeBaseInfo(db *gorm.DB, knowledgeKey string) (string, error) {
	return GetKnowledgeBaseInfoWithQuery(db, knowledgeKey, "Please provide information from this knowledge base")
}

// GetKnowledgeBaseInfoWithQuery gets information from knowledge base based on query
func GetKnowledgeBaseInfoWithQuery(db *gorm.DB, knowledgeKey string, query string) (string, error) {
	// Get knowledge base information from database
	k, err := GetKnowledge(db, knowledgeKey)
	if err != nil {
		return "", err
	}

	// Parse configuration information
	var config map[string]interface{}
	if k.Config != "" {
		if err := json.Unmarshal([]byte(k.Config), &config); err != nil {
			return "", fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Get knowledge base instance
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		return "", fmt.Errorf("failed to create knowledge base instance: %w", err)
	}

	// Execute search
	options := knowledge.SearchOptions{
		Query: query,
		TopK:  10, // Default return top 10 results
	}
	results, err := kb.Search(nil, knowledgeKey, options)
	if err != nil {
		return "", fmt.Errorf("failed to search knowledge base: %w", err)
	}

	// Concatenate results (maintain backward compatibility)
	if len(results) == 0 {
		return "", fmt.Errorf("no valid text content found in knowledge base")
	}

	var messages string
	for _, result := range results {
		messages += result.Content + "\n"
	}

	return messages, nil
}

// SearchKnowledgeBase searches knowledge base and returns structured results
func SearchKnowledgeBase(db *gorm.DB, knowledgeKey string, query string, topK int) ([]knowledge.SearchResult, error) {
	log.Printf("DEBUG: SearchKnowledgeBase called with key: %s, query: %s, topK: %d", knowledgeKey, query, topK)

	// Get knowledge base information from database
	k, err := GetKnowledge(db, knowledgeKey)
	if err != nil {
		log.Printf("DEBUG: Failed to get knowledge base: %v", err)
		return nil, err
	}

	log.Printf("DEBUG: Found knowledge base - ID: %d, Provider: %s, KnowledgeKey: %s, IndexId: %s",
		k.ID, k.Provider, k.KnowledgeKey, k.IndexId)

	// Parse configuration information
	var config map[string]interface{}
	if k.Config != "" {
		if err := json.Unmarshal([]byte(k.Config), &config); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	// Get knowledge base instance
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge base instance: %w", err)
	}

	// Execute search
	ctx := context.Background()

	// Generate embedding using professional API (with fallback to simple embedding)
	// Use the dimension from knowledge base config, or from embedding provider
	var embedding []float32
	var embeddingDim int

	// Try to get dimension from knowledge base config first
	if config != nil {
		if dim, ok := config["dimension"].(int); ok && dim > 0 {
			embeddingDim = dim
			log.Printf("DEBUG: Using dimension from knowledge base config: %d", embeddingDim)
		} else if dim, ok := config["dimension"].(float64); ok && dim > 0 {
			embeddingDim = int(dim)
			log.Printf("DEBUG: Using dimension from knowledge base config: %d", embeddingDim)
		}
	}

	// If not found in config, try to get from embedding provider
	if embeddingDim == 0 {
		provider, err := knowledge.GetEmbeddingProvider()
		if err == nil {
			embeddingDim = provider.Dimension()
			log.Printf("DEBUG: Using dimension from embedding provider: %d", embeddingDim)
		} else {
			// Fallback to default dimension (384 for compatibility with most vector DBs)
			embeddingDim = 384
			log.Printf("DEBUG: Using default dimension: %d", embeddingDim)
		}
	}

	embedding = knowledge.GenerateEmbedding(query, embeddingDim)
	log.Printf("DEBUG: Generated embedding with dimension: %d", len(embedding))

	// Use the correct key for search
	// - Aliyun: use IndexId (the actual index ID in Bailian)
	// - Others (Qdrant, Milvus, etc.): use KnowledgeKey (the collection name)
	searchKey := k.KnowledgeKey
	if k.Provider == knowledge.ProviderAliyun && k.IndexId != "" {
		searchKey = k.IndexId
	}

	log.Printf("DEBUG: Using searchKey: %s for provider: %s", searchKey, k.Provider)

	// Retrieve more results for reranking (2x topK)
	retrieveTopK := topK * 2
	if retrieveTopK < 10 {
		retrieveTopK = 10
	}

	options := knowledge.SearchOptions{
		Query: query,
		TopK:  retrieveTopK,
		Filter: map[string]interface{}{
			"embedding": embedding,
		},
	}

	results, err := kb.Search(ctx, searchKey, options)
	if err != nil {
		// Check if index doesn't exist in provider - sync delete local record
		if err.Error() == knowledge.ErrIndexNotExist || strings.Contains(err.Error(), "index not exist") {
			log.Printf("DEBUG: Index not exist in provider, deleting local record: %s", knowledgeKey)
			// Delete the local knowledge base record
			if delErr := DeleteKnowledge(db, knowledgeKey); delErr != nil {
				log.Printf("ERROR: Failed to delete local knowledge record: %v", delErr)
			}
			return nil, fmt.Errorf("知识库索引已不存在，已自动删除本地记录")
		}
		return nil, err
	}

	// Apply reranking if configured
	rerankedResults, err := knowledge.RerankSearchResults(query, results, topK)
	if err != nil {
		// If reranking fails, return original results (truncated to topK)
		if len(results) > topK {
			results = results[:topK]
		}
		return results, nil
	}

	return rerankedResults, nil
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

// GenerateKnowledgeName generates knowledge base name with length constraint for Aliyun (max 20 chars)
func GenerateKnowledgeName(userID int, name string) string {
	// 阿里云要求名称长度 1-20 个字符
	// 格式: {userID}_{name}_{sequence}
	// 例如: 2_Git_1, 10_MyKB_1

	// 截断名称，确保总长度不超过 20
	// 最坏情况: 10_<name>_999 = 3 + len(name) + 4 = 7 + len(name)
	// 所以 name 最多 13 个字符
	maxNameLen := 13

	runes := []rune(name)
	if len(runes) > maxNameLen {
		runes = runes[:maxNameLen]
	}
	truncatedName := string(runes)

	// 如果截断后的名称为空，使用默认名称
	if truncatedName == "" {
		truncatedName = "kb"
	}

	// 使用时间戳的最后 3 位作为序列号（0-999）
	sequence := time.Now().UnixNano() % 1000

	result := fmt.Sprintf("%d_%s_%d", userID, truncatedName, sequence)

	// 最后再检查一遍长度（以防万一）
	if len(result) > 20 {
		result = result[:20]
	}

	return result
}

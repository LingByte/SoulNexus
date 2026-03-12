package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/response"
	bailian20231229 "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	teaUtil "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

// getAliyunConfig gets Aliyun knowledge base config from config file
func getAliyunConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyAliyunAccessKeyID:     cfg.Services.KnowledgeBase.Bailian.AccessKeyId,
		knowledge.ConfigKeyAliyunAccessKeySecret: cfg.Services.KnowledgeBase.Bailian.AccessKeySecret,
		knowledge.ConfigKeyAliyunEndpoint:        models.GetStringOrDefault(cfg.Services.KnowledgeBase.Bailian.Endpoint, knowledge.DefaultAliyunEndpoint),
		knowledge.ConfigKeyAliyunWorkspaceID:     cfg.Services.KnowledgeBase.Bailian.WorkspaceId,
		knowledge.ConfigKeyAliyunCategoryID:      cfg.Services.KnowledgeBase.Bailian.CategoryId,
		knowledge.ConfigKeyAliyunSourceType:      models.GetStringOrDefault(cfg.Services.KnowledgeBase.Bailian.SourceType, knowledge.DefaultAliyunSourceType),
		knowledge.ConfigKeyAliyunParser:          models.GetStringOrDefault(cfg.Services.KnowledgeBase.Bailian.Parser, knowledge.DefaultAliyunParser),
		knowledge.ConfigKeyAliyunStructType:      models.GetStringOrDefault(cfg.Services.KnowledgeBase.Bailian.StructType, knowledge.DefaultAliyunStructType),
		knowledge.ConfigKeyAliyunSinkType:        models.GetStringOrDefault(cfg.Services.KnowledgeBase.Bailian.SinkType, knowledge.DefaultAliyunSinkType),
	}
}

// getMilvusConfig gets Milvus knowledge base config from config file
func getMilvusConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyMilvusAddress:        cfg.Services.KnowledgeBase.Milvus.Address,
		knowledge.ConfigKeyMilvusUsername:       cfg.Services.KnowledgeBase.Milvus.Username,
		knowledge.ConfigKeyMilvusPassword:       cfg.Services.KnowledgeBase.Milvus.Password,
		knowledge.ConfigKeyMilvusCollectionName: cfg.Services.KnowledgeBase.Milvus.Collection,
		knowledge.ConfigKeyMilvusDimension:      cfg.Services.KnowledgeBase.Milvus.Dimension,
	}
}

// getQdrantConfig gets Qdrant knowledge base config from config file
func getQdrantConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	baseURL := cfg.Services.KnowledgeBase.Qdrant.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}

	return map[string]interface{}{
		knowledge.ConfigKeyQdrantHost:      baseURL,
		knowledge.ConfigKeyQdrantAPIKey:    cfg.Services.KnowledgeBase.Qdrant.APIKey,
		knowledge.ConfigKeyQdrantDimension: cfg.Services.KnowledgeBase.Qdrant.Dimension,
	}
}

// getElasticsearchConfig gets Elasticsearch knowledge base config from config file
func getElasticsearchConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyElasticsearchBaseURL:   cfg.Services.KnowledgeBase.Elasticsearch.BaseURL,
		knowledge.ConfigKeyElasticsearchUsername:  cfg.Services.KnowledgeBase.Elasticsearch.Username,
		knowledge.ConfigKeyElasticsearchPassword:  cfg.Services.KnowledgeBase.Elasticsearch.Password,
		knowledge.ConfigKeyElasticsearchIndexName: cfg.Services.KnowledgeBase.Elasticsearch.Index,
	}
}

// getPineconeConfig gets Pinecone knowledge base config from config file
func getPineconeConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyPineconeApiKey:    cfg.Services.KnowledgeBase.Pinecone.APIKey,
		knowledge.ConfigKeyPineconeBaseURL:   cfg.Services.KnowledgeBase.Pinecone.BaseURL,
		knowledge.ConfigKeyPineconeIndexName: cfg.Services.KnowledgeBase.Pinecone.IndexName,
		knowledge.ConfigKeyPineconeDimension: cfg.Services.KnowledgeBase.Pinecone.Dimension,
	}
}

// getKnowledgeBaseConfig gets config for the specified provider
func getKnowledgeBaseConfig(provider string) map[string]interface{} {
	switch provider {
	case knowledge.ProviderAliyun:
		return getAliyunConfig()
	case knowledge.ProviderZilliz:
		return getMilvusConfig()
	case knowledge.ProviderQdrant:
		return getQdrantConfig()
	case knowledge.ProviderElasticsearch:
		return getElasticsearchConfig()
	case knowledge.ProviderPinecone:
		return getPineconeConfig()
	default:
		return getAliyunConfig() // default to Aliyun config
	}
}

// getKnowledgeBase gets knowledge base instance from config file
func getKnowledgeBase(provider string) (knowledge.KnowledgeBase, error) {
	cfg := config.GlobalConfig

	// Return error if knowledge base feature is not enabled
	if !cfg.Services.KnowledgeBase.Enabled {
		return nil, fmt.Errorf(knowledge.ErrKnowledgeBaseDisabled)
	}

	// Use default provider if not specified
	if provider == "" {
		provider = knowledge.DefaultProvider // default to Aliyun
	}

	// Get config for the specified provider
	kbConfig := getKnowledgeBaseConfig(provider)
	return knowledge.GetKnowledgeBaseByProvider(provider, kbConfig)
}

// getStringFromConfig gets string value from config map
func getStringFromConfig(config map[string]interface{}, key string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// CreateKnowledgeBase creates a knowledge base
func (h *Handlers) CreateKnowledgeBase(c *gin.Context) {
	// 1. Receive file (optional)
	var file multipart.File
	var header *multipart.FileHeader
	var err error

	file, header, err = c.Request.FormFile(constants.FormFieldFile)
	if err != nil && err != http.ErrMissingFile {
		response.Fail(c, knowledge.ErrFileReceiveFailed, err)
		return
	}

	// If file is provided, defer close
	if file != nil {
		defer file.Close()
	}

	// 2. Receive request parameters
	knowledgeName := c.PostForm(constants.FormFieldKnowledgeName)
	provider := c.PostForm(constants.FormFieldProvider) // optional, default to aliyun
	if provider == "" {
		provider = knowledge.DefaultProvider
	}

	// Get organization ID (optional)
	var groupID *uint
	if groupIDStr := c.PostForm("group_id"); groupIDStr != "" {
		if parsedID, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			groupIDVal := uint(parsedID)
			groupID = &groupIDVal
		}
	}

	log.Printf("Creating knowledge base - name: %s, provider: %s, groupID: %v, hasFile: %v", knowledgeName, provider, groupID, file != nil)
	user := models.CurrentUser(c)
	userId := int(user.ID)

	// If organization ID is specified, verify user has permission to create shared knowledge base in that organization
	if groupID != nil {
		var group models.Group
		if err := h.db.First(&group, *groupID).Error; err != nil {
			response.Fail(c, "Organization not found", nil)
			return
		}
		// Check if user is the creator or admin of the organization
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *groupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "Insufficient permissions", "Only creator or admin can create organization-shared knowledge base")
				return
			}
		}
	}

	// 3. Generate unique name for Aliyun (only used for Aliyun index creation)
	generatedName := models.GenerateKnowledgeName(userId, knowledgeName)

	// 4. Get knowledge base instance
	kb, err := getKnowledgeBase(provider)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeBaseInitFailed, err)
		return
	}

	// 检查 kb 是否为 nil
	if kb == nil {
		response.Fail(c, knowledge.ErrKnowledgeBaseInitFailed, fmt.Errorf("knowledge base provider is nil"))
		return
	}

	// 5. Get config for the specified provider
	config := getKnowledgeBaseConfig(provider)
	if config == nil {
		config = make(map[string]interface{})
	}

	// 6. Generate knowledge base key (use generated name for all providers)
	var knowledgeKey string
	knowledgeKey = models.GenerateKnowledgeKey(userId, generatedName)

	// 7. Handle file upload and index creation based on provider
	var indexId string

	// If no file is provided, still need to create index (with auto-generated initial document)
	if file == nil {
		log.Printf("No file provided, creating knowledge base with auto-generated initial document")

		// For Aliyun, create index with auto-generated txt file
		if provider == knowledge.ProviderAliyun {
			indexId, err = kb.CreateIndex(context.Background(), generatedName, nil)
			if err != nil {
				log.Printf("ERROR: Failed to create Aliyun index - error: %v, knowledgeKey: %s", err, knowledgeKey)
				response.Fail(c, knowledge.ErrIndexCreateFailed, err)
				return
			}
			log.Printf("SUCCESS: Aliyun index created (empty) - indexId: %s, knowledgeKey: %s", indexId, knowledgeKey)
		} else {
			// Other providers: create empty index
			indexId, err = kb.CreateIndex(context.Background(), knowledgeKey, config)
			if err != nil {
				log.Printf("ERROR: Failed to create index for provider %s - error: %v, knowledgeKey: %s", provider, err, knowledgeKey)
				response.Fail(c, knowledge.ErrIndexCreateFailed, err)
				return
			}
			log.Printf("SUCCESS: Index created for provider %s (empty) - indexId: %s, knowledgeKey: %s", provider, indexId, knowledgeKey)
		}
	} else if provider == knowledge.ProviderAliyun {
		// Aliyun: need to upload file first to get fileId, then create index
		aliyunConfig := getAliyunConfig()
		if aliyunConfig == nil {
			response.Fail(c, "Aliyun config not found", nil)
			return
		}

		client, err := createAliyunClient(aliyunConfig)
		if err != nil {
			response.Fail(c, "failed to create Aliyun client", err)
			return
		}

		// Upload file to get fileId
		fileId, ok := uploadFileToAliyunWithClient(client, file, header, aliyunConfig)
		if !ok {
			response.Fail(c, knowledge.ErrFileUploadFailed, nil)
			return
		}

		// Build index creation config
		structType := getStringFromConfig(aliyunConfig, knowledge.ConfigKeyAliyunStructType)
		if structType == "" {
			structType = knowledge.DefaultAliyunStructType
		}
		sourceType := getStringFromConfig(aliyunConfig, knowledge.ConfigKeyAliyunSourceType)
		if sourceType == "" {
			sourceType = knowledge.DefaultAliyunSourceType
		}
		sinkType := getStringFromConfig(aliyunConfig, knowledge.ConfigKeyAliyunSinkType)
		if sinkType == "" {
			sinkType = knowledge.DefaultAliyunSinkType
		}
		createConfig := map[string]interface{}{
			"file_id":                           fileId,
			knowledge.ConfigKeyAliyunStructType: structType,
			knowledge.ConfigKeyAliyunSourceType: sourceType,
			knowledge.ConfigKeyAliyunSinkType:   sinkType,
		}

		log.Printf("DEBUG: Creating Aliyun index with config - fileId: %s, structType: %s, sourceType: %s, sinkType: %s",
			fileId, structType, sourceType, sinkType)

		// 使用 generatedName 作为 Aliyun 索引名（长度限制在 20 以内）
		indexId, err = kb.CreateIndex(context.Background(), generatedName, createConfig)
		if err != nil {
			log.Printf("ERROR: Failed to create Aliyun index - error: %v, knowledgeKey: %s", err, knowledgeKey)
			response.Fail(c, knowledge.ErrIndexCreateFailed, err)
			return
		}

		// 检查 indexId 是否为空
		if indexId == "" {
			log.Printf("ERROR: Index id is empty after Aliyun creation")
			response.Fail(c, knowledge.ErrIndexCreateFailed, fmt.Errorf("index id is empty"))
			return
		}

		log.Printf("SUCCESS: Aliyun index created - indexId: %s, knowledgeKey: %s", indexId, knowledgeKey)
	} else {
		// Other providers: upload document first, then create index
		metadata := map[string]interface{}{
			knowledge.MetadataKeyUserID: userId,
			knowledge.MetadataKeyName:   knowledgeName,
			knowledge.MetadataKeySource: knowledge.MetadataSourceAPICreate,
		}
		err = kb.UploadDocument(context.Background(), knowledgeKey, file, header, metadata)
		if err != nil {
			response.Fail(c, knowledge.ErrFileUploadFailed, err)
			return
		}

		// Build index creation config (copy config and add specific parameters)
		createConfig := make(map[string]interface{})
		for k, v := range config {
			createConfig[k] = v
		}
		// Add specific parameters if they exist
		if collectionName, ok := config[knowledge.ConfigKeyMilvusCollectionName]; ok {
			createConfig[knowledge.ConfigKeyMilvusCollectionName] = collectionName
		}
		if indexName, ok := config[knowledge.ConfigKeyElasticsearchIndexName]; ok {
			createConfig[knowledge.ConfigKeyElasticsearchIndexName] = indexName
		}
		if pineconeIndexName, ok := config[knowledge.ConfigKeyPineconeIndexName]; ok {
			createConfig[knowledge.ConfigKeyPineconeIndexName] = pineconeIndexName
		}

		indexId, err = kb.CreateIndex(context.Background(), knowledgeKey, createConfig)
		if err != nil {
			log.Printf("ERROR: Failed to create index for provider %s - error: %v, knowledgeKey: %s", provider, err, knowledgeKey)
			response.Fail(c, knowledge.ErrIndexCreateFailed, err)
			return
		}

		// 检查 indexId 是否为空
		if indexId == "" {
			log.Printf("ERROR: Index id is empty after creation for provider %s", provider)
			response.Fail(c, knowledge.ErrIndexCreateFailed, fmt.Errorf("index id is empty"))
			return
		}

		log.Printf("SUCCESS: Index created for provider %s - indexId: %s, knowledgeKey: %s", provider, indexId, knowledgeKey)
	}

	// 8. Call models layer to create knowledge base record (use original knowledgeName, not generated name)
	log.Printf("DEBUG: About to create knowledge base record - indexId: %s, knowledgeName: %s, provider: %s", indexId, knowledgeName, provider)

	// Check if knowledge base with same key already exists (idempotency check)
	var existingKB models.Knowledge
	if err := h.db.Where("knowledge_key = ? AND user_id = ?", knowledgeKey, userId).First(&existingKB).Error; err == nil {
		// Knowledge base already exists, return it
		log.Printf("INFO: Knowledge base already exists - ID: %d, Key: %s", existingKB.ID, existingKB.KnowledgeKey)
		responseData := map[string]interface{}{
			"id":             existingKB.ID,
			"user_id":        existingKB.UserID,
			"group_id":       existingKB.GroupID,
			"knowledge_key":  existingKB.KnowledgeKey,
			"knowledge_name": existingKB.KnowledgeName,
			"provider":       existingKB.Provider,
			"created_at":     existingKB.CreatedAt,
			"updated_at":     existingKB.UpdateAt,
		}
		response.Success(c, "created successfully", responseData)
		return
	}

	knowledgeRecord, err := models.CreateKnowledgeWithIndexId(h.db, int(userId), knowledgeKey, knowledgeName, provider, config, groupID, indexId)
	if err != nil {
		log.Printf("ERROR: Failed to create knowledge base record - error: %v", err)
		response.Fail(c, err.Error(), nil)
		return
	}

	log.Printf("DEBUG: Knowledge base record created - ID: %d, Key: %s", knowledgeRecord.ID, knowledgeRecord.KnowledgeKey)

	// 9. Return success response
	// 构建返回数据，确保所有字段都有效
	responseData := map[string]interface{}{
		"id":             knowledgeRecord.ID,
		"user_id":        knowledgeRecord.UserID,
		"group_id":       knowledgeRecord.GroupID,
		"knowledge_key":  knowledgeRecord.KnowledgeKey,
		"knowledge_name": knowledgeRecord.KnowledgeName,
		"provider":       knowledgeRecord.Provider,
		"created_at":     knowledgeRecord.CreatedAt,
		"updated_at":     knowledgeRecord.UpdateAt,
	}

	log.Printf("SUCCESS: Knowledge base created successfully - ID: %d, Name: %s", knowledgeRecord.ID, knowledgeRecord.KnowledgeName)
	response.Success(c, "created successfully", responseData)
}

// createAliyunClient creates Aliyun client from config
func createAliyunClient(config map[string]interface{}) (*bailian20231229.Client, error) {
	accessKeyId := getStringFromConfig(config, knowledge.ConfigKeyAliyunAccessKeyID)
	accessKeySecret := getStringFromConfig(config, knowledge.ConfigKeyAliyunAccessKeySecret)
	endpoint := getStringFromConfig(config, knowledge.ConfigKeyAliyunEndpoint)
	if endpoint == "" {
		endpoint = knowledge.DefaultAliyunEndpoint
	}

	if accessKeyId == "" || accessKeySecret == "" {
		return nil, fmt.Errorf(knowledge.ErrAccessKeyRequired)
	}

	openapiConfig := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}

	return bailian20231229.NewClient(openapiConfig)
}

// uploadFileToAliyunWithClient uploads file to Aliyun using specified client (for CreateKnowledgeBase)
func uploadFileToAliyunWithClient(client *bailian20231229.Client, file multipart.File, header *multipart.FileHeader, config map[string]interface{}) (string, bool) {
	// Reset file pointer (may have been read already)
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	// Calculate MD5
	calculateMD5, err := CalculateMD5(file)
	if err != nil {
		return "failed to calculate MD5", false
	}

	// Reset file pointer
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart)
	}

	size, err := GetFileSize(header)
	if err != nil {
		return "failed to get file size", false
	}

	// Get parameters from config
	categoryId := getStringFromConfig(config, knowledge.ConfigKeyAliyunCategoryID)
	workspaceId := getStringFromConfig(config, knowledge.ConfigKeyAliyunWorkspaceID)
	parser := getStringFromConfig(config, knowledge.ConfigKeyAliyunParser)
	if parser == "" {
		parser = knowledge.DefaultAliyunParser
	}

	// Apply for upload lease
	lease, err := ApplyLease(client, categoryId, header, calculateMD5, size, workspaceId)
	if err != nil {
		log.Println("failed to apply upload lease: ", err)
		return "failed to apply upload lease", false
	}

	leaseBody := lease.GetBody()
	if leaseBody == nil || leaseBody.Data == nil || leaseBody.Data.Param == nil {
		return "failed to apply upload lease: incomplete data", false
	}

	if leaseBody.Message != nil && *leaseBody.Message != "" {
		return *leaseBody.Message, false
	}

	// Get presigned URL and headers
	preSignedUrl := leaseBody.Data.Param.Url
	headers := leaseBody.Data.Param.Headers
	result := make(map[string]string)
	if value, ok := headers.(map[string]interface{}); ok {
		for key, val := range value {
			if strVal, ok := val.(string); ok {
				result[key] = strVal
			}
		}
	}

	// Upload file to presigned URL
	err = UploadFile(*preSignedUrl, result, file)
	if err != nil {
		return "failed to upload file", false
	}

	// Call AddFile API
	leaseId := *leaseBody.Data.FileUploadLeaseId
	fileResult, err := AddFile(client, leaseId, parser, categoryId, workspaceId)
	if err != nil {
		return "failed to add file", false
	}
	fileId := *fileResult.GetBody().Data.FileId

	// Get document info
	_, err = DescribeFile(client, workspaceId, fileId)
	if err != nil {
		return "failed to get document info", false
	}

	return fileId, true
}

// UploadFileToKnowledgeBase uploads file to knowledge base (supports multiple providers)
func (h *Handlers) UploadFileToKnowledgeBase(c *gin.Context) {
	// 1. Receive file
	file, header, err := c.Request.FormFile(constants.FormFieldFile)
	if err != nil {
		response.Fail(c, knowledge.ErrFileReceiveFailed, err)
		return
	}
	defer file.Close()

	// 2. Receive knowledge base key
	knowledgeKey := c.PostForm(constants.FormFieldKnowledgeKey)
	if knowledgeKey == "" {
		response.Fail(c, knowledge.ErrKnowledgeKeyRequired, nil)
		return
	}

	log.Printf("Uploading file to knowledge base - key: %s, filename: %s, size: %d bytes", knowledgeKey, header.Filename, header.Size)

	// 3. Get knowledge base info from database
	k, err := models.GetKnowledge(h.db, knowledgeKey)
	if err != nil {
		// If exact match fails, try to find by IndexId (new format)
		log.Printf("DEBUG: Exact key not found: %s, trying IndexId search", knowledgeKey)
		var kb models.Knowledge
		if err := h.db.Where("index_id = ?", knowledgeKey).First(&kb).Error; err == nil {
			k = &kb
			knowledgeKey = kb.KnowledgeKey // Update knowledgeKey to the one found in DB
			log.Printf("DEBUG: Found knowledge base by IndexId: %s", knowledgeKey)
		} else {
			// If IndexId search fails, try prefix search (for truncated keys)
			log.Printf("DEBUG: IndexId not found, trying prefix search")
			if err := h.db.Where("knowledge_key LIKE ?", knowledgeKey+"%").First(&kb).Error; err != nil {
				response.Fail(c, knowledge.ErrKnowledgeNotFound, err)
				return
			}
			k = &kb
			knowledgeKey = kb.KnowledgeKey // Update knowledgeKey to the one found in DB
			log.Printf("DEBUG: Found knowledge base by prefix: %s", knowledgeKey)
		}
	}

	// 4. Parse config
	config, err := models.GetKnowledgeConfigOrDefault(k.Provider, k.Config, getKnowledgeBaseConfig)
	if err != nil {
		response.Fail(c, knowledge.ErrConfigParseFailed, err)
		return
	}

	// 5. Get knowledge base instance
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeBaseInitFailed, err)
		return
	}

	// 6. For Aliyun, check if index needs to be created (on first file upload)
	if k.Provider == knowledge.ProviderAliyun {
		log.Printf("DEBUG: Aliyun upload - knowledgeKey: %s, IndexId: %s", knowledgeKey, k.IndexId)
		log.Printf("Preparing to upload file to Aliyun knowledge base")
	}

	// 7. Upload document to knowledge base
	metadata := map[string]interface{}{
		knowledge.MetadataKeyUserID: k.UserID,
		knowledge.MetadataKeyName:   k.KnowledgeName,
		knowledge.MetadataKeySource: knowledge.MetadataSourceAPIUpload,
	}

	// For Aliyun, use the stored indexId (which is the generatedName with length ≤ 20)
	// For other providers, use knowledgeKey
	uploadKey := knowledgeKey
	if k.Provider == knowledge.ProviderAliyun && k.IndexId != "" {
		uploadKey = k.IndexId
	}

	err = kb.UploadDocument(context.Background(), uploadKey, file, header, metadata)
	if err != nil {
		log.Printf("ERROR: Failed to upload file - error: %v", err)
		response.Fail(c, knowledge.ErrFileUploadFailed, err)
		return
	}

	log.Printf("File uploaded successfully - key: %s, filename: %s. Note: Indexing is asynchronous, may take a few seconds", knowledgeKey, header.Filename)
	response.Success(c, "uploaded successfully", nil)
}

// GetKnowledgeBase gets knowledge base list for the current user
func (h *Handlers) GetKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	userID := int(user.ID)

	// Query knowledge base list by user ID
	knowledgeList, err := models.GetKnowledgeByUserID(h.db, userID)
	if err != nil {
		response.Fail(c, knowledge.ErrQueryKnowledgeListFailed, err)
		return
	}

	// Build response data with essential information only
	result := make([]map[string]interface{}, 0, len(knowledgeList))
	for _, kb := range knowledgeList {
		// Generate new simplified key format for frontend
		// If IndexId exists (new format), use it; otherwise generate from name
		displayKey := kb.IndexId
		if displayKey == "" {
			// For old knowledge bases without IndexId, generate new format
			displayKey = models.GenerateKnowledgeName(kb.UserID, kb.KnowledgeName)
		}

		item := map[string]interface{}{
			"id":             kb.ID,
			"user_id":        kb.UserID,
			"group_id":       kb.GroupID,
			"knowledge_key":  displayKey,
			"knowledge_name": kb.KnowledgeName,
			"provider":       kb.Provider,
			"created_at":     kb.CreatedAt,
			"updated_at":     kb.UpdateAt,
		}
		result = append(result, item)
	}

	response.Success(c, "retrieved successfully", result)
}

// DeleteKnowledgeBase deletes a knowledge base
func (h *Handlers) DeleteKnowledgeBase(c *gin.Context) {
	// Receive knowledge base key
	knowledgeKey := c.Query(constants.QueryParamKnowledgeKey)
	if knowledgeKey == "" {
		response.Fail(c, knowledge.ErrKnowledgeKeyRequired, nil)
		return
	}

	// 1. Get knowledge base info
	k, err := models.GetKnowledge(h.db, knowledgeKey)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeNotFound, err)
		return
	}

	// 2. Parse config
	config, err := models.GetKnowledgeConfigOrDefault(k.Provider, k.Config, getKnowledgeBaseConfig)
	if err != nil {
		response.Fail(c, knowledge.ErrConfigParseFailed, err)
		return
	}

	// 3. Get knowledge base instance and delete
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeBaseInitFailed, err)
		return
	}

	err = kb.DeleteIndex(context.Background(), knowledgeKey)
	if err != nil {
		response.Fail(c, knowledge.ErrIndexDeleteFailed, err)
		return
	}

	// 4. Delete database record
	err = models.DeleteKnowledge(h.db, knowledgeKey)
	if err != nil {
		response.Fail(c, knowledge.ErrDatabaseDeleteFailed, err)
		return
	}

	response.Success(c, "deleted successfully", nil)
}

// SearchKnowledgeBase searches for documents in a knowledge base (retrieval test)
func (h *Handlers) SearchKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	userID := int(user.ID)

	// Get parameters
	knowledgeKey := c.Query(constants.QueryParamKnowledgeKey)
	message := c.Query("message")
	topKStr := c.DefaultQuery("topK", "5")

	// Validate parameters
	if knowledgeKey == "" {
		response.Fail(c, knowledge.ErrKnowledgeKeyRequired, nil)
		return
	}
	if message == "" {
		response.Fail(c, "message parameter is required", nil)
		return
	}

	// Parse topK
	topK := 5
	if val, err := strconv.Atoi(topKStr); err == nil && val > 0 {
		topK = val
	}

	// Verify knowledge base belongs to user
	k, err := models.GetKnowledge(h.db, knowledgeKey)
	if err != nil {
		// If exact match fails, try to find by IndexId (new format)
		log.Printf("DEBUG: Exact key not found: %s, trying IndexId search", knowledgeKey)
		var kb models.Knowledge
		if err := h.db.Where("index_id = ?", knowledgeKey).First(&kb).Error; err == nil {
			k = &kb
			knowledgeKey = kb.KnowledgeKey // Update knowledgeKey to the one found in DB
			log.Printf("DEBUG: Found knowledge base by IndexId: %s", knowledgeKey)
		} else {
			// If IndexId search fails, try prefix search (for truncated keys)
			log.Printf("DEBUG: IndexId not found, trying prefix search")
			if err := h.db.Where("knowledge_key LIKE ?", knowledgeKey+"%").First(&kb).Error; err != nil {
				response.Fail(c, knowledge.ErrKnowledgeNotFound, err)
				return
			}
			k = &kb
			knowledgeKey = kb.KnowledgeKey // Update knowledgeKey to the one found in DB
			log.Printf("DEBUG: Found knowledge base by prefix: %s", knowledgeKey)
		}
	}
	if k.UserID != userID {
		response.Fail(c, "unauthorized access to knowledge base", nil)
		return
	}

	// Search knowledge base
	log.Printf("Searching knowledge base - key: %s, query: %s, topK: %d", knowledgeKey, message, topK)
	results, err := models.SearchKnowledgeBase(h.db, knowledgeKey, message, topK)
	if err != nil {
		log.Printf("ERROR: Failed to search knowledge base - error: %v", err)
		response.Fail(c, "failed to search knowledge base", err)
		return
	}

	// Build response
	resultList := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		item := map[string]interface{}{
			"content":  result.Content,
			"score":    result.Score,
			"metadata": result.Metadata,
		}
		resultList = append(resultList, item)
	}

	response.Success(c, "search completed", map[string]interface{}{
		"knowledge_key": knowledgeKey,
		"query":         message,
		"total":         len(resultList),
		"results":       resultList,
	})
}

// ListKnowledgeBaseContent lists all content in a knowledge base (documents and chunks)
func (h *Handlers) ListKnowledgeBaseContent(c *gin.Context) {
	user := models.CurrentUser(c)
	userID := int(user.ID)

	// Get parameters
	knowledgeKey := c.Query(constants.QueryParamKnowledgeKey)

	// Validate parameters
	if knowledgeKey == "" {
		response.Fail(c, knowledge.ErrKnowledgeKeyRequired, nil)
		return
	}

	// Verify knowledge base belongs to user
	k, err := models.GetKnowledge(h.db, knowledgeKey)
	if err != nil {
		// If exact match fails, try to find by IndexId (new format)
		log.Printf("DEBUG: Exact key not found: %s, trying IndexId search", knowledgeKey)
		var kb models.Knowledge
		if err := h.db.Where("index_id = ?", knowledgeKey).First(&kb).Error; err == nil {
			k = &kb
			knowledgeKey = kb.KnowledgeKey // Update knowledgeKey to the one found in DB
			log.Printf("DEBUG: Found knowledge base by IndexId: %s", knowledgeKey)
		} else {
			// If IndexId search fails, try prefix search (for truncated keys)
			log.Printf("DEBUG: IndexId not found, trying prefix search")
			if err := h.db.Where("knowledge_key LIKE ?", knowledgeKey+"%").First(&kb).Error; err != nil {
				response.Fail(c, knowledge.ErrKnowledgeNotFound, err)
				return
			}
			k = &kb
			knowledgeKey = kb.KnowledgeKey // Update knowledgeKey to the one found in DB
			log.Printf("DEBUG: Found knowledge base by prefix: %s", knowledgeKey)
		}
	}
	if k.UserID != userID {
		response.Fail(c, "unauthorized access to knowledge base", nil)
		return
	}

	// Parse config
	config, err := models.GetKnowledgeConfigOrDefault(k.Provider, k.Config, getKnowledgeBaseConfig)
	if err != nil {
		response.Fail(c, knowledge.ErrConfigParseFailed, err)
		return
	}

	// Get knowledge base instance
	kb, err := knowledge.GetKnowledgeBaseByProvider(k.Provider, config)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeBaseInitFailed, err)
		return
	}

	// For Aliyun, we use a wildcard search to get all content
	// This is a workaround since Aliyun doesn't have a direct ListDocuments API
	log.Printf("Listing knowledge base content - key: %s, provider: %s", knowledgeKey, k.Provider)

	// Use the correct key for search (IndexId for Aliyun, knowledgeKey for others)
	searchKey := knowledgeKey
	if k.Provider == knowledge.ProviderAliyun && k.IndexId != "" {
		searchKey = k.IndexId
		log.Printf("DEBUG: Using IndexId for Aliyun search: %s", searchKey)
	}

	// Use a broad search query to retrieve all chunks
	// For Aliyun, use a generic query instead of wildcard
	query := "*"
	if k.Provider == knowledge.ProviderAliyun {
		query = "content" // Use a generic query for Aliyun
	}

	options := knowledge.SearchOptions{
		Query: query,
		TopK:  1000, // Get up to 1000 results
	}
	results, err := kb.Search(context.Background(), searchKey, options)
	if err != nil {
		log.Printf("ERROR: Failed to list knowledge base content - error: %v", err)
		// Return empty results instead of error for empty knowledge bases
		response.Success(c, "content listed successfully", map[string]interface{}{
			"knowledge_key":  knowledgeKey,
			"document_count": 0,
			"total_chunks":   0,
			"documents":      []map[string]interface{}{},
		})
		return
	}

	// Group results by source (document)
	documentMap := make(map[string][]map[string]interface{})
	for _, result := range results {
		source := result.Source
		if source == "" {
			source = "unknown"
		}

		chunk := map[string]interface{}{
			"content":  result.Content,
			"score":    result.Score,
			"metadata": result.Metadata,
		}

		documentMap[source] = append(documentMap[source], chunk)
	}

	// Build response
	documents := make([]map[string]interface{}, 0)
	for source, chunks := range documentMap {
		doc := map[string]interface{}{
			"source":      source,
			"chunk_count": len(chunks),
			"chunks":      chunks,
		}
		documents = append(documents, doc)
	}

	response.Success(c, "content listed successfully", map[string]interface{}{
		"knowledge_key":  knowledgeKey,
		"document_count": len(documents),
		"total_chunks":   len(results),
		"documents":      documents,
	})
}

// Note: Old uploadFile and initKnowledgeBase functions have been removed
// Now using unified knowledge base interface and config parameters, no longer using global variables

// Note: Old initKnowledgeBase function has been removed
// Now using knowledge base interface kb.CreateIndex() to handle index creation

// CreateIndex creates a knowledge base in Aliyun Bailian service (initialization).
//
// Parameters:
//   - client (bailian20231229.Client): Client instance.
//   - workspaceId (string): Workspace ID.
//   - fileId (string): Document ID.
//   - name (string): Knowledge base name.
//   - structureType (string): Knowledge base data type.
//   - sourceType (string): Application data type, supports category type and document type.
//   - sinkType (string): Knowledge base vector storage type.
//
// Returns:
//   - *bailian20231229.CreateIndexResponse: Aliyun Bailian service response.
//   - error: Error information.
func CreateIndex(client *bailian20231229.Client, workspaceId, fileId, name, structureType, sourceType, sinkType string) (_result *bailian20231229.CreateIndexResponse, _err error) {
	headers := make(map[string]*string)
	log.Println("Creating knowledge base")
	createIndexRequest := &bailian20231229.CreateIndexRequest{
		StructureType: tea.String(structureType),
		Name:          tea.String(name),
		SourceType:    tea.String(sourceType),
		SinkType:      tea.String(sinkType),
		DocumentIds:   []*string{tea.String(fileId)},
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.CreateIndexWithOptions(tea.String(workspaceId), createIndexRequest, headers, runtime)
}

// SubmitIndex submits an index job.
//
// Parameters:
//   - client (bailian20231229.Client): Client instance.
//   - workspaceId (string): Workspace ID.
//   - indexId (string): Knowledge base ID.
//
// Returns:
//   - *bailian20231229.SubmitIndexJobResponse: Aliyun Bailian service response.
//   - error: Error information.
func SubmitIndex(client *bailian20231229.Client, workspaceId, indexId string) (_result *bailian20231229.SubmitIndexJobResponse, _err error) {
	headers := make(map[string]*string)
	submitIndexJobRequest := &bailian20231229.SubmitIndexJobRequest{
		IndexId: tea.String(indexId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.SubmitIndexJobWithOptions(tea.String(workspaceId), submitIndexJobRequest, headers, runtime)
}

// GetIndexJobStatus queries index job status.
//
// Parameters:
//   - client (bailian20231229.Client): Client instance.
//   - workspaceId (string): Workspace ID.
//   - jobId (string): Job ID.
//   - indexId (string): Knowledge base ID.
//
// Returns:
//   - *bailian20231229.GetIndexJobStatusResponse: Aliyun Bailian service response.
//   - error: Error information.
func GetIndexJobStatus(client *bailian20231229.Client, workspaceId, jobId, indexId string) (_result *bailian20231229.GetIndexJobStatusResponse, _err error) {
	headers := make(map[string]*string)
	getIndexJobStatusRequest := &bailian20231229.GetIndexJobStatusRequest{
		JobId:   tea.String(jobId),
		IndexId: tea.String(indexId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.GetIndexJobStatusWithOptions(tea.String(workspaceId), getIndexJobStatusRequest, headers, runtime)
}

// CalculateMD5 calculates MD5 value of the document
//
// Parameters:
//   - file (multipart.File): Uploaded file object
//
// Returns:
//   - string: Document MD5 value
//   - error: Error information
func CalculateMD5(file multipart.File) (_result string, _err error) {
	// Reset file pointer to beginning, just in case
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	md5Hash := md5.New()
	_, err = io.Copy(md5Hash, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", md5Hash.Sum(nil)), nil
}

// GetFileSize gets the size of uploaded file (in bytes)
//
// Parameters:
//   - header (*multipart.FileHeader): Uploaded file header information
//
// Returns:
//   - string: File size (in bytes)
//   - error: Error information
func GetFileSize(header *multipart.FileHeader) (_result string, _err error) {
	return fmt.Sprintf("%d", header.Size), nil
}

// ApplyLease applies for document upload lease from Aliyun Bailian service.
//
// Parameters:
//   - client (bailian20231229.Client): Client instance.
//   - categoryId (string): Category ID.
//   - fileName (string): Document name.
//   - fileMD5 (string): Document MD5 value.
//   - fileSize (string): Document size (in bytes).
//   - workspaceId (string): Workspace ID.
//
// Returns:
//   - *bailian20231229.ApplyFileUploadLeaseResponse: Aliyun Bailian service response.
//   - error: Error information.
func ApplyLease(client *bailian20231229.Client, categoryId string, header *multipart.FileHeader, fileMD5 string, fileSize string, workspaceId string) (_result *bailian20231229.ApplyFileUploadLeaseResponse, _err error) {
	headers := make(map[string]*string)
	applyFileUploadLeaseRequest := &bailian20231229.ApplyFileUploadLeaseRequest{
		FileName:    tea.String(header.Filename),
		Md5:         tea.String(fileMD5),
		SizeInBytes: tea.String(fileSize),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.ApplyFileUploadLeaseWithOptions(tea.String(categoryId), tea.String(workspaceId), applyFileUploadLeaseRequest, headers, runtime)
}

// UploadFile sends uploaded file data to presigned URL
//
// Parameters:
//   - preSignedUrl (string): URL in upload lease
//   - headers (map[string]string): Upload request headers
//   - file (multipart.File): Uploaded file object
//
// Returns:
//   - error: Returns error information if upload fails, otherwise returns nil
func UploadFile(preSignedUrl string, headers map[string]string, file multipart.File) error {
	// Reset file pointer to beginning (ensure complete read)
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Create Resty client
	client := resty.New()

	// Send PUT request (direct streaming upload, avoid loading entire file into memory)
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(file). // Directly pass file stream
		Put(preSignedUrl)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Check HTTP status code
	if resp.IsError() {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode())
	}

	return nil
}

func AddFile(client *bailian20231229.Client, leaseId, parser, categoryId, workspaceId string) (_result *bailian20231229.AddFileResponse, _err error) {
	headers := make(map[string]*string)
	addFileRequest := &bailian20231229.AddFileRequest{
		LeaseId:    tea.String(leaseId),
		Parser:     tea.String(parser),
		CategoryId: tea.String(categoryId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.AddFileWithOptions(tea.String(workspaceId), addFileRequest, headers, runtime)
}

// DescribeFile gets basic information of the document.
//
// Parameters:
//   - client (bailian20231229.Client): Client instance.
//   - workspaceId (string): Workspace ID.
//   - fileId (string): Document ID.
//
// Returns:
//   - any: Aliyun Bailian service response.
//   - error: Error information.
func DescribeFile(client *bailian20231229.Client, workspaceId, fileId string) (_result *bailian20231229.DescribeFileResponse, _err error) {
	headers := make(map[string]*string)
	runtime := &teaUtil.RuntimeOptions{}
	return client.DescribeFileWithOptions(tea.String(workspaceId), tea.String(fileId), headers, runtime)
}

// deleteIndex permanently deletes the specified knowledge base.
//
// Parameters:
//   - client      *bailian20231229.Client: Client instance.
//   - workspaceId string: Workspace ID.
//   - indexId     string: Knowledge base ID.
//
// Returns:
//   - *bailian20231229.DeleteIndexResponse: Aliyun Bailian service response.
//   - error: Error information.
func deleteIndex(client *bailian20231229.Client, workspaceId, indexId string) (_result *bailian20231229.DeleteIndexResponse, _err error) {
	headers := make(map[string]*string)
	deleteIndexRequest := &bailian20231229.DeleteIndexRequest{
		IndexId: tea.String(indexId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.DeleteIndexWithOptions(tea.String(workspaceId), deleteIndexRequest, headers, runtime)

}

// SubmitIndexAddDocumentsJob appends parsed documents to an unstructured knowledge base.
//
// Parameters:
//   - client (bailian20231229.Client): Client instance.
//   - workspaceId (string): Workspace ID.
//   - indexId (string): Knowledge base ID.
//   - fileId (string): Document ID.
//   - sourceType (string): Data type.
//
// Returns:
//   - *bailian20231229.SubmitIndexAddDocumentsJobResponse: Aliyun Bailian service response.
//   - error: Error information.
func SubmitIndexAddDocumentsJob(client *bailian20231229.Client, workspaceId, indexId, fileId, sourceType string) (_result *bailian20231229.SubmitIndexAddDocumentsJobResponse, _err error) {
	headers := make(map[string]*string)
	submitIndexAddDocumentsJobRequest := &bailian20231229.SubmitIndexAddDocumentsJobRequest{
		IndexId:     tea.String(indexId),
		SourceType:  tea.String(sourceType),
		DocumentIds: []*string{tea.String(fileId)},
	}
	runtime := &teaUtil.RuntimeOptions{}
	return client.SubmitIndexAddDocumentsJobWithOptions(tea.String(workspaceId), submitIndexAddDocumentsJobRequest, headers, runtime)
}

// fileWrapper wraps io.Reader to implement multipart.File interface
type fileWrapper struct {
	Reader io.Reader
	data   []byte
	offset int64
}

func (fw *fileWrapper) Read(p []byte) (n int, err error) {
	return fw.Reader.Read(p)
}

func (fw *fileWrapper) ReadAt(p []byte, off int64) (n int, err error) {
	if fw.data == nil {
		// Read all data into memory on first ReadAt call
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, fw.Reader); err != nil {
			return 0, err
		}
		fw.data = buf.Bytes()
	}

	if off >= int64(len(fw.data)) {
		return 0, io.EOF
	}

	n = copy(p, fw.data[off:])
	return n, nil
}

func (fw *fileWrapper) Seek(offset int64, whence int) (int64, error) {
	if fw.data == nil {
		// Read all data into memory on first Seek call
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, fw.Reader); err != nil {
			return 0, err
		}
		fw.data = buf.Bytes()
	}

	switch whence {
	case io.SeekStart:
		fw.offset = offset
	case io.SeekCurrent:
		fw.offset += offset
	case io.SeekEnd:
		fw.offset = int64(len(fw.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence")
	}

	if fw.offset < 0 {
		fw.offset = 0
	}
	if fw.offset > int64(len(fw.data)) {
		fw.offset = int64(len(fw.data))
	}

	return fw.offset, nil
}

func (fw *fileWrapper) Close() error {
	return nil
}

// simplifyKnowledgeKey simplifies knowledge key to max 20 characters
// If key is longer than 20 chars, it extracts the userID and generates a short hash
func simplifyKnowledgeKey(key string) string {
	if len(key) <= 20 {
		return key
	}

	// Extract userID (before the first dash)
	parts := strings.Split(key, "-")
	if len(parts) < 2 {
		// No dash found, just truncate
		return key[:20]
	}

	userID := parts[0]
	// Generate a short hash from the rest of the key
	hash := md5.Sum([]byte(key))
	hashStr := fmt.Sprintf("%x", hash)[:8] // Use first 8 chars of hex hash

	// Format: {userID}-{hash}
	result := fmt.Sprintf("%s-%s", userID, hashStr)
	if len(result) > 20 {
		result = result[:20]
	}

	return result
}

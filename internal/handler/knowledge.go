package handlers

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strconv"

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
		knowledge.ConfigKeyAliyunAccessKeyID:     cfg.BailianAccessKeyId,
		knowledge.ConfigKeyAliyunAccessKeySecret: cfg.BailianAccessKeySecret,
		knowledge.ConfigKeyAliyunEndpoint:        models.GetStringOrDefault(cfg.BailianEndpoint, knowledge.DefaultAliyunEndpoint),
		knowledge.ConfigKeyAliyunWorkspaceID:     cfg.BailianWorkspaceId,
		knowledge.ConfigKeyAliyunCategoryID:      cfg.BailianCategoryId,
		knowledge.ConfigKeyAliyunSourceType:      models.GetStringOrDefault(cfg.BailianSourceType, knowledge.DefaultAliyunSourceType),
		knowledge.ConfigKeyAliyunParser:          models.GetStringOrDefault(cfg.BailianParser, knowledge.DefaultAliyunParser),
		knowledge.ConfigKeyAliyunStructType:      models.GetStringOrDefault(cfg.BailianStructType, knowledge.DefaultAliyunStructType),
		knowledge.ConfigKeyAliyunSinkType:        models.GetStringOrDefault(cfg.BailianSinkType, knowledge.DefaultAliyunSinkType),
	}
}

// getMilvusConfig gets Milvus knowledge base config from config file
func getMilvusConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyMilvusAddress:        cfg.MilvusAddress,
		knowledge.ConfigKeyMilvusUsername:       cfg.MilvusUsername,
		knowledge.ConfigKeyMilvusPassword:       cfg.MilvusPassword,
		knowledge.ConfigKeyMilvusCollectionName: cfg.MilvusCollection,
		knowledge.ConfigKeyMilvusDimension:      cfg.MilvusDimension,
	}
}

// getQdrantConfig gets Qdrant knowledge base config from config file
func getQdrantConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyQdrantBaseURL:        cfg.QdrantBaseURL,
		knowledge.ConfigKeyQdrantApiKey:         cfg.QdrantApiKey,
		knowledge.ConfigKeyQdrantCollectionName: cfg.QdrantCollection,
		knowledge.ConfigKeyQdrantDimension:      cfg.QdrantDimension,
	}
}

// getElasticsearchConfig gets Elasticsearch knowledge base config from config file
func getElasticsearchConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyElasticsearchBaseURL:   cfg.ElasticsearchBaseURL,
		knowledge.ConfigKeyElasticsearchUsername:  cfg.ElasticsearchUsername,
		knowledge.ConfigKeyElasticsearchPassword:  cfg.ElasticsearchPassword,
		knowledge.ConfigKeyElasticsearchIndexName: cfg.ElasticsearchIndex,
	}
}

// getPineconeConfig gets Pinecone knowledge base config from config file
func getPineconeConfig() map[string]interface{} {
	cfg := config.GlobalConfig
	return map[string]interface{}{
		knowledge.ConfigKeyPineconeApiKey:    cfg.PineconeApiKey,
		knowledge.ConfigKeyPineconeBaseURL:   cfg.PineconeBaseURL,
		knowledge.ConfigKeyPineconeIndexName: cfg.PineconeIndexName,
		knowledge.ConfigKeyPineconeDimension: cfg.PineconeDimension,
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
	if !cfg.KnowledgeBaseEnabled {
		return nil, fmt.Errorf(knowledge.ErrKnowledgeBaseDisabled)
	}

	// Use provider from config if not specified
	if provider == "" {
		provider = cfg.KnowledgeBaseProvider
		if provider == "" {
			provider = knowledge.DefaultProvider // default to Aliyun
		}
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
	// 1. Receive file
	file, header, err := c.Request.FormFile(constants.FormFieldFile)
	if err != nil {
		response.Fail(c, knowledge.ErrFileReceiveFailed, err)
		return
	}
	if file == nil || header == nil {
		response.Fail(c, knowledge.ErrFileEmpty, nil)
		return
	}
	defer file.Close()

	// 2. Receive request parameters
	knowledgeName := c.PostForm(constants.FormFieldKnowledgeName)
	provider := c.PostForm(constants.FormFieldProvider) // optional, default to aliyun
	if provider == "" {
		provider = knowledge.DefaultProvider
	}

	// 获取组织ID（可选）
	var groupID *uint
	if groupIDStr := c.PostForm("group_id"); groupIDStr != "" {
		if parsedID, err := strconv.ParseUint(groupIDStr, 10, 32); err == nil {
			groupIDVal := uint(parsedID)
			groupID = &groupIDVal
		}
	}

	log.Printf("Creating knowledge base - name: %s, provider: %s, groupID: %v", knowledgeName, provider, groupID)
	user := models.CurrentUser(c)
	userId := int(user.ID)

	// 如果指定了组织ID，验证用户是否有权限在该组织创建共享知识库
	if groupID != nil {
		var group models.Group
		if err := h.db.First(&group, *groupID).Error; err != nil {
			response.Fail(c, "组织不存在", nil)
			return
		}
		// 检查用户是否是组织的创建者或管理员
		if group.CreatorID != user.ID {
			var member models.GroupMember
			if err := h.db.Where("group_id = ? AND user_id = ? AND role = ?", *groupID, user.ID, models.GroupRoleAdmin).First(&member).Error; err != nil {
				response.Fail(c, "权限不足", "只有创建者或管理员可以创建组织共享的知识库")
				return
			}
		}
	}

	// 3. Process knowledge base name (prefix with userID)
	knowledgeName = models.GenerateKnowledgeName(userId, knowledgeName)

	// 4. Get knowledge base instance
	kb, err := getKnowledgeBase(provider)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeBaseInitFailed, err)
		return
	}

	// 5. Get config for the specified provider
	config := getKnowledgeBaseConfig(provider)

	// 6. Generate knowledge base key (userID + knowledge name)
	knowledgeKey := models.GenerateKnowledgeKey(userId, knowledgeName)

	// 7. Handle file upload and index creation based on provider
	var indexId string
	if provider == knowledge.ProviderAliyun {
		// Aliyun: need to upload file first to get fileId, then create index
		aliyunConfig := getAliyunConfig()
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
		indexId, err = kb.CreateIndex(context.Background(), knowledgeKey, createConfig)
		if err != nil {
			response.Fail(c, knowledge.ErrIndexCreateFailed, err)
			return
		}
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
		if qdrantCollectionName, ok := config[knowledge.ConfigKeyQdrantCollectionName]; ok {
			createConfig[knowledge.ConfigKeyQdrantCollectionName] = qdrantCollectionName
		}
		if indexName, ok := config[knowledge.ConfigKeyElasticsearchIndexName]; ok {
			createConfig[knowledge.ConfigKeyElasticsearchIndexName] = indexName
		}
		if pineconeIndexName, ok := config[knowledge.ConfigKeyPineconeIndexName]; ok {
			createConfig[knowledge.ConfigKeyPineconeIndexName] = pineconeIndexName
		}

		indexId, err = kb.CreateIndex(context.Background(), knowledgeKey, createConfig)
		if err != nil {
			response.Fail(c, knowledge.ErrIndexCreateFailed, err)
			return
		}
	}

	// 8. Call models layer to create knowledge base record (use indexId as knowledgeKey)
	knowledgeRecord, err := models.CreateKnowledge(h.db, int(userId), indexId, knowledgeName, provider, config, groupID)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	// 9. Return success response
	response.Success(c, "created successfully", knowledgeRecord)
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

	// 3. Get knowledge base info from database
	k, err := models.GetKnowledge(h.db, knowledgeKey)
	if err != nil {
		response.Fail(c, knowledge.ErrKnowledgeNotFound, err)
		return
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

	// 6. Upload document to knowledge base
	metadata := map[string]interface{}{
		knowledge.MetadataKeyUserID: k.UserID,
		knowledge.MetadataKeyName:   k.KnowledgeName,
		knowledge.MetadataKeySource: knowledge.MetadataSourceAPIUpload,
	}

	err = kb.UploadDocument(context.Background(), knowledgeKey, file, header, metadata)
	if err != nil {
		response.Fail(c, knowledge.ErrFileUploadFailed, err)
		return
	}

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

	// Build response data with complete information
	result := make([]map[string]interface{}, 0, len(knowledgeList))
	for _, kb := range knowledgeList {
		// Parse config
		config, _ := models.ParseKnowledgeConfig(kb.Config)
		if config == nil {
			config = make(map[string]interface{})
		}

		item := map[string]interface{}{
			"id":             kb.ID,
			"user_id":        kb.UserID,
			"group_id":       kb.GroupID,
			"knowledge_key":  kb.KnowledgeKey,
			"knowledge_name": kb.KnowledgeName,
			"provider":       kb.Provider,
			"config":         config,
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

package knowledge

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"

	bailian20231229 "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	teaUtil "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/go-resty/resty/v2"
)

// AliyunConfig Aliyun knowledge base configuration
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	WorkspaceID     string
	CategoryID      string
	SourceType      string
	Parser          string
	StructType      string
	SinkType        string
}

// aliyunKnowledgeBase Aliyun knowledge base implementation
type aliyunKnowledgeBase struct {
	client      *bailian20231229.Client
	workspaceId string
	categoryId  string
	sourceType  string
	parser      string
	structType  string
	sinkType    string
}

// NewAliyunKnowledgeBase creates Aliyun knowledge base instance
func NewAliyunKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	// Extract parameters from config
	accessKeyID := getStringFromConfig(config, ConfigKeyAliyunAccessKeyID)
	accessKeySecret := getStringFromConfig(config, ConfigKeyAliyunAccessKeySecret)
	endpoint := getStringFromConfig(config, ConfigKeyAliyunEndpoint)
	if endpoint == "" {
		endpoint = DefaultAliyunEndpoint
	}
	workspaceID := getStringFromConfig(config, ConfigKeyAliyunWorkspaceID)
	categoryID := getStringFromConfig(config, ConfigKeyAliyunCategoryID)
	sourceType := getStringFromConfig(config, ConfigKeyAliyunSourceType)
	if sourceType == "" {
		sourceType = DefaultAliyunSourceType
	}
	parser := getStringFromConfig(config, ConfigKeyAliyunParser)
	if parser == "" {
		parser = DefaultAliyunParser
	}
	structType := getStringFromConfig(config, ConfigKeyAliyunStructType)
	if structType == "" {
		structType = DefaultAliyunStructType
	}
	sinkType := getStringFromConfig(config, ConfigKeyAliyunSinkType)
	if sinkType == "" {
		sinkType = DefaultAliyunSinkType
	}

	if accessKeyID == "" || accessKeySecret == "" {
		return nil, fmt.Errorf(ErrAccessKeyRequired)
	}

	// Create Aliyun client
	openapiConfig := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String(endpoint),
	}
	client, err := bailian20231229.NewClient(openapiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun client: %w", err)
	}

	return &aliyunKnowledgeBase{
		client:      client,
		workspaceId: workspaceID,
		categoryId:  categoryID,
		sourceType:  sourceType,
		parser:      parser,
		structType:  structType,
		sinkType:    sinkType,
	}, nil
}

// Provider returns provider name
func (a *aliyunKnowledgeBase) Provider() string {
	return ProviderAliyun
}

// Search searches for relevant information in the knowledge base
func (a *aliyunKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	if options.Query == "" {
		options.Query = "Please give me information from this knowledge base"
	}

	// Call Aliyun search API
	headers := make(map[string]*string)
	request := &bailian20231229.RetrieveRequest{
		IndexId: tea.String(knowledgeKey),
		Query:   tea.String(options.Query),
	}
	// Note: Aliyun RetrieveRequest may not support TopK field, commented out
	// if options.TopK > 0 {
	// 	request.TopK = tea.Int32(int32(options.TopK))
	// }

	runtime := &teaUtil.RuntimeOptions{}
	response, err := a.client.RetrieveWithOptions(tea.String(a.workspaceId), request, headers, runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to search knowledge base: %w", err)
	}

	// Check response result
	if response.GetBody() == nil || response.GetBody().Data == nil {
		return nil, fmt.Errorf("knowledge base returned empty data")
	}

	// Convert to unified result format
	var results []SearchResult
	// Determine max results to return (limit to TopK if specified, default to 5)
	maxResults := options.TopK
	if maxResults <= 0 {
		maxResults = 5 // Default to 5 if not specified
	}

	for _, node := range response.GetBody().Data.GetNodes() {
		// Limit results to maxResults
		if len(results) >= maxResults {
			break
		}

		if node.GetText() == nil {
			continue
		}

		text := *node.GetText()
		// Calculate relevance score (score returned by Aliyun)
		score := 1.0
		if node.GetScore() != nil {
			score = *node.GetScore()
		}

		// Apply threshold filter
		if options.Threshold > 0 && score < options.Threshold {
			continue
		}

		result := SearchResult{
			Content:  text,
			Score:    score,
			Metadata: make(map[string]interface{}),
		}

		// Add metadata
		if node.GetMetadata() != nil {
			if meta, ok := node.GetMetadata().(map[string]interface{}); ok {
				result.Metadata = meta
			}
		}

		// Add source information (if API supports)
		// Note: Aliyun RetrieveResponseBodyDataNodes may not have GetSource method
		// Can get source information from metadata
		if result.Metadata != nil {
			if source, ok := result.Metadata["source"].(string); ok {
				result.Source = source
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// CreateIndex creates knowledge base index
func (a *aliyunKnowledgeBase) CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	// Get fileId from config (if provided)
	fileId := getStringFromConfig(config, "file_id")
	if fileId == "" {
		return "", fmt.Errorf("file_id is required for creating index")
	}

	// Use type from config or default value
	structType := getStringFromConfig(config, ConfigKeyAliyunStructType)
	if structType == "" {
		structType = a.structType
	}
	sourceType := getStringFromConfig(config, ConfigKeyAliyunSourceType)
	if sourceType == "" {
		sourceType = a.sourceType
	}
	sinkType := getStringFromConfig(config, ConfigKeyAliyunSinkType)
	if sinkType == "" {
		sinkType = a.sinkType
	}

	headers := make(map[string]*string)
	createIndexRequest := &bailian20231229.CreateIndexRequest{
		StructureType: tea.String(structType),
		Name:          tea.String(name),
		SourceType:    tea.String(sourceType),
		SinkType:      tea.String(sinkType),
		DocumentIds:   []*string{tea.String(fileId)},
	}
	runtime := &teaUtil.RuntimeOptions{}

	response, err := a.client.CreateIndexWithOptions(tea.String(a.workspaceId), createIndexRequest, headers, runtime)
	if err != nil {
		return "", fmt.Errorf("failed to create knowledge base: %w", err)
	}

	if response.GetBody() == nil || response.GetBody().Data == nil || response.GetBody().Data.Id == nil {
		message := ""
		if response.GetBody() != nil && response.GetBody().Message != nil {
			message = *response.GetBody().Message
		}
		return "", fmt.Errorf("failed to create knowledge base: %s", message)
	}

	indexId := *response.GetBody().Data.Id

	// Submit index job
	submitResponse, err := a.submitIndex(indexId)
	if err != nil {
		return "", fmt.Errorf("failed to submit index: %w", err)
	}

	jobId := *submitResponse.GetBody().Data.Id

	// Wait for index job to complete (simplified handling here, should poll status in practice)
	_, err = a.getIndexJobStatus(jobId, indexId)
	if err != nil {
		return "", fmt.Errorf("failed to get index status: %w", err)
	}

	return indexId, nil
}

// DeleteIndex deletes knowledge base index
func (a *aliyunKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	headers := make(map[string]*string)
	deleteIndexRequest := &bailian20231229.DeleteIndexRequest{
		IndexId: tea.String(knowledgeKey),
	}
	runtime := &teaUtil.RuntimeOptions{}

	_, err := a.client.DeleteIndexWithOptions(tea.String(a.workspaceId), deleteIndexRequest, headers, runtime)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge base: %w", err)
	}

	return nil
}

// UploadDocument uploads document to knowledge base
func (a *aliyunKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	// 1. Calculate file MD5 and size
	md5Hash, err := calculateMD5(file)
	if err != nil {
		return fmt.Errorf("failed to calculate MD5: %w", err)
	}

	fileSize := fmt.Sprintf("%d", header.Size)

	// 2. Apply for upload lease
	lease, err := a.applyLease(header, md5Hash, fileSize)
	if err != nil {
		return fmt.Errorf("failed to apply upload lease: %w", err)
	}

	leaseBody := lease.GetBody()
	if leaseBody == nil || leaseBody.Data == nil || leaseBody.Data.Param == nil {
		return fmt.Errorf("申请上传租约失败: 响应数据不完整")
	}

	// 3. 上传文件到预签名URL
	preSignedUrl := leaseBody.Data.Param.Url
	headers := make(map[string]string)
	if leaseBody.Data.Param.Headers != nil {
		if headerMap, ok := leaseBody.Data.Param.Headers.(map[string]interface{}); ok {
			for k, v := range headerMap {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}
	}

	err = uploadFileToURL(*preSignedUrl, headers, file)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	// 4. 调用AddFile接口
	leaseId := *leaseBody.Data.FileUploadLeaseId
	fileResponse, err := a.addFile(leaseId)
	if err != nil {
		return fmt.Errorf("添加文件失败: %w", err)
	}

	fileId := *fileResponse.GetBody().Data.FileId

	// 5. 将文件添加到知识库
	_, err = a.submitIndexAddDocumentsJob(knowledgeKey, fileId)
	if err != nil {
		return fmt.Errorf("提交添加文档任务失败: %w", err)
	}

	return nil
}

// DeleteDocument 从知识库删除文档（阿里云不支持单独删除文档，需要删除整个知识库）
func (a *aliyunKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	// 阿里云百炼不支持单独删除文档，需要删除整个知识库或使用其他方式
	return fmt.Errorf("aliyun knowledge base does not support deleting individual documents")
}

// ListDocuments 列出知识库中的所有文档
func (a *aliyunKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	// 阿里云百炼没有直接的API列出文档列表
	// 可以通过DescribeIndex获取索引信息，但可能不包含文档列表
	return nil, fmt.Errorf("aliyun knowledge base does not support listing documents")
}

// GetDocument 获取文档内容
func (a *aliyunKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	// 阿里云百炼没有直接的API获取文档内容
	return nil, fmt.Errorf("aliyun knowledge base does not support getting document content")
}

// 辅助方法

func (a *aliyunKnowledgeBase) submitIndex(indexId string) (*bailian20231229.SubmitIndexJobResponse, error) {
	headers := make(map[string]*string)
	request := &bailian20231229.SubmitIndexJobRequest{
		IndexId: tea.String(indexId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return a.client.SubmitIndexJobWithOptions(tea.String(a.workspaceId), request, headers, runtime)
}

func (a *aliyunKnowledgeBase) getIndexJobStatus(jobId, indexId string) (*bailian20231229.GetIndexJobStatusResponse, error) {
	headers := make(map[string]*string)
	request := &bailian20231229.GetIndexJobStatusRequest{
		JobId:   tea.String(jobId),
		IndexId: tea.String(indexId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return a.client.GetIndexJobStatusWithOptions(tea.String(a.workspaceId), request, headers, runtime)
}

func (a *aliyunKnowledgeBase) applyLease(header *multipart.FileHeader, fileMD5, fileSize string) (*bailian20231229.ApplyFileUploadLeaseResponse, error) {
	headers := make(map[string]*string)
	request := &bailian20231229.ApplyFileUploadLeaseRequest{
		FileName:    tea.String(header.Filename),
		Md5:         tea.String(fileMD5),
		SizeInBytes: tea.String(fileSize),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return a.client.ApplyFileUploadLeaseWithOptions(tea.String(a.categoryId), tea.String(a.workspaceId), request, headers, runtime)
}

func (a *aliyunKnowledgeBase) addFile(leaseId string) (*bailian20231229.AddFileResponse, error) {
	headers := make(map[string]*string)
	request := &bailian20231229.AddFileRequest{
		LeaseId:    tea.String(leaseId),
		Parser:     tea.String(a.parser),
		CategoryId: tea.String(a.categoryId),
	}
	runtime := &teaUtil.RuntimeOptions{}
	return a.client.AddFileWithOptions(tea.String(a.workspaceId), request, headers, runtime)
}

func (a *aliyunKnowledgeBase) submitIndexAddDocumentsJob(indexId, fileId string) (*bailian20231229.SubmitIndexAddDocumentsJobResponse, error) {
	headers := make(map[string]*string)
	request := &bailian20231229.SubmitIndexAddDocumentsJobRequest{
		IndexId:     tea.String(indexId),
		SourceType:  tea.String(a.sourceType),
		DocumentIds: []*string{tea.String(fileId)},
	}
	runtime := &teaUtil.RuntimeOptions{}
	return a.client.SubmitIndexAddDocumentsJobWithOptions(tea.String(a.workspaceId), request, headers, runtime)
}

// 注意：getStringFromConfig 已移至 config.go

func calculateMD5(file multipart.File) (string, error) {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func uploadFileToURL(url string, headers map[string]string, file multipart.File) error {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("重置文件指针失败: %w", err)
	}

	client := resty.New()
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(file).
		Put(url)

	if err != nil {
		return fmt.Errorf("请求发送失败: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf("HTTP错误: %d", resp.StatusCode())
	}

	return nil
}

// 注册阿里云提供者
func init() {
	RegisterKnowledgeBaseProvider(ProviderAliyun, func(config map[string]interface{}) (KnowledgeBase, error) {
		return NewAliyunKnowledgeBase(config)
	})
}

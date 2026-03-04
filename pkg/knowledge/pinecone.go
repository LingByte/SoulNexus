package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// pineconeKnowledgeBase Pinecone向量数据库实现
type pineconeKnowledgeBase struct {
	apiKey     string
	baseURL    string
	indexName  string
	dimension  int
	httpClient *http.Client
}

// NewPineconeKnowledgeBase 创建Pinecone知识库实例
func NewPineconeKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	apiKey := getStringFromConfig(config, "api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	baseURL := getStringFromConfig(config, "base_url")
	if baseURL == "" {
		baseURL = "https://api.pinecone.io"
	}

	indexName := getStringFromConfig(config, "index_name")
	if indexName == "" {
		return nil, fmt.Errorf("index_name is required")
	}

	dimension := getIntFromConfig(config, "dimension")
	if dimension == 0 {
		dimension = 1536 // Pinecone默认维度
	}

	return &pineconeKnowledgeBase{
		apiKey:     apiKey,
		baseURL:    baseURL,
		indexName:  indexName,
		dimension:  dimension,
		httpClient: &http.Client{},
	}, nil
}

func (p *pineconeKnowledgeBase) Provider() string {
	return ProviderPinecone
}

func (p *pineconeKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 获取embedding向量
	queryEmbedding := getFloatVectorFromConfig(options.Filter, "embedding")
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("embedding vector is required for pinecone search")
	}

	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}

	// 构建Pinecone查询请求
	queryReq := map[string]interface{}{
		"vector":          queryEmbedding,
		"topK":            topK,
		"includeMetadata": true,
	}

	if options.Threshold > 0 {
		queryReq["minScore"] = options.Threshold
	}

	reqBody, _ := json.Marshal(queryReq)
	url := fmt.Sprintf("%s/indexes/%s/query", p.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Api-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pinecone query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pinecone query failed with status: %d", resp.StatusCode)
	}

	var queryResp struct {
		Matches []struct {
			ID       string                 `json:"id"`
			Score    float64                `json:"score"`
			Metadata map[string]interface{} `json:"metadata"`
			Values   []float32              `json:"values"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var results []SearchResult
	for _, match := range queryResp.Matches {
		// 从metadata中提取content
		content := ""
		if contentField, ok := match.Metadata["content"].(string); ok {
			content = contentField
		}

		score := float64(match.Score)
		if options.Threshold > 0 && score < options.Threshold {
			continue
		}

		result := SearchResult{
			Content:  content,
			Score:    score,
			Metadata: match.Metadata,
			Source:   match.ID,
		}
		results = append(results, result)
	}

	return results, nil
}

func (p *pineconeKnowledgeBase) CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	// Pinecone索引通过其控制台创建，这里返回索引名称
	indexName := name
	if in := getStringFromConfig(config, "index_name"); in != "" {
		indexName = in
	}

	// 检查索引是否存在（简化实现）
	// 实际应该调用Pinecone API检查
	return indexName, nil
}

func (p *pineconeKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	// Pinecone索引删除需要通过API
	url := fmt.Sprintf("%s/indexes/%s", p.baseURL, knowledgeKey)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Api-Key", p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete index with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *pineconeKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	// Pinecone上传需要先进行文本处理和embedding
	return fmt.Errorf("pinecone upload document not fully implemented, requires text processing and embedding generation")
}

func (p *pineconeKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// 构建删除请求
	deleteReq := map[string]interface{}{
		"ids": []string{documentID},
	}

	reqBody, _ := json.Marshal(deleteReq)
	url := fmt.Sprintf("%s/indexes/%s/vectors/delete", p.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Api-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete document with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *pineconeKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	// Pinecone不直接支持列出所有文档ID
	// 需要通过metadata查询或使用其他方式
	return nil, fmt.Errorf("pinecone does not support listing all documents directly")
}

func (p *pineconeKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	// 通过ID查询向量数据
	queryReq := map[string]interface{}{
		"ids":             []string{documentID},
		"includeMetadata": true,
		"includeValues":   false,
	}

	reqBody, _ := json.Marshal(queryReq)
	url := fmt.Sprintf("%s/indexes/%s/vectors/fetch", p.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Api-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch document with status: %d", resp.StatusCode)
	}

	var fetchResp struct {
		Vectors map[string]struct {
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"vectors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&fetchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if vec, ok := fetchResp.Vectors[documentID]; ok {
		content := ""
		if contentField, ok := vec.Metadata["content"].(string); ok {
			content = contentField
		}
		return io.NopCloser(strings.NewReader(content)), nil
	}

	return nil, fmt.Errorf("document not found")
}

// 注册Pinecone提供者
func init() {
	RegisterKnowledgeBaseProvider(ProviderPinecone, func(config map[string]interface{}) (KnowledgeBase, error) {
		return NewPineconeKnowledgeBase(config)
	})
}

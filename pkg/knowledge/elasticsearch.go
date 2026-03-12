package knowledge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// elasticsearchKnowledgeBase Elasticsearch全文搜索实现
type elasticsearchKnowledgeBase struct {
	baseURL    string
	username   string
	password   string
	indexName  string
	httpClient *http.Client
}

// NewElasticsearchKnowledgeBase 创建Elasticsearch知识库实例
func NewElasticsearchKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	baseURL := getStringFromConfig(config, "base_url")
	if baseURL == "" {
		baseURL = "http://localhost:9200"
	}

	username := getStringFromConfig(config, "username")
	password := getStringFromConfig(config, "password")

	indexName := getStringFromConfig(config, "index_name")
	if indexName == "" {
		return nil, fmt.Errorf("index_name is required")
	}

	return &elasticsearchKnowledgeBase{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		indexName:  indexName,
		httpClient: &http.Client{},
	}, nil
}

func (e *elasticsearchKnowledgeBase) Provider() string {
	return ProviderElasticsearch
}

func (e *elasticsearchKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}

	// 构建Elasticsearch查询请求
	query := map[string]interface{}{
		"size": topK,
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  options.Query,
				"fields": []string{"content^2", "title", "metadata"},
			},
		},
		"_source": []string{"content", "metadata", "title"},
	}

	if options.Threshold > 0 {
		query["min_score"] = options.Threshold
	}

	reqBody, _ := json.Marshal(query)
	url := fmt.Sprintf("%s/%s/_search", e.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elasticsearch search failed with status: %d", resp.StatusCode)
	}

	var searchResp struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Score  float64                `json:"_score"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var results []SearchResult
	for _, hit := range searchResp.Hits.Hits {
		// 从source中提取content
		content := ""
		if contentField, ok := hit.Source["content"].(string); ok {
			content = contentField
		}

		// 提取metadata
		metadata := make(map[string]interface{})
		if metaField, ok := hit.Source["metadata"].(map[string]interface{}); ok {
			metadata = metaField
		}

		// 归一化分数（Elasticsearch分数可能很大，需要归一化到0-1）
		score := hit.Score / (hit.Score + 1.0) // 简单的归一化
		if score > 1.0 {
			score = 1.0
		}

		if options.Threshold > 0 && score < options.Threshold {
			continue
		}

		result := SearchResult{
			Content:  content,
			Score:    score,
			Metadata: metadata,
			Source:   hit.ID,
		}
		results = append(results, result)
	}

	return results, nil
}

func (e *elasticsearchKnowledgeBase) CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	indexName := name
	if in := getStringFromConfig(config, "index_name"); in != "" {
		indexName = in
	}

	// 检查索引是否存在
	url := fmt.Sprintf("%s/%s", e.baseURL, indexName)
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to check index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// 索引已存在
		return indexName, nil
	}

	// 创建新的索引
	indexSettings := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":     "text",
					"analyzer": "ik_max_word",
				},
				"title": map[string]interface{}{
					"type": "text",
				},
				"metadata": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	reqBody, _ := json.Marshal(indexSettings)
	url = fmt.Sprintf("%s/%s", e.baseURL, indexName)

	req, err = http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	e.addAuth(req)

	resp, err = e.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create index with status: %d", resp.StatusCode)
	}

	return indexName, nil
}

func (e *elasticsearchKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/%s", e.baseURL, knowledgeKey)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete index with status: %d", resp.StatusCode)
	}

	return nil
}

func (e *elasticsearchKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// 读取文件内容
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 构建文档
	doc := map[string]interface{}{
		"content":  string(fileContent),
		"title":    header.Filename,
		"metadata": metadata,
	}

	reqBody, _ := json.Marshal(doc)
	// 使用文档ID或者让ES自动生成
	url := fmt.Sprintf("%s/%s/_doc", e.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload document with status: %d", resp.StatusCode)
	}

	return nil
}

func (e *elasticsearchKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", e.baseURL, knowledgeKey, documentID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete document with status: %d", resp.StatusCode)
	}

	return nil
}

func (e *elasticsearchKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 使用search API获取所有文档ID
	query := map[string]interface{}{
		"size":    10000,
		"_source": false,
	}

	reqBody, _ := json.Marshal(query)
	url := fmt.Sprintf("%s/%s/_search", e.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list documents with status: %d", resp.StatusCode)
	}

	var searchResp struct {
		Hits struct {
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var ids []string
	for _, hit := range searchResp.Hits.Hits {
		ids = append(ids, hit.ID)
	}

	return ids, nil
}

func (e *elasticsearchKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", e.baseURL, knowledgeKey, documentID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	e.addAuth(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get document with status: %d", resp.StatusCode)
	}

	var getResp struct {
		Source map[string]interface{} `json:"_source"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	content := ""
	if contentField, ok := getResp.Source["content"].(string); ok {
		content = contentField
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

// addAuth 添加HTTP基本认证
func (e *elasticsearchKnowledgeBase) addAuth(req *http.Request) {
	if e.username != "" && e.password != "" {
		auth := e.username + ":" + e.password
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encoded)
	}
}

// 注册Elasticsearch提供者
func init() {
	RegisterKnowledgeBaseProvider(ProviderElasticsearch, func(config map[string]interface{}) (KnowledgeBase, error) {
		return NewElasticsearchKnowledgeBase(config)
	})
}

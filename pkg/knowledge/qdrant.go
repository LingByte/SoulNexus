package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// qdrantKnowledgeBase Qdrant向量数据库实现
type qdrantKnowledgeBase struct {
	apiKey         string
	baseURL        string
	collectionName string
	dimension      int
	httpClient     *http.Client
}

// NewQdrantKnowledgeBase 创建Qdrant知识库实例
func NewQdrantKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	baseURL := getStringFromConfig(config, "base_url")
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}

	apiKey := getStringFromConfig(config, "api_key")
	collectionName := getStringFromConfig(config, "collection_name")
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}

	dimension := getIntFromConfig(config, "dimension")
	if dimension == 0 {
		dimension = 384 // Qdrant默认维度
	}

	httpClient := &http.Client{}
	if apiKey != "" {
		// 可以使用自定义transport添加认证
	}

	return &qdrantKnowledgeBase{
		apiKey:         apiKey,
		baseURL:        baseURL,
		collectionName: collectionName,
		dimension:      dimension,
		httpClient:     httpClient,
	}, nil
}

func (q *qdrantKnowledgeBase) Provider() string {
	return ProviderQdrant
}

func (q *qdrantKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 获取embedding向量
	queryEmbedding := getFloatVectorFromConfig(options.Filter, "embedding")
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("embedding vector is required for qdrant search")
	}

	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}

	// 构建Qdrant搜索请求
	searchReq := map[string]interface{}{
		"vector":       queryEmbedding,
		"limit":        topK,
		"with_payload": true,
		"with_vector":  false,
	}

	if options.Threshold > 0 {
		searchReq["score_threshold"] = options.Threshold
	}

	reqBody, _ := json.Marshal(searchReq)
	url := fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant search failed with status: %d", resp.StatusCode)
	}

	var searchResp struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var results []SearchResult
	for _, hit := range searchResp.Result {
		// 从payload中提取content
		content := ""
		if contentField, ok := hit.Payload["content"].(string); ok {
			content = contentField
		}

		score := hit.Score
		if options.Threshold > 0 && score < options.Threshold {
			continue
		}

		// 转换ID为字符串
		idStr := fmt.Sprintf("%v", hit.ID)

		result := SearchResult{
			Content:  content,
			Score:    score,
			Metadata: hit.Payload,
			Source:   idStr,
		}
		results = append(results, result)
	}

	return results, nil
}

func (q *qdrantKnowledgeBase) CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	collectionName := name
	if cn := getStringFromConfig(config, "collection_name"); cn != "" {
		collectionName = cn
	}

	// 检查collection是否存在
	url := fmt.Sprintf("%s/collections/%s", q.baseURL, collectionName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Collection已存在
		return collectionName, nil
	}

	// 创建新的collection
	dimension := q.dimension
	if d := getIntFromConfig(config, "dimension"); d > 0 {
		dimension = d
	}

	createReq := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	reqBody, _ := json.Marshal(createReq)
	url = fmt.Sprintf("%s/collections/%s", q.baseURL, collectionName)

	req, err = http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err = q.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to create collection with status: %d", resp.StatusCode)
	}

	return collectionName, nil
}

func (q *qdrantKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/collections/%s", q.baseURL, knowledgeKey)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete collection with status: %d", resp.StatusCode)
	}

	return nil
}

func (q *qdrantKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	// Qdrant上传需要先进行文本处理和embedding
	return fmt.Errorf("qdrant upload document not fully implemented, requires text processing and embedding generation")
}

func (q *qdrantKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	deleteReq := map[string]interface{}{
		"points": []string{documentID},
	}

	reqBody, _ := json.Marshal(deleteReq)
	url := fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete document with status: %d", resp.StatusCode)
	}

	return nil
}

func (q *qdrantKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Qdrant可以通过scroll API获取所有点
	scrollReq := map[string]interface{}{
		"limit":        1000,
		"with_payload": false,
		"with_vector":  false,
	}

	reqBody, _ := json.Marshal(scrollReq)
	url := fmt.Sprintf("%s/collections/%s/points/scroll", q.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list documents with status: %d", resp.StatusCode)
	}

	var scrollResp struct {
		Result struct {
			Points []struct {
				ID interface{} `json:"id"`
			} `json:"points"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&scrollResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var ids []string
	for _, point := range scrollResp.Result.Points {
		ids = append(ids, fmt.Sprintf("%v", point.ID))
	}

	return ids, nil
}

func (q *qdrantKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/collections/%s/points/%s", q.baseURL, knowledgeKey, documentID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get document with status: %d", resp.StatusCode)
	}

	var getResp struct {
		Result struct {
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	content := ""
	if contentField, ok := getResp.Result.Payload["content"].(string); ok {
		content = contentField
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

// 注册Qdrant提供者
func init() {
	RegisterKnowledgeBaseProvider(ProviderQdrant, func(config map[string]interface{}) (KnowledgeBase, error) {
		return NewQdrantKnowledgeBase(config)
	})
}

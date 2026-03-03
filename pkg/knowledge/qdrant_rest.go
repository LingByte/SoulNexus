package knowledge

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// qdrantRESTKnowledgeBase Qdrant REST API实现
// 用于gRPC连接失败的情况
type qdrantRESTKnowledgeBase struct {
	baseURL   string
	apiKey    string
	dimension int
	client    *http.Client
}

// NewQdrantRESTKnowledgeBase 创建Qdrant REST知识库实例
func NewQdrantRESTKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	// 从 config 中获取 baseURL
	baseURL := getStringFromConfig(config, ConfigKeyQdrantHost)
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}

	// 确保baseURL格式正确
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}

	apiKey := getStringFromConfig(config, ConfigKeyQdrantAPIKey)

	dimension := getIntFromConfig(config, ConfigKeyQdrantDimension)
	if dimension == 0 {
		dimension = 384
	}

	return &qdrantRESTKnowledgeBase{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		apiKey:    apiKey,
		dimension: dimension,
		client:    &http.Client{},
	}, nil
}

func (q *qdrantRESTKnowledgeBase) Provider() string {
	return ProviderQdrant
}

func (q *qdrantRESTKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	// 获取embedding向量
	queryEmbedding := getFloatVectorFromConfig(options.Filter, "embedding")

	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}

	if len(queryEmbedding) == 0 {
		// 没有embedding，列出所有文档
		return q.listAllPoints(ctx, knowledgeKey, topK)
	}

	// 有embedding，执行向量搜索
	return q.searchPoints(ctx, knowledgeKey, queryEmbedding, topK, options.Threshold)
}

func (q *qdrantRESTKnowledgeBase) searchPoints(ctx context.Context, collectionName string, vector []float32, limit int, threshold float64) ([]SearchResult, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, collectionName)

	payload := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}

	if threshold > 0 {
		payload["score_threshold"] = threshold
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID      interface{}            `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	var results []SearchResult
	for _, item := range result.Result {
		content := ""
		if payload, ok := item.Payload["content"].(string); ok {
			content = payload
		}

		results = append(results, SearchResult{
			Content:  content,
			Score:    item.Score,
			Metadata: item.Payload,
			Source:   fmt.Sprintf("%v", item.ID),
		})
	}

	return results, nil
}

func (q *qdrantRESTKnowledgeBase) listAllPoints(ctx context.Context, collectionName string, limit int) ([]SearchResult, error) {
	// 使用 scroll 端点来列出所有点
	url := fmt.Sprintf("%s/collections/%s/points/scroll", q.baseURL, collectionName)

	payload := map[string]interface{}{
		"limit":        limit,
		"with_payload": true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scroll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scroll failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result struct {
			Points []struct {
				ID      interface{}            `json:"id"`
				Payload map[string]interface{} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode scroll response: %w", err)
	}

	var results []SearchResult
	for _, point := range result.Result.Points {
		content := ""
		if payload, ok := point.Payload["content"]; ok {
			if contentStr, ok := payload.(string); ok {
				content = contentStr
			}
		}

		result := SearchResult{
			Content:  content,
			Score:    1.0,
			Metadata: make(map[string]interface{}),
			Source:   fmt.Sprintf("%v", point.ID),
		}
		results = append(results, result)
	}

	return results, nil
}

func (q *qdrantRESTKnowledgeBase) CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/collections/%s", q.baseURL, name)

	// 检查collection是否存在
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Collection已存在
		return name, nil
	}

	// 创建新collection
	dimension := q.dimension
	if d := getIntFromConfig(config, ConfigKeyQdrantDimension); d > 0 {
		dimension = d
	}

	createPayload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	body, err := json.Marshal(createPayload)
	if err != nil {
		return "", err
	}

	req, err = http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err = q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create collection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create collection failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return name, nil
}

func (q *qdrantRESTKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	url := fmt.Sprintf("%s/collections/%s", q.baseURL, knowledgeKey)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete collection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete collection failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (q *qdrantRESTKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	// 先检查 collection 是否存在，如果不存在则创建
	exists, err := q.collectionExists(ctx, knowledgeKey)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		_, err := q.CreateIndex(ctx, knowledgeKey, map[string]interface{}{
			ConfigKeyQdrantDimension: q.dimension,
		})
		if err != nil {
			return fmt.Errorf("failed to create collection before upload: %w", err)
		}
	}

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 生成embedding
	embedding := GenerateEmbedding(string(content), q.dimension)

	// 计算ID
	hash := md5.Sum(content)
	pointID := fmt.Sprintf("%x", hash)

	// 创建payload
	payload := map[string]interface{}{
		"content":  string(content),
		"filename": header.Filename,
		"size":     header.Size,
	}

	// 添加元数据
	if metadata != nil {
		for k, v := range metadata {
			payload[k] = v
		}
	}

	// 上传点
	url := fmt.Sprintf("%s/collections/%s/points", q.baseURL, knowledgeKey)

	upsertPayload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      pointID,
				"vector":  embedding,
				"payload": payload,
			},
		},
	}

	body, err := json.Marshal(upsertPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("upsert request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (q *qdrantRESTKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	url := fmt.Sprintf("%s/collections/%s/points/%s", q.baseURL, knowledgeKey, documentID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete point request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete point failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (q *qdrantRESTKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	results, err := q.listAllPoints(ctx, knowledgeKey, 1000)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, r := range results {
		ids = append(ids, r.Source)
	}

	return ids, nil
}

func (q *qdrantRESTKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/collections/%s/points/%s", q.baseURL, knowledgeKey, documentID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get point request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("get point failed with status %d", resp.StatusCode)
	}

	var result struct {
		Result struct {
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()

	content := ""
	if payload, ok := result.Result.Payload["content"].(string); ok {
		content = payload
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

// collectionExists 检查 collection 是否存在
func (q *qdrantRESTKnowledgeBase) collectionExists(ctx context.Context, collectionName string) (bool, error) {
	url := fmt.Sprintf("%s/collections/%s", q.baseURL, collectionName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	if q.apiKey != "" {
		req.Header.Set("api-key", q.apiKey)
	}

	resp, err := q.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("check collection failed with status %d: %s", resp.StatusCode, string(body))
}

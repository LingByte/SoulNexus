package knowledge

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

// qdrantKnowledgeBase Qdrant向量数据库实现
type qdrantKnowledgeBase struct {
	client    *qdrant.Client
	dimension int
}

// NewQdrantKnowledgeBase 创建Qdrant知识库实例
func NewQdrantKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	host := getStringFromConfig(config, ConfigKeyQdrantHost)
	if host == "" {
		host = "localhost"
	}

	port := getIntFromConfig(config, ConfigKeyQdrantPort)
	if port == 0 {
		port = 6334
	}

	apiKey := getStringFromConfig(config, ConfigKeyQdrantAPIKey)
	useTLS := false
	if tlsStr := getStringFromConfig(config, ConfigKeyQdrantUseTLS); tlsStr == "true" {
		useTLS = true
	}

	dimension := getIntFromConfig(config, ConfigKeyQdrantDimension)
	if dimension == 0 {
		dimension = 384 // 默认维度
	}

	// 创建Qdrant客户端
	cfg := &qdrant.Config{
		Host: host,
		Port: port,
	}

	if apiKey != "" {
		cfg.APIKey = apiKey
	}

	if useTLS {
		cfg.UseTLS = true
	}

	client, err := qdrant.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	return &qdrantKnowledgeBase{
		client:    client,
		dimension: dimension,
	}, nil
}

func (q *qdrantKnowledgeBase) Provider() string {
	return ProviderQdrant
}

func (q *qdrantKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 如果没有提供embedding，则列出所有文档
	queryEmbedding := getFloatVectorFromConfig(options.Filter, "embedding")
	if len(queryEmbedding) == 0 {
		// 没有embedding，使用Scroll API列出所有文档
		limit := uint32(options.TopK)
		if limit <= 0 {
			limit = 10
		}

		points, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: knowledgeKey,
			Limit:          &limit,
			WithPayload:    qdrant.NewWithPayload(true),
		})
		if err != nil {
			return nil, fmt.Errorf("qdrant scroll failed: %w", err)
		}

		var results []SearchResult
		for _, point := range points {
			// 从payload中提取content
			content := ""
			if payload := point.GetPayload(); payload != nil {
				if contentField, ok := payload["content"]; ok {
					if contentStr := contentField.GetStringValue(); contentStr != "" {
						content = contentStr
					}
				}
			}

			result := SearchResult{
				Content:  content,
				Score:    1.0, // 列出所有文档时，分数为1.0
				Metadata: make(map[string]interface{}),
				Source:   fmt.Sprintf("%v", point.GetId()),
			}
			results = append(results, result)
		}

		return results, nil
	}

	// 有embedding，执行向量搜索
	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}

	// 使用Qdrant客户端搜索
	limit := uint64(topK)
	searchResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: knowledgeKey,
		Query:          qdrant.NewQuery(queryEmbedding...),
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}

	var results []SearchResult
	for _, scored := range searchResult {
		// 从payload中提取content
		content := ""
		if payload := scored.GetPayload(); payload != nil {
			if contentField, ok := payload["content"]; ok {
				if contentStr := contentField.GetStringValue(); contentStr != "" {
					content = contentStr
				}
			}
		}

		score := float64(scored.GetScore())
		if options.Threshold > 0 && score < options.Threshold {
			continue
		}

		// 获取ID
		idStr := fmt.Sprintf("%v", scored.GetId())

		result := SearchResult{
			Content:  content,
			Score:    score,
			Metadata: make(map[string]interface{}),
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

	// 检查collection是否存在
	exists, err := q.client.CollectionExists(ctx, collectionName)
	if err == nil && exists {
		// Collection已存在
		return collectionName, nil
	}

	// 创建新的collection
	dimension := q.dimension
	if d := getIntFromConfig(config, ConfigKeyQdrantDimension); d > 0 {
		dimension = d
	}

	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     uint64(dimension),
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create collection: %w", err)
	}

	return collectionName, nil
}

func (q *qdrantKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	err := q.client.DeleteCollection(ctx, knowledgeKey)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

func (q *qdrantKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 计算文件MD5作为ID
	hash := md5.Sum(content)
	pointID := qdrant.NewIDNum(parsePointID(fmt.Sprintf("%x", hash)))

	// 生成embedding向量
	embedding := GenerateEmbedding(string(content), q.dimension)

	// 创建payload
	payload := map[string]*qdrant.Value{
		"content":  {Kind: &qdrant.Value_StringValue{StringValue: string(content)}},
		"filename": {Kind: &qdrant.Value_StringValue{StringValue: header.Filename}},
		"size":     {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(header.Size)}},
	}

	// 添加元数据
	if metadata != nil {
		for k, v := range metadata {
			payload[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
		}
	}

	// 创建点并上传到Qdrant
	point := &qdrant.PointStruct{
		Id:      pointID,
		Vectors: qdrant.NewVectors(embedding...),
		Payload: payload,
	}

	_, err = q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: knowledgeKey,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert point to qdrant: %w", err)
	}

	return nil
}

func (q *qdrantKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	pointID := qdrant.NewIDNum(parsePointID(documentID))

	_, err := q.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: knowledgeKey,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{pointID},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

func (q *qdrantKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 使用Scroll API获取所有点
	limit := uint32(1000)
	points, err := q.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: knowledgeKey,
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	var ids []string
	for _, point := range points {
		if point != nil {
			ids = append(ids, fmt.Sprintf("%v", point.GetId()))
		}
	}

	return ids, nil
}

func (q *qdrantKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	pointID := qdrant.NewIDNum(parsePointID(documentID))

	// 获取点数据
	points, err := q.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: knowledgeKey,
		Ids:            []*qdrant.PointId{pointID},
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("document not found")
	}

	// 从payload中提取content
	content := ""
	if payload := points[0].GetPayload(); payload != nil {
		if contentField, ok := payload["content"]; ok {
			if contentStr := contentField.GetStringValue(); contentStr != "" {
				content = contentStr
			}
		}
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

// parsePointID 将字符串ID转换为uint64
func parsePointID(id string) uint64 {
	var result uint64
	fmt.Sscanf(id, "%d", &result)
	return result
}

// 注册Qdrant提供者
func init() {
	RegisterKnowledgeBaseProvider(ProviderQdrant, func(config map[string]interface{}) (KnowledgeBase, error) {
		return NewQdrantRESTKnowledgeBase(config)
	})
}

package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// milvusKnowledgeBase Milvus/Zilliz向量数据库实现
type milvusKnowledgeBase struct {
	client         client.Client
	collectionName string
	dimension      int
}

// GetClient 获取 Milvus 客户端（用于直接操作 Milvus）
func (m *milvusKnowledgeBase) GetClient() client.Client {
	return m.client
}

// NewMilvusKnowledgeBase 创建Milvus知识库实例
// 配置选项：
//   - address: Milvus 服务器地址（默认: localhost:19530）
//   - username: 用户名（可选）
//   - password: 密码（可选）
//   - collection_name: 集合名称（必需）
//   - dimension: 向量维度（默认: 768）
func NewMilvusKnowledgeBase(config map[string]interface{}) (KnowledgeBase, error) {
	addr := getStringFromConfig(config, "address")
	if addr == "" {
		addr = "localhost:19530"
	}

	username := getStringFromConfig(config, "username")
	password := getStringFromConfig(config, "password")

	collectionName := getStringFromConfig(config, "collection_name")
	if collectionName == "" {
		return nil, fmt.Errorf("collection_name is required")
	}

	dimension := getIntFromConfig(config, "dimension")
	if dimension == 0 {
		dimension = 768 // 默认维度，通常与embedding模型相关
	}

	// 创建Milvus客户端
	clientConfig := client.Config{
		Address:  addr,
		Username: username,
		Password: password,
	}

	milvusClient, err := client.NewClient(context.Background(), clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create milvus client: %w", err)
	}

	return &milvusKnowledgeBase{
		client:         milvusClient,
		collectionName: collectionName,
		dimension:      dimension,
	}, nil
}

func (m *milvusKnowledgeBase) Provider() string {
	return ProviderZilliz
}

func (m *milvusKnowledgeBase) Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 获取embedding（这里需要外部提供embedding服务）
	// 简化实现：假设query已经转换为embedding
	queryEmbedding := getFloatVectorFromConfig(options.Filter, "embedding")
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("embedding vector is required for milvus search")
	}

	// 构建搜索参数
	topK := options.TopK
	if topK <= 0 {
		topK = 10
	}

	vectors := []entity.Vector{
		entity.FloatVector(queryEmbedding),
	}

	// 执行向量搜索
	searchResults, err := m.client.Search(ctx, knowledgeKey, nil, "", []string{"id", "content", "metadata"}, vectors, "embedding", entity.L2, topK, nil)
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}

	// 转换结果 - Search返回[]SearchResult，每个SearchResult包含多个结果
	var results []SearchResult
	for _, searchResult := range searchResults {
		// SearchResult包含多个ID、Score和Fields
		ids := searchResult.IDs
		scores := searchResult.Scores

		// 遍历每个结果
		resultCount := searchResult.ResultCount
		for i := 0; i < resultCount; i++ {
			// 提取内容
			content := ""
			if contentCol := searchResult.Fields.GetColumn("content"); contentCol != nil {
				if contentStr, err := contentCol.GetAsString(i); err == nil {
					content = contentStr
				}
			}

			// 提取元数据
			metadata := make(map[string]interface{})
			if metaCol := searchResult.Fields.GetColumn("metadata"); metaCol != nil {
				// 尝试解析JSON
				if metaStr, err := metaCol.GetAsString(i); err == nil && metaStr != "" {
					if err := json.Unmarshal([]byte(metaStr), &metadata); err == nil {
						// 成功解析
					}
				}
			}

			// 计算相关性分数
			score := 1.0
			if scores != nil && i < len(scores) {
				distance := scores[i]
				score = 1.0 - float64(distance)/2.0 // 简单的归一化
				if score < 0 {
					score = 0
				}
				if score > 1 {
					score = 1
				}
			}

			// 应用阈值过滤
			if options.Threshold > 0 && score < options.Threshold {
				continue
			}

			// 获取ID
			idStr := ""
			if ids != nil {
				if idVal, err := ids.Get(i); err == nil {
					idStr = fmt.Sprintf("%v", idVal)
				}
			}

			result := SearchResult{
				Content:  content,
				Score:    score,
				Metadata: metadata,
				Source:   idStr,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

func (m *milvusKnowledgeBase) CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 在Milvus中，collection名称就是知识库标识
	collectionName := name
	if cn := getStringFromConfig(config, "collection_name"); cn != "" {
		collectionName = cn
	}

	// 检查collection是否存在
	has, err := m.client.HasCollection(ctx, collectionName)
	if err != nil {
		return "", fmt.Errorf("failed to check collection: %w", err)
	}

	if !has {
		// 创建新的collection
		schema := &entity.Schema{
			CollectionName: collectionName,
			Description:    fmt.Sprintf("Knowledge base: %s", name),
			Fields: []*entity.Field{
				{
					Name:       "id",
					DataType:   entity.FieldTypeVarChar,
					PrimaryKey: true,
					TypeParams: map[string]string{
						"max_length": "100",
					},
				},
				{
					Name:     "content",
					DataType: entity.FieldTypeVarChar,
					TypeParams: map[string]string{
						"max_length": "65535",
					},
				},
				{
					Name:     "metadata",
					DataType: entity.FieldTypeJSON,
				},
				{
					Name:     "embedding",
					DataType: entity.FieldTypeFloatVector,
					TypeParams: map[string]string{
						"dim": fmt.Sprintf("%d", m.dimension),
					},
				},
			},
		}

		err = m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
		if err != nil {
			return "", fmt.Errorf("failed to create collection: %w", err)
		}
	}

	return collectionName, nil
}

func (m *milvusKnowledgeBase) DeleteIndex(ctx context.Context, knowledgeKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	collectionName := knowledgeKey
	err := m.client.DropCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	return nil
}

func (m *milvusKnowledgeBase) UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error {
	// Milvus需要先进行文本分割和embedding生成
	// 这里简化实现，实际需要集成文本处理和embedding服务
	return fmt.Errorf("milvus upload document not fully implemented, requires text processing and embedding generation")
}

func (m *milvusKnowledgeBase) DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	collectionName := knowledgeKey
	// 使用表达式删除
	expr := fmt.Sprintf(`id == "%s"`, documentID)

	err := m.client.Delete(ctx, collectionName, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

func (m *milvusKnowledgeBase) ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	collectionName := knowledgeKey

	// 查询所有文档ID
	queryResults, err := m.client.Query(ctx, collectionName, nil, "", []string{"id"})
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}

	var ids []string
	if idCol := queryResults.GetColumn("id"); idCol != nil {
		for i := 0; i < idCol.Len(); i++ {
			if idVal, err := idCol.Get(i); err == nil {
				ids = append(ids, fmt.Sprintf("%v", idVal))
			}
		}
	}

	return ids, nil
}

func (m *milvusKnowledgeBase) GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error) {
	// Milvus不直接存储文档内容，需要通过ID查询
	if ctx == nil {
		ctx = context.Background()
	}

	collectionName := knowledgeKey
	expr := fmt.Sprintf(`id == "%s"`, documentID)

	queryResults, err := m.client.Query(ctx, collectionName, nil, expr, []string{"content"})
	if err != nil {
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	if queryResults.Len() == 0 {
		return nil, fmt.Errorf("document not found")
	}

	content := ""
	if contentCol := queryResults.GetColumn("content"); contentCol != nil && contentCol.Len() > 0 {
		if contentStr, err := contentCol.GetAsString(0); err == nil {
			content = contentStr
		}
	}

	return io.NopCloser(strings.NewReader(content)), nil
}

// 工具函数

func getFloatVectorFromConfig(filter map[string]interface{}, key string) []float32 {
	if filter == nil {
		return nil
	}

	if val, ok := filter[key]; ok {
		if vec, ok := val.([]float32); ok {
			return vec
		}
		if vec, ok := val.([]interface{}); ok {
			result := make([]float32, 0, len(vec))
			for _, v := range vec {
				switch f := v.(type) {
				case float32:
					result = append(result, f)
				case float64:
					result = append(result, float32(f))
				case int:
					result = append(result, float32(f))
				}
			}
			return result
		}
	}
	return nil
}

// 注意：getIntFromConfig 和 getBoolFromConfig 已移至 config.go

// 注册Milvus提供者
func init() {
	RegisterKnowledgeBaseProvider(ProviderZilliz, func(config map[string]interface{}) (KnowledgeBase, error) {
		return NewMilvusKnowledgeBase(config)
	})
}

package knowledge

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
)

// EmbeddingProvider interface for embedding services
type EmbeddingProvider interface {
	// GenerateEmbedding generates embedding vector for text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	// GenerateEmbeddings generates embedding vectors for multiple texts
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
	// Dimension returns the dimension of the embedding vectors
	Dimension() int
}

// NvidiaEmbeddingProvider NVIDIA embedding provider
type NvidiaEmbeddingProvider struct {
	baseURL   string
	apiKey    string
	model     string
	dimension int
	client    *http.Client
}

// NewNvidiaEmbeddingProvider creates a new NVIDIA embedding provider
func NewNvidiaEmbeddingProvider(cfg config.EmbeddingConfig) *NvidiaEmbeddingProvider {
	return &NvidiaEmbeddingProvider{
		baseURL:   cfg.BaseURL,
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		dimension: cfg.Dimension,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// nvidiaEmbeddingRequest NVIDIA API request structure
type nvidiaEmbeddingRequest struct {
	Input          interface{} `json:"input"` // Can be string or []string
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format"` // "float" or "base64"
	InputType      string      `json:"input_type"`      // "query" or "passage"
}

// nvidiaEmbeddingResponse NVIDIA API response structure
type nvidiaEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateEmbedding generates embedding vector for text
func (p *NvidiaEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// GenerateEmbeddings generates embedding vectors for multiple texts
func (p *NvidiaEmbeddingProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	// Build request
	reqBody := nvidiaEmbeddingRequest{
		Input:          texts,
		Model:          p.model,
		EncodingFormat: "float",
		InputType:      "passage", // Use "passage" for document embedding
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/embeddings", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var embeddingResp nvidiaEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(texts))
	for _, data := range embeddingResp.Data {
		if data.Index >= 0 && data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}

// Dimension returns the dimension of the embedding vectors
func (p *NvidiaEmbeddingProvider) Dimension() int {
	return p.dimension
}

// GetEmbeddingProvider gets the configured embedding provider
func GetEmbeddingProvider() (EmbeddingProvider, error) {
	cfg := config.GlobalConfig.Services.KnowledgeBase.Embedding

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("embedding API key not configured")
	}

	switch cfg.Provider {
	case "nvidia":
		return NewNvidiaEmbeddingProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
}

// GenerateEmbedding generates embedding vector for text using configured provider
func GenerateEmbedding(text string, dimension int) []float32 {
	provider, err := GetEmbeddingProvider()
	if err != nil {
		// Fallback to simple embedding if provider not configured
		log.Printf("DEBUG: Embedding provider not available, using simple embedding with dimension: %d", dimension)
		return generateSimpleEmbedding(text, dimension)
	}

	providerDim := provider.Dimension()
	log.Printf("DEBUG: Embedding provider dimension: %d, requested dimension: %d", providerDim, dimension)

	// Check if dimensions match
	if providerDim != dimension {
		// Dimensions don't match, use simple embedding with correct dimension
		log.Printf("DEBUG: Dimension mismatch! Provider: %d, Requested: %d. Using simple embedding.", providerDim, dimension)
		return generateSimpleEmbedding(text, dimension)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	embedding, err := provider.GenerateEmbedding(ctx, text)
	if err != nil {
		// Fallback to simple embedding on error
		log.Printf("DEBUG: Embedding API error: %v. Using simple embedding.", err)
		return generateSimpleEmbedding(text, dimension)
	}

	log.Printf("DEBUG: Successfully generated embedding from provider with dimension: %d", len(embedding))
	return embedding
}

// GenerateEmbeddingFromBytes generates embedding from bytes
func GenerateEmbeddingFromBytes(data []byte, dimension int) []float32 {
	return GenerateEmbedding(string(data), dimension)
}

// generateSimpleEmbedding is the fallback simple embedding (original implementation)
func generateSimpleEmbedding(text string, dimension int) []float32 {
	// Original simple TF-IDF implementation as fallback
	// (keeping the original code for backward compatibility)
	return generateSimpleEmbeddingInternal(text, dimension)
}

// tokenize 简单的分词函数
func tokenize(text string) []string {
	// 转换为小写
	text = strings.ToLower(text)

	// 替换标点符号为空格
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		";", " ", ":", " ", "'", " ", "\"", " ",
		"(", " ", ")", " ", "[", " ", "]", " ",
		"{", " ", "}", " ", "-", " ", "_", " ",
		"/", " ", "\\", " ", "|", " ", "@", " ",
		"#", " ", "$", " ", "%", " ", "^", " ",
		"&", " ", "*", " ", "+", " ", "=", " ",
	)
	text = replacer.Replace(text)

	// 按空格分割
	words := strings.Fields(text)

	// 过滤空字符串和短词
	var result []string
	for _, word := range words {
		if len(word) > 2 { // 只保留长度大于2的词
			result = append(result, word)
		}
	}

	return result
}

// generateSimpleEmbeddingInternal simple TF-IDF embedding (fallback)
func generateSimpleEmbeddingInternal(text string, dimension int) []float32 {
	if dimension <= 0 {
		dimension = 384
	}

	// 简单的分词（按空格和标点符号分割）
	words := tokenize(text)
	if len(words) == 0 {
		// 如果没有单词，返回零向量
		return make([]float32, dimension)
	}

	// 计算词频
	wordFreq := make(map[string]int)
	for _, word := range words {
		wordFreq[word]++
	}

	// 生成向量：使用词频和词的哈希值
	embedding := make([]float32, dimension)

	for word, freq := range wordFreq {
		// 计算词的哈希值
		hash := md5.Sum([]byte(word))

		// 使用哈希值确定这个词对哪些维度有贡献
		for i := 0; i < dimension; i++ {
			// 使用哈希的字节生成伪随机索引
			idx := (int(hash[i%len(hash)]) + i) % dimension

			// 计算这个词对该维度的贡献
			// 使用词频和哈希值的组合
			contribution := float32(freq) * float32(hash[i%len(hash)]) / 255.0
			embedding[idx] += contribution
		}
	}

	// L2 归一化
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/knowledge/chunk"
	"github.com/LingByte/SoulNexus/pkg/knowledge/parser"
	"github.com/gin-gonic/gin"
)

type kbProfile struct {
	Name         string
	Provider     string
	QdrantURL    string
	QdrantAPIKey string
	Collection   string

	EmbeddingURL   string
	EmbeddingKey   string
	EmbeddingModel string
}

type queryResultView struct {
	Score   float64
	ID      string
	Title   string
	Preview string
}

type pageData struct {
	Profiles      []kbProfile
	Default       kbProfile
	SelectedKB    string
	Message       string
	Query         string
	QueryResults  []queryResultView
	LastUploadKB  string
	LastUploadMsg string
}

var (
	profilesMu sync.RWMutex
	profiles   = map[string]kbProfile{}
)

const (
	defaultKBName           = "default_qdrant_kb"
	defaultProvider         = knowledge.KnowledgeQdrant
	defaultQdrantURL        = "http://localhost:6333"
	defaultQdrantCollection = "ling_kb"
	defaultEmbeddingURL     = "https://integrate.api.nvidia.com/v1/embeddings"
	defaultEmbeddingModel   = "nvidia/nv-embed-v1"
	defaultEmbeddingAPIKey  = "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"
	defaultRerankURL        = "https://api.siliconflow.cn/v1/rerank"
	defaultRerankModel      = "BAAI/bge-reranker-v2-m3"
	defaultRerankAPIKey     = "sk-yzbotvukjnjarxnpkponblqnalkzrunhasovgpmjsnugajsn"
)

func main() {
	ensureDefaultProfile()
	r := gin.Default()
	loadHTMLTemplates(r)

	r.GET("/", func(c *gin.Context) {
		def, _ := getProfile(defaultKBName)
		c.HTML(http.StatusOK, "index.tmpl", pageData{
			Profiles:      listProfiles(),
			Default:       def,
			SelectedKB:    c.Query("selected_kb"),
			Message:       c.Query("message"),
			Query:         c.Query("query"),
			LastUploadKB:  c.Query("upload_kb"),
			LastUploadMsg: c.Query("upload_msg"),
		})
	})

	r.POST("/knowledge/create", func(c *gin.Context) {
		profile := kbProfile{
			Name:           strings.TrimSpace(c.PostForm("name")),
			Provider:       strings.ToLower(strings.TrimSpace(c.PostForm("provider"))),
			QdrantURL:      strings.TrimSpace(c.PostForm("qdrant_url")),
			QdrantAPIKey:   strings.TrimSpace(c.PostForm("qdrant_api_key")),
			Collection:     strings.TrimSpace(c.PostForm("collection")),
			EmbeddingURL:   strings.TrimSpace(c.PostForm("embedding_url")),
			EmbeddingKey:   strings.TrimSpace(c.PostForm("embedding_key")),
			EmbeddingModel: strings.TrimSpace(c.PostForm("embedding_model")),
		}
		if profile.Name == "" {
			redirectMsg(c, "知识库名称不能为空")
			return
		}
		if profile.Provider == "" {
			profile.Provider = knowledge.KnowledgeQdrant
		}
		if profile.Provider == knowledge.KnowledgeQdrant {
			if profile.QdrantURL == "" || profile.Collection == "" {
				redirectMsg(c, "Qdrant 类型需要填写 qdrant_url 和 collection")
				return
			}
			if profile.EmbeddingURL == "" || profile.EmbeddingKey == "" || profile.EmbeddingModel == "" {
				redirectMsg(c, "Qdrant 类型需要填写 embedding_url/embedding_key/embedding_model")
				return
			}
		}

		profilesMu.Lock()
		profiles[profile.Name] = profile
		profilesMu.Unlock()
		redirectMsg(c, "知识库配置已保存: "+profile.Name)
	})

	r.POST("/knowledge/upload", func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("kb_name"))
		if name == "" {
			name = defaultKBName
		}
		p, ok := getProfile(name)
		if !ok {
			redirectWithUpload(c, name, "请选择已创建的知识库")
			return
		}
		file, err := c.FormFile("file")
		if err != nil {
			redirectWithUpload(c, name, "读取上传文件失败: "+err.Error())
			return
		}
		f, err := file.Open()
		if err != nil {
			redirectWithUpload(c, name, "打开上传文件失败: "+err.Error())
			return
		}
		defer f.Close()

		buf, err := io.ReadAll(f)
		if err != nil {
			redirectWithUpload(c, name, "读取上传文件内容失败: "+err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
		defer cancel()
		h, err := buildKnowledgeHandler(p)
		if err != nil {
			redirectWithUpload(c, name, err.Error())
			return
		}

		parseRes, err := parser.ParseBytes(ctx, file.Filename, buf, &parser.ParseOptions{MaxTextLength: 500_000, PreserveLineBreaks: true})
		if err != nil {
			redirectWithUpload(c, name, "解析文件失败: "+err.Error())
			return
		}
		chunker, _ := chunk.New(chunk.ChunkerTypeRule, nil)
		chunks, err := chunker.Chunk(ctx, parseRes.Text, &chunk.ChunkOptions{MaxChars: 800, OverlapChars: 80, MinChars: 80, DocumentTitle: file.Filename})
		if err != nil {
			redirectWithUpload(c, name, "切块失败: "+err.Error())
			return
		}

		now := time.Now()
		docID := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
		recs := make([]knowledge.Record, 0, len(chunks))
		for _, ch := range chunks {
			recs = append(recs, knowledge.Record{
				ID:        fmt.Sprintf("%s#chunk_%d", docID, ch.Index),
				Source:    "web_upload",
				Title:     file.Filename,
				Content:   ch.Text,
				Tags:      []string{"web", "upload"},
				Metadata:  map[string]any{"chunk_index": ch.Index, "provider": p.Provider},
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if err := h.Upsert(ctx, recs, &knowledge.UpsertOptions{}); err != nil {
			redirectWithUpload(c, name, "写入知识库失败: "+err.Error())
			return
		}
		redirectWithUpload(c, name, fmt.Sprintf("上传成功: %s, 切块并入库 %d 条", file.Filename, len(recs)))
	})

	r.POST("/knowledge/query", func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("kb_name"))
		if name == "" {
			name = defaultKBName
		}
		query := strings.TrimSpace(c.PostForm("query"))
		if query == "" {
			def, _ := getProfile(defaultKBName)
			renderPage(c, pageData{Profiles: listProfiles(), Default: def, SelectedKB: name, Message: "查询内容不能为空"})
			return
		}
		p, ok := getProfile(name)
		if !ok {
			def, _ := getProfile(defaultKBName)
			renderPage(c, pageData{Profiles: listProfiles(), Default: def, SelectedKB: name, Query: query, Message: "请选择已创建的知识库"})
			return
		}
		h, err := buildKnowledgeHandler(p)
		if err != nil {
			def, _ := getProfile(defaultKBName)
			renderPage(c, pageData{Profiles: listProfiles(), Default: def, SelectedKB: name, Query: query, Message: err.Error()})
			return
		}
		topK := 5
		if v := strings.TrimSpace(c.PostForm("topk")); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 {
				topK = n
			}
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()
		results, err := h.Query(ctx, query, &knowledge.QueryOptions{TopK: topK})
		if err != nil {
			def, _ := getProfile(defaultKBName)
			renderPage(c, pageData{Profiles: listProfiles(), Default: def, SelectedKB: name, Query: query, Message: "召回失败: " + err.Error()})
			return
		}
		views := make([]queryResultView, 0, len(results))
		for _, r := range results {
			preview := strings.TrimSpace(r.Record.Content)
			if len(preview) > 220 {
				preview = preview[:220] + "..."
			}
			views = append(views, queryResultView{
				Score:   r.Score,
				ID:      r.Record.ID,
				Title:   r.Record.Title,
				Preview: preview,
			})
		}
		def, _ := getProfile(defaultKBName)
		renderPage(c, pageData{
			Profiles:     listProfiles(),
			Default:      def,
			SelectedKB:   name,
			Query:        query,
			QueryResults: views,
			Message:      fmt.Sprintf("召回完成，共 %d 条", len(views)),
		})
	})

	addr := ":8080"
	log.Printf("server started at %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func loadHTMLTemplates(r *gin.Engine) {
	candidates := []string{
		filepath.Join("templates", "*.tmpl"),
		filepath.Join("cmd", "knowledge", "templates", "*.tmpl"),
		filepath.Join("cmd", "server", "templates", "*.tmpl"),
	}
	for _, pattern := range candidates {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			r.LoadHTMLGlob(pattern)
			return
		}
	}
	panic("no templates found for knowledge app")
}

func renderPage(c *gin.Context, data pageData) {
	c.HTML(http.StatusOK, "index.tmpl", data)
}

func redirectMsg(c *gin.Context, msg string) {
	c.Redirect(http.StatusFound, "/?message="+msg)
}

func redirectWithUpload(c *gin.Context, kb, msg string) {
	c.Redirect(http.StatusFound, "/?upload_kb="+kb+"&upload_msg="+msg+"&selected_kb="+kb)
}

func listProfiles() []kbProfile {
	profilesMu.RLock()
	defer profilesMu.RUnlock()
	out := make([]kbProfile, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p)
	}
	return out
}

func getProfile(name string) (kbProfile, bool) {
	profilesMu.RLock()
	defer profilesMu.RUnlock()
	p, ok := profiles[name]
	return p, ok
}

func buildKnowledgeHandler(p kbProfile) (knowledge.KnowledgeHandler, error) {
	switch p.Provider {
	case knowledge.KnowledgeAliyun:
		return nil, fmt.Errorf("aliyun provider 当前在 Ling 里还未实现，先使用 qdrant")
	case knowledge.KnowledgeQdrant:
		embed := &knowledge.NvidiaEmbedClient{
			BaseURL: p.EmbeddingURL,
			APIKey:  p.EmbeddingKey,
			Model:   p.EmbeddingModel,
		}
		return knowledge.New(knowledge.KnowledgeQdrant, &knowledge.FactoryOptions{
			Qdrant: &knowledge.QdrantOptions{
				BaseURL:    p.QdrantURL,
				APIKey:     p.QdrantAPIKey,
				Collection: p.Collection,
				Embedder:   embed,
			},
		})
	default:
		return nil, fmt.Errorf("unsupported provider: %s", p.Provider)
	}
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v != "" {
		return v
	}
	return fallback
}

func ensureDefaultProfile() {
	profilesMu.Lock()
	defer profilesMu.Unlock()
	if _, ok := profiles[defaultKBName]; ok {
		return
	}
	profiles[defaultKBName] = kbProfile{
		Name:           defaultKBName,
		Provider:       defaultProvider,
		QdrantURL:      envOrDefault("QDRANT_URL", defaultQdrantURL),
		QdrantAPIKey:   envOrDefault("QDRANT_API_KEY", ""),
		Collection:     envOrDefault("QDRANT_COLLECTION", defaultQdrantCollection),
		EmbeddingURL:   envOrDefault("NVIDIA_EMBEDDINGS_URL", defaultEmbeddingURL),
		EmbeddingKey:   envOrDefault("NVIDIA_API_KEY", defaultEmbeddingAPIKey),
		EmbeddingModel: envOrDefault("NVIDIA_EMBEDDINGS_MODEL", defaultEmbeddingModel),
	}
	_ = envOrDefault("SILICONFLOW_RERANK_URL", defaultRerankURL)
	_ = envOrDefault("SILICONFLOW_RERANK_MODEL", defaultRerankModel)
	_ = envOrDefault("SILICONFLOW_API_KEY", defaultRerankAPIKey)
}

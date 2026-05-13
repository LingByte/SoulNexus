// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Knowledge base HTTP handler.
//
// Architecture overview:
//   - One knowledge base (KnowledgeNamespace) maps to one Qdrant/Milvus collection.
//   - One document (KnowledgeDocument) is one file the user uploaded;
//     parsing -> chunking -> embedding -> upsert to vector backend happens
//     in a background goroutine (status: processing -> active / failed).
//   - Markdown form of each document is uploaded to object storage; the URL is
//     stored in KnowledgeDocument.TextURL for later viewing / editing.
//   - Multi-tenancy is enforced by GroupID: every list/create/update/delete
//     check `models.UserIsGroupMember`.

package handlers

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/logger"
	knowledgeParser "github.com/LingByte/SoulNexus/pkg/parser"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/search"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ============================================================================
// Helpers (shared)
// ============================================================================

func normalizeVec64InPlace(v []float64) {
	if len(v) == 0 {
		return
	}
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	if sum <= 0 {
		return
	}
	n := math.Sqrt(sum)
	if n == 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return
	}
	for i := range v {
		v[i] = v[i] / n
	}
}

func knowledgeProviderForLog(ns *models.KnowledgeNamespace) string {
	if ns == nil {
		return ""
	}
	p := strings.TrimSpace(strings.ToLower(ns.VectorProvider))
	if p == "" {
		p = models.KnowledgeVectorProviderQdrant
	}
	return p
}

var (
	knowledgeSearchOnce   sync.Once
	knowledgeSearchEngine search.Engine
	knowledgeSearchErr    error
)

func knowledgeSearchFromEnv() (search.Engine, error) {
	knowledgeSearchOnce.Do(func() {
		path := strings.TrimSpace(utils.GetEnv("KNOWLEDGE_SEARCH_INDEX_PATH"))
		if path == "" {
			path = "./data/knowledge_search.bleve"
		}
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		eng, err := search.New(search.Config{
			IndexPath:           path,
			DefaultAnalyzer:     "standard",
			DefaultSearchFields: []string{"title", "content"},
			OpenTimeout:         5 * time.Second,
			QueryTimeout:        5 * time.Second,
			BatchSize:           200,
		}, search.BuildIndexMapping(""))
		if err != nil {
			knowledgeSearchErr = err
			return
		}
		knowledgeSearchEngine = eng
	})
	return knowledgeSearchEngine, knowledgeSearchErr
}

func knowledgeHandlerForNS(ns *models.KnowledgeNamespace, embedder knowledge.Embedder) (knowledge.KnowledgeHandler, error) {
	if ns == nil {
		return nil, errors.New("nil namespace")
	}
	return knowledge.NewKnowledgeHandler(knowledge.HandlerFactoryParams{
		Provider:  ns.VectorProvider,
		Namespace: strings.TrimSpace(ns.Namespace),
		Embedder:  embedder,
	})
}

// embedderFromEnv builds an embedding client from env vars (EMBED_*).
func embedderFromEnv() (knowledge.Embedder, error) {
	baseURL := strings.TrimSpace(utils.GetEnv("EMBED_BASEURL"))
	apiKey := strings.TrimSpace(utils.GetEnv("EMBED_API_KEY"))
	model := strings.TrimSpace(utils.GetEnv("EMBED_MODEL"))
	inputKey := strings.TrimSpace(utils.GetEnv("EMBED_INPUT_KEY"))
	embPath := strings.TrimSpace(utils.GetEnv("EMBED_EMBEDDINGS_PATH"))
	if baseURL == "" || apiKey == "" || model == "" {
		return nil, errors.New("embedder env required: EMBED_BASEURL, EMBED_API_KEY, EMBED_MODEL")
	}
	timeoutSec := int64(30)
	if raw := strings.TrimSpace(utils.GetEnv("EMBED_TIMEOUT_SECONDS")); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			timeoutSec = n
		}
	}
	return &knowledge.NvidiaEmbedClient{
		BaseURL:        baseURL,
		APIKey:         apiKey,
		Model:          model,
		InputKey:       inputKey,
		EmbeddingsPath: embPath,
		HTTPClient:     &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}, nil
}

// chunkerFromEnv returns a routing chunker, optionally with an LLM arm for unstructured docs.
func chunkerFromEnv() (knowledge.Chunker, string, error) {
	var llmArm knowledge.Chunker
	provider := strings.TrimSpace(utils.GetEnv("LLM_PROVIDER"))
	apiKey := strings.TrimSpace(utils.GetEnv("LLM_API_KEY"))
	baseURL := strings.TrimSpace(utils.GetEnv("LLM_BASEURL"))
	model := strings.TrimSpace(utils.GetEnv("LLM_MODEL"))
	if provider != "" && apiKey != "" && baseURL != "" && model != "" {
		systemPrompt := strings.TrimSpace(utils.GetEnv("LLM_SYSTEM_PROMPT"))
		h, err := llm.NewLLMProvider(context.Background(), provider, apiKey, baseURL, systemPrompt)
		if err == nil {
			llmArm = &knowledge.LLMChunker{LLM: h, Model: model}
		}
	}
	r := knowledge.DefaultRoutingChunker(llmArm)
	var msg string
	if llmArm != nil {
		msg = fmt.Sprintf("RoutingChunker: structured+table_kv+llm (LLM provider=%s model=%s)", provider, model)
	} else {
		msg = "RoutingChunker: structured+table_kv+rules"
	}
	return r, msg, nil
}

func knowledgeChunkOpts(docTitle string) *knowledge.ChunkOptions {
	return &knowledge.ChunkOptions{
		DocumentTitle: strings.TrimSpace(docTitle),
		MaxChars:      knowledge.DefaultChunkMaxChars,
		OverlapChars:  knowledge.DefaultChunkOverlapChars,
		MinChars:      knowledge.DefaultChunkMinChars,
	}
}

func parseRecordIDs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		var arr []string
		if err := json.Unmarshal([]byte(s), &arr); err == nil {
			out := make([]string, 0, len(arr))
			for _, it := range arr {
				if v := strings.TrimSpace(it); v != "" {
					out = append(out, v)
				}
			}
			return out
		}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// uploadMarkdownToStore uploads a markdown text blob to object storage and returns its URL.
func uploadMarkdownToStore(groupID uint, namespace string, docID int64, filename, mdText string) (string, error) {
	if config.GlobalConfig == nil {
		return "", errors.New("storage not initialized")
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	baseName := strings.TrimSpace(filename)
	if baseName == "" {
		baseName = "document"
	}
	baseName = filepath.Base(baseName)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if baseName == "" || baseName == "." {
		baseName = "document"
	}
	mdName := baseName + ".md"
	key := fmt.Sprintf("knowledge/%d/%s/%d/%d-%s", groupID, ns, docID, time.Now().UnixNano(), mdName)
	st := stores.Default()
	if err := st.Write(key, bytes.NewReader([]byte(mdText))); err != nil {
		return "", err
	}
	u := strings.TrimSpace(st.PublicURL(key))
	return u, nil
}

type mdChunkRef struct {
	Idx  int
	Text string
}

func buildStructuredMarkdown(title, namespace, fileHash, source string, chunks []mdChunkRef) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Document"
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	fh := strings.TrimSpace(fileHash)
	src := strings.TrimSpace(source)
	if src == "" {
		src = "upload"
	}
	var b strings.Builder
	_, _ = b.WriteString("# " + title + "\n\n")
	_, _ = b.WriteString("- namespace: `" + ns + "`\n")
	if fh != "" {
		_, _ = b.WriteString("- file_hash: `" + fh + "`\n")
	}
	_, _ = b.WriteString("- source: `" + src + "`\n")
	_, _ = b.WriteString(fmt.Sprintf("- chunks: `%d`\n", len(chunks)))
	_, _ = b.WriteString("- generated_at: `" + time.Now().UTC().Format(time.RFC3339) + "`\n\n")
	for i, ch := range chunks {
		txt := strings.TrimSpace(ch.Text)
		if txt == "" {
			continue
		}
		_, _ = b.WriteString(fmt.Sprintf("## Chunk %d\n\n", i+1))
		_, _ = b.WriteString(fmt.Sprintf("- index: `%d`\n- length: `%d`\n\n", ch.Idx, len(txt)))
		_, _ = b.WriteString(txt + "\n\n")
	}
	return b.String()
}

func (h *Handlers) knowledgeDocFinalizeSuccess(docID int64, recordIDs []string, textURL, storedMarkdown string) {
	if h == nil || h.db == nil || docID == 0 {
		return
	}
	textURL = strings.TrimSpace(textURL)
	storedMarkdown = strings.TrimSpace(storedMarkdown)
	// When object storage did not yield a URL but we have markdown, persist a sentinel in text_url
	// so APIs and operators see a non-empty marker (content lives in stored_markdown).
	if textURL == "" && storedMarkdown != "" {
		textURL = models.KnowledgeTextURLInline
	}
	rawIDs, _ := json.Marshal(recordIDs)
	updates := map[string]any{
		"record_ids": string(rawIDs),
		"status":     models.KnowledgeStatusActive,
	}
	if textURL != "" {
		updates["text_url"] = textURL
	}
	if storedMarkdown != "" {
		updates["stored_markdown"] = storedMarkdown
	}
	_ = h.db.Model(&models.KnowledgeDocument{}).Where("id = ?", docID).Updates(updates).Error
}

func (h *Handlers) knowledgeDocFinalizeFailed(docID int64) {
	if h == nil || h.db == nil || docID == 0 {
		return
	}
	_ = h.db.Model(&models.KnowledgeDocument{}).Where("id = ?", docID).
		Update("status", models.KnowledgeStatusFailed).Error
}

// ============================================================================
// Request DTOs
// ============================================================================

// KnowledgeNamespaceUpsertReq create/update knowledge base.
type KnowledgeNamespaceUpsertReq struct {
	Namespace      string  `json:"namespace" binding:"required,max=128"`
	Name           string  `json:"name" binding:"required,max=255"`
	Description    string  `json:"description"`
	VectorProvider string  `json:"vectorProvider"`
	EmbedModel     string  `json:"embedModel" binding:"max=64"`
	VectorDim      int     `json:"vectorDim"`
	Status         *string `json:"status"`
	GroupID        *uint   `json:"groupId,omitempty"`
}

// KnowledgeDocumentUpsertReq metadata-only document upsert.
type KnowledgeDocumentUpsertReq struct {
	Namespace string  `json:"namespace" binding:"required,max=128"`
	Title     string  `json:"title" binding:"required,max=255"`
	Source    string  `json:"source" binding:"max=128"`
	FileHash  string  `json:"fileHash" binding:"required,max=64"`
	RecordIDs string  `json:"recordIds"`
	Status    *string `json:"status"`
}

// KnowledgeRecallTestReq RAG quality probe request.
type KnowledgeRecallTestReq struct {
	Query    string  `json:"query" binding:"required"`
	TopK     int     `json:"topK"`
	DocID    *string `json:"docId"`
	MinScore float64 `json:"minScore"`
}

type knowledgeDocumentTextPutReq struct {
	Markdown string `json:"markdown" binding:"required"`
}

// knowledgeDocumentCreateReq registers document metadata under a group (vector rows may be filled later via upload).
type knowledgeDocumentCreateReq struct {
	GroupID   *uint   `json:"groupId"`
	Namespace string  `json:"namespace" binding:"required,max=128"`
	Title     string  `json:"title" binding:"required,max=255"`
	Source    string  `json:"source" binding:"max=128"`
	FileHash  string  `json:"fileHash" binding:"required,max=64"`
	RecordIDs string  `json:"recordIds"`
	Status    *string `json:"status"`
}

// ============================================================================
// Auth helpers
// ============================================================================

// knowledgeAuthAndScope ensures the user is logged in and returns (user, allowedGroupIDs).
func (h *Handlers) knowledgeAuthAndScope(c *gin.Context) (*models.User, []uint, bool) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return nil, nil, false
	}
	gids, err := models.MemberGroupIDs(h.db, user.ID)
	if err != nil {
		response.Fail(c, "list groups failed", err.Error())
		return nil, nil, false
	}
	return user, gids, true
}

func (h *Handlers) knowledgeLoadNamespaceForUser(c *gin.Context, user *models.User, gids []uint, id int64) (*models.KnowledgeNamespace, bool) {
	row, err := models.GetKnowledgeNamespace(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithCode(c, 404, "not found", "knowledge base not found")
			return nil, false
		}
		response.Fail(c, "query failed", err.Error())
		return nil, false
	}
	if !containsUint(gids, row.GroupID) {
		response.FailWithCode(c, 403, "forbidden", "no permission")
		return nil, false
	}
	_ = user
	return row, true
}

func (h *Handlers) knowledgeLoadDocumentForUser(c *gin.Context, user *models.User, gids []uint, id int64) (*models.KnowledgeDocument, bool) {
	row, err := models.GetKnowledgeDocument(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithCode(c, 404, "not found", "document not found")
			return nil, false
		}
		response.Fail(c, "query failed", err.Error())
		return nil, false
	}
	if !containsUint(gids, row.GroupID) {
		response.FailWithCode(c, 403, "forbidden", "no permission")
		return nil, false
	}
	if strings.EqualFold(strings.TrimSpace(row.Status), models.KnowledgeStatusDeleted) && !models.UserHasAdminAccess(h.db, user.ID) {
		response.FailWithCode(c, 404, "not found", "document not found")
		return nil, false
	}
	return row, true
}

func containsUint(xs []uint, v uint) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func parseInt64Param(c *gin.Context, key string) (int64, bool) {
	raw := strings.TrimSpace(c.Param(key))
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		response.FailWithCode(c, 400, "invalid id", raw)
		return 0, false
	}
	return n, true
}

func parseQueryInt(c *gin.Context, key string, def int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

func clampPageSize(n int) int {
	if n <= 0 {
		return 20
	}
	if n > 200 {
		return 200
	}
	return n
}

// ============================================================================
// Route registration
// ============================================================================

func (h *Handlers) registerKnowledgeRoutes(api *gin.RouterGroup) {
	ns := api.Group("/knowledge-namespaces")
	ns.Use(models.AuthRequired)
	{
		ns.GET("", h.knowledgeNamespacesListHandler)
		ns.POST("", h.knowledgeNamespaceCreateHandler)
		ns.GET("/:id", h.knowledgeNamespaceDetailHandler)
		ns.PUT("/:id", h.knowledgeNamespaceUpdateHandler)
		ns.POST("/:id/upload", h.knowledgeNamespaceUploadHandler)
		ns.POST("/:id/recall-test", h.knowledgeNamespaceRecallTestHandler)
		ns.DELETE("/:id", h.knowledgeNamespaceDeleteHandler)
	}
	docs := api.Group("/knowledge-documents")
	docs.Use(models.AuthRequired)
	{
		docs.GET("", h.knowledgeDocumentsListHandler)
		docs.POST("", h.knowledgeDocumentCreateOrUpsertHandler)
		docs.GET("/:id", h.knowledgeDocumentDetailHandler)
		docs.PUT("/:id", h.knowledgeDocumentUpdateHandler)
		docs.GET("/:id/text", h.knowledgeDocumentTextGetHandler)
		docs.PUT("/:id/text", h.knowledgeDocumentTextPutHandler)
		docs.POST("/:id/upload", h.knowledgeDocumentReuploadHandler)
		docs.DELETE("/:id", h.knowledgeDocumentDeleteHandler)
	}
}

// ============================================================================
// Namespace handlers
// ============================================================================

func (h *Handlers) knowledgeNamespacesListHandler(c *gin.Context) {
	_, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	page := parseQueryInt(c, "page", 1)
	pageSize := clampPageSize(parseQueryInt(c, "pageSize", 20))
	statusRaw := strings.TrimSpace(c.Query("status"))
	var status string
	if statusRaw == "" {
		status = models.KnowledgeStatusActive
	} else if strings.EqualFold(statusRaw, "all") {
		status = ""
	} else {
		status = statusRaw
	}
	keyword := strings.TrimSpace(c.Query("q"))
	out, err := models.ListKnowledgeNamespaces(h.db, gids, status, keyword, page, pageSize)
	if err != nil {
		response.Fail(c, "query failed", err.Error())
		return
	}
	response.Success(c, "ok", out)
}

func (h *Handlers) knowledgeNamespaceDetailHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	row, ok := h.knowledgeLoadNamespaceForUser(c, user, gids, id)
	if !ok {
		return
	}
	response.Success(c, "ok", gin.H{"namespace": row})
}

func (h *Handlers) knowledgeNamespaceCreateHandler(c *gin.Context) {
	user, _, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	var req KnowledgeNamespaceUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, 400, "bad request", err.Error())
		return
	}
	if strings.TrimSpace(req.Namespace) == "" {
		response.FailWithCode(c, 400, "namespace required", nil)
		return
	}
	gid, err := models.ResolveWriteGroupID(h.db, user.ID, req.GroupID)
	if err != nil {
		response.Fail(c, "resolve group failed", err.Error())
		return
	}
	vp := models.NormalizeVectorProvider(req.VectorProvider)
	var status string
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	embedder, err := embedderFromEnv()
	if err != nil {
		response.Fail(c, "embedder not configured", err.Error())
		return
	}
	probe, err := embedder.Embed(ctx, []string{"dimension_probe"})
	if err != nil || len(probe) == 0 || len(probe[0]) == 0 {
		response.Fail(c, "embedding model unavailable", fmt.Sprintf("%v", err))
		return
	}
	realDim := len(probe[0])

	tmpNS := &models.KnowledgeNamespace{VectorProvider: vp, Namespace: strings.TrimSpace(req.Namespace)}
	kh, err := knowledgeHandlerForNS(tmpNS, embedder)
	if err != nil {
		response.Fail(c, "vector backend unavailable", err.Error())
		return
	}
	if err := kh.Ping(ctx); err != nil {
		response.Fail(c, "vector backend ping failed", gin.H{"provider": vp, "error": err.Error()})
		return
	}
	if vp == models.KnowledgeVectorProviderQdrant {
		if err := kh.CreateNamespace(ctx, strings.TrimSpace(req.Namespace)); err != nil {
			response.Fail(c, "create namespace failed (qdrant)", err.Error())
			return
		}
	}

	row, err := models.UpsertKnowledgeNamespace(h.db, gid, user.ID, 0, &models.KnowledgeNamespaceCreateUpdate{
		Namespace:      req.Namespace,
		Name:           req.Name,
		Description:    req.Description,
		VectorProvider: vp,
		EmbedModel:     req.EmbedModel,
		VectorDim:      realDim,
		Status:         status,
	})
	if err != nil {
		if vp == models.KnowledgeVectorProviderQdrant {
			_ = kh.DeleteNamespace(context.Background(), strings.TrimSpace(req.Namespace))
		}
		response.Fail(c, "create failed", err.Error())
		return
	}
	response.Success(c, "created", row)
}

func (h *Handlers) knowledgeNamespaceUpdateHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	var req KnowledgeNamespaceUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, 400, "bad request", err.Error())
		return
	}
	existing, ok := h.knowledgeLoadNamespaceForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, existing.GroupID, existing.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to update")
		return
	}
	if strings.TrimSpace(existing.Namespace) != strings.TrimSpace(req.Namespace) {
		response.FailWithCode(c, 400, "namespace immutable", "vector backend collection cannot be renamed")
		return
	}
	if models.NormalizeVectorProvider(req.VectorProvider) != models.NormalizeVectorProvider(existing.VectorProvider) {
		response.FailWithCode(c, 400, "vector_provider immutable", nil)
		return
	}
	if existing.VectorDim > 0 && req.VectorDim > 0 && existing.VectorDim != req.VectorDim {
		response.FailWithCode(c, 400, "vector_dim immutable", gin.H{"current": existing.VectorDim, "got": req.VectorDim})
		return
	}
	var status string
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}
	row, err := models.UpsertKnowledgeNamespace(h.db, existing.GroupID, existing.CreatedBy, id, &models.KnowledgeNamespaceCreateUpdate{
		Namespace:      req.Namespace,
		Name:           req.Name,
		Description:    req.Description,
		VectorProvider: existing.VectorProvider,
		EmbedModel:     req.EmbedModel,
		VectorDim:      existing.VectorDim,
		Status:         status,
	})
	if err != nil {
		response.Fail(c, "update failed", err.Error())
		return
	}
	response.Success(c, "updated", row)
}

func (h *Handlers) knowledgeNamespaceDeleteHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	row, ok := h.knowledgeLoadNamespaceForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, row.GroupID, row.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to delete")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	kh, err := knowledgeHandlerForNS(row, nil)
	if err != nil {
		response.Fail(c, "vector backend unavailable", err.Error())
		return
	}
	if err := kh.DeleteNamespace(ctx, strings.TrimSpace(row.Namespace)); err != nil {
		response.Fail(c, "delete failed (vector backend)", err.Error())
		return
	}
	if err := models.SoftDeleteKnowledgeNamespace(h.db, id); err != nil {
		response.Fail(c, "delete failed", err.Error())
		return
	}
	response.Success(c, "deleted", gin.H{"id": id})
}

func (h *Handlers) knowledgeDocumentCreateOrUpsertHandler(c *gin.Context) {
	user, _, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	var req knowledgeDocumentCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, 400, "bad request", err.Error())
		return
	}
	gid, err := models.ResolveWriteGroupID(h.db, user.ID, req.GroupID)
	if err != nil {
		response.Fail(c, "resolve group failed", err.Error())
		return
	}
	ns := strings.TrimSpace(req.Namespace)
	if _, err := models.GetKnowledgeNamespaceByGroupAndNamespace(h.db, gid, ns); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithCode(c, 404, "namespace not found", "create a knowledge base in this organization first")
			return
		}
		response.Fail(c, "query namespace failed", err.Error())
		return
	}
	var status string
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}
	row, err := models.UpsertKnowledgeDocument(h.db, gid, user.ID, 0, &models.KnowledgeDocumentUpsertReq{
		Namespace: req.Namespace,
		Title:     req.Title,
		Source:    req.Source,
		FileHash:  req.FileHash,
		RecordIDs: req.RecordIDs,
		Status:    status,
	})
	if err != nil {
		response.Fail(c, "write failed", err.Error())
		return
	}
	response.Success(c, "ok", row)
}

func (h *Handlers) knowledgeDocumentsListHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	page := parseQueryInt(c, "page", 1)
	pageSize := clampPageSize(parseQueryInt(c, "pageSize", 20))
	namespace := strings.TrimSpace(c.Query("namespace"))
	statusRaw := strings.TrimSpace(c.Query("status"))
	var status string
	if statusRaw == "" {
		status = models.KnowledgeStatusActive
	} else if strings.EqualFold(statusRaw, "all") {
		status = ""
	} else {
		status = statusRaw
	}
	keyword := strings.TrimSpace(c.Query("q"))
	excludeDeleted := !models.UserHasAdminAccess(h.db, user.ID)
	out, err := models.ListKnowledgeDocuments(h.db, gids, namespace, status, keyword, page, pageSize, excludeDeleted)
	if err != nil {
		response.Fail(c, "query failed", err.Error())
		return
	}
	response.Success(c, "ok", out)
}

func (h *Handlers) knowledgeDocumentDetailHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	row, ok := h.knowledgeLoadDocumentForUser(c, user, gids, id)
	if !ok {
		return
	}
	out := gin.H{"document": row}
	if nsRow, err := models.GetKnowledgeNamespaceByGroupAndNamespace(h.db, row.GroupID, row.Namespace); err == nil && nsRow != nil {
		out["vectorProvider"] = nsRow.VectorProvider
	}
	response.Success(c, "ok", out)
}

func (h *Handlers) knowledgeDocumentUpdateHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	doc, ok := h.knowledgeLoadDocumentForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to update")
		return
	}
	var req KnowledgeDocumentUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, 400, "bad request", err.Error())
		return
	}
	var status string
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}
	row, err := models.UpsertKnowledgeDocument(h.db, doc.GroupID, doc.CreatedBy, id, &models.KnowledgeDocumentUpsertReq{
		Namespace: req.Namespace,
		Title:     req.Title,
		Source:    req.Source,
		FileHash:  req.FileHash,
		RecordIDs: req.RecordIDs,
		Status:    status,
	})
	if err != nil {
		response.Fail(c, "update failed", err.Error())
		return
	}
	response.Success(c, "updated", row)
}

func (h *Handlers) knowledgeDocumentDeleteHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	doc, ok := h.knowledgeLoadDocumentForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to delete")
		return
	}
	ids := parseRecordIDs(doc.RecordIDs)
	if len(ids) > 0 {
		nsRow, err := models.GetKnowledgeNamespaceByGroupAndNamespace(h.db, doc.GroupID, doc.Namespace)
		if err != nil {
			response.Fail(c, "load namespace failed", err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		kh, err := knowledgeHandlerForNS(nsRow, nil)
		if err != nil {
			response.Fail(c, "vector backend unavailable", err.Error())
			return
		}
		if err := kh.Delete(ctx, ids, &knowledge.DeleteOptions{Namespace: strings.TrimSpace(doc.Namespace)}); err != nil {
			response.Fail(c, "delete failed (vector backend)", err.Error())
			return
		}
		if eng, err := knowledgeSearchFromEnv(); err == nil && eng != nil {
			for _, rid := range ids {
				_ = eng.Delete(ctx, rid)
			}
		}
	}
	if err := models.SoftDeleteKnowledgeDocument(h.db, id); err != nil {
		response.Fail(c, "delete failed", err.Error())
		return
	}
	response.Success(c, "deleted", gin.H{"id": id})
}

func (h *Handlers) knowledgeDocumentTextGetHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	doc, ok := h.knowledgeLoadDocumentForUser(c, user, gids, id)
	if !ok {
		return
	}
	stored := strings.TrimSpace(doc.StoredMarkdown)
	url := strings.TrimSpace(doc.TextURL)
	if url == "" || models.IsKnowledgeInlineTextURL(url) {
		response.Success(c, "ok", gin.H{"url": url, "markdown": stored})
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		response.Fail(c, "fetch failed", err.Error())
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if stored != "" {
			response.Success(c, "ok", gin.H{"url": url, "markdown": stored})
			return
		}
		response.Fail(c, "fetch failed", err.Error())
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if stored != "" {
			response.Success(c, "ok", gin.H{"url": url, "markdown": stored})
			return
		}
		response.FailWithCode(c, 502, "storage fetch failed", gin.H{"status": resp.StatusCode, "body": string(b)})
		return
	}
	body := string(b)
	if strings.TrimSpace(body) == "" && stored != "" {
		response.Success(c, "ok", gin.H{"url": url, "markdown": stored})
		return
	}
	response.Success(c, "ok", gin.H{"url": doc.TextURL, "markdown": body})
}

func (h *Handlers) knowledgeDocumentTextPutHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	doc, ok := h.knowledgeLoadDocumentForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to edit")
		return
	}
	var req knowledgeDocumentTextPutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, 400, "bad request", err.Error())
		return
	}
	mdText := strings.TrimSpace(req.Markdown)
	if mdText == "" {
		response.FailWithCode(c, 400, "markdown required", nil)
		return
	}
	nsRow, err := models.GetKnowledgeNamespaceByGroupAndNamespace(h.db, doc.GroupID, doc.Namespace)
	if err != nil {
		response.Fail(c, "load namespace failed", err.Error())
		return
	}
	// Fast path: upload markdown immediately + mark processing.
	textURL := strings.TrimSpace(doc.TextURL)
	if u, upErr := uploadMarkdownToStore(doc.GroupID, doc.Namespace, doc.ID, doc.Title, mdText+"\n"); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("knowledge.text_put.upload_markdown_failed",
			zap.Error(upErr),
			zap.Int64("doc_id", doc.ID),
			zap.Uint("group_id", doc.GroupID),
		)
	}
	upd := map[string]any{
		"status": models.KnowledgeStatusProcessing,
	}
	if textURL != "" && !models.IsKnowledgeInlineTextURL(textURL) {
		upd["text_url"] = textURL
	} else {
		upd["text_url"] = models.KnowledgeTextURLInline
		upd["stored_markdown"] = mdText
	}
	_ = h.db.Model(&models.KnowledgeDocument{}).Where("id = ?", doc.ID).Updates(upd).Error
	doc.TextURL = textURL
	doc.Status = models.KnowledgeStatusProcessing

	go func(orgGID uint, ns models.KnowledgeNamespace, d models.KnowledgeDocument, md string) {
		ctx := context.Background()
		if err := h.runKnowledgeTextPutJob(ctx, orgGID, ns, d, md); err != nil {
			logger.Error("knowledge.text_put.job.failed", zap.Error(err), zap.Int64("doc_id", d.ID))
		}
	}(doc.GroupID, *nsRow, *doc, mdText)

	response.Success(c, "submitted", gin.H{"document": doc})
}

// ============================================================================
// Upload (new file) handler
// ============================================================================

func (h *Handlers) knowledgeNamespaceUploadHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	nsRow, ok := h.knowledgeLoadNamespaceForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, nsRow.GroupID, nsRow.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to upload")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		response.FailWithCode(c, 400, "file missing", err.Error())
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Fail(c, "open file failed", err.Error())
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		response.Fail(c, "read file failed", err.Error())
		return
	}
	sum := md5.Sum(b)
	fileHash := fmt.Sprintf("%x", sum[:])

	docRow, err := models.UpsertKnowledgeDocument(h.db, nsRow.GroupID, user.ID, 0, &models.KnowledgeDocumentUpsertReq{
		Namespace: nsRow.Namespace,
		Title:     fh.Filename,
		Source:    "upload",
		FileHash:  fileHash,
		RecordIDs: "",
		Status:    models.KnowledgeStatusProcessing,
	})
	if err != nil {
		response.Fail(c, "write document failed", err.Error())
		return
	}

	go func(gid uint, ns models.KnowledgeNamespace, docID int64, fileName, fileHashStr string, content []byte) {
		ctx := context.Background()
		if err := h.runKnowledgeUploadJob(ctx, gid, ns, docID, fileName, fileHashStr, content); err != nil {
			logger.Error("knowledge.upload.job.failed", zap.Error(err), zap.Int64("doc_id", docID))
		}
	}(nsRow.GroupID, *nsRow, docRow.ID, fh.Filename, fileHash, b)

	response.Success(c, "submitted", gin.H{"document": docRow})
}

// ============================================================================
// Re-upload (replace file)
// ============================================================================

func (h *Handlers) knowledgeDocumentReuploadHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	doc, ok := h.knowledgeLoadDocumentForUser(c, user, gids, id)
	if !ok {
		return
	}
	if !models.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to reupload")
		return
	}
	nsRow, err := models.GetKnowledgeNamespaceByGroupAndNamespace(h.db, doc.GroupID, doc.Namespace)
	if err != nil {
		response.Fail(c, "load namespace failed", err.Error())
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		response.FailWithCode(c, 400, "file missing", err.Error())
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Fail(c, "open file failed", err.Error())
		return
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		response.Fail(c, "read file failed", err.Error())
		return
	}
	sum := md5.Sum(b)
	fileHash := fmt.Sprintf("%x", sum[:])
	oldRecordIDs := doc.RecordIDs

	_ = h.db.Model(&models.KnowledgeDocument{}).Where("id = ?", doc.ID).
		Updates(map[string]any{
			"title":      fh.Filename,
			"file_hash":  fileHash,
			"source":     "upload",
			"record_ids": "",
			"status":     models.KnowledgeStatusProcessing,
		}).Error
	doc.Title = fh.Filename
	doc.FileHash = fileHash
	doc.Source = "upload"
	doc.RecordIDs = ""
	doc.Status = models.KnowledgeStatusProcessing

	go func(gid uint, ns models.KnowledgeNamespace, d models.KnowledgeDocument, old, fileName, fileHashStr string, content []byte) {
		ctx := context.Background()
		if err := h.runKnowledgeReuploadJob(ctx, gid, ns, d, old, fileName, fileHashStr, content); err != nil {
			logger.Error("knowledge.reupload.job.failed", zap.Error(err), zap.Int64("doc_id", d.ID))
		}
	}(doc.GroupID, *nsRow, *doc, oldRecordIDs, fh.Filename, fileHash, b)

	response.Success(c, "submitted", gin.H{"document": doc})
}

// ============================================================================
// Recall test (hybrid: vector + keyword RRF)
// ============================================================================

func (h *Handlers) knowledgeNamespaceRecallTestHandler(c *gin.Context) {
	user, gids, ok := h.knowledgeAuthAndScope(c)
	if !ok {
		return
	}
	id, ok := parseInt64Param(c, "id")
	if !ok {
		return
	}
	nsRow, ok := h.knowledgeLoadNamespaceForUser(c, user, gids, id)
	if !ok {
		return
	}
	var req KnowledgeRecallTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, 400, "bad request", err.Error())
		return
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	minScore := req.MinScore
	if minScore < 0 {
		minScore = 0
	}
	if minScore > 1 {
		response.FailWithCode(c, 400, "minScore must be in [0,1]", gin.H{"got": req.MinScore})
		return
	}

	var docRow *models.KnowledgeDocument
	expected := map[string]struct{}{}
	if req.DocID != nil && strings.TrimSpace(*req.DocID) != "" {
		n, err := strconv.ParseInt(strings.TrimSpace(*req.DocID), 10, 64)
		if err != nil || n <= 0 {
			response.FailWithCode(c, 400, "invalid docId", *req.DocID)
			return
		}
		d, err := models.GetKnowledgeDocument(h.db, n)
		if err != nil {
			response.Fail(c, "query document failed", err.Error())
			return
		}
		if d.GroupID != nsRow.GroupID {
			response.FailWithCode(c, 403, "forbidden", "document not in this knowledge base")
			return
		}
		docRow = d
		for _, rid := range parseRecordIDs(d.RecordIDs) {
			expected[rid] = struct{}{}
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	embedder, err := embedderFromEnv()
	if err != nil {
		response.Fail(c, "embedder not configured", err.Error())
		return
	}
	kh, err := knowledgeHandlerForNS(nsRow, embedder)
	if err != nil {
		response.Fail(c, "vector backend unavailable", err.Error())
		return
	}
	vecResults, err := kh.Query(ctx, strings.TrimSpace(req.Query), &knowledge.QueryOptions{
		Namespace: strings.TrimSpace(nsRow.Namespace),
		TopK:      topK,
		MinScore:  minScore,
		Filters: func() []knowledge.Filter {
			if docRow == nil {
				return nil
			}
			return []knowledge.Filter{
				{Field: "doc_id", Operator: knowledge.FilterOpEqual, Value: []any{fmt.Sprintf("%d", docRow.ID)}},
			}
		}(),
	})
	if err != nil {
		response.Fail(c, "recall failed", err.Error())
		return
	}

	// Keyword retrieval (best-effort).
	type kwItem struct {
		ID      string
		Score   float64
		Title   string
		Content string
	}
	kwItems := make([]kwItem, 0, topK)
	if eng, e2 := knowledgeSearchFromEnv(); e2 == nil && eng != nil {
		must := map[string][]string{
			"group_id":  {fmt.Sprintf("%d", nsRow.GroupID)},
			"namespace": {strings.TrimSpace(nsRow.Namespace)},
		}
		if docRow != nil {
			must["doc_id"] = []string{fmt.Sprintf("%d", docRow.ID)}
		}
		kwRes, e3 := eng.Search(ctx, search.SearchRequest{
			Keyword:       strings.TrimSpace(req.Query),
			SearchFields:  []string{"title", "content"},
			MustTerms:     must,
			From:          0,
			Size:          topK,
			IncludeFields: []string{"title", "content"},
		})
		if e3 == nil {
			for _, hit := range kwRes.Hits {
				title, _ := hit.Fields["title"].(string)
				content, _ := hit.Fields["content"].(string)
				kwItems = append(kwItems, kwItem{ID: hit.ID, Score: hit.Score, Title: title, Content: content})
			}
		}
	}

	// RRF fusion.
	const rrfK = 60.0
	type fused struct {
		ID       string
		Record   knowledge.Record
		Score    float64
		VecScore *float64
		KwScore  *float64
		VecRank  *int
		KwRank   *int
	}
	fusedMap := map[string]*fused{}
	for i, r := range vecResults {
		rank := i + 1
		recID := r.Record.ID
		vs := r.Score
		item := fusedMap[recID]
		if item == nil {
			rec := r.Record
			item = &fused{ID: recID, Record: rec}
			fusedMap[recID] = item
		}
		item.Score += 1.0 / (rrfK + float64(rank))
		item.VecScore = &vs
		item.VecRank = &rank
	}
	for i, hh := range kwItems {
		rank := i + 1
		recID := hh.ID
		ks := hh.Score
		item := fusedMap[recID]
		if item == nil {
			item = &fused{
				ID: recID,
				Record: knowledge.Record{
					ID:      recID,
					Source:  "keyword",
					Title:   hh.Title,
					Content: hh.Content,
					Metadata: map[string]any{
						"group_id":  fmt.Sprintf("%d", nsRow.GroupID),
						"namespace": strings.TrimSpace(nsRow.Namespace),
					},
				},
			}
			fusedMap[recID] = item
		}
		item.Score += 1.0 / (rrfK + float64(rank))
		item.KwScore = &ks
		item.KwRank = &rank
	}
	fusedList := make([]*fused, 0, len(fusedMap))
	for _, it := range fusedMap {
		if it.Record.Metadata == nil {
			it.Record.Metadata = map[string]any{}
		}
		it.Record.Metadata["hybrid_score"] = it.Score
		if it.VecScore != nil {
			it.Record.Metadata["vec_score"] = *it.VecScore
		}
		if it.KwScore != nil {
			it.Record.Metadata["kw_score"] = *it.KwScore
		}
		if it.VecRank != nil {
			it.Record.Metadata["vec_rank"] = *it.VecRank
		}
		if it.KwRank != nil {
			it.Record.Metadata["kw_rank"] = *it.KwRank
		}
		fusedList = append(fusedList, it)
	}
	sort.Slice(fusedList, func(i, j int) bool { return fusedList[i].Score > fusedList[j].Score })
	if len(fusedList) > topK {
		fusedList = fusedList[:topK]
	}
	results := make([]knowledge.QueryResult, 0, len(fusedList))
	for _, it := range fusedList {
		score := it.Score
		if it.VecScore != nil {
			score = *it.VecScore
		} else if it.KwScore != nil {
			score = *it.KwScore
		}
		results = append(results, knowledge.QueryResult{Record: it.Record, Score: score})
	}

	hits := 0
	if len(expected) > 0 {
		for _, r := range results {
			if _, ok := expected[r.Record.ID]; ok {
				hits++
			}
		}
	}
	recallAtK := 0.0
	precisionAtK := 0.0
	if len(expected) > 0 {
		recallAtK = float64(hits) / float64(len(expected))
	}
	if len(results) > 0 && len(expected) > 0 {
		precisionAtK = float64(hits) / float64(len(results))
	}

	response.Success(c, "ok", gin.H{
		"namespace":      nsRow,
		"query":          req.Query,
		"topK":           topK,
		"minScore":       minScore,
		"score_note":     "Cosine score in [0,1]; higher = more similar.",
		"document":       docRow,
		"hits":           hits,
		"expected":       len(expected),
		"recall_at_k":    recallAtK,
		"precision_at_k": precisionAtK,
		"results":        results,
	})
}

// ============================================================================
// Background jobs (parsing + chunking + embedding + upsert)
// ============================================================================

func (h *Handlers) runKnowledgeUploadJob(ctx context.Context, groupID uint, ns models.KnowledgeNamespace, docID int64, fileName, fileHash string, content []byte) error {
	start := time.Now()
	provider := knowledgeProviderForLog(&ns)
	logger.Info("knowledge.upload.job.start",
		zap.String("provider", provider),
		zap.Uint("group_id", groupID),
		zap.String("namespace", ns.Namespace),
		zap.Int64("doc_id", docID),
		zap.String("file_name", fileName),
		zap.Int("bytes", len(content)),
	)
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	embedder, err := embedderFromEnv()
	if err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	parsed, err := knowledgeParser.ParseBytes(ctx, fileName, content, &knowledgeParser.ParseOptions{MaxTextLength: 200000})
	if err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	chunks := buildChunks(ctx, fileName, parsed)
	if len(chunks) == 0 {
		h.knowledgeDocFinalizeFailed(docID)
		return errors.New("knowledge: empty text")
	}
	mdText := buildStructuredMarkdown(fileName, ns.Namespace, fileHash, "upload", chunks)
	inputs := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		inputs = append(inputs, ch.Text)
	}
	vecs, err := embedder.Embed(ctx, inputs)
	if err != nil || len(vecs) != len(chunks) || len(vecs) == 0 || len(vecs[0]) == 0 {
		h.knowledgeDocFinalizeFailed(docID)
		if err == nil {
			return errors.New("knowledge: invalid embeddings")
		}
		return err
	}
	for i := range vecs {
		normalizeVec64InPlace(vecs[i])
	}
	gotDim := len(vecs[0])
	if ns.VectorDim > 0 && gotDim != ns.VectorDim {
		h.knowledgeDocFinalizeFailed(docID)
		return errors.New("knowledge: vector dim mismatch")
	}
	now := time.Now().UTC()
	records := make([]knowledge.Record, 0, len(chunks))
	recordIDs := make([]string, 0, len(chunks))
	for i, ch := range chunks {
		rid := uuid.NewString()
		recordIDs = append(recordIDs, rid)
		v32 := vec64To32(vecs[i])
		records = append(records, knowledge.Record{
			ID:      rid,
			Source:  "upload",
			Title:   fileName,
			Content: ch.Text,
			Vector:  v32,
			Metadata: map[string]any{
				"group_id":      fmt.Sprintf("%d", groupID),
				"doc_id":        fmt.Sprintf("%d", docID),
				"file_name":     fileName,
				"file_hash":     fileHash,
				"section_index": ch.Idx,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	kh, err := knowledgeHandlerForNS(&ns, embedder)
	if err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	if err := kh.Upsert(ctx, records, &knowledge.UpsertOptions{Namespace: ns.Namespace, BatchSize: 64}); err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	textURL := ""
	if u, upErr := uploadMarkdownToStore(groupID, ns.Namespace, docID, fileName, mdText); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("knowledge.upload.job.markdown_upload_failed",
			zap.Error(upErr),
			zap.Int64("doc_id", docID),
			zap.Uint("group_id", groupID),
		)
	}
	stored := ""
	if strings.TrimSpace(textURL) == "" {
		stored = mdText
	}
	indexKeywordRecords(ctx, groupID, ns.Namespace, docID, fileHash, "upload", records)
	h.knowledgeDocFinalizeSuccess(docID, recordIDs, textURL, stored)
	logger.Info("knowledge.upload.job.done",
		zap.Int64("doc_id", docID),
		zap.Int("records", len(recordIDs)),
		zap.Duration("elapsed", time.Since(start)),
	)
	return nil
}

func (h *Handlers) runKnowledgeReuploadJob(ctx context.Context, groupID uint, ns models.KnowledgeNamespace, doc models.KnowledgeDocument, oldRecordIDs, fileName, fileHash string, content []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	embedder, err := embedderFromEnv()
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	parsed, err := knowledgeParser.ParseBytes(ctx, fileName, content, &knowledgeParser.ParseOptions{MaxTextLength: 200000})
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	chunks := buildChunks(ctx, fileName, parsed)
	if len(chunks) == 0 {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return errors.New("knowledge: empty text")
	}
	mdText := buildStructuredMarkdown(fileName, doc.Namespace, fileHash, "reupload", chunks)
	inputs := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		inputs = append(inputs, ch.Text)
	}
	vecs, err := embedder.Embed(ctx, inputs)
	if err != nil || len(vecs) != len(chunks) || len(vecs) == 0 || len(vecs[0]) == 0 {
		h.knowledgeDocFinalizeFailed(doc.ID)
		if err == nil {
			return errors.New("knowledge: invalid embeddings")
		}
		return err
	}
	for i := range vecs {
		normalizeVec64InPlace(vecs[i])
	}
	gotDim := len(vecs[0])
	if ns.VectorDim > 0 && gotDim != ns.VectorDim {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return errors.New("knowledge: vector dim mismatch")
	}
	kh, err := knowledgeHandlerForNS(&ns, embedder)
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	oldIDs := parseRecordIDs(oldRecordIDs)
	if len(oldIDs) > 0 {
		_ = kh.Delete(ctx, oldIDs, &knowledge.DeleteOptions{Namespace: doc.Namespace})
		if eng, err := knowledgeSearchFromEnv(); err == nil && eng != nil {
			for _, rid := range oldIDs {
				_ = eng.Delete(ctx, rid)
			}
		}
	}
	now := time.Now().UTC()
	records := make([]knowledge.Record, 0, len(chunks))
	recordIDs := make([]string, 0, len(chunks))
	for i, ch := range chunks {
		rid := uuid.NewString()
		recordIDs = append(recordIDs, rid)
		v32 := vec64To32(vecs[i])
		records = append(records, knowledge.Record{
			ID:      rid,
			Source:  "upload",
			Title:   fileName,
			Content: ch.Text,
			Vector:  v32,
			Metadata: map[string]any{
				"group_id":      fmt.Sprintf("%d", groupID),
				"doc_id":        fmt.Sprintf("%d", doc.ID),
				"file_name":     fileName,
				"file_hash":     fileHash,
				"section_index": ch.Idx,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := kh.Upsert(ctx, records, &knowledge.UpsertOptions{Namespace: doc.Namespace, Overwrite: true, BatchSize: 64}); err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	textURL := ""
	if u, upErr := uploadMarkdownToStore(groupID, doc.Namespace, doc.ID, fileName, mdText); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("knowledge.reupload.job.markdown_upload_failed",
			zap.Error(upErr),
			zap.Int64("doc_id", doc.ID),
			zap.Uint("group_id", groupID),
		)
	}
	stored := ""
	if strings.TrimSpace(textURL) == "" {
		stored = mdText
	}
	indexKeywordRecords(ctx, groupID, doc.Namespace, doc.ID, fileHash, "upload", records)
	h.knowledgeDocFinalizeSuccess(doc.ID, recordIDs, textURL, stored)
	return nil
}

func (h *Handlers) runKnowledgeTextPutJob(ctx context.Context, groupID uint, ns models.KnowledgeNamespace, doc models.KnowledgeDocument, mdText string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	embedder, err := embedderFromEnv()
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	chunks := buildChunksFromMarkdown(ctx, doc.Title, mdText)
	if len(chunks) == 0 {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return errors.New("knowledge: empty chunks from markdown")
	}
	inputs := make([]string, 0, len(chunks))
	for _, ch := range chunks {
		inputs = append(inputs, ch.Text)
	}
	vecs, err := embedder.Embed(ctx, inputs)
	if err != nil || len(vecs) != len(chunks) || len(vecs) == 0 || len(vecs[0]) == 0 {
		h.knowledgeDocFinalizeFailed(doc.ID)
		if err == nil {
			return errors.New("knowledge: invalid embeddings")
		}
		return err
	}
	for i := range vecs {
		normalizeVec64InPlace(vecs[i])
	}
	gotDim := len(vecs[0])
	if ns.VectorDim > 0 && gotDim != ns.VectorDim {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return errors.New("knowledge: vector dim mismatch")
	}
	kh, err := knowledgeHandlerForNS(&ns, embedder)
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	oldIDs := parseRecordIDs(doc.RecordIDs)
	if len(oldIDs) > 0 {
		_ = kh.Delete(ctx, oldIDs, &knowledge.DeleteOptions{Namespace: doc.Namespace})
		if eng, err := knowledgeSearchFromEnv(); err == nil && eng != nil {
			for _, rid := range oldIDs {
				_ = eng.Delete(ctx, rid)
			}
		}
	}
	now := time.Now().UTC()
	records := make([]knowledge.Record, 0, len(chunks))
	recordIDs := make([]string, 0, len(chunks))
	for i, ch := range chunks {
		rid := uuid.NewString()
		recordIDs = append(recordIDs, rid)
		v32 := vec64To32(vecs[i])
		records = append(records, knowledge.Record{
			ID:      rid,
			Source:  "edit",
			Title:   doc.Title,
			Content: ch.Text,
			Vector:  v32,
			Metadata: map[string]any{
				"group_id":      fmt.Sprintf("%d", groupID),
				"doc_id":        fmt.Sprintf("%d", doc.ID),
				"file_name":     doc.Title,
				"section_index": ch.Idx,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := kh.Upsert(ctx, records, &knowledge.UpsertOptions{Namespace: doc.Namespace, Overwrite: true, BatchSize: 64}); err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	indexKeywordRecords(ctx, groupID, doc.Namespace, doc.ID, strings.TrimSpace(doc.FileHash), "edit", records)
	textURL := ""
	if u, upErr := uploadMarkdownToStore(groupID, doc.Namespace, doc.ID, doc.Title, mdText); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("knowledge.text_put.job.markdown_upload_failed",
			zap.Error(upErr),
			zap.Int64("doc_id", doc.ID),
			zap.Uint("group_id", groupID),
		)
	}
	stored := ""
	if strings.TrimSpace(textURL) == "" {
		stored = mdText
	}
	h.knowledgeDocFinalizeSuccess(doc.ID, recordIDs, textURL, stored)
	return nil
}

// ============================================================================
// Chunking helpers
// ============================================================================

func buildChunks(ctx context.Context, fileName string, parsed *knowledgeParser.ParseResult) []mdChunkRef {
	chunks := make([]mdChunkRef, 0, 16)
	ch, _, _ := chunkerFromEnv()
	if ch != nil && parsed != nil {
		raw := strings.TrimSpace(parsed.Text)
		if raw != "" {
			routed, err := ch.Chunk(ctx, raw, knowledgeChunkOpts(fileName))
			if err == nil {
				for _, it := range routed {
					if strings.TrimSpace(it.Text) == "" {
						continue
					}
					chunks = append(chunks, mdChunkRef{Idx: it.Index, Text: it.Text})
				}
			}
		}
	}
	if len(chunks) == 0 && parsed != nil {
		for _, s := range parsed.Sections {
			txt := strings.TrimSpace(s.Text)
			if txt == "" {
				continue
			}
			chunks = append(chunks, mdChunkRef{Idx: s.Index, Text: txt})
		}
	}
	if len(chunks) == 0 && parsed != nil {
		txt := strings.TrimSpace(parsed.Text)
		if txt != "" {
			chunks = append(chunks, mdChunkRef{Idx: 0, Text: txt})
		}
	}
	return chunks
}

func buildChunksFromMarkdown(ctx context.Context, title, mdText string) []mdChunkRef {
	chunks := make([]mdChunkRef, 0, 16)
	ch, _, _ := chunkerFromEnv()
	if ch != nil {
		routed, err := ch.Chunk(ctx, mdText, knowledgeChunkOpts(title))
		if err == nil {
			for _, it := range routed {
				if strings.TrimSpace(it.Text) == "" {
					continue
				}
				chunks = append(chunks, mdChunkRef{Idx: it.Index, Text: it.Text})
			}
		}
	}
	if len(chunks) > 0 {
		return chunks
	}
	// Fallback: fixed-size sliding window.
	const maxChars = knowledge.DefaultChunkMaxChars
	const overlap = knowledge.DefaultChunkOverlapChars
	s := mdText
	idx := 0
	for chunkStart := 0; chunkStart < len(s); {
		end := chunkStart + maxChars
		if end > len(s) {
			end = len(s)
		}
		part := strings.TrimSpace(s[chunkStart:end])
		if part != "" {
			chunks = append(chunks, mdChunkRef{Idx: idx, Text: part})
			idx++
		}
		if end == len(s) {
			break
		}
		chunkStart = end - overlap
		if chunkStart < 0 {
			chunkStart = 0
		}
	}
	return chunks
}

func vec64To32(v []float64) []float32 {
	out := make([]float32, 0, len(v))
	for _, x := range v {
		out = append(out, float32(x))
	}
	return out
}

func indexKeywordRecords(ctx context.Context, groupID uint, namespace string, docID int64, fileHash, source string, records []knowledge.Record) {
	eng, err := knowledgeSearchFromEnv()
	if err != nil || eng == nil {
		return
	}
	docs := make([]search.Doc, 0, len(records))
	for _, r := range records {
		docs = append(docs, search.Doc{
			ID:   r.ID,
			Type: "knowledge_record",
			Fields: map[string]any{
				"group_id":  fmt.Sprintf("%d", groupID),
				"namespace": strings.TrimSpace(namespace),
				"doc_id":    fmt.Sprintf("%d", docID),
				"title":     r.Title,
				"content":   r.Content,
				"file_hash": fileHash,
				"source":    source,
			},
		})
	}
	_ = eng.IndexBatch(ctx, docs)
}

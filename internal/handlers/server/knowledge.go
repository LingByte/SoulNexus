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
//     check `svcmodels.UserIsGroupMember`.

package server

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

	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	knowledgeParser "github.com/LingByte/SoulNexus/pkg/parser"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils/search"
	lingembedder "github.com/LingByte/lingllm/embedder"
	lingknowledge "github.com/LingByte/lingllm/knowledge"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func normalizeVec32InPlace(v []float32) {
	if len(v) == 0 {
		return
	}
	var sum float64
	for _, x := range v {
		f := float64(x)
		sum += f * f
	}
	if sum <= 0 {
		return
	}
	n := math.Sqrt(sum)
	if n == 0 || math.IsNaN(n) || math.IsInf(n, 0) {
		return
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / n)
	}
}

func knowledgeProviderForLog(_ *svcmodels.KnowledgeNamespace) string {
	return config.VectorProviderFromEnv()
}

var (
	knowledgeSearchOnce   sync.Once
	knowledgeSearchEngine search.Engine
	knowledgeSearchErr    error
)

func knowledgeSearchFromEnv() (search.Engine, error) {
	knowledgeSearchOnce.Do(func() {
		path := config.SearchIndexPath()
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

func knowledgeHandlerForNS(ns *svcmodels.KnowledgeNamespace, emb lingembedder.Embedder) (lingknowledge.KnowledgeHandler, error) {
	if ns == nil {
		return nil, errors.New("nil namespace")
	}
	return config.NewHandler(ns.Namespace, emb)
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
		textURL = svcmodels.KnowledgeTextURLInline
	}
	rawIDs, _ := json.Marshal(recordIDs)
	updates := map[string]any{
		"record_ids": string(rawIDs),
		"status":     svcmodels.KnowledgeStatusActive,
	}
	if textURL != "" {
		updates["text_url"] = textURL
	}
	if storedMarkdown != "" {
		updates["stored_markdown"] = storedMarkdown
	}
	_ = h.db.Model(&svcmodels.KnowledgeDocument{}).Where("id = ?", docID).Updates(updates).Error
}

func (h *Handlers) knowledgeDocFinalizeFailed(docID int64) {
	if h == nil || h.db == nil || docID == 0 {
		return
	}
	_ = h.db.Model(&svcmodels.KnowledgeDocument{}).Where("id = ?", docID).
		Update("status", svcmodels.KnowledgeStatusFailed).Error
}

// ============================================================================
// Request DTOs
// ============================================================================

// KnowledgeNamespaceUpsertReq create/update knowledge base.
type KnowledgeNamespaceUpsertReq struct {
	Namespace   string  `json:"namespace" binding:"required,max=128"`
	Name        string  `json:"name" binding:"required,max=255"`
	Description string  `json:"description"`
	VectorDim   int     `json:"vectorDim"`
	Status      *string `json:"status"`
	GroupID     *uint   `json:"groupId,omitempty"`
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
func (h *Handlers) knowledgeAuthAndScope(c *gin.Context) (*auth.User, []uint, bool) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return nil, nil, false
	}
	gids, err := svcmodels.MemberGroupIDs(h.db, user.ID)
	if err != nil {
		response.Fail(c, "list groups failed", err.Error())
		return nil, nil, false
	}
	return user, gids, true
}

func (h *Handlers) knowledgeLoadNamespaceForUser(c *gin.Context, user *auth.User, gids []uint, id int64) (*svcmodels.KnowledgeNamespace, bool) {
	row, err := svcmodels.GetKnowledgeNamespace(h.db, id)
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

func (h *Handlers) knowledgeLoadDocumentForUser(c *gin.Context, user *auth.User, gids []uint, id int64) (*svcmodels.KnowledgeDocument, bool) {
	row, err := svcmodels.GetKnowledgeDocument(h.db, id)
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
	if strings.EqualFold(strings.TrimSpace(row.Status), svcmodels.KnowledgeStatusDeleted) && !user.HasAdminAccess() {
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
	ns.Use(auth.AuthRequired)
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
	docs.Use(auth.AuthRequired)
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
		status = svcmodels.KnowledgeStatusActive
	} else if strings.EqualFold(statusRaw, "all") {
		status = ""
	} else {
		status = statusRaw
	}
	keyword := strings.TrimSpace(c.Query("q"))
	out, err := svcmodels.ListKnowledgeNamespaces(h.db, gids, status, keyword, page, pageSize)
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
	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, req.GroupID)
	if err != nil {
		response.Fail(c, "resolve group failed", err.Error())
		return
	}
	vp := config.VectorProviderFromEnv()
	var status string
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	embedder, err := config.EmbedderFromEnv()
	if err != nil {
		response.Fail(c, "embedder not configured", err.Error())
		return
	}
	realDim, err := config.ProbeEmbeddingDimension(ctx, embedder)
	if err != nil {
		response.Fail(c, "embedding model unavailable", err.Error())
		return
	}

	tmpNS := &svcmodels.KnowledgeNamespace{Namespace: strings.TrimSpace(req.Namespace)}
	kh, err := knowledgeHandlerForNS(tmpNS, embedder)
	if err != nil {
		response.Fail(c, "vector backend unavailable", err.Error())
		return
	}
	if err := kh.Ping(ctx); err != nil {
		response.Fail(c, "vector backend ping failed", gin.H{"provider": vp, "error": err.Error()})
		return
	}
	if config.UsesQdrant() {
		if err := config.ValidateEmbeddingDim(ctx, req.Namespace, realDim, realDim, kh); err != nil {
			response.Fail(c, "vector dimension incompatible with existing qdrant collection", err.Error())
			return
		}
		if err := kh.CreateNamespace(ctx, strings.TrimSpace(req.Namespace)); err != nil {
			response.Fail(c, "create namespace failed (qdrant)", err.Error())
			return
		}
	}

	row, err := svcmodels.UpsertKnowledgeNamespace(h.db, gid, user.ID, 0, &svcmodels.KnowledgeNamespaceCreateUpdate{
		Namespace:   req.Namespace,
		Name:        req.Name,
		Description: req.Description,
		VectorDim:   realDim,
		Status:      status,
	})
	if err != nil {
		if config.UsesQdrant() {
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, existing.GroupID, existing.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to update")
		return
	}
	if strings.TrimSpace(existing.Namespace) != strings.TrimSpace(req.Namespace) {
		response.FailWithCode(c, 400, "namespace immutable", "vector backend collection cannot be renamed")
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
	row, err := svcmodels.UpsertKnowledgeNamespace(h.db, existing.GroupID, existing.CreatedBy, id, &svcmodels.KnowledgeNamespaceCreateUpdate{
		Namespace:   req.Namespace,
		Name:        req.Name,
		Description: req.Description,
		VectorDim:   existing.VectorDim,
		Status:      status,
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, row.GroupID, row.CreatedBy) {
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
	if err := svcmodels.SoftDeleteKnowledgeNamespace(h.db, id); err != nil {
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
	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, req.GroupID)
	if err != nil {
		response.Fail(c, "resolve group failed", err.Error())
		return
	}
	ns := strings.TrimSpace(req.Namespace)
	if _, err := svcmodels.GetKnowledgeNamespaceByGroupAndNamespace(h.db, gid, ns); err != nil {
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
	row, err := svcmodels.UpsertKnowledgeDocument(h.db, gid, user.ID, 0, &svcmodels.KnowledgeDocumentUpsertReq{
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
		status = svcmodels.KnowledgeStatusActive
	} else if strings.EqualFold(statusRaw, "all") {
		status = ""
	} else {
		status = statusRaw
	}
	keyword := strings.TrimSpace(c.Query("q"))
	excludeDeleted := !user.HasAdminAccess()
	out, err := svcmodels.ListKnowledgeDocuments(h.db, gids, namespace, status, keyword, page, pageSize, excludeDeleted)
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
	if nsRow, err := svcmodels.GetKnowledgeNamespaceByGroupAndNamespace(h.db, row.GroupID, row.Namespace); err == nil && nsRow != nil {
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
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
	row, err := svcmodels.UpsertKnowledgeDocument(h.db, doc.GroupID, doc.CreatedBy, id, &svcmodels.KnowledgeDocumentUpsertReq{
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to delete")
		return
	}
	ids := parseRecordIDs(doc.RecordIDs)
	if len(ids) > 0 {
		nsRow, err := svcmodels.GetKnowledgeNamespaceByGroupAndNamespace(h.db, doc.GroupID, doc.Namespace)
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
		if err := kh.Delete(ctx, ids, &lingknowledge.DeleteOptions{Namespace: strings.TrimSpace(doc.Namespace)}); err != nil {
			response.Fail(c, "delete failed (vector backend)", err.Error())
			return
		}
		if eng, err := knowledgeSearchFromEnv(); err == nil && eng != nil {
			for _, rid := range ids {
				_ = eng.Delete(ctx, rid)
			}
		}
	}
	if err := svcmodels.SoftDeleteKnowledgeDocument(h.db, id); err != nil {
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
	if url == "" || svcmodels.IsKnowledgeInlineTextURL(url) {
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
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
	nsRow, err := svcmodels.GetKnowledgeNamespaceByGroupAndNamespace(h.db, doc.GroupID, doc.Namespace)
	if err != nil {
		response.Fail(c, "load namespace failed", err.Error())
		return
	}
	// Fast path: upload markdown immediately + mark processing.
	textURL := strings.TrimSpace(doc.TextURL)
	if u, upErr := uploadMarkdownToStore(doc.GroupID, doc.Namespace, doc.ID, doc.Title, mdText+"\n"); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("lingknowledge.text_put.upload_markdown_failed",
			zap.Error(upErr),
			zap.Int64("doc_id", doc.ID),
			zap.Uint("group_id", doc.GroupID),
		)
	}
	upd := map[string]any{
		"status": svcmodels.KnowledgeStatusProcessing,
	}
	if textURL != "" && !svcmodels.IsKnowledgeInlineTextURL(textURL) {
		upd["text_url"] = textURL
	} else {
		upd["text_url"] = svcmodels.KnowledgeTextURLInline
		upd["stored_markdown"] = mdText
	}
	_ = h.db.Model(&svcmodels.KnowledgeDocument{}).Where("id = ?", doc.ID).Updates(upd).Error
	doc.TextURL = textURL
	doc.Status = svcmodels.KnowledgeStatusProcessing

	go func(orgGID uint, ns svcmodels.KnowledgeNamespace, d svcmodels.KnowledgeDocument, md string) {
		ctx := context.Background()
		if err := h.runKnowledgeTextPutJob(ctx, orgGID, ns, d, md); err != nil {
			logger.Error("lingknowledge.text_put.job.failed", zap.Error(err), zap.Int64("doc_id", d.ID))
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, nsRow.GroupID, nsRow.CreatedBy) {
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

	docRow, err := svcmodels.UpsertKnowledgeDocument(h.db, nsRow.GroupID, user.ID, 0, &svcmodels.KnowledgeDocumentUpsertReq{
		Namespace: nsRow.Namespace,
		Title:     fh.Filename,
		Source:    "upload",
		FileHash:  fileHash,
		RecordIDs: "",
		Status:    svcmodels.KnowledgeStatusProcessing,
	})
	if err != nil {
		response.Fail(c, "write document failed", err.Error())
		return
	}

	go func(gid uint, ns svcmodels.KnowledgeNamespace, docID int64, fileName, fileHashStr string, content []byte) {
		ctx := context.Background()
		if err := h.runKnowledgeUploadJob(ctx, gid, ns, docID, fileName, fileHashStr, content); err != nil {
			logger.Error("lingknowledge.upload.job.failed", zap.Error(err), zap.Int64("doc_id", docID))
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
	if !svcmodels.CanManageTenantResource(h.db, user.ID, doc.GroupID, doc.CreatedBy) {
		response.FailWithCode(c, 403, "forbidden", "no permission to reupload")
		return
	}
	nsRow, err := svcmodels.GetKnowledgeNamespaceByGroupAndNamespace(h.db, doc.GroupID, doc.Namespace)
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

	_ = h.db.Model(&svcmodels.KnowledgeDocument{}).Where("id = ?", doc.ID).
		Updates(map[string]any{
			"title":      fh.Filename,
			"file_hash":  fileHash,
			"source":     "upload",
			"record_ids": "",
			"status":     svcmodels.KnowledgeStatusProcessing,
		}).Error
	doc.Title = fh.Filename
	doc.FileHash = fileHash
	doc.Source = "upload"
	doc.RecordIDs = ""
	doc.Status = svcmodels.KnowledgeStatusProcessing

	go func(gid uint, ns svcmodels.KnowledgeNamespace, d svcmodels.KnowledgeDocument, old, fileName, fileHashStr string, content []byte) {
		ctx := context.Background()
		if err := h.runKnowledgeReuploadJob(ctx, gid, ns, d, old, fileName, fileHashStr, content); err != nil {
			logger.Error("lingknowledge.reupload.job.failed", zap.Error(err), zap.Int64("doc_id", d.ID))
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

	var docRow *svcmodels.KnowledgeDocument
	expected := map[string]struct{}{}
	if req.DocID != nil && strings.TrimSpace(*req.DocID) != "" {
		n, err := strconv.ParseInt(strings.TrimSpace(*req.DocID), 10, 64)
		if err != nil || n <= 0 {
			response.FailWithCode(c, 400, "invalid docId", *req.DocID)
			return
		}
		d, err := svcmodels.GetKnowledgeDocument(h.db, n)
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

	embedder, err := config.EmbedderFromEnv()
	if err != nil {
		response.Fail(c, "embedder not configured", err.Error())
		return
	}
	kh, err := knowledgeHandlerForNS(nsRow, embedder)
	if err != nil {
		response.Fail(c, "vector backend unavailable", err.Error())
		return
	}
	vecResults, err := kh.Query(ctx, strings.TrimSpace(req.Query), &lingknowledge.QueryOptions{
		Namespace: strings.TrimSpace(nsRow.Namespace),
		TopK:      topK,
		MinScore:  minScore,
		Filters: func() []lingknowledge.Filter {
			if docRow == nil {
				return nil
			}
			return []lingknowledge.Filter{
				{Field: "doc_id", Operator: lingknowledge.FilterOpEqual, Value: []any{fmt.Sprintf("%d", docRow.ID)}},
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
		Record   lingknowledge.Record
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
				Record: lingknowledge.Record{
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
	results := make([]lingknowledge.QueryResult, 0, len(fusedList))
	for _, it := range fusedList {
		score := it.Score
		if it.VecScore != nil {
			score = *it.VecScore
		} else if it.KwScore != nil {
			score = *it.KwScore
		}
		results = append(results, lingknowledge.QueryResult{Record: it.Record, Score: score})
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

func (h *Handlers) runKnowledgeUploadJob(ctx context.Context, groupID uint, ns svcmodels.KnowledgeNamespace, docID int64, fileName, fileHash string, content []byte) error {
	start := time.Now()
	provider := knowledgeProviderForLog(&ns)
	logger.Info("lingknowledge.upload.job.start",
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

	if ns.ID > 0 {
		if fresh, err := svcmodels.GetKnowledgeNamespace(h.db, ns.ID); err == nil && fresh != nil {
			ns = *fresh
		}
	}

	embedder, err := config.EmbedderFromEnv()
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
	vecs, err := config.EmbedTextsBatched(ctx, embedder, inputs)
	if err != nil || len(vecs) != len(chunks) || len(vecs) == 0 || len(vecs[0]) == 0 {
		h.knowledgeDocFinalizeFailed(docID)
		if err == nil {
			return errors.New("knowledge: invalid embeddings")
		}
		return err
	}
	for i := range vecs {
		normalizeVec32InPlace(vecs[i])
	}
	gotDim := len(vecs[0])
	kh, err := knowledgeHandlerForNS(&ns, embedder)
	if err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	if err := config.ValidateEmbeddingDim(ctx, ns.Namespace, ns.VectorDim, gotDim, kh); err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	now := time.Now().UTC()
	records := make([]lingknowledge.Record, 0, len(chunks))
	recordIDs := make([]string, 0, len(chunks))
	for i, ch := range chunks {
		rid := uuid.NewString()
		recordIDs = append(recordIDs, rid)
		records = append(records, lingknowledge.Record{
			ID:      rid,
			Source:  "upload",
			Title:   fileName,
			Content: ch.Text,
			Vector:  vecs[i],
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
	if err := kh.Upsert(ctx, records, &lingknowledge.UpsertOptions{Namespace: ns.Namespace, BatchSize: 64}); err != nil {
		h.knowledgeDocFinalizeFailed(docID)
		return err
	}
	textURL := ""
	if u, upErr := uploadMarkdownToStore(groupID, ns.Namespace, docID, fileName, mdText); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("lingknowledge.upload.job.markdown_upload_failed",
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
	logger.Info("lingknowledge.upload.job.done",
		zap.Int64("doc_id", docID),
		zap.Int("records", len(recordIDs)),
		zap.Duration("elapsed", time.Since(start)),
	)
	return nil
}

func (h *Handlers) runKnowledgeReuploadJob(ctx context.Context, groupID uint, ns svcmodels.KnowledgeNamespace, doc svcmodels.KnowledgeDocument, oldRecordIDs, fileName, fileHash string, content []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	embedder, err := config.EmbedderFromEnv()
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
	vecs, err := config.EmbedTextsBatched(ctx, embedder, inputs)
	if err != nil || len(vecs) != len(chunks) || len(vecs) == 0 || len(vecs[0]) == 0 {
		h.knowledgeDocFinalizeFailed(doc.ID)
		if err == nil {
			return errors.New("knowledge: invalid embeddings")
		}
		return err
	}
	for i := range vecs {
		normalizeVec32InPlace(vecs[i])
	}
	gotDim := len(vecs[0])
	kh, err := knowledgeHandlerForNS(&ns, embedder)
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	if err := config.ValidateEmbeddingDim(ctx, doc.Namespace, ns.VectorDim, gotDim, kh); err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	oldIDs := parseRecordIDs(oldRecordIDs)
	if len(oldIDs) > 0 {
		_ = kh.Delete(ctx, oldIDs, &lingknowledge.DeleteOptions{Namespace: doc.Namespace})
		if eng, err := knowledgeSearchFromEnv(); err == nil && eng != nil {
			for _, rid := range oldIDs {
				_ = eng.Delete(ctx, rid)
			}
		}
	}
	now := time.Now().UTC()
	records := make([]lingknowledge.Record, 0, len(chunks))
	recordIDs := make([]string, 0, len(chunks))
	for i, ch := range chunks {
		rid := uuid.NewString()
		recordIDs = append(recordIDs, rid)
		records = append(records, lingknowledge.Record{
			ID:      rid,
			Source:  "upload",
			Title:   fileName,
			Content: ch.Text,
			Vector:  vecs[i],
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
	if err := kh.Upsert(ctx, records, &lingknowledge.UpsertOptions{Namespace: doc.Namespace, Overwrite: true, BatchSize: 64}); err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	textURL := ""
	if u, upErr := uploadMarkdownToStore(groupID, doc.Namespace, doc.ID, fileName, mdText); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("lingknowledge.reupload.job.markdown_upload_failed",
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

func (h *Handlers) runKnowledgeTextPutJob(ctx context.Context, groupID uint, ns svcmodels.KnowledgeNamespace, doc svcmodels.KnowledgeDocument, mdText string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	embedder, err := config.EmbedderFromEnv()
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
	vecs, err := config.EmbedTextsBatched(ctx, embedder, inputs)
	if err != nil || len(vecs) != len(chunks) || len(vecs) == 0 || len(vecs[0]) == 0 {
		h.knowledgeDocFinalizeFailed(doc.ID)
		if err == nil {
			return errors.New("knowledge: invalid embeddings")
		}
		return err
	}
	for i := range vecs {
		normalizeVec32InPlace(vecs[i])
	}
	gotDim := len(vecs[0])
	kh, err := knowledgeHandlerForNS(&ns, embedder)
	if err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	if err := config.ValidateEmbeddingDim(ctx, doc.Namespace, ns.VectorDim, gotDim, kh); err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	oldIDs := parseRecordIDs(doc.RecordIDs)
	if len(oldIDs) > 0 {
		_ = kh.Delete(ctx, oldIDs, &lingknowledge.DeleteOptions{Namespace: doc.Namespace})
		if eng, err := knowledgeSearchFromEnv(); err == nil && eng != nil {
			for _, rid := range oldIDs {
				_ = eng.Delete(ctx, rid)
			}
		}
	}
	now := time.Now().UTC()
	records := make([]lingknowledge.Record, 0, len(chunks))
	recordIDs := make([]string, 0, len(chunks))
	for i, ch := range chunks {
		rid := uuid.NewString()
		recordIDs = append(recordIDs, rid)
		records = append(records, lingknowledge.Record{
			ID:      rid,
			Source:  "edit",
			Title:   doc.Title,
			Content: ch.Text,
			Vector:  vecs[i],
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
	if err := kh.Upsert(ctx, records, &lingknowledge.UpsertOptions{Namespace: doc.Namespace, Overwrite: true, BatchSize: 64}); err != nil {
		h.knowledgeDocFinalizeFailed(doc.ID)
		return err
	}
	indexKeywordRecords(ctx, groupID, doc.Namespace, doc.ID, strings.TrimSpace(doc.FileHash), "edit", records)
	textURL := ""
	if u, upErr := uploadMarkdownToStore(groupID, doc.Namespace, doc.ID, doc.Title, mdText); upErr == nil && u != "" {
		textURL = u
	} else if upErr != nil {
		logger.Warn("lingknowledge.text_put.job.markdown_upload_failed",
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

// splitTextWindow splits long text into overlapping segments (character-based).
func splitTextWindow(text string, maxChars, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" || maxChars <= 0 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxChars {
		overlap = maxChars / 5
	}
	out := make([]string, 0, 8)
	for start := 0; start < len(text); {
		end := start + maxChars
		if end > len(text) {
			end = len(text)
		}
		part := strings.TrimSpace(text[start:end])
		if part != "" {
			out = append(out, part)
		}
		if end >= len(text) {
			break
		}
		start = end - overlap
		if start <= 0 {
			start = end
		}
	}
	return out
}

// normalizeKnowledgeChunks enforces max chunk size so embedders (Ollama/vLLM) never see whole files.
func normalizeKnowledgeChunks(chunks []mdChunkRef) []mdChunkRef {
	maxChars := config.DefaultChunkMaxChars
	overlap := config.DefaultChunkOverlapChars
	out := make([]mdChunkRef, 0, len(chunks))
	idx := 0
	for _, ch := range chunks {
		parts := splitTextWindow(ch.Text, maxChars, overlap)
		if len(parts) == 0 {
			continue
		}
		for _, p := range parts {
			out = append(out, mdChunkRef{Idx: idx, Text: p})
			idx++
		}
	}
	return out
}

func buildChunks(ctx context.Context, fileName string, parsed *knowledgeParser.ParseResult) []mdChunkRef {
	chunks := make([]mdChunkRef, 0, 16)
	ch, _ := config.ChunkerFromEnv()
	if ch != nil && parsed != nil {
		raw := strings.TrimSpace(parsed.Text)
		if raw != "" {
			routed, err := ch.Chunk(ctx, raw, config.ChunkOptions(fileName))
			if err != nil {
				logger.Warn("knowledge.chunker.routed_failed",
					zap.String("file", fileName),
					zap.Error(err),
				)
			} else {
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
	return normalizeKnowledgeChunks(chunks)
}

func buildChunksFromMarkdown(ctx context.Context, title, mdText string) []mdChunkRef {
	chunks := make([]mdChunkRef, 0, 16)
	ch, _ := config.ChunkerFromEnv()
	if ch != nil {
		routed, err := ch.Chunk(ctx, mdText, config.ChunkOptions(title))
		if err != nil {
			logger.Warn("knowledge.chunker.routed_failed",
				zap.String("title", title),
				zap.Error(err),
			)
		} else {
			for _, it := range routed {
				if strings.TrimSpace(it.Text) == "" {
					continue
				}
				chunks = append(chunks, mdChunkRef{Idx: it.Index, Text: it.Text})
			}
		}
	}
	if len(chunks) == 0 {
		parts := splitTextWindow(mdText, config.DefaultChunkMaxChars, config.DefaultChunkOverlapChars)
		for i, p := range parts {
			chunks = append(chunks, mdChunkRef{Idx: i, Text: p})
		}
	}
	return normalizeKnowledgeChunks(chunks)
}

func indexKeywordRecords(ctx context.Context, groupID uint, namespace string, docID int64, fileHash, source string, records []lingknowledge.Record) {
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

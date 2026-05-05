package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/knowledge/chunk"
	parser2 "github.com/LingByte/SoulNexus/pkg/parser"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

func (h *Handlers) CreateKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}

	var req struct {
		Name           string                 `json:"name" binding:"required"`
		Description    string                 `json:"description"`
		Provider       string                 `json:"provider" binding:"required"`
		EndpointURL    string                 `json:"endpointUrl"`
		APIKey         string                 `json:"apiKey"`
		APISecret      string                 `json:"apiSecret"`
		IndexName      string                 `json:"indexName"`
		Namespace      string                 `json:"namespace"`
		GroupID        *uint                  `json:"groupId"`
		EmbeddingURL   string                 `json:"embeddingUrl"`
		EmbeddingKey   string                 `json:"embeddingKey"`
		EmbeddingModel string                 `json:"embeddingModel"`
		ExtraConfig    map[string]interface{} `json:"extraConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "parameter error", err.Error())
		return
	}

	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		response.Fail(c, "provider is required", nil)
		return
	}

	if req.ExtraConfig == nil {
		req.ExtraConfig = map[string]interface{}{}
	}
	if req.EmbeddingURL != "" {
		req.ExtraConfig["embeddingUrl"] = req.EmbeddingURL
	}
	if req.EmbeddingKey != "" {
		req.ExtraConfig["embeddingKey"] = req.EmbeddingKey
	}
	if req.EmbeddingModel != "" {
		req.ExtraConfig["embeddingModel"] = req.EmbeddingModel
	}

	rawConfig, err := json.Marshal(req.ExtraConfig)
	if err != nil {
		response.Fail(c, "invalid extraConfig", err.Error())
		return
	}

	gid, err := models.ResolveWriteGroupID(h.db, user.ID, req.GroupID)
	if err != nil {
		response.Fail(c, err.Error(), nil)
		return
	}

	entity := models.KnowledgeBase{
		GroupID:     gid,
		CreatedBy:   user.ID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Provider:    provider,
		EndpointURL: strings.TrimSpace(req.EndpointURL),
		APIKey:      strings.TrimSpace(req.APIKey),
		APISecret:   strings.TrimSpace(req.APISecret),
		IndexName:   strings.TrimSpace(req.IndexName),
		Namespace:   strings.TrimSpace(req.Namespace),
		ExtraConfig: datatypes.JSON(rawConfig),
		IsActive:    true,
	}
	if err := models.CreateKnowledgeBase(h.db, &entity); err != nil {
		response.Fail(c, "create knowledge base failed", err.Error())
		return
	}

	response.Success(c, "create knowledge base successful", entity)
}

func (h *Handlers) ListKnowledgeBases(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	list, err := models.ListKnowledgeBasesByUser(h.db, user.ID)
	if err != nil {
		response.Fail(c, "list knowledge bases failed", err.Error())
		return
	}
	response.Success(c, "list knowledge bases successful", list)
}

func (h *Handlers) GetKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}
	kb, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "knowledge base not found", err.Error())
		return
	}
	response.Success(c, "get knowledge base successful", kb)
}

func (h *Handlers) UpdateKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}

	var req struct {
		Name           *string                `json:"name"`
		Description    *string                `json:"description"`
		Provider       *string                `json:"provider"`
		EndpointURL    *string                `json:"endpointUrl"`
		APIKey         *string                `json:"apiKey"`
		APISecret      *string                `json:"apiSecret"`
		IndexName      *string                `json:"indexName"`
		Namespace      *string                `json:"namespace"`
		GroupID        *uint                  `json:"groupId"`
		IsActive       *bool                  `json:"isActive"`
		EmbeddingURL   *string                `json:"embeddingUrl"`
		EmbeddingKey   *string                `json:"embeddingKey"`
		EmbeddingModel *string                `json:"embeddingModel"`
		ExtraConfig    map[string]interface{} `json:"extraConfig"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "parameter error", err.Error())
		return
	}

	kb, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "knowledge base not found", err.Error())
		return
	}

	var extraConfig map[string]interface{}
	if len(kb.ExtraConfig) > 0 {
		_ = json.Unmarshal(kb.ExtraConfig, &extraConfig)
	}
	if extraConfig == nil {
		extraConfig = map[string]interface{}{}
	}
	if req.ExtraConfig != nil {
		for k, v := range req.ExtraConfig {
			extraConfig[k] = v
		}
	}
	if req.EmbeddingURL != nil {
		extraConfig["embeddingUrl"] = strings.TrimSpace(*req.EmbeddingURL)
	}
	if req.EmbeddingKey != nil {
		extraConfig["embeddingKey"] = strings.TrimSpace(*req.EmbeddingKey)
	}
	if req.EmbeddingModel != nil {
		extraConfig["embeddingModel"] = strings.TrimSpace(*req.EmbeddingModel)
	}
	rawConfig, err := json.Marshal(extraConfig)
	if err != nil {
		response.Fail(c, "invalid extraConfig", err.Error())
		return
	}

	updates := map[string]interface{}{
		"extra_config": datatypes.JSON(rawConfig),
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Provider != nil {
		updates["provider"] = strings.ToLower(strings.TrimSpace(*req.Provider))
	}
	if req.EndpointURL != nil {
		updates["endpoint_url"] = strings.TrimSpace(*req.EndpointURL)
	}
	if req.APIKey != nil {
		updates["api_key"] = strings.TrimSpace(*req.APIKey)
	}
	if req.APISecret != nil {
		updates["api_secret"] = strings.TrimSpace(*req.APISecret)
	}
	if req.IndexName != nil {
		updates["index_name"] = strings.TrimSpace(*req.IndexName)
	}
	if req.Namespace != nil {
		updates["namespace"] = strings.TrimSpace(*req.Namespace)
	}
	if req.GroupID != nil {
		updates["group_id"] = req.GroupID
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := models.UpdateKnowledgeBase(h.db, id, user.ID, updates); err != nil {
		response.Fail(c, "update knowledge base failed", err.Error())
		return
	}
	updated, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "get updated knowledge base failed", err.Error())
		return
	}
	response.Success(c, "update knowledge base successful", updated)
}

func (h *Handlers) DeleteKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}
	if err := models.DeleteKnowledgeBase(h.db, id, user.ID); err != nil {
		response.Fail(c, "delete knowledge base failed", err.Error())
		return
	}
	response.Success(c, "delete knowledge base successful", nil)
}

func (h *Handlers) ListKnowledgeDocumentFormats(c *gin.Context) {
	response.Success(c, "supported document formats", gin.H{
		"formats": parser2.SupportedDocumentFormats(),
		"notes":   parser2.SupportedDocumentNotes(),
	})
}

func kbUploadNDJSONLine(c *gin.Context, v map[string]any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, _ = c.Writer.Write(b)
	_, _ = c.Writer.Write([]byte("\n"))
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

func (h *Handlers) UploadKnowledgeDocument(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}
	kb, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "knowledge base not found", err.Error())
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Fail(c, "read upload file failed", err.Error())
		return
	}
	f, err := file.Open()
	if err != nil {
		response.Fail(c, "open upload file failed", err.Error())
		return
	}
	defer f.Close()

	buf, err := io.ReadAll(f)
	if err != nil {
		response.Fail(c, "read upload file content failed", err.Error())
		return
	}

	stream := strings.Contains(strings.ToLower(c.GetHeader("Accept")), "application/x-ndjson") || c.Query("stream") == "1"

	handler, err := buildKnowledgeHandlerFromEntity(kb)
	if err != nil {
		if stream {
			c.Writer.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
			c.Writer.Header().Set("Cache-Control", "no-cache")
			c.Writer.Header().Set("X-Accel-Buffering", "no")
			c.Status(http.StatusOK)
			kbUploadNDJSONLine(c, map[string]any{"type": "error", "message": err.Error()})
			return
		}
		response.Fail(c, "build knowledge handler failed", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	fail := func(msg string, errVal error) {
		errMsg := msg
		if errVal != nil {
			errMsg = errVal.Error()
		}
		if stream {
			kbUploadNDJSONLine(c, map[string]any{"type": "error", "message": errMsg})
			return
		}
		response.Fail(c, msg, errVal)
	}

	if stream {
		c.Writer.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		kbUploadNDJSONLine(c, map[string]any{
			"type":    "progress",
			"phase":   "received",
			"percent": 15,
			"message": fmt.Sprintf("已接收文件（%d KB）", len(buf)/1024),
		})
	}

	parseRes, err := parser2.ParseBytes(ctx, file.Filename, buf, &parser2.ParseOptions{MaxTextLength: 500_000, PreserveLineBreaks: true})
	if err != nil {
		fail("parse file failed", err)
		return
	}
	if stream {
		kbUploadNDJSONLine(c, map[string]any{
			"type":    "progress",
			"phase":   "parse",
			"percent": 35,
			"message": "文档解析完成",
		})
	}

	chunker, _ := chunk.New(chunk.ChunkerTypeRule, nil)
	chunks, err := chunker.Chunk(ctx, parseRes.Text, &chunk.ChunkOptions{MaxChars: 800, OverlapChars: 80, MinChars: 80, DocumentTitle: file.Filename})
	if err != nil {
		fail("chunk file failed", err)
		return
	}
	if stream {
		kbUploadNDJSONLine(c, map[string]any{
			"type":    "progress",
			"phase":   "chunk",
			"percent": 55,
			"message": fmt.Sprintf("切块完成，共 %d 段", len(chunks)),
			"chunks":  len(chunks),
		})
	}

	now := time.Now()
	docID := strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	recs := make([]knowledge.Record, 0, len(chunks))
	for _, ch := range chunks {
		recs = append(recs, knowledge.Record{
			ID:        fmt.Sprintf("%s#chunk_%d", docID, ch.Index),
			Source:    "api_upload",
			Title:     file.Filename,
			Content:   ch.Text,
			Tags:      []string{"api", "upload"},
			Metadata:  map[string]any{"chunk_index": ch.Index, "doc_id": docID, "filename": file.Filename},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if stream {
		kbUploadNDJSONLine(c, map[string]any{
			"type":    "progress",
			"phase":   "index",
			"percent": 80,
			"message": "正在向量化并写入向量库",
		})
	}
	if err := handler.Upsert(ctx, recs, &knowledge.UpsertOptions{Namespace: kb.Namespace, Overwrite: true}); err != nil {
		fail("upsert document failed", err)
		return
	}

	donePayload := gin.H{
		"knowledgeBaseId": kb.ID,
		"docId":           docID,
		"chunks":          len(recs),
		"filename":        file.Filename,
	}
	if stream {
		kbUploadNDJSONLine(c, map[string]any{
			"type":    "done",
			"percent": 100,
			"message": "上传完成",
			"data":    donePayload,
		})
		return
	}
	response.Success(c, "upload document successful", donePayload)
}

type knowledgeDocSummary struct {
	DocID    string `json:"docId"`
	Filename string `json:"filename"`
	Chunks   int    `json:"chunks"`
	Source   string `json:"source"`
}

func recordDocKey(rec knowledge.Record) (docID string, filename string) {
	if rec.Metadata != nil {
		if v, ok := rec.Metadata["doc_id"].(string); ok {
			docID = strings.TrimSpace(v)
		}
		if v, ok := rec.Metadata["filename"].(string); ok && strings.TrimSpace(v) != "" {
			filename = strings.TrimSpace(v)
		}
	}
	if docID == "" {
		id := strings.TrimSpace(rec.ID)
		if i := strings.LastIndex(id, "#chunk_"); i > 0 {
			docID = id[:i]
		} else if id != "" {
			docID = id
		}
	}
	if filename == "" {
		filename = strings.TrimSpace(rec.Title)
	}
	return docID, filename
}

// ListKnowledgeDocuments GET /knowledge-base/:id/documents — aggregates vectors by doc_id / chunk id prefix.
func (h *Handlers) ListKnowledgeDocuments(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}
	kb, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "knowledge base not found", err.Error())
		return
	}

	handler, err := buildKnowledgeHandlerFromEntity(kb)
	if err != nil {
		response.Fail(c, "build knowledge handler failed", err.Error())
		return
	}

	pageSize := 500
	if v := strings.TrimSpace(c.Query("pageSize")); v != "" {
		if n, errConv := strconv.Atoi(v); errConv == nil && n > 0 && n <= 1000 {
			pageSize = n
		}
	}

	const maxPoints = 50000
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	var records []knowledge.Record
	offset := ""
	for len(records) < maxPoints {
		listRes, errL := handler.List(ctx, &knowledge.ListOptions{Namespace: kb.Namespace, Limit: pageSize, Offset: offset})
		if errL != nil {
			response.Fail(c, "list documents failed", errL.Error())
			return
		}
		records = append(records, listRes.Records...)
		if strings.TrimSpace(listRes.NextOffset) == "" {
			break
		}
		offset = listRes.NextOffset
	}

	agg := make(map[string]*knowledgeDocSummary)
	for _, r := range records {
		docID, fn := recordDocKey(r)
		if docID == "" {
			continue
		}
		if agg[docID] == nil {
			agg[docID] = &knowledgeDocSummary{
				DocID:    docID,
				Filename: fn,
				Source:   strings.TrimSpace(r.Source),
			}
		}
		agg[docID].Chunks++
		if agg[docID].Filename == "" && fn != "" {
			agg[docID].Filename = fn
		}
	}

	out := make([]knowledgeDocSummary, 0, len(agg))
	for _, v := range agg {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Filename != out[j].Filename {
			return out[i].Filename < out[j].Filename
		}
		return out[i].DocID < out[j].DocID
	})

	response.Success(c, "list documents successful", gin.H{
		"items":       out,
		"totalChunks": len(records),
		"totalDocs":   len(out),
	})
}

func (h *Handlers) DeleteKnowledgeDocument(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}
	kb, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "knowledge base not found", err.Error())
		return
	}

	var req struct {
		DocID string   `json:"docId"`
		IDs   []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "parameter error", err.Error())
		return
	}
	if strings.TrimSpace(req.DocID) == "" && len(req.IDs) == 0 {
		response.Fail(c, "docId or ids is required", nil)
		return
	}

	handler, err := buildKnowledgeHandlerFromEntity(kb)
	if err != nil {
		response.Fail(c, "build knowledge handler failed", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	targetIDs := req.IDs
	if len(targetIDs) == 0 {
		listRes, err := handler.List(ctx, &knowledge.ListOptions{Namespace: kb.Namespace, Limit: 500})
		if err != nil {
			response.Fail(c, "list documents failed", err.Error())
			return
		}
		for _, r := range listRes.Records {
			if strings.HasPrefix(r.ID, req.DocID+"#chunk_") {
				targetIDs = append(targetIDs, r.ID)
			}
		}
	}
	if len(targetIDs) == 0 {
		response.Success(c, "no matched documents to delete", gin.H{"deleted": 0})
		return
	}

	if err := handler.Delete(ctx, targetIDs, &knowledge.DeleteOptions{Namespace: kb.Namespace}); err != nil {
		response.Fail(c, "delete document failed", err.Error())
		return
	}
	response.Success(c, "delete document successful", gin.H{"deleted": len(targetIDs), "ids": targetIDs})
}

func (h *Handlers) RecallTestKnowledgeBase(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "unauthorized", "User not logged in")
		return
	}
	id, ok := parseUintParam(c, "id")
	if !ok {
		response.Fail(c, "invalid id", nil)
		return
	}
	kb, err := models.GetKnowledgeBaseByID(h.db, id, user.ID)
	if err != nil {
		response.Fail(c, "knowledge base not found", err.Error())
		return
	}

	var req struct {
		Query string `json:"query" binding:"required"`
		TopK  int    `json:"topK"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "parameter error", err.Error())
		return
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}

	handler, err := buildKnowledgeHandlerFromEntity(kb)
	if err != nil {
		response.Fail(c, "build knowledge handler failed", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	results, err := handler.Query(ctx, strings.TrimSpace(req.Query), &knowledge.QueryOptions{Namespace: kb.Namespace, TopK: topK})
	if err != nil {
		response.Fail(c, "recall test failed", err.Error())
		return
	}
	response.Success(c, "recall test successful", gin.H{"total": len(results), "items": results})
}

func buildKnowledgeHandlerFromEntity(kb *models.KnowledgeBase) (knowledge.KnowledgeHandler, error) {
	provider := strings.ToLower(strings.TrimSpace(kb.Provider))
	if provider != knowledge.KnowledgeQdrant {
		return nil, fmt.Errorf("unsupported provider currently: %s", kb.Provider)
	}

	var cfg map[string]interface{}
	if len(kb.ExtraConfig) > 0 {
		_ = json.Unmarshal(kb.ExtraConfig, &cfg)
	}
	getCfg := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := cfg[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		return ""
	}

	embeddingURL := getCfg("embeddingUrl", "embedding_url")
	embeddingKey := getCfg("embeddingKey", "embedding_key")
	embeddingModel := getCfg("embeddingModel", "embedding_model")

	if embeddingURL == "" || embeddingKey == "" || embeddingModel == "" {
		return nil, fmt.Errorf("missing embedding config in extraConfig (embeddingUrl/embeddingKey/embeddingModel)")
	}

	embedder := &knowledge.NvidiaEmbedClient{
		BaseURL: embeddingURL,
		APIKey:  embeddingKey,
		Model:   embeddingModel,
	}
	return knowledge.New(knowledge.KnowledgeQdrant, &knowledge.FactoryOptions{
		Qdrant: &knowledge.QdrantOptions{
			BaseURL:    kb.EndpointURL,
			APIKey:     kb.APIKey,
			Collection: kb.IndexName,
			Embedder:   embedder,
		},
	})
}

func parseUintParam(c *gin.Context, key string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(key))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}

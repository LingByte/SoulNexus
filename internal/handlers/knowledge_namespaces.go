package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	knworker "github.com/LingByte/SoulNexus/pkg/knowledge/worker"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	llmparser "github.com/LingByte/lingllm/parser"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxKnowledgeDocBytes = 50 << 20 // 50 MiB

type knowledgeNamespaceCreateReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type knowledgeNamespaceUpdateReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type knowledgeDocumentUploadReq struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	DocType     string   `json:"docType"`
	Tags        []string `json:"tags"`
	CampaignID  string   `json:"campaignId"`
	ProductLine string   `json:"productLine"`
	IndexMode   string   `json:"indexMode"`
}

type knowledgeUploadPayload struct {
	Title       string
	FileName    string
	RawBytes    []byte
	DocType     string
	Tags        []string
	CampaignID  string
	ProductLine string
	IndexMode   string
}

type knowledgeDocumentUpdateReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type knowledgeRecallReq struct {
	Query       string   `json:"query"`
	TopK        int      `json:"topK"`
	MinScore    float64  `json:"minScore"`
	DocIDs      []string `json:"docIds"`
	DocTypes    []string `json:"docTypes"`
	Tags        []string `json:"tags"`
	CampaignID  string   `json:"campaignId"`
	ProductLine string   `json:"productLine"`
	CreatedFrom string   `json:"createdFrom"`
	CreatedTo   string   `json:"createdTo"`
}

// listKnowledgeNamespaces lists all knowledge namespaces for the current tenant group.
//
//	Endpoint: GET /knowledge/namespaces (or similar tenant-scoped route).
//
// Path parameters: none.
//
// Query parameters: none.
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200,
//	  "msg": "success",
//	  "data": [
//	    {
//	      "id": string(numeric id),
//	      "groupId": string(tenant id),
//	      "createdBy": string(user id),
//	      "namespace": "kb-<groupId>-<slug>",
//	      "name": "display name",
//	      "description": "string",
//	      "vectorProvider": "qdrant",
//	      "embedModel": "text-embedding-v3",
//	      "vectorDim": 1024,
//	      "status": "active",
//	      "createdAt": time.Time,
//	      "updatedAt": time.Time
//	    }
//	  ]
//	}
func (h *Handlers) listKnowledgeNamespaces(c *gin.Context) {
	groupID := middleware.CurrentTenantID(c)
	rows, err := knmodels.ListKnowledgeNamespacesByGroup(h.db, groupID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "knowledge namespaces not found")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, knowledgeNamespaceDTO(r))
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

// getKnowledgeNamespace loads one knowledge namespace by numeric ID for the current tenant.
//
//	Endpoint: GET /knowledge/namespaces/:id
//
// Path parameters:
//
//	id (uint) - the knowledge namespace row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200,
//	  "msg": "success",
//	  "data": knowledgeNamespaceDTO (see listKnowledgeNamespaces for shape)
//	}
func (h *Handlers) getKnowledgeNamespace(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, knowledgeNamespaceDTO(ns))
}

// createKnowledgeNamespace creates a new knowledge namespace for the current tenant and
// initializes the underlying vector collection.
//
//	Endpoint: POST /knowledge/namespaces
//
// Path parameters: none.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{
//	  "name": "display name (required, non-empty)",
//	  "description": "optional long description"
//	}
//
// Vector provider, embed model, and dimension are taken from server env
// (KNOWLEDGE_VECTOR_PROVIDER, EMBED_MODEL, EMBED_DIMENSION) — not user-selectable.
//
// Response:
//
//	{
//	  "code": 200,
//	  "msg": "success",
//	  "data": knowledgeNamespaceDTO
//	}
func (h *Handlers) createKnowledgeNamespace(c *gin.Context) {
	groupID := middleware.CurrentTenantID(c)
	var req knowledgeNamespaceCreateReq
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyNameRequired))
		return
	}

	if h.kb == nil {
		response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyKnowledgeUnavailable))
		return
	}

	provider := h.kb.VectorProvider()
	embedModel := h.kb.EmbedModel()
	vectorDim := h.kb.EmbedDim()
	if vectorDim <= 0 {
		vectorDim = 1024
	}

	ctx := c.Request.Context()
	if err := h.kb.Ping(ctx); err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "vector backend unreachable; check KNOWLEDGE_VECTOR_PROVIDER and related env", err))
		return
	}

	namespace := generateNamespace(name, groupID)
	namespace = ensureUniqueKnowledgeNamespace(h.db, groupID, namespace)

	if err := h.kb.EnsureNamespace(ctx, namespace, vectorDim); err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "failed to create vector namespace", err))
		return
	}

	row := &knmodels.KnowledgeNamespace{
		GroupID:        groupID,
		CreatedBy:      uint(middleware.AuthUserID(c)),
		Namespace:      namespace,
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		VectorProvider: provider,
		EmbedModel:     embedModel,
		VectorDim:      vectorDim,
		Status:         "active",
	}

	if err := h.db.Create(row).Error; err != nil {
		_ = h.kb.DeleteNamespace(context.Background(), namespace)
		ginutil.WriteGORMError(c, err, "failed to create knowledge namespace")
		return
	}

	response.SuccessI18n(c, i18n.KeySuccess, knowledgeNamespaceDTO(*row))
}

// updateKnowledgeNamespace updates the mutable fields (name/description) of an
// existing knowledge namespace.
//
//	Endpoint: PUT /knowledge/namespaces/:id
//
// Path parameters:
//
//	id (uint) - the knowledge namespace row id.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{
//	  "name": "new display name (optional)",
//	  "description": "new description (optional)"
//	}
//
// Response:
//
//	{ "code": 200, "msg": "success", "data": knowledgeNamespaceDTO }
func (h *Handlers) updateKnowledgeNamespace(c *gin.Context) {
	groupID := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}

	row, err := knmodels.GetKnowledgeNamespaceByIDAndGroup(h.db, id, groupID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "knowledge namespace not found")
		return
	}

	var req knowledgeNamespaceUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}

	if len(updates) == 0 {
		response.SuccessI18n(c, i18n.KeySuccess, knowledgeNamespaceDTO(row))
		return
	}

	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		ginutil.WriteGORMError(c, err, "failed to update knowledge namespace")
		return
	}

	response.SuccessI18n(c, i18n.KeySuccess, knowledgeNamespaceDTO(row))
}

// deleteKnowledgeNamespace soft-deletes a knowledge namespace and removes the
// underlying vector collection.
//
//	Endpoint: DELETE /knowledge/namespaces/:id
//
// Path parameters:
//
//	id (uint) - the knowledge namespace row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": null }
func (h *Handlers) deleteKnowledgeNamespace(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}

	oldSlug := ns.Namespace
	if h.kb != nil {
		if err := h.kb.DeleteNamespace(c.Request.Context(), oldSlug); err != nil {
			response.Render(c, response.Wrap(response.CodeInternal, "failed to delete vector namespace", err))
			return
		}
	}

	// Soft-delete still occupies UNIQUE(group_id, namespace). Rename the slug first so the
	// same display name (and derived slug) can be recreated immediately.
	if err := retireKnowledgeNamespaceSlug(h.db, ns); err != nil {
		ginutil.WriteGORMError(c, err, "failed to delete knowledge namespace")
		return
	}

	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

// listKnowledgeDocuments lists documents stored inside a knowledge namespace.
//
//	Endpoint: GET /knowledge/namespaces/:id/documents
//
// Path parameters:
//
//	id (uint) - the knowledge namespace row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": [ knowledgeDocumentDTO, ... ]
//	}
//
//	knowledgeDocumentDTO shape:
//
//	{
//	  "id": string(docId),
//	  "namespaceId": string(namespaceId),
//	  "title": "document title",
//	  "source": "upload | ...",
//	  "sourceType": "upload",
//	  "sourceFileName": "original.pdf",
//	  "textUrl": "knowledge/<groupId>/<ns>/<docId>.txt",
//	  "chunkCount": 0,
//	  "chunkStrategy": "",
//	  "charCount": 0,
//	  "status": "processing | active | failed",
//	  "indexError": "last error message if failed",
//	  "createdAt": time.Time,
//	  "updatedAt": time.Time
//	}
func (h *Handlers) listKnowledgeDocuments(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}

	rows, err := knmodels.ListKnowledgeDocumentsByNamespace(h.db, ns.Namespace, ns.GroupID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "documents not found")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, doc := range rows {
		out = append(out, knowledgeDocumentDTO(ns.ID, doc))
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

// getKnowledgeDocument loads metadata about one document by docId within a namespace.
//
//	Endpoint: GET /knowledge/namespaces/:id/documents/:docId
//
// Path parameters:
//
//	id (uint)    - the knowledge namespace row id.
//	docId (uint) - the knowledge document row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": knowledgeDocumentDTO }
func (h *Handlers) getKnowledgeDocument(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}
	docID, ok := ginutil.ParamID(c, "docId")
	if !ok {
		return
	}

	doc, err := knmodels.GetKnowledgeDocumentByID(h.db, docID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "document not found")
		return
	}
	if doc.GroupID != ns.GroupID || doc.Namespace != ns.Namespace {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "document not found"})
		return
	}

	response.SuccessI18n(c, i18n.KeySuccess, knowledgeDocumentDTO(ns.ID, doc))
}

// getKnowledgeDocumentContent returns the extracted plain-text body of a document
// from object storage (falls back to stored markdown or a re-parse of the raw file).
//
//	Endpoint: GET /knowledge/namespaces/:id/documents/:docId/content
//
// Path parameters:
//
//	id (uint)    - the knowledge namespace row id.
//	docId (uint) - the knowledge document row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": { "id": string(docId), "title": "title", "content": "plain text" }
//	}
func (h *Handlers) getKnowledgeDocumentContent(c *gin.Context) {
	_, doc, ok := h.loadKnowledgeDocument(c)
	if !ok {
		return
	}
	content, err := h.readKnowledgeDocumentBody(doc)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "failed to read document content", err))
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"id":      fmt.Sprintf("%d", doc.ID),
		"title":   doc.Title,
		"content": content,
	})
}

// uploadKnowledgeDocument stores an uploaded raw file (or pasted JSON text) in
// object storage, creates a document row in the "processing" state, and enqueues
// an async job to parse/vector-index the file.
//
//	Endpoint: POST /knowledge/namespaces/:id/documents
//
// Path parameters:
//
//	id (uint) - the knowledge namespace row id.
//
// Query parameters: none.
//
// Request body: one of
//   - multipart/form-data with a form field "file" (max 5 MiB) and
//     an optional form field "title";
//   - application/json: { "title": "optional", "content": "plain text (required)" }
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": { ...knowledgeDocumentDTO, "chunkCount": 0 }
//	}
func (h *Handlers) uploadKnowledgeDocument(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}
	if h.kbWorker == nil {
		response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyKnowledgeWorkerUnavail))
		return
	}

	payload, err := readKnowledgeUpload(c)
	if err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, err.Error(), err))
		return
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = strings.TrimSpace(payload.FileName)
	}
	if title == "" {
		title = "Untitled"
	}

	fileHash := sha256Hex(payload.RawBytes)

	docIDNum := nextKnowledgeDocID()
	rawKey := knowledgeDocRawKey(ns.GroupID, ns.Namespace, docIDNum, payload.FileName)
	textKey := knowledgeDocTextKey(ns.GroupID, ns.Namespace, docIDNum)

	if err := writeKnowledgeDocBytes(rawKey, payload.RawBytes); err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "failed to store uploaded file", err))
		return
	}

	row := &knmodels.KnowledgeDocument{
		BaseModel:      common.BaseModel{ID: docIDNum},
		GroupID:        ns.GroupID,
		CreatedBy:      middleware.AuthUserID(c),
		Namespace:      ns.Namespace,
		Title:          title,
		Source:         "upload",
		SourceFileName: payload.FileName,
		RawFileURL:     rawKey,
		FileHash:       fileHash,
		Status:         knconst.KnowledgeDocStatusQueued,
		DocType:        payload.DocType,
		TagsJSON:       knowledge.EncodeTagsJSON(payload.Tags),
		CampaignID:     payload.CampaignID,
		ProductLine:    payload.ProductLine,
		IndexMode:      payload.IndexMode,
	}
	if err := h.db.Create(row).Error; err != nil {
		_ = deleteKnowledgeDocContent(rawKey)
		ginutil.WriteGORMError(c, err, "failed to save document")
		return
	}

	h.enqueueKnowledgeDocumentIngest(knworker.IngestJob{
		DocID:      docIDNum,
		Namespace:  ns.Namespace,
		Title:      title,
		FileName:   payload.FileName,
		RawFileURL: rawKey,
		TextURL:    textKey,
	})

	dto := knowledgeDocumentDTO(ns.ID, *row)
	dto["chunkCount"] = 0
	response.SuccessI18n(c, i18n.KeySuccess, dto)
}

// updateKnowledgeDocument replaces the text content of a document by overwriting
// its stored raw file and enqueues a re-indexing job that will replace the existing
// vectors.
//
//	Endpoint: PUT /knowledge/namespaces/:id/documents/:docId
//
// Path parameters:
//
//	id (uint)    - the knowledge namespace row id.
//	docId (uint) - the knowledge document row id.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{ "title": "optional", "content": "plain text (required, non-empty)" }
//
// Response: { "code": 200, "msg": "success", "data": knowledgeDocumentDTO }
func (h *Handlers) updateKnowledgeDocument(c *gin.Context) {
	ns, doc, ok := h.loadKnowledgeDocument(c)
	if !ok {
		return
	}
	if h.kb == nil {
		response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyKnowledgeUnavailable))
		return
	}
	if h.kbWorker == nil {
		response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyKnowledgeWorkerUnavail))
		return
	}

	var req knowledgeDocumentUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = doc.Title
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyDocumentEmpty))
		return
	}

	textKey := strings.TrimSpace(doc.TextURL)
	if textKey == "" {
		textKey = knowledgeDocTextKey(ns.GroupID, ns.Namespace, doc.ID)
	}
	rawKey := strings.TrimSpace(doc.RawFileURL)
	if rawKey == "" {
		rawKey = knowledgeDocRawKey(ns.GroupID, ns.Namespace, doc.ID, "paste.txt")
	}
	if err := writeKnowledgeDocBytes(rawKey, []byte(content)); err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "failed to store document", err))
		return
	}

	newHash := sha256Hex([]byte(content))
	oldIDs := knowledge.DecodeRecordIDs(doc.RecordIDs)
	updates := map[string]any{
		"title":          title,
		"raw_file_url":   rawKey,
		"file_hash":      newHash,
		"status":         knconst.KnowledgeDocStatusProcessing,
		"index_error":    "",
		"record_ids":     "[]",
		"chunk_count":    0,
		"chunk_strategy": "",
		"chunks_json":    "[]",
	}
	if err := h.db.Model(&doc).Updates(updates).Error; err != nil {
		ginutil.WriteGORMError(c, err, "failed to update document")
		return
	}
	doc, _ = knmodels.GetKnowledgeDocumentByID(h.db, doc.ID)

	h.enqueueKnowledgeDocumentIngest(knworker.IngestJob{
		DocID:        doc.ID,
		Namespace:    ns.Namespace,
		Title:        title,
		FileName:     "paste.txt",
		RawFileURL:   rawKey,
		TextURL:      textKey,
		OldRecordIDs: oldIDs,
	})

	response.SuccessI18n(c, i18n.KeySuccess, knowledgeDocumentDTO(ns.ID, doc))
}

// recallKnowledgeDocuments performs a top-K vector search over the namespace
// collection and returns the hit chunks with preview and score.
//
//	Endpoint: POST /knowledge/namespaces/:id/recall
//
// Path parameters:
//
//	id (uint) - the knowledge namespace row id.
//
// Query parameters: none.
//
// Request body (application/json):
//
//	{
//	  "query": "user query (required, non-empty)",
//	  "topK": 5 (optional; clamped to [1,20]),
//	  "minScore": 0.0 (optional; only return hits with score >= minScore)
//	}
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": {
//	    "query": "user query",
//	    "topK": 5,
//	    "strategy": "retrieve strategy name",
//	    "pipeline": "recall pipeline description",
//	    "results": [ { "docId": "...", "chunkIndex": 0, "score": 0.87, "title": "...", "preview": "...", "text": "..." } ],
//	    "elapsed": "12.34ms",
//	    "hitCount": 5
//	  }
//	}
func (h *Handlers) recallKnowledgeDocuments(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}
	if h.kb == nil {
		response.Render(c, response.NewI18n(response.CodeInternal, i18n.KeyKnowledgeUnavailable))
		return
	}

	var req knowledgeRecallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Render(c, response.Wrap(response.CodeBadRequest, "invalid body", err))
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyQueryRequired))
		return
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	start := time.Now()
	hits, err := h.kb.RecallWithOptions(c.Request.Context(), ns.Namespace, query, knowledge.RecallOptions{
		TopK:     topK,
		MinScore: req.MinScore,
		Filter: knowledge.RecallFilter{
			DocIDs:      req.DocIDs,
			DocTypes:    req.DocTypes,
			Tags:        req.Tags,
			CampaignID:  req.CampaignID,
			ProductLine: req.ProductLine,
			CreatedFrom: req.CreatedFrom,
			CreatedTo:   req.CreatedTo,
		},
	})
	elapsed := time.Since(start)

	if err != nil {
		response.Render(c, response.Wrap(response.CodeInternal, "recall failed", err))
		return
	}
	results := make([]map[string]any, 0, len(hits))
	for _, hit := range hits {
		results = append(results, knowledge.RecallHitPayload(hit))
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"query":    query,
		"topK":     topK,
		"strategy": h.kb.RetrieveStrategy(),
		"pipeline": h.kb.RecallPipelineInfo(),
		"results":  results,
		"elapsed":  fmt.Sprintf("%.2fms", float64(elapsed.Microseconds())/1000.0),
		"hitCount": len(results),
	})
}

// deleteKnowledgeDocument permanently removes a document's DB row and enqueues
// an async purge to remove its vectors and object-storage files.
//
//	Endpoint: DELETE /knowledge/namespaces/:id/documents/:docId
//
// Path parameters:
//
//	id (uint)    - the knowledge namespace row id.
//	docId (uint) - the knowledge document row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response: { "code": 200, "msg": "success", "data": null }
func (h *Handlers) deleteKnowledgeDocument(c *gin.Context) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return
	}
	docID, ok := ginutil.ParamID(c, "docId")
	if !ok {
		return
	}

	doc, err := knmodels.GetKnowledgeDocumentByID(h.db, docID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "document not found")
		return
	}
	if doc.GroupID != ns.GroupID || doc.Namespace != ns.Namespace {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "document not found"})
		return
	}

	recordIDs := knowledge.DecodeRecordIDs(doc.RecordIDs)
	rawURL := doc.RawFileURL
	textURL := doc.TextURL
	namespace := ns.Namespace
	groupID := doc.GroupID

	_ = knmodels.SoftDeleteKnowledgeChunksByDocID(h.db, docID, groupID)

	if err := knmodels.HardDeleteKnowledgeDocument(h.db, doc.ID); err != nil {
		ginutil.WriteGORMError(c, err, "failed to delete document")
		return
	}

	h.enqueueKnowledgeDocumentPurge(knworker.PurgeJob{
		Namespace:  namespace,
		RecordIDs:  recordIDs,
		DocID:      docID,
		GroupID:    groupID,
		RawFileURL: rawURL,
		TextURL:    textURL,
	})

	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

func (h *Handlers) loadKnowledgeNamespace(c *gin.Context) (knmodels.KnowledgeNamespace, bool) {
	groupID := middleware.CurrentTenantID(c)
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return knmodels.KnowledgeNamespace{}, false
	}
	row, err := knmodels.GetKnowledgeNamespaceByIDAndGroup(h.db, id, groupID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "knowledge namespace not found")
		return knmodels.KnowledgeNamespace{}, false
	}
	return row, true
}

func (h *Handlers) loadKnowledgeDocument(c *gin.Context) (knmodels.KnowledgeNamespace, knmodels.KnowledgeDocument, bool) {
	ns, ok := h.loadKnowledgeNamespace(c)
	if !ok {
		return knmodels.KnowledgeNamespace{}, knmodels.KnowledgeDocument{}, false
	}
	docID, ok := ginutil.ParamID(c, "docId")
	if !ok {
		return knmodels.KnowledgeNamespace{}, knmodels.KnowledgeDocument{}, false
	}
	doc, err := knmodels.GetKnowledgeDocumentByID(h.db, docID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "document not found")
		return knmodels.KnowledgeNamespace{}, knmodels.KnowledgeDocument{}, false
	}
	if doc.GroupID != ns.GroupID || doc.Namespace != ns.Namespace {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "document not found"})
		return knmodels.KnowledgeNamespace{}, knmodels.KnowledgeDocument{}, false
	}
	return ns, doc, true
}

func (h *Handlers) readKnowledgeDocumentBody(doc knmodels.KnowledgeDocument) (string, error) {
	if key := strings.TrimSpace(doc.TextURL); key != "" {
		content, err := readKnowledgeDocContent(key)
		if err == nil && strings.TrimSpace(content) != "" {
			return content, nil
		}
	}
	if key := strings.TrimSpace(doc.RawFileURL); key != "" {
		raw, err := readKnowledgeDocBytes(key)
		if err == nil {
			name := strings.TrimSpace(doc.SourceFileName)
			if name == "" {
				name = "upload.txt"
			}
			if text, _, parseErr := llmparser.ParseBytesWithMeta(context.Background(), name, raw, nil); parseErr == nil {
				return text, nil
			}
		}
	}
	return doc.StoredMarkdown, nil
}

func readKnowledgeUpload(c *gin.Context) (*knowledgeUploadPayload, error) {
	if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil {
		title := strings.TrimSpace(fh.Filename)
		f, openErr := fh.Open()
		if openErr != nil {
			return nil, openErr
		}
		defer f.Close()
		raw, readErr := io.ReadAll(io.LimitReader(f, maxKnowledgeDocBytes+1))
		if readErr != nil {
			return nil, readErr
		}
		if len(raw) > maxKnowledgeDocBytes {
			return nil, fmt.Errorf("file too large (max %d bytes)", maxKnowledgeDocBytes)
		}
		if t := strings.TrimSpace(c.PostForm("title")); t != "" {
			title = t
		}
		meta := readKnowledgeUploadMeta(c)
		return &knowledgeUploadPayload{
			Title:       title,
			FileName:    fh.Filename,
			RawBytes:    raw,
			DocType:     meta.DocType,
			Tags:        meta.Tags,
			CampaignID:  meta.CampaignID,
			ProductLine: meta.ProductLine,
			IndexMode:   meta.IndexMode,
		}, nil
	}

	var req knowledgeDocumentUploadReq
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		return nil, fmt.Errorf("upload requires multipart file or JSON body with content")
	}
	text := strings.TrimSpace(req.Content)
	if text == "" {
		return nil, fmt.Errorf("document content is empty")
	}
	title := strings.TrimSpace(req.Title)
	fileName := "paste.txt"
	if title == "" {
		title = "Untitled"
	}
	return &knowledgeUploadPayload{
		Title:       title,
		FileName:    fileName,
		RawBytes:    []byte(text),
		DocType:     strings.TrimSpace(req.DocType),
		Tags:        req.Tags,
		CampaignID:  strings.TrimSpace(req.CampaignID),
		ProductLine: strings.TrimSpace(req.ProductLine),
		IndexMode:   strings.TrimSpace(req.IndexMode),
	}, nil
}

func readKnowledgeUploadMeta(c *gin.Context) knowledgeUploadPayload {
	tagsRaw := strings.TrimSpace(c.PostForm("tags"))
	var tags []string
	if tagsRaw != "" {
		for _, part := range strings.Split(tagsRaw, ",") {
			if t := strings.TrimSpace(part); t != "" {
				tags = append(tags, t)
			}
		}
	}
	return knowledgeUploadPayload{
		DocType:     strings.TrimSpace(c.PostForm("docType")),
		Tags:        tags,
		CampaignID:  strings.TrimSpace(c.PostForm("campaignId")),
		ProductLine: strings.TrimSpace(c.PostForm("productLine")),
		IndexMode:   strings.TrimSpace(c.PostForm("indexMode")),
	}
}

func knowledgeNamespaceDTO(r knmodels.KnowledgeNamespace) map[string]any {
	return map[string]any{
		"id":             fmt.Sprintf("%d", r.ID),
		"groupId":        fmt.Sprintf("%d", r.GroupID),
		"createdBy":      fmt.Sprintf("%d", r.CreatedBy),
		"namespace":      r.Namespace,
		"name":           r.Name,
		"description":    r.Description,
		"vectorProvider": r.VectorProvider,
		"embedModel":     r.EmbedModel,
		"vectorDim":      r.VectorDim,
		"status":         r.Status,
		"createdAt":      r.CreatedAt,
		"updatedAt":      r.UpdatedAt,
	}
}

func knowledgeDocumentDTO(namespaceID uint, doc knmodels.KnowledgeDocument) map[string]any {
	chunkCount := doc.ChunkCount
	if chunkCount <= 0 {
		chunkCount = len(knowledge.DecodeRecordIDs(doc.RecordIDs))
	}
	chunks := knowledge.DecodeChunkSummaries(doc.ChunksJSON)
	if chunkCount <= 0 && len(chunks) > 0 {
		chunkCount = len(chunks)
	}
	return map[string]any{
		"id":             fmt.Sprintf("%d", doc.ID),
		"namespaceId":    fmt.Sprintf("%d", namespaceID),
		"title":          doc.Title,
		"source":         doc.Source,
		"sourceType":     doc.Source,
		"sourceFileName": doc.SourceFileName,
		"textUrl":        doc.TextURL,
		"chunkCount":     chunkCount,
		"chunkStrategy":  doc.ChunkStrategy,
		"charCount":      len(doc.StoredMarkdown),
		"status":         doc.Status,
		"indexError":     doc.IndexError,
		"indexMode":      doc.IndexMode,
		"docType":        doc.DocType,
		"tags":           knowledge.DecodeTagsJSON(doc.TagsJSON),
		"campaignId":     doc.CampaignID,
		"productLine":    doc.ProductLine,
		"summaryText":    doc.SummaryText,
		"createdAt":      doc.CreatedAt,
		"updatedAt":      doc.UpdatedAt,
	}
}

// listKnowledgeDocumentChunks returns the stored chunk summary list (preview +
// index + charCount) for a document.
//
//	Endpoint: GET /knowledge/namespaces/:id/documents/:docId/chunks
//
// Path parameters:
//
//	id (uint)    - the knowledge namespace row id.
//	docId (uint) - the knowledge document row id.
//
// Query parameters: none.
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": {
//	    "docId": string(docId),
//	    "chunkCount": int,
//	    "chunkStrategy": "recursive | ...",
//	    "chunks": [ { "index": 0, "title": "...", "preview": "...", "charCount": 120 } ]
//	  }
//	}
func (h *Handlers) listKnowledgeDocumentChunks(c *gin.Context) {
	_, doc, ok := h.loadKnowledgeDocument(c)
	if !ok {
		return
	}
	chunks := knowledge.DecodeChunkSummaries(doc.ChunksJSON)
	out := make([]map[string]any, 0, len(chunks))
	for _, ch := range chunks {
		out = append(out, map[string]any{
			"index":     ch.Index,
			"title":     ch.Title,
			"preview":   ch.Preview,
			"charCount": ch.CharCount,
		})
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"docId":         fmt.Sprintf("%d", doc.ID),
		"chunkCount":    doc.ChunkCount,
		"chunkStrategy": doc.ChunkStrategy,
		"chunks":        out,
	})
}

// getKnowledgeDocumentChunk returns a single chunk's full content by its index.
//
//	Endpoint: GET /knowledge/namespaces/:id/documents/:docId/chunks/:chunkIndex
//
// Path parameters:
//
//	id (uint)        - the knowledge namespace row id.
//	docId (uint)     - the knowledge document row id.
//	chunkIndex (int) - 0-based chunk index.
//
// Query parameters: none.
//
// Request body: none.
//
// Response:
//
//	{
//	  "code": 200, "msg": "success",
//	  "data": {
//	    "docId": string(docId),
//	    "index": int,
//	    "title": "chunk title",
//	    "preview": "short preview",
//	    "content": "full chunk text (falls back to preview)",
//	    "charCount": int,
//	    "chunkStrategy": "recursive | ..."
//	  }
//	}
func (h *Handlers) getKnowledgeDocumentChunk(c *gin.Context) {
	_, doc, ok := h.loadKnowledgeDocument(c)
	if !ok {
		return
	}
	index, err := strconv.Atoi(strings.TrimSpace(c.Param("chunkIndex")))
	if err != nil || index < 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidChunkIndex))
		return
	}
	ch, found := knowledge.ChunkSummaryByIndex(doc.ChunksJSON, index)
	if !found {
		response.Render(c, response.NewI18n(response.CodeNotFound, i18n.KeyChunkNotFound))
		return
	}
	content := strings.TrimSpace(ch.Content)
	if content == "" {
		content = ch.Preview
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"docId":         fmt.Sprintf("%d", doc.ID),
		"index":         ch.Index,
		"title":         ch.Title,
		"preview":       ch.Preview,
		"content":       content,
		"charCount":     ch.CharCount,
		"chunkStrategy": doc.ChunkStrategy,
	})
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func nextKnowledgeDocID() uint {
	if id := utils.NextSnowflakeUint(); id > 0 {
		return id
	}
	// Fallback when snowflake is not initialized (tests): keep within signed int64 range.
	return uint(time.Now().UnixNano() & 0x7FFFFFFFFFFFFFFF)
}

// generateNamespace generates a vector collection slug from display name.
func generateNamespace(name string, groupID uint) string {
	slug := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "-"))
	slug = strings.ReplaceAll(slug, "_", "-")
	var b strings.Builder
	for _, c := range slug {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	slug = strings.Trim(b.String(), "-")
	if slug == "" {
		// Chinese or symbol-only names: use stable hash suffix instead of generic "kb".
		h := fnv.New32a()
		_, _ = h.Write([]byte(strings.TrimSpace(name)))
		slug = fmt.Sprintf("n%x", h.Sum32())
	}
	base := fmt.Sprintf("kb-%d-%s", groupID, slug)
	if len(base) > 120 {
		base = base[:120]
	}
	return base
}

// ensureUniqueKnowledgeNamespace appends -2, -3… when slug collides within tenant.
// Uses Unscoped so soft-deleted rows that still hold the unique key are visible.
func ensureUniqueKnowledgeNamespace(db *gorm.DB, groupID uint, base string) string {
	if db == nil {
		return base
	}
	candidate := base
	for i := 0; i < 100; i++ {
		var count int64
		_ = db.Unscoped().Model(&knmodels.KnowledgeNamespace{}).
			Where("group_id = ? AND namespace = ?", groupID, candidate).
			Count(&count)
		if count == 0 {
			return candidate
		}
		suffix := fmt.Sprintf("-%d", i+2)
		if len(base)+len(suffix) > 128 {
			candidate = base[:128-len(suffix)] + suffix
		} else {
			candidate = base + suffix
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

// retireKnowledgeNamespaceSlug renames the collection slug (and related rows) then soft-deletes
// the namespace so UNIQUE(group_id, namespace) is freed for recreate.
func retireKnowledgeNamespaceSlug(db *gorm.DB, ns knmodels.KnowledgeNamespace) error {
	if db == nil || ns.ID == 0 {
		return fmt.Errorf("invalid namespace")
	}
	oldSlug := strings.TrimSpace(ns.Namespace)
	if oldSlug == "" {
		return db.Delete(&ns).Error
	}
	freed := fmt.Sprintf("%s-del-%d", oldSlug, ns.ID)
	if len(freed) > 128 {
		freed = freed[:128]
	}
	return db.Transaction(func(tx *gorm.DB) error {
		groupID := ns.GroupID
		if err := tx.Model(&knmodels.KnowledgeDocument{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error; err != nil {
			return err
		}
		if err := tx.Model(&knmodels.KnowledgeChunk{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error; err != nil {
			return err
		}
		// Best-effort: keep ops tables consistent with the retired slug.
		_ = tx.Model(&knmodels.KnowledgeUnansweredQuestion{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error
		_ = tx.Model(&knmodels.KnowledgeAnsweredQuestion{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error
		_ = tx.Model(&knmodels.KnowledgeTypicalQuestion{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error
		_ = tx.Model(&knmodels.KnowledgeSyncSource{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error
		_ = tx.Model(&knmodels.KnowledgeEvalDataset{}).
			Where("group_id = ? AND namespace = ?", groupID, oldSlug).
			Update("namespace", freed).Error
		if err := tx.Model(&knmodels.KnowledgeNamespace{}).
			Where("id = ?", ns.ID).
			Update("namespace", freed).Error; err != nil {
			return err
		}
		return tx.Delete(&knmodels.KnowledgeNamespace{}, ns.ID).Error
	})
}

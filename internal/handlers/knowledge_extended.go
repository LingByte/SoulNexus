package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	llmeval "github.com/LingByte/lingllm/retrieve/eval"
	llmutils "github.com/LingByte/lingllm/utils"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

type knowledgeChunkCreateReq struct {
	DocID   uint   `json:"docId,string"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type knowledgeChunkUpdateReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type knowledgeUnansweredResolveReq struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type knowledgeEvalRunReq struct {
	DatasetID uint    `json:"datasetId,string"`
	Strategy  string  `json:"strategy"`
	TopK      int     `json:"topK"`
	MinScore  float64 `json:"minScore"`
}

type knowledgeSyncSourceReq struct {
	Name            string         `json:"name"`
	SourceType      string         `json:"sourceType"`
	SourceURL       string         `json:"sourceUrl"`
	IntervalMinutes int            `json:"intervalMinutes"`
	ChunkConfig     map[string]any `json:"chunkConfig"`
}

type knowledgeQuoteRateReq struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func (h *Handlers) requireKnowledgeNamespace(c *gin.Context) (knmodels.KnowledgeNamespace, bool) {
	groupID := middleware.CurrentTenantID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailWithCode(c, http.StatusBadRequest, "invalid namespace id", nil)
		return knmodels.KnowledgeNamespace{}, false
	}
	ns, err := knmodels.GetKnowledgeNamespaceByIDAndGroup(h.db, uint(id), groupID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "knowledge namespace not found")
		return knmodels.KnowledgeNamespace{}, false
	}
	return ns, true
}

// listKnowledgeChunks lists all slices in a namespace (optionally filtered by docId).
func (h *Handlers) listKnowledgeChunks(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	var docID uint
	if raw := strings.TrimSpace(c.Query("docId")); raw != "" {
		if n, err := strconv.ParseUint(raw, 10, 64); err == nil {
			docID = uint(n)
		}
	}
	rows, err := knmodels.ListKnowledgeChunksByNamespace(h.db, ns.Namespace, groupID, docID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "list chunks failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, rows)
}

// createKnowledgeChunk creates a manual slice or appends to a document.
func (h *Handlers) createKnowledgeChunk(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok || h.kb == nil {
		if h.kb == nil {
			response.FailWithCode(c, http.StatusServiceUnavailable, "knowledge service unavailable", nil)
		}
		return
	}
	var req knowledgeChunkCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.FailWithCode(c, http.StatusBadRequest, "content is required", nil)
		return
	}
	groupID := middleware.CurrentTenantID(c)
	ctx := c.Request.Context()
	title := strings.TrimSpace(req.Title)

	if req.DocID > 0 {
		doc, err := knmodels.GetKnowledgeDocumentByID(h.db, req.DocID)
		if err != nil || doc.GroupID != groupID || doc.Namespace != ns.Namespace {
			response.FailWithCode(c, http.StatusNotFound, "document not found", nil)
			return
		}
		chunks := knowledge.DecodeChunkSummaries(doc.ChunksJSON)
		idx := len(chunks)
		for _, ch := range chunks {
			if ch.Index >= idx {
				idx = ch.Index + 1
			}
		}
		recordID, err := knowledge.LoadChunkRecordID(doc, idx)
		if err != nil {
			response.FailWithCode(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		if err := h.kb.UpsertChunk(ctx, ns.Namespace, fmt.Sprintf("%d", doc.ID), recordID, title, content, idx); err != nil {
			response.FailWithCode(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		chunks = append(chunks, knowledge.ChunkSummary{Index: idx, Title: title, Content: content, Preview: llmutils.PreviewText(content, 240), CharCount: len([]rune(content))})
		ids := knowledge.DecodeRecordIDs(doc.RecordIDs)
		ids = append(ids, recordID)
		_ = knowledge.UpdateDocumentChunksJSON(h.db, doc.ID, chunks, ids, doc.ChunkStrategy)
		row := knmodels.KnowledgeChunk{GroupID: groupID, Namespace: ns.Namespace, DocID: doc.ID, ChunkIndex: idx, RecordID: recordID, Title: title, Content: content, SourceType: knconst.KnowledgeChunkSourceManual, Status: knconst.KnowledgeChunkStatusActive}
		if err := h.db.Create(&row).Error; err != nil {
			ginutil.WriteGORMError(c, err, "create chunk failed")
			return
		}
		response.SuccessI18n(c, i18n.KeySuccess, row)
		return
	}

	recordID, err := h.kb.UpsertManualChunk(ctx, ns.Namespace, "", "", title, content)
	if err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	row := knmodels.KnowledgeChunk{GroupID: groupID, Namespace: ns.Namespace, ChunkIndex: 0, RecordID: recordID, Title: title, Content: content, SourceType: knconst.KnowledgeChunkSourceManual, Status: knconst.KnowledgeChunkStatusActive}
	if err := h.db.Create(&row).Error; err != nil {
		ginutil.WriteGORMError(c, err, "create chunk failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// updateKnowledgeChunk edits one slice in-place (re-embed).
func (h *Handlers) updateKnowledgeChunk(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	if h.kb == nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "knowledge service unavailable", nil)
		return
	}
	chunkID, _ := strconv.ParseUint(c.Param("chunkId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	row, err := knmodels.GetKnowledgeChunkByID(h.db, uint(chunkID), groupID)
	if err != nil || row.Namespace != ns.Namespace {
		response.FailWithCode(c, http.StatusNotFound, "chunk not found", nil)
		return
	}
	var req knowledgeChunkUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.FailWithCode(c, http.StatusBadRequest, "content is required", nil)
		return
	}
	docID := fmt.Sprintf("manual-%d", row.ID)
	if row.DocID > 0 {
		docID = fmt.Sprintf("%d", row.DocID)
	}
	recordID := strings.TrimSpace(row.RecordID)
	if recordID == "" {
		response.FailWithCode(c, http.StatusInternalServerError, "chunk record id is empty", nil)
		return
	}
	if err := h.kb.UpsertChunk(c.Request.Context(), ns.Namespace, docID, recordID, title, content, row.ChunkIndex); err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	updates := map[string]any{"title": title, "content": content}
	if err := h.db.Model(&row).Updates(updates).Error; err != nil {
		ginutil.WriteGORMError(c, err, "update chunk failed")
		return
	}
	if row.DocID > 0 {
		_ = knowledge.PatchDocumentChunkSummary(h.db, row.DocID, row.ChunkIndex, title, content)
	}
	_ = h.db.Where("id = ?", row.ID).First(&row)
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// deleteKnowledgeChunk removes one slice.
func (h *Handlers) deleteKnowledgeChunk(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok || h.kb == nil {
		return
	}
	chunkID, _ := strconv.ParseUint(c.Param("chunkId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	row, err := knmodels.GetKnowledgeChunkByID(h.db, uint(chunkID), groupID)
	if err != nil || row.Namespace != ns.Namespace {
		response.FailWithCode(c, http.StatusNotFound, "chunk not found", nil)
		return
	}
	if err := h.kb.DeleteChunk(c.Request.Context(), ns.Namespace, []string{row.RecordID}); err != nil {
		response.FailWithCode(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	_ = h.db.Model(&row).Update("status", knconst.KnowledgeChunkStatusDeleted).Error
	if row.DocID > 0 {
		doc, err := knmodels.GetKnowledgeDocumentByID(h.db, row.DocID)
		if err == nil {
			chunks := knowledge.DecodeChunkSummaries(doc.ChunksJSON)
			filtered := make([]knowledge.ChunkSummary, 0, len(chunks))
			ids := knowledge.DecodeRecordIDs(doc.RecordIDs)
			newIDs := make([]string, 0, len(ids))
			for _, ch := range chunks {
				if ch.Index == row.ChunkIndex {
					continue
				}
				filtered = append(filtered, ch)
			}
			for _, id := range ids {
				if id != row.RecordID {
					newIDs = append(newIDs, id)
				}
			}
			_ = knowledge.UpdateDocumentChunksJSON(h.db, doc.ID, filtered, newIDs, doc.ChunkStrategy)
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

// exportKnowledgeChunks exports slices as Excel.
func (h *Handlers) exportKnowledgeChunks(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	rows, err := knmodels.ListKnowledgeChunksByNamespace(h.db, ns.Namespace, groupID, 0)
	if err != nil {
		ginutil.WriteGORMError(c, err, "export failed")
		return
	}
	f := excelize.NewFile()
	sheet := "Chunks"
	_ = f.SetSheetName("Sheet1", sheet)
	headers := []string{"DocID", "ChunkIndex", "Title", "Content", "RecordID", "SourceType"}
	for i, hname := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, hname)
	}
	for i, row := range rows {
		vals := []any{row.DocID, row.ChunkIndex, row.Title, row.Content, row.RecordID, row.SourceType}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", ns.Namespace+"-chunks.xlsx"))
	_ = f.Write(c.Writer)
}

// listKnowledgeUnansweredQuestions lists open unanswered questions.
func (h *Handlers) listKnowledgeUnansweredQuestions(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = knconst.KnowledgeUnansweredStatusOpen
	}
	rows, total, err := knmodels.ListKnowledgeUnansweredPage(h.db, ns.ID, groupID, status, page, size)
	if err != nil {
		ginutil.WriteGORMError(c, err, "list unanswered failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": rows, "total": total, "page": page, "size": size})
}

// countKnowledgeUnansweredQuestions returns counts by status.
func (h *Handlers) countKnowledgeUnansweredQuestions(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	open, _ := knmodels.CountKnowledgeUnansweredByNamespace(h.db, ns.ID, groupID, knconst.KnowledgeUnansweredStatusOpen)
	resolved, _ := knmodels.CountKnowledgeUnansweredByNamespace(h.db, ns.ID, groupID, knconst.KnowledgeUnansweredStatusResolved)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"open": open, "resolved": resolved})
}

// resolveKnowledgeUnansweredQuestion promotes an unanswered question into KB.
// Embedding is async; the HTTP response returns after DB rows are written.
func (h *Handlers) resolveKnowledgeUnansweredQuestion(c *gin.Context) {
	_, ok := h.requireKnowledgeNamespace(c)
	if !ok || h.kb == nil {
		return
	}
	qid, _ := strconv.ParseUint(c.Param("questionId"), 10, 64)
	var req knowledgeUnansweredResolveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		response.FailWithCode(c, http.StatusBadRequest, "content is required", nil)
		return
	}
	title := strings.TrimSpace(req.Title)
	groupID := middleware.CurrentTenantID(c)
	if err := knowledge.ResolveUnansweredToChunk(c.Request.Context(), h.db, h.kb, groupID, uint(qid), title, content); err != nil {
		ginutil.WriteGORMError(c, err, "resolve failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"queued": true})
}

// draftKnowledgeUnansweredAnswer prefills title/content for 「入库为切片」 using call context + ops LLM.
func (h *Handlers) draftKnowledgeUnansweredAnswer(c *gin.Context) {
	_, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	qid, _ := strconv.ParseUint(c.Param("questionId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	draft, err := knowledge.DraftUnansweredAnswer(c.Request.Context(), h.db, h.kb, groupID, uint(qid))
	if err != nil {
		ginutil.WriteGORMError(c, err, "draft failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, draft)
}

// deleteKnowledgeUnansweredQuestion ignores an unanswered question.
func (h *Handlers) deleteKnowledgeUnansweredQuestion(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	qid, _ := strconv.ParseUint(c.Param("questionId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	if err := h.db.Model(&knmodels.KnowledgeUnansweredQuestion{}).
		Where("id = ? AND group_id = ? AND namespace_id = ?", qid, groupID, ns.ID).
		Update("status", knconst.KnowledgeUnansweredStatusIgnored).Error; err != nil {
		ginutil.WriteGORMError(c, err, "delete failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

// listKnowledgeHFQuestions lists high-frequency typical questions.
func (h *Handlers) listKnowledgeHFQuestions(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	rows, total, err := knmodels.ListKnowledgeTypicalQuestionsPage(h.db, ns.ID, groupID, page, size)
	if err != nil {
		ginutil.WriteGORMError(c, err, "list hf questions failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": rows, "total": total, "page": page, "size": size})
}

// getKnowledgeHFDailySummary returns aggregated daily HF stats for a namespace.
func (h *Handlers) getKnowledgeHFDailySummary(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	from, to := timeutil.DefaultRollingDateRange(c.Query("from"), c.Query("to"), 7)
	rows, err := knowledge.ListHFDailySummary(h.db, ns.ID, groupID, from, to)
	if err != nil {
		ginutil.WriteGORMError(c, err, "hf daily summary failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": rows, "from": from.Format("2006-01-02"), "to": to.Add(-24 * time.Hour).Format("2006-01-02")})
}

// getKnowledgeHFQuestionStats returns per-day stats for one typical question.
func (h *Handlers) getKnowledgeHFQuestionStats(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	typicalID, _ := strconv.ParseUint(c.Param("typicalId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	if _, err := knmodels.GetKnowledgeTypicalQuestionByID(h.db, uint(typicalID), ns.ID, groupID); err != nil {
		ginutil.WriteGORMError(c, err, "typical question not found")
		return
	}
	from, to := timeutil.DefaultRollingDateRange(c.Query("from"), c.Query("to"), 7)
	rows, err := knowledge.ListTypicalQuestionDailyStats(h.db, uint(typicalID), ns.ID, groupID, from, to)
	if err != nil {
		ginutil.WriteGORMError(c, err, "hf question stats failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": rows, "typicalId": typicalID})
}

// listKnowledgeHFQuestionAnswers lists answered records for drill-down.
func (h *Handlers) listKnowledgeHFQuestionAnswers(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	typicalID, _ := strconv.ParseUint(c.Param("typicalId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	if _, err := knmodels.GetKnowledgeTypicalQuestionByID(h.db, uint(typicalID), ns.ID, groupID); err != nil {
		ginutil.WriteGORMError(c, err, "typical question not found")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	day := strings.TrimSpace(c.Query("day"))
	rows, total, err := knowledge.ListTypicalQuestionAnswersPage(h.db, uint(typicalID), ns.ID, groupID, day, page, size)
	if err != nil {
		ginutil.WriteGORMError(c, err, "list hf answers failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": rows, "total": total, "page": page, "size": size, "day": day})
}

// getKnowledgeQuoteRateReport returns knowledge citation analytics.
func (h *Handlers) getKnowledgeQuoteRateReport(c *gin.Context) {
	groupID := middleware.CurrentTenantID(c)
	var req knowledgeQuoteRateReq
	_ = c.ShouldBindJSON(&req)
	from, to := timeutil.DefaultRollingDateRange(req.From, req.To, 7)
	overview, err := knowledge.ComputeQuoteRateOverview(h.db, groupID, from, to)
	if err != nil {
		ginutil.WriteGORMError(c, err, "quote rate failed")
		return
	}
	nsStats, _ := knowledge.CollectNamespaceQuoteStats(c.Request.Context(), h.db, groupID, from, to)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"overview": overview, "namespaces": nsStats, "from": from, "to": to})
}

// runKnowledgeEval executes retrieval evaluation for a dataset.
func (h *Handlers) runKnowledgeEval(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	if h.kb == nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "knowledge service unavailable", nil)
		return
	}
	var req knowledgeEvalRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	groupID := middleware.CurrentTenantID(c)
	var ds knmodels.KnowledgeEvalDataset
	if err := h.db.Where("id = ? AND group_id = ? AND namespace_id = ?", req.DatasetID, groupID, ns.ID).First(&ds).Error; err != nil {
		ginutil.WriteGORMError(c, err, "dataset not found")
		return
	}
	var samples []llmeval.Sample
	if err := jsonUnmarshalSamples(ds.SamplesJSON, &samples); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "invalid dataset samples", nil)
		return
	}
	opts := llmeval.Options{Strategy: req.Strategy, TopK: req.TopK, MinScore: req.MinScore}
	if opts.TopK <= 0 {
		opts.TopK = 5
	}
	jobID := knowledge.StartRetrievalEvalJob(h.kb, ns.Namespace, samples, opts, false)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"jobId": jobID, "status": knowledge.EvalJobPending})
}

// compareKnowledgeEvalStrategies compares vector/keyword/hybrid on one dataset.
func (h *Handlers) compareKnowledgeEvalStrategies(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	if h.kb == nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "knowledge service unavailable", nil)
		return
	}
	var req knowledgeEvalRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	groupID := middleware.CurrentTenantID(c)
	var ds knmodels.KnowledgeEvalDataset
	if err := h.db.Where("id = ? AND group_id = ? AND namespace_id = ?", req.DatasetID, groupID, ns.ID).First(&ds).Error; err != nil {
		ginutil.WriteGORMError(c, err, "dataset not found")
		return
	}
	var samples []llmeval.Sample
	if err := jsonUnmarshalSamples(ds.SamplesJSON, &samples); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, "invalid dataset samples", nil)
		return
	}
	opts := llmeval.Options{TopK: req.TopK, MinScore: req.MinScore}
	if opts.TopK <= 0 {
		opts.TopK = 5
	}
	jobID := knowledge.StartRetrievalEvalJob(h.kb, ns.Namespace, samples, opts, true)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"jobId": jobID, "status": knowledge.EvalJobPending})
}

// getKnowledgeEvalJob returns async eval job status/result.
func (h *Handlers) getKnowledgeEvalJob(c *gin.Context) {
	if _, ok := h.requireKnowledgeNamespace(c); !ok {
		return
	}
	jobID := strings.TrimSpace(c.Param("jobId"))
	if jobID == "" {
		response.FailWithCode(c, http.StatusBadRequest, "jobId is required", nil)
		return
	}
	job, ok := knowledge.GetEvalJob(jobID)
	if !ok {
		response.FailWithCode(c, http.StatusNotFound, "eval job not found", nil)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, job)
}

// listKnowledgeSyncSources lists URL/API sync sources.
func (h *Handlers) listKnowledgeSyncSources(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	rows, err := knmodels.ListKnowledgeSyncSourcesByNamespace(h.db, ns.ID, groupID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "list sync sources failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, rows)
}

// createKnowledgeSyncSource adds a recurring sync source.
func (h *Handlers) createKnowledgeSyncSource(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	var req knowledgeSyncSourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if strings.TrimSpace(req.SourceURL) == "" {
		response.FailWithCode(c, http.StatusBadRequest, "sourceUrl is required", nil)
		return
	}
	groupID := middleware.CurrentTenantID(c)
	row := knmodels.KnowledgeSyncSource{
		GroupID:         groupID,
		NamespaceID:     ns.ID,
		Namespace:       ns.Namespace,
		Name:            strings.TrimSpace(req.Name),
		SourceType:      strings.TrimSpace(req.SourceType),
		SourceURL:       strings.TrimSpace(req.SourceURL),
		IntervalMinutes: req.IntervalMinutes,
		Status:          knconst.KnowledgeSyncStatusActive,
	}
	if row.SourceType == "" {
		row.SourceType = knconst.KnowledgeSyncTypeURL
	}
	if row.IntervalMinutes <= 0 {
		row.IntervalMinutes = 1440
	}
	if req.ChunkConfig != nil {
		if b, err := json.Marshal(req.ChunkConfig); err == nil {
			row.ChunkConfig = b
		}
	}
	if err := h.db.Create(&row).Error; err != nil {
		ginutil.WriteGORMError(c, err, "create sync source failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

func applyKnowledgeSyncSourceReq(row *knmodels.KnowledgeSyncSource, req knowledgeSyncSourceReq) {
	row.Name = strings.TrimSpace(req.Name)
	if st := strings.TrimSpace(req.SourceType); st != "" {
		row.SourceType = st
	}
	row.SourceURL = strings.TrimSpace(req.SourceURL)
	if req.IntervalMinutes > 0 {
		row.IntervalMinutes = req.IntervalMinutes
	}
	if req.ChunkConfig != nil {
		if b, err := json.Marshal(req.ChunkConfig); err == nil {
			row.ChunkConfig = b
		}
	}
}

// updateKnowledgeSyncSource edits a sync source.
func (h *Handlers) updateKnowledgeSyncSource(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	sourceID, _ := strconv.ParseUint(c.Param("sourceId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	var row knmodels.KnowledgeSyncSource
	if err := h.db.Where("id = ? AND group_id = ? AND namespace_id = ?", sourceID, groupID, ns.ID).First(&row).Error; err != nil {
		ginutil.WriteGORMError(c, err, "sync source not found")
		return
	}
	var req knowledgeSyncSourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	if strings.TrimSpace(req.SourceURL) == "" {
		response.FailWithCode(c, http.StatusBadRequest, "sourceUrl is required", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.FailWithCode(c, http.StatusBadRequest, "name is required", nil)
		return
	}
	applyKnowledgeSyncSourceReq(&row, req)
	if err := h.db.Save(&row).Error; err != nil {
		ginutil.WriteGORMError(c, err, "update sync source failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// deleteKnowledgeSyncSource removes a sync source.
func (h *Handlers) deleteKnowledgeSyncSource(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	sourceID, _ := strconv.ParseUint(c.Param("sourceId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	res := h.db.Where("id = ? AND group_id = ? AND namespace_id = ?", sourceID, groupID, ns.ID).Delete(&knmodels.KnowledgeSyncSource{})
	if res.Error != nil {
		ginutil.WriteGORMError(c, res.Error, "delete sync source failed")
		return
	}
	if res.RowsAffected == 0 {
		response.FailWithCode(c, http.StatusNotFound, "sync source not found", nil)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

// triggerKnowledgeSyncNow runs one sync immediately.
func (h *Handlers) triggerKnowledgeSyncNow(c *gin.Context) {
	if h.kbWorker == nil {
		response.FailWithCode(c, http.StatusServiceUnavailable, "worker unavailable", nil)
		return
	}
	sourceID, _ := strconv.ParseUint(c.Param("sourceId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	var src knmodels.KnowledgeSyncSource
	if err := h.db.Where("id = ? AND group_id = ?", sourceID, groupID).First(&src).Error; err != nil {
		ginutil.WriteGORMError(c, err, "sync source not found")
		return
	}
	h.enqueueKnowledgeSyncJob(src.ID)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"queued": true})
}

// getKnowledgeWorkerStats returns ingest queue depth and pending job list.
func (h *Handlers) getKnowledgeWorkerStats(c *gin.Context) {
	if _, ok := h.requireKnowledgeNamespace(c); !ok {
		return
	}
	if h.kbWorker == nil {
		response.SuccessI18n(c, i18n.KeySuccess, gin.H{"workers": 0, "queued": 0, "running": 0, "unfinished": 0, "jobs": []any{}})
		return
	}
	snap := h.kbWorker.Snapshot()
	response.SuccessI18n(c, i18n.KeySuccess, snap)
}

// getKnowledgeDocumentProgress returns document indexing status and worker queue position.
func (h *Handlers) getKnowledgeDocumentProgress(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	docID, ok := ginutil.ParamID(c, "docId")
	if !ok {
		return
	}
	doc, err := knmodels.GetKnowledgeDocumentByID(h.db, docID)
	if err != nil {
		ginutil.WriteGORMError(c, err, "knowledge document not found")
		return
	}
	if doc.GroupID != ns.GroupID || doc.Namespace != ns.Namespace {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"code": 404, "msg": "document not found"})
		return
	}
	out := gin.H{
		"docId":          doc.ID,
		"documentStatus": doc.Status,
		"inWorker":       false,
	}
	if h.kbWorker != nil {
		if jp, found := h.kbWorker.DocProgress(doc.ID); found {
			out["inWorker"] = true
			out["taskId"] = jp.TaskID
			out["taskStatus"] = jp.TaskStatus
			out["queueAhead"] = jp.QueueAhead
			out["queuedTotal"] = jp.QueuedTotal
			out["runningWorkers"] = jp.RunningWorkers
			out["unfinishedEstimate"] = jp.UnfinishedEstimate
			out["submittedAt"] = jp.SubmittedAt
		}
	}
	response.SuccessI18n(c, i18n.KeySuccess, out)
}

func jsonUnmarshalSamples(raw []byte, out *[]llmeval.Sample) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty")
	}
	return knowledge.UnmarshalEvalSamples(raw, out)
}

type knowledgeEvalDatasetReq struct {
	Name    string              `json:"name"`
	Samples string              `json:"samples"`
	Items   []evalSampleItemReq `json:"items"`
}

type evalSampleItemReq struct {
	Query        string   `json:"query"`
	RelevantIDs  []string `json:"relevant_ids"`
	RelevantIDs2 []string `json:"relevantIds"`
}

// listKnowledgeEvalDatasets lists labeled eval datasets for a namespace.
func (h *Handlers) listKnowledgeEvalDatasets(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	groupID := middleware.CurrentTenantID(c)
	var rows []knmodels.KnowledgeEvalDataset
	if err := h.db.Where("namespace_id = ? AND group_id = ?", ns.ID, groupID).Order("id DESC").Find(&rows).Error; err != nil {
		ginutil.WriteGORMError(c, err, "list eval datasets failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, rows)
}

// createKnowledgeEvalDataset uploads a labeled retrieval eval dataset.
func (h *Handlers) createKnowledgeEvalDataset(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	var req knowledgeEvalDatasetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.FailWithCode(c, http.StatusBadRequest, "name is required", nil)
		return
	}
	var samples []llmeval.Sample
	var samplesJSON []byte
	if len(req.Items) > 0 {
		samples = make([]llmeval.Sample, 0, len(req.Items))
		for i, it := range req.Items {
			ids := it.RelevantIDs
			if len(ids) == 0 {
				ids = it.RelevantIDs2
			}
			q := strings.TrimSpace(it.Query)
			if q == "" {
				response.FailWithCode(c, http.StatusBadRequest, fmt.Sprintf("sample %d: query is required", i+1), nil)
				return
			}
			if len(ids) == 0 {
				response.FailWithCode(c, http.StatusBadRequest, fmt.Sprintf("sample %d: relevant record id is required", i+1), nil)
				return
			}
			samples = append(samples, llmeval.Sample{Query: q, RelevantIDs: ids})
		}
		var err error
		samplesJSON, err = json.Marshal(samples)
		if err != nil {
			response.FailWithCode(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	} else {
		samplesRaw := strings.TrimSpace(req.Samples)
		if samplesRaw == "" {
			response.FailWithCode(c, http.StatusBadRequest, "items or samples is required", nil)
			return
		}
		if err := jsonUnmarshalSamples([]byte(samplesRaw), &samples); err != nil {
			response.FailWithCode(c, http.StatusBadRequest, "invalid samples: "+err.Error(), nil)
			return
		}
		samplesJSON = []byte(samplesRaw)
	}
	groupID := middleware.CurrentTenantID(c)
	row := knmodels.KnowledgeEvalDataset{
		GroupID:     groupID,
		NamespaceID: ns.ID,
		Namespace:   ns.Namespace,
		Name:        name,
		SamplesJSON: samplesJSON,
		SampleCount: len(samples),
	}
	if err := h.db.Create(&row).Error; err != nil {
		ginutil.WriteGORMError(c, err, "create eval dataset failed")
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, row)
}

// deleteKnowledgeEvalDataset removes one eval dataset.
func (h *Handlers) deleteKnowledgeEvalDataset(c *gin.Context) {
	ns, ok := h.requireKnowledgeNamespace(c)
	if !ok {
		return
	}
	datasetID, _ := strconv.ParseUint(c.Param("datasetId"), 10, 64)
	groupID := middleware.CurrentTenantID(c)
	res := h.db.Where("id = ? AND namespace_id = ? AND group_id = ?", datasetID, ns.ID, groupID).Delete(&knmodels.KnowledgeEvalDataset{})
	if res.Error != nil {
		ginutil.WriteGORMError(c, res.Error, "delete eval dataset failed")
		return
	}
	if res.RowsAffected == 0 {
		response.FailWithCode(c, http.StatusNotFound, "dataset not found", nil)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, nil)
}

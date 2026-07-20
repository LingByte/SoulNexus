package handlers

import (
	"context"
	"fmt"
	"strings"

	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	knworker "github.com/LingByte/SoulNexus/pkg/knowledge/worker"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	llmparser "github.com/LingByte/lingllm/parser"
	"go.uber.org/zap"
)

func (h *Handlers) updateKnowledgeDocumentStatus(docID uint, status string, extra map[string]any) {
	if h == nil || h.db == nil {
		return
	}
	updates := map[string]any{"status": status}
	for k, v := range extra {
		updates[k] = v
	}
	_ = h.db.Model(&knmodels.KnowledgeDocument{}).Where("id = ?", utils.ClampSnowflakeUint(docID)).Updates(updates).Error
}

func (h *Handlers) processKnowledgeIngest(ctx context.Context, job knworker.IngestJob) error {
	if h.kb == nil {
		h.markKnowledgeDocumentFailed(job.DocID, fmt.Errorf("knowledge base service unavailable"))
		return fmt.Errorf("knowledge base service unavailable")
	}

	if job.ConfirmIndex {
		return h.processKnowledgeConfirmIndex(ctx, job)
	}

	h.updateKnowledgeDocumentStatus(job.DocID, knconst.KnowledgeDocStatusParsing, nil)

	raw, err := readKnowledgeDocBytes(job.RawFileURL)
	if err != nil {
		h.markKnowledgeDocumentFailed(job.DocID, fmt.Errorf("read uploaded file: %w", err))
		return err
	}

	text, parseMeta, err := llmparser.ParseBytesWithMeta(ctx, job.FileName, raw, nil)
	if err != nil {
		h.markKnowledgeDocumentFailed(job.DocID, fmt.Errorf("parse document: %w", err))
		return err
	}

	if textKey := strings.TrimSpace(job.TextURL); textKey != "" {
		if err := writeKnowledgeDocContent(textKey, text); err != nil {
			h.markKnowledgeDocumentFailed(job.DocID, fmt.Errorf("store parsed text: %w", err))
			return err
		}
	}

	var doc knmodels.KnowledgeDocument
	if err := h.db.Where("id = ?", utils.ClampSnowflakeUint(job.DocID)).First(&doc).Error; err != nil {
		h.markKnowledgeDocumentFailed(job.DocID, err)
		return err
	}

	indexMode := strings.TrimSpace(doc.IndexMode)
	if indexMode == "" {
		indexMode = knconst.KnowledgeIndexModeParentChild
	}

	preview, err := h.kb.BuildDocumentPreview(ctx, job.Title, text, indexMode)
	if err != nil {
		h.markKnowledgeDocumentFailed(job.DocID, fmt.Errorf("chunk preview: %w", err))
		return err
	}
	if parseMeta != nil {
		preview.Parse = parseMeta
	}

	previewJSON := knowledge.EncodeDocumentPreview(preview)
	updates := map[string]any{
		"preview_json": previewJSON,
		"summary_text": preview.Summary,
		"text_url":     job.TextURL,
		"index_error":  "",
	}
	if h.kb != nil && h.kb.Config() != nil && h.kb.Config().IndexPreviewRequired {
		h.updateKnowledgeDocumentStatus(job.DocID, knconst.KnowledgeDocStatusPreview, updates)
		return nil
	}

	h.updateKnowledgeDocumentStatus(job.DocID, knconst.KnowledgeDocStatusIndexing, updates)
	return h.indexKnowledgeDocumentFromPreview(ctx, job, doc, preview)
}

func (h *Handlers) processKnowledgeConfirmIndex(ctx context.Context, job knworker.IngestJob) error {
	var doc knmodels.KnowledgeDocument
	if err := h.db.Where("id = ?", utils.ClampSnowflakeUint(job.DocID)).First(&doc).Error; err != nil {
		h.markKnowledgeDocumentFailed(job.DocID, err)
		return err
	}
	preview := knowledge.DecodeDocumentPreview(doc.PreviewJSON)
	if preview == nil || len(preview.Children) == 0 {
		err := fmt.Errorf("document preview is empty; re-upload or regenerate preview")
		h.markKnowledgeDocumentFailed(job.DocID, err)
		return err
	}
	h.updateKnowledgeDocumentStatus(job.DocID, knconst.KnowledgeDocStatusIndexing, nil)
	return h.indexKnowledgeDocumentFromPreview(ctx, job, doc, preview)
}

func (h *Handlers) indexKnowledgeDocumentFromPreview(ctx context.Context, job knworker.IngestJob, doc knmodels.KnowledgeDocument, preview *knowledge.DocumentPreview) error {
	if len(job.OldRecordIDs) > 0 {
		if err := h.kb.DeleteVectors(ctx, job.Namespace, job.OldRecordIDs); err != nil {
			logger.WarnCtx(ctx, "knowledge doc delete old vectors failed",
				zap.Uint("docId", job.DocID),
				zap.String("namespace", job.Namespace),
				zap.Error(err),
			)
		}
	}

	docIDStr := fmt.Sprintf("%d", utils.ClampSnowflakeUint(job.DocID))
	meta := knowledge.DocumentMetadata{
		DocType:     doc.DocType,
		Tags:        knowledge.DecodeTagsJSON(doc.TagsJSON),
		CampaignID:  doc.CampaignID,
		ProductLine: doc.ProductLine,
		CreatedAt:   doc.CreatedAt,
	}
	result, err := h.kb.IngestFromPreview(ctx, job.Namespace, docIDStr, job.Title, preview, meta)
	if err != nil {
		h.markKnowledgeDocumentFailed(job.DocID, err)
		return err
	}

	res := h.db.Model(&knmodels.KnowledgeDocument{}).Where("id = ?", utils.ClampSnowflakeUint(job.DocID)).Updates(map[string]any{
		"status":         knconst.KnowledgeDocStatusActive,
		"record_ids":     knowledge.EncodeRecordIDs(result.RecordIDs),
		"chunk_count":    result.ChunkCount,
		"chunk_strategy": result.ChunkStrategy,
		"chunks_json":    knowledge.EncodeChunkSummaries(result.Chunks),
		"summary_text":   result.SummaryText,
		"index_error":    "",
		"text_url":       job.TextURL,
	})
	if res.Error != nil {
		_ = h.kb.DeleteVectors(context.Background(), job.Namespace, result.RecordIDs)
		h.markKnowledgeDocumentFailed(job.DocID, res.Error)
		return res.Error
	}
	if res.RowsAffected == 0 {
		_ = h.kb.DeleteVectors(context.Background(), job.Namespace, result.RecordIDs)
	}
	if err := h.db.Where("id = ?", job.DocID).First(&doc).Error; err == nil {
		_ = knowledge.SyncChunkRegistryFromDocument(h.db, doc, result.Chunks, result.RecordIDs, knconst.KnowledgeChunkSourceIngest)
	}
	return nil
}

func (h *Handlers) enqueueKnowledgeDocumentIngest(job knworker.IngestJob) {
	if h == nil || h.db == nil {
		return
	}
	if h.kbWorker == nil {
		h.markKnowledgeDocumentFailed(job.DocID, fmt.Errorf("knowledge worker unavailable"))
		return
	}
	h.kbWorker.Enqueue(knworker.DocumentJob{Ingest: &job})
}

func (h *Handlers) enqueueKnowledgeDocumentPurge(job knworker.PurgeJob) {
	if h == nil || h.kbWorker == nil {
		return
	}
	h.kbWorker.Enqueue(knworker.DocumentJob{Purge: &job})
}

func (h *Handlers) processKnowledgeDocumentJob(ctx context.Context, job knworker.DocumentJob) error {
	if job.SyncSourceID > 0 {
		return h.processKnowledgeSyncSource(ctx, job.SyncSourceID)
	}
	if job.Purge != nil {
		return h.processKnowledgePurge(ctx, *job.Purge)
	}
	if job.Ingest != nil {
		return h.processKnowledgeIngest(ctx, *job.Ingest)
	}
	return nil
}

func (h *Handlers) processKnowledgePurge(ctx context.Context, job knworker.PurgeJob) error {
	if job.DocID > 0 && h.db != nil {
		if err := knmodels.SoftDeleteKnowledgeChunksByDocID(h.db, job.DocID, job.GroupID); err != nil {
			logger.WarnCtx(ctx, "knowledge doc purge chunks failed",
				zap.Uint("docId", job.DocID),
				zap.Error(err),
			)
		}
	}
	if h.kb != nil && len(job.RecordIDs) > 0 {
		if err := h.kb.DeleteVectors(ctx, job.Namespace, job.RecordIDs); err != nil {
			logger.WarnCtx(ctx, "knowledge doc purge vectors failed",
				zap.String("namespace", job.Namespace),
				zap.Error(err),
			)
			return err
		}
	}
	deleteKnowledgeDocFiles(job.RawFileURL, job.TextURL)
	return nil
}

func (h *Handlers) enqueueKnowledgeSyncJob(sourceID uint) {
	if h == nil || h.kbWorker == nil || sourceID == 0 {
		return
	}
	h.kbWorker.Enqueue(knworker.DocumentJob{SyncSourceID: sourceID})
}

func (h *Handlers) markKnowledgeDocumentFailed(docID uint, err error) {
	if h == nil || h.db == nil {
		return
	}
	msg := knowledge.FormatIndexError(err)
	res := h.db.Model(&knmodels.KnowledgeDocument{}).Where("id = ?", utils.ClampSnowflakeUint(docID)).Updates(map[string]any{
		"status":      knconst.KnowledgeDocStatusFailed,
		"index_error": msg,
	})
	if res.Error != nil {
		logger.Warn("knowledge doc mark failed update error",
			zap.Uint("docId", docID),
			zap.Error(res.Error),
		)
		return
	}
	if res.RowsAffected == 0 {
		logger.Warn("knowledge doc mark failed: document row not found", zap.Uint("docId", docID))
	}
}

// StopKnowledgeWorker stops the background document worker pool (process shutdown).
func (h *Handlers) StopKnowledgeWorker() {
	if h == nil || h.kbWorker == nil {
		return
	}
	_ = h.kbWorker.Stop()
	h.kbWorker = nil
}

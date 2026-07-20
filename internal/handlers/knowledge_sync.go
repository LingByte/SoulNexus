package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	knowledge "github.com/LingByte/SoulNexus/pkg/knowledge/service"
	knworker "github.com/LingByte/SoulNexus/pkg/knowledge/worker"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
)

func (h *Handlers) processKnowledgeSyncSource(ctx context.Context, sourceID uint) error {
	if h == nil || h.db == nil || h.kb == nil {
		return fmt.Errorf("knowledge sync unavailable")
	}
	var src knmodels.KnowledgeSyncSource
	if err := h.db.WithContext(ctx).Where("id = ?", sourceID).First(&src).Error; err != nil {
		return err
	}
	if src.Status != knconst.KnowledgeSyncStatusActive {
		return nil
	}
	url := strings.TrimSpace(src.SourceURL)
	if url == "" {
		return fmt.Errorf("empty source url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_ = h.db.Model(&src).Update("last_sync_error", err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("fetch status %d", resp.StatusCode)
		_ = h.db.Model(&src).Update("last_sync_error", err.Error())
		return err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxKnowledgeDocBytes))
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(src.SourceType), knconst.KnowledgeSyncTypeTable) {
		return h.processKnowledgeTableSync(ctx, src, body)
	}
	return h.processKnowledgeURLSync(ctx, src, body)
}

func (h *Handlers) processKnowledgeURLSync(ctx context.Context, src knmodels.KnowledgeSyncSource, body []byte) error {
	title := strings.TrimSpace(src.Name)
	if title == "" {
		title = src.SourceURL
	}
	var doc knmodels.KnowledgeDocument
	if src.DocumentID > 0 {
		if err := h.db.Where("id = ?", src.DocumentID).First(&doc).Error; err != nil {
			return err
		}
		oldIDs := knowledge.DecodeRecordIDs(doc.RecordIDs)
		job := knworker.IngestJob{
			DocID:        doc.ID,
			Namespace:    src.Namespace,
			Title:        title,
			FileName:     "sync.html",
			OldRecordIDs: oldIDs,
		}
		if err := writeKnowledgeDocContent(doc.RawFileURL, string(body)); err == nil {
			job.RawFileURL = doc.RawFileURL
		}
		err := h.processKnowledgeIngest(ctx, job)
		if err == nil {
			now := time.Now()
			_ = h.db.Model(&src).Updates(map[string]any{"last_sync_at": &now, "last_sync_error": ""})
		}
		return err
	}
	doc = knmodels.KnowledgeDocument{
		GroupID:   src.GroupID,
		Namespace: src.Namespace,
		Title:     title,
		Source:    knconst.KnowledgeSyncTypeURL,
		Status:    knconst.KnowledgeDocStatusProcessing,
	}
	if err := h.db.Create(&doc).Error; err != nil {
		return err
	}
	_ = h.db.Model(&src).Updates(map[string]any{"document_id": doc.ID}).Error
	h.enqueueKnowledgeDocumentIngest(knworker.IngestJob{
		DocID:     doc.ID,
		Namespace: src.Namespace,
		Title:     title,
		FileName:  "sync.html",
	})
	now := time.Now()
	_ = h.db.Model(&src).Updates(map[string]any{"last_sync_at": &now, "last_sync_error": ""})
	return nil
}

func (h *Handlers) processKnowledgeTableSync(ctx context.Context, src knmodels.KnowledgeSyncSource, body []byte) error {
	cfg := knowledge.ParseTableSyncConfig(src.ChunkConfig)
	rows, err := knowledge.ParseTableSyncBody(body, cfg)
	if err != nil {
		_ = h.db.Model(&src).Update("last_sync_error", err.Error())
		return err
	}
	docID := src.DocumentID
	if docID == 0 {
		doc := knmodels.KnowledgeDocument{
			GroupID:   src.GroupID,
			Namespace: src.Namespace,
			Title:     strings.TrimSpace(src.Name),
			Source:    knconst.KnowledgeSyncTypeTable,
			Status:    knconst.KnowledgeDocStatusActive,
		}
		if doc.Title == "" {
			doc.Title = fmt.Sprintf("table-sync-%d", src.ID)
		}
		if err := h.db.Create(&doc).Error; err != nil {
			return err
		}
		docID = doc.ID
		_ = h.db.Model(&src).Update("document_id", docID)
	}
	docIDStr := fmt.Sprintf("table-sync-%d", src.ID)
	prefix := fmt.Sprintf("table-%d-", src.ID)
	existing, _ := knmodels.ListKnowledgeChunksByRecordPrefix(h.db, src.Namespace, src.GroupID, prefix)
	existingByRecord := map[string]knmodels.KnowledgeChunk{}
	for _, ch := range existing {
		existingByRecord[ch.RecordID] = ch
	}
	newIDs := map[string]struct{}{}
	for i, row := range rows {
		recordID := knowledge.TableRowRecordID(src.ID, row.Key)
		newIDs[recordID] = struct{}{}
		if err := h.kb.UpsertChunk(ctx, src.Namespace, docIDStr, recordID, row.Title, row.Content, i); err != nil {
			_ = h.db.Model(&src).Update("last_sync_error", err.Error())
			return err
		}
		if ch, ok := existingByRecord[recordID]; ok {
			_ = h.db.Model(&ch).Updates(map[string]any{
				"title": row.Title, "content": row.Content, "chunk_index": i, "doc_id": docID,
			}).Error
		} else {
			_ = h.db.Create(&knmodels.KnowledgeChunk{
				GroupID: src.GroupID, Namespace: src.Namespace, DocID: docID,
				ChunkIndex: i, RecordID: recordID, Title: row.Title, Content: row.Content,
				SourceType: knconst.KnowledgeChunkSourceTable, Status: knconst.KnowledgeChunkStatusActive,
			}).Error
		}
	}
	for _, ch := range existing {
		if _, keep := newIDs[ch.RecordID]; keep {
			continue
		}
		_ = h.kb.DeleteChunk(ctx, src.Namespace, []string{ch.RecordID})
		_ = h.db.Model(&ch).Update("status", knconst.KnowledgeChunkStatusDeleted)
	}
	_ = h.db.Model(&knmodels.KnowledgeDocument{}).Where("id = ?", docID).Updates(map[string]any{
		"status": knconst.KnowledgeDocStatusActive, "chunk_count": len(rows), "index_error": "",
	}).Error
	now := time.Now()
	return h.db.Model(&src).Updates(map[string]any{"last_sync_at": &now, "last_sync_error": ""}).Error
}

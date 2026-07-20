package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	"github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"github.com/LingByte/lingllm/utils"
	"gorm.io/gorm"
	"strings"
	"time"
)

// QuoteRateOverview is knowledge citation analytics for a time range.
type QuoteRateOverview struct {
	TotalCalls int64   `json:"totalCalls"`
	QuotedCalls int64  `json:"quotedCalls"`
	QuoteRate  float64 `json:"quoteRate"`
	HitRate    float64 `json:"hitRate"`
}

// QuoteRateBucket is one time bucket in the quote-rate trend.
type QuoteRateBucket struct {
	Label       string  `json:"label"`
	TotalCalls  int64   `json:"totalCalls"`
	QuotedCalls int64   `json:"quotedCalls"`
	QuoteRate   float64 `json:"quoteRate"`
}

// ComputeQuoteRateOverview aggregates knowledge usage (turn analytics removed).
func ComputeQuoteRateOverview(_ *gorm.DB, _ uint, _, _ time.Time) (QuoteRateOverview, error) {
	return QuoteRateOverview{}, nil
}

// SyncChunkRegistryFromDocument upserts knowledge_chunks rows from a document ingest result.
func SyncChunkRegistryFromDocument(db *gorm.DB, doc models.KnowledgeDocument, summaries []ChunkSummary, recordIDs []string, sourceType string) error {
	if db == nil || doc.ID == 0 {
		return nil
	}
	if sourceType == "" {
		sourceType = constants.KnowledgeChunkSourceIngest
	}
	// Soft-delete old chunks for this document
	if err := db.Model(&models.KnowledgeChunk{}).
		Where("doc_id = ? AND group_id = ?", doc.ID, doc.GroupID).
		Update("status", constants.KnowledgeChunkStatusDeleted).Error; err != nil {
		return err
	}
	for i, ch := range summaries {
		recordID := ""
		if i < len(recordIDs) {
			recordID = recordIDs[i]
		}
		row := models.KnowledgeChunk{
			GroupID:    doc.GroupID,
			Namespace:  doc.Namespace,
			DocID:      doc.ID,
			ChunkIndex: ch.Index,
			RecordID:   recordID,
			Title:      ch.Title,
			Content:    ch.Content,
			SourceType: sourceType,
			Status:     constants.KnowledgeChunkStatusActive,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// UpdateDocumentChunksJSON refreshes ChunksJSON and RecordIDs on a document row.
func UpdateDocumentChunksJSON(db *gorm.DB, docID uint, summaries []ChunkSummary, recordIDs []string, chunkStrategy string) error {
	updates := map[string]any{
		"chunks_json":    EncodeChunkSummaries(summaries),
		"record_ids":     EncodeRecordIDs(recordIDs),
		"chunk_count":    len(summaries),
		"chunk_strategy": chunkStrategy,
		"status":         constants.KnowledgeDocStatusActive,
		"index_error":    "",
	}
	return db.Model(&models.KnowledgeDocument{}).Where("id = ?", docID).Updates(updates).Error
}

// PatchDocumentChunkSummary updates one chunk in document JSON after slice edit.
func PatchDocumentChunkSummary(db *gorm.DB, docID uint, index int, title, content string) error {
	var doc models.KnowledgeDocument
	if err := db.Where("id = ?", docID).First(&doc).Error; err != nil {
		return err
	}
	chunks := DecodeChunkSummaries(doc.ChunksJSON)
	found := false
	for i := range chunks {
		if chunks[i].Index == index {
			chunks[i].Title = title
			chunks[i].Content = content
			chunks[i].Preview = utils.PreviewText(content, chunkPreviewMaxRunes)
			chunks[i].CharCount = len([]rune(content))
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("chunk index %d not found", index)
	}
	return db.Model(&doc).Update("chunks_json", EncodeChunkSummaries(chunks)).Error
}

// LoadChunkRecordID resolves vector point id for a document chunk index.
func LoadChunkRecordID(doc models.KnowledgeDocument, index int) (string, error) {
	ids := DecodeRecordIDs(doc.RecordIDs)
	if index >= 0 && index < len(ids) {
		return ids[index], nil
	}
	return chunkPointID(fmt.Sprintf("%d", doc.ID), index), nil
}

// EncodeQuoteChunkIDs serializes quoted chunk ids for answered question rows.
func EncodeQuoteChunkIDs(ids []uint) []byte {
	if len(ids) == 0 {
		return []byte("[]")
	}
	b, _ := json.Marshal(ids)
	return b
}

// DecodeQuoteChunkIDs parses quoted chunk ids.
func DecodeQuoteChunkIDs(raw []byte) []uint {
	if len(raw) == 0 {
		return nil
	}
	var ids []uint
	_ = json.Unmarshal(raw, &ids)
	return ids
}

// CollectNamespaceQuoteStats returns per-namespace quote stats for overview dashboard.
type NamespaceQuoteStat struct {
	NamespaceID   uint    `json:"namespaceId"`
	NamespaceName string  `json:"namespaceName"`
	QuoteRate     float64 `json:"quoteRate"`
	HitRate       float64 `json:"hitRate"`
	TotalCalls    int64   `json:"totalCalls"`
}

func CollectNamespaceQuoteStats(ctx context.Context, db *gorm.DB, groupID uint, from, to time.Time) ([]NamespaceQuoteStat, error) {
	if db == nil {
		return nil, nil
	}
	namespaces, err := models.ListKnowledgeNamespacesByGroup(db.WithContext(ctx), groupID)
	if err != nil {
		return nil, err
	}
	overview, err := ComputeQuoteRateOverview(db.WithContext(ctx), groupID, from, to)
	if err != nil {
		return nil, err
	}
	_ = overview
	out := make([]NamespaceQuoteStat, 0, len(namespaces))
	for _, ns := range namespaces {
		out = append(out, NamespaceQuoteStat{
			NamespaceID:   ns.ID,
			NamespaceName: ns.Name,
			QuoteRate:     overview.QuoteRate,
			HitRate:       overview.HitRate,
			TotalCalls:    overview.TotalCalls,
		})
	}
	return out, nil
}

// TypicalQuestionDailyStat is one day of HF question metrics.
type TypicalQuestionDailyStat struct {
	StatDate    string `json:"statDate"`
	Count       int    `json:"count"`
	QuotedCount int    `json:"quotedCount"`
}

// HFDailySummary aggregates all typical questions per day.
type HFDailySummary struct {
	StatDate    string `json:"statDate"`
	Count       int    `json:"count"`
	QuotedCount int    `json:"quotedCount"`
}

// ListTypicalQuestionDailyStats returns per-day stats for one typical question.
func ListTypicalQuestionDailyStats(db *gorm.DB, typicalID, namespaceID, groupID uint, from, to time.Time) ([]TypicalQuestionDailyStat, error) {
	if db == nil || typicalID == 0 {
		return nil, nil
	}
	var rows []models.KnowledgeTypicalQuestionStat
	q := db.Where("typical_question_id = ? AND namespace_id = ? AND group_id = ?", typicalID, namespaceID, groupID)
	if !from.IsZero() {
		q = q.Where("stat_date >= ?", from.Truncate(24*time.Hour))
	}
	if !to.IsZero() {
		q = q.Where("stat_date < ?", to.Truncate(24*time.Hour))
	}
	if err := q.Order("stat_date ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TypicalQuestionDailyStat, 0, len(rows))
	for _, r := range rows {
		out = append(out, TypicalQuestionDailyStat{
			StatDate:    r.StatDate.Format("2006-01-02"),
			Count:       r.Count,
			QuotedCount: r.QuotedCount,
		})
	}
	return out, nil
}

// ListHFDailySummary aggregates daily counts for a namespace.
func ListHFDailySummary(db *gorm.DB, namespaceID, groupID uint, from, to time.Time) ([]HFDailySummary, error) {
	if db == nil {
		return nil, nil
	}
	type aggRow struct {
		StatDate    time.Time
		Count       int
		QuotedCount int
	}
	var rows []aggRow
	q := db.Model(&models.KnowledgeTypicalQuestionStat{}).
		Select("stat_date, SUM(count) as count, SUM(quoted_count) as quoted_count").
		Where("namespace_id = ? AND group_id = ?", namespaceID, groupID).
		Group("stat_date").
		Order("stat_date ASC")
	if !from.IsZero() {
		q = q.Where("stat_date >= ?", from.Truncate(24*time.Hour))
	}
	if !to.IsZero() {
		q = q.Where("stat_date < ?", to.Truncate(24*time.Hour))
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]HFDailySummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, HFDailySummary{
			StatDate:    r.StatDate.Format("2006-01-02"),
			Count:       r.Count,
			QuotedCount: r.QuotedCount,
		})
	}
	return out, nil
}

// ListTypicalQuestionAnswersPage lists answered question records for drill-down.
func ListTypicalQuestionAnswersPage(db *gorm.DB, typicalID, namespaceID, groupID uint, day string, page, size int) ([]models.KnowledgeAnsweredQuestion, int64, error) {
	q := db.Model(&models.KnowledgeAnsweredQuestion{}).
		Where("typical_question_id = ? AND namespace_id = ? AND group_id = ?", typicalID, namespaceID, groupID)
	if strings.TrimSpace(day) != "" {
		loc := time.Local
		if t, err := time.ParseInLocation("2006-01-02", day, loc); err == nil {
			end := t.Add(24 * time.Hour)
			q = q.Where("created_at >= ? AND created_at < ?", t, end)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	var rows []models.KnowledgeAnsweredQuestion
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

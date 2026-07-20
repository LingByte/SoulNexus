package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	knconst "github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// KnowledgeNamespace represents a knowledge base (maps to a vector backend collection).
type KnowledgeNamespace struct {
	common.BaseModel
	GroupID        uint   `json:"groupId" gorm:"uniqueIndex:idx_kn_group_ns;not null;default:0;comment:tenant group id"`
	CreatedBy      uint   `json:"createdBy" gorm:"index;comment:user who created the namespace"`
	Namespace      string `json:"namespace" gorm:"type:varchar(128);uniqueIndex:idx_kn_group_ns;not null;comment:vector backend collection name"`
	Name           string `json:"name" gorm:"type:varchar(255);not null;comment:display name"`
	Description    string `json:"description" gorm:"type:text"`
	VectorProvider string `json:"-" gorm:"type:varchar(32);not null;default:'qdrant';index;comment:server-configured vector backend snapshot at create time"`
	EmbedModel     string `json:"-" gorm:"type:varchar(64);not null;comment:embedding model"`
	VectorDim      int    `json:"-" gorm:"not null;comment:embedding dim"`
	Status         string `json:"-" gorm:"type:varchar(20);index;not null;default:'active'"`
}

func (KnowledgeNamespace) TableName() string {
	return constants.KNOWLEDGE_NAMESPACE_TABLE_NAME
}

// GetKnowledgeNamespaceByIDAndGroup loads one namespace scoped to group.
func GetKnowledgeNamespaceByIDAndGroup(db *gorm.DB, id, groupID uint) (KnowledgeNamespace, error) {
	var row KnowledgeNamespace
	err := db.Where("id = ? AND group_id = ?", id, groupID).First(&row).Error
	return row, err
}

// GetKnowledgeNamespaceByNamespaceAndGroup loads one namespace by collection slug.
func GetKnowledgeNamespaceByNamespaceAndGroup(db *gorm.DB, namespace string, groupID uint) (KnowledgeNamespace, error) {
	var row KnowledgeNamespace
	err := db.Where("namespace = ? AND group_id = ?", namespace, groupID).First(&row).Error
	return row, err
}

// ListKnowledgeNamespacesByGroup lists all namespaces for a group.
func ListKnowledgeNamespacesByGroup(db *gorm.DB, groupID uint) ([]KnowledgeNamespace, error) {
	var rows []KnowledgeNamespace
	err := db.Where("group_id = ?", groupID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// KnowledgeDocument represents a user-uploaded document to a knowledge namespace, corresponding to a batch of vector records.
type KnowledgeDocument struct {
	common.BaseModel

	GroupID        uint   `json:"groupId" gorm:"index;not null;default:0;comment:tenant group id"`
	CreatedBy      uint   `json:"createdBy" gorm:"index"`
	Namespace      string `json:"namespace" gorm:"type:varchar(128);index;not null"`
	Title          string `json:"title" gorm:"type:varchar(255);not null"`
	Source         string `json:"source" gorm:"type:varchar(128);comment:upload|url|api|..."`
	SourceFileName string `json:"sourceFileName,omitempty" gorm:"type:varchar(255);comment:original upload filename"`
	RawFileURL     string `json:"rawFileUrl,omitempty" gorm:"type:text;comment:original file object key"`
	FileHash       string `json:"fileHash" gorm:"type:varchar(64);index;not null"`
	TextURL        string `json:"textUrl,omitempty" gorm:"type:text;comment:parsed text object key"`
	StoredMarkdown string `json:"storedMarkdown,omitempty" gorm:"type:longtext;comment:fallback parsed text when text_url empty"`
	RecordIDs      string `json:"recordIds" gorm:"type:text;comment:related vector ids (csv or json)"`
	ChunkCount     int    `json:"chunkCount" gorm:"not null;default:0;comment:vector chunk count"`
	ChunkStrategy  string `json:"chunkStrategy,omitempty" gorm:"type:varchar(64);comment:lingllm chunk strategy"`
	ChunksJSON     string `json:"-" gorm:"type:longtext;comment:chunk preview json"`
	PreviewJSON    string `json:"-" gorm:"type:longtext;comment:parse+chunk preview before confirm"`
	IndexMode      string `json:"indexMode,omitempty" gorm:"type:varchar(32);default:'parent_child';comment:flat|parent_child"`
	DocType        string `json:"docType,omitempty" gorm:"type:varchar(64);index;comment:document type for metadata filter"`
	TagsJSON       string `json:"-" gorm:"type:text;comment:json array of tags"`
	CampaignID     string `json:"campaignId,omitempty" gorm:"type:varchar(128);index;comment:outbound campaign filter"`
	ProductLine    string `json:"productLine,omitempty" gorm:"type:varchar(128);index;comment:product line filter"`
	SummaryText    string `json:"summaryText,omitempty" gorm:"type:longtext;comment:document summary for summary index"`
	IndexError     string `json:"indexError,omitempty" gorm:"type:varchar(1024);comment:last vector index error"`
	Status         string `json:"-" gorm:"type:varchar(20);index;not null;default:'processing'"`
}

func (KnowledgeDocument) TableName() string {
	return constants.KNOWLEDGE_DOCUMENT_TABLE_NAME
}

// RepairKnowledgeSnowflakeNegativeIDs clears the sign bit on snowflake IDs that were
// written as signed SQLite/MySQL INTEGER (unreadable into Go uint).
func RepairKnowledgeSnowflakeNegativeIDs(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	const mask = int64(0x7FFFFFFFFFFFFFFF)
	stmts := []string{
		`UPDATE knowledge_documents SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_chunks SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_chunks SET doc_id = (doc_id & ?) WHERE doc_id < 0`,
		`UPDATE knowledge_namespaces SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_unanswered_questions SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_answered_questions SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_typical_questions SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_sync_sources SET id = (id & ?) WHERE id < 0`,
		`UPDATE knowledge_sync_sources SET document_id = (document_id & ?) WHERE document_id < 0`,
		`UPDATE knowledge_eval_datasets SET id = (id & ?) WHERE id < 0`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt, mask).Error; err != nil {
			// Table may not exist yet on fresh installs; ignore.
			continue
		}
	}
	return nil
}

// GetKnowledgeDocumentByID loads one document by ID.
func GetKnowledgeDocumentByID(db *gorm.DB, id uint) (KnowledgeDocument, error) {
	var row KnowledgeDocument
	err := db.Where("id = ?", utils.ClampSnowflakeUint(id)).First(&row).Error
	return row, err
}

// ListKnowledgeDocumentsByNamespace lists documents for a namespace.
func ListKnowledgeDocumentsByNamespace(db *gorm.DB, namespace string, groupID uint) ([]KnowledgeDocument, error) {
	var rows []KnowledgeDocument
	err := db.Where("namespace = ? AND group_id = ?", namespace, groupID).Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// HardDeleteKnowledgeDocument permanently removes a document row.
func HardDeleteKnowledgeDocument(db *gorm.DB, id uint) error {
	return db.Unscoped().Delete(&KnowledgeDocument{}, id).Error
}

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT
// KnowledgeChunk is the canonical slice registry (vector point + metadata).
type KnowledgeChunk struct {
	common.BaseModel
	GroupID    uint   `json:"groupId" gorm:"index;not null;default:0"`
	Namespace  string `json:"namespace" gorm:"type:varchar(128);index;not null"`
	DocID      uint   `json:"docId,string" gorm:"index;default:0;comment:0 for manual slices"`
	ChunkIndex int    `json:"chunkIndex" gorm:"index;not null;default:0"`
	RecordID   string `json:"recordId" gorm:"type:varchar(64);index;not null;comment:vector point id"`
	Title      string `json:"title" gorm:"type:varchar(512)"`
	Content    string `json:"content" gorm:"type:longtext"`
	SourceType string `json:"sourceType" gorm:"type:varchar(32);index;not null;default:'ingest'"`
	Status     string `json:"status" gorm:"type:varchar(20);index;not null;default:'active'"`
}

func (KnowledgeChunk) TableName() string { return constants.KNOWLEDGE_CHUNK_TABLE_NAME }

// LookupChunkIDsByRecordIDsInNamespace maps vector record ids to chunk ids within a namespace.
func LookupChunkIDsByRecordIDsInNamespace(db *gorm.DB, namespace string, recordIDs []string) map[string]uint {
	out := map[string]uint{}
	if db == nil || namespace == "" || len(recordIDs) == 0 {
		return out
	}
	var rows []KnowledgeChunk
	if err := db.Where("namespace = ? AND record_id IN ? AND status = ?",
		namespace, recordIDs, knconst.KnowledgeChunkStatusActive).Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		if r.RecordID != "" {
			out[r.RecordID] = r.ID
		}
	}
	return out
}

// LookupChunkIDsByRecordIDs maps vector record ids to knowledge_chunks.id.
func LookupChunkIDsByRecordIDs(db *gorm.DB, namespace string, groupID uint, recordIDs []string) map[string]uint {
	out := map[string]uint{}
	if db == nil || len(recordIDs) == 0 {
		return out
	}
	var rows []KnowledgeChunk
	if err := db.Where("namespace = ? AND group_id = ? AND record_id IN ? AND status = ?",
		namespace, groupID, recordIDs, knconst.KnowledgeChunkStatusActive).Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		if r.RecordID != "" {
			out[r.RecordID] = r.ID
		}
	}
	return out
}

// SoftDeleteKnowledgeChunksByDocID marks all chunks of a document as deleted.
func SoftDeleteKnowledgeChunksByDocID(db *gorm.DB, docID, groupID uint) error {
	if db == nil || docID == 0 {
		return nil
	}
	q := db.Model(&KnowledgeChunk{}).Where("doc_id = ?", docID)
	if groupID > 0 {
		q = q.Where("group_id = ?", groupID)
	}
	return q.Update("status", knconst.KnowledgeChunkStatusDeleted).Error
}

// ListKnowledgeChunksByNamespace lists active chunks for a namespace.
func ListKnowledgeChunksByNamespace(db *gorm.DB, namespace string, groupID uint, docID uint) ([]KnowledgeChunk, error) {
	var rows []KnowledgeChunk
	q := db.Where("namespace = ? AND group_id = ? AND status = ?", namespace, groupID, knconst.KnowledgeChunkStatusActive)
	if docID > 0 {
		q = q.Where("doc_id = ?", docID)
	}
	err := q.Order("doc_id ASC, chunk_index ASC").Find(&rows).Error
	return rows, err
}

// GetKnowledgeChunkByID loads one chunk scoped to group.
func GetKnowledgeChunkByID(db *gorm.DB, id, groupID uint) (KnowledgeChunk, error) {
	var row KnowledgeChunk
	err := db.Where("id = ? AND group_id = ?", utils.ClampSnowflakeUint(id), groupID).First(&row).Error
	return row, err
}

// ListKnowledgeChunksByRecordPrefix lists active chunks whose record_id starts with prefix.
func ListKnowledgeChunksByRecordPrefix(db *gorm.DB, namespace string, groupID uint, prefix string) ([]KnowledgeChunk, error) {
	var rows []KnowledgeChunk
	err := db.Where("namespace = ? AND group_id = ? AND status = ? AND record_id LIKE ?",
		namespace, groupID, knconst.KnowledgeChunkStatusActive, prefix+"%").Find(&rows).Error
	return rows, err
}

// KnowledgeUnansweredQuestion tracks user questions the bot failed to answer.
type KnowledgeUnansweredQuestion struct {
	common.BaseModel

	GroupID           uint       `json:"groupId" gorm:"index;not null;default:0"`
	NamespaceID       uint       `json:"namespaceId,string" gorm:"index;not null;default:0"`
	Namespace         string     `json:"namespace" gorm:"type:varchar(128);index"`
	CallID            string     `json:"callId" gorm:"type:varchar(128);index"`
	AssistantID       uint       `json:"assistantId,string" gorm:"index;default:0"`
	Question          string     `json:"question" gorm:"type:text;not null"`
	TypicalQuestionID uint       `json:"typicalQuestionId,string" gorm:"index;default:0"`
	ClusterChunkID    uint       `json:"clusterChunkId,string" gorm:"index;default:0;comment:cluster slice in collect namespace"`
	OccurrenceCount   int        `json:"occurrenceCount" gorm:"not null;default:1"`
	Status            string     `json:"status" gorm:"type:varchar(20);index;not null;default:'open'"`
	ResolvedChunkID   uint       `json:"resolvedChunkId,string" gorm:"index;default:0"`
	ResolvedAt        *time.Time `json:"resolvedAt,omitempty"`
}

func (KnowledgeUnansweredQuestion) TableName() string {
	return constants.KNOWLEDGE_UNANSWERED_QUESTION_TABLE_NAME
}

// KnowledgeAnsweredQuestion records individual answered user questions.
type KnowledgeAnsweredQuestion struct {
	common.BaseModel

	GroupID           uint           `json:"groupId" gorm:"index;not null;default:0"`
	NamespaceID       uint           `json:"namespaceId,string" gorm:"index;not null;default:0"`
	Namespace         string         `json:"namespace" gorm:"type:varchar(128);index"`
	CallID            string         `json:"callId" gorm:"type:varchar(128);index"`
	AssistantID       uint           `json:"assistantId,string" gorm:"index;default:0"`
	Question          string         `json:"question" gorm:"type:text;not null"`
	TypicalQuestionID uint           `json:"typicalQuestionId,string" gorm:"index;default:0"`
	KnowledgeQuoted   bool           `json:"knowledgeQuoted" gorm:"not null;default:false"`
	QuoteChunkIDs     datatypes.JSON `json:"quoteChunkIds,omitempty" gorm:"type:json"`
	RetrievalHitCount int            `json:"retrievalHitCount" gorm:"not null;default:0"`
	MatchScore        float64        `json:"matchScore" gorm:"not null;default:0"`
}

func (KnowledgeAnsweredQuestion) TableName() string {
	return constants.KNOWLEDGE_ANSWERED_QUESTION_TABLE_NAME
}

// KnowledgeTypicalQuestion is a clustered canonical question for HF analytics.
type KnowledgeTypicalQuestion struct {
	common.BaseModel

	GroupID        uint   `json:"groupId" gorm:"index;not null;default:0"`
	NamespaceID    uint   `json:"namespaceId,string" gorm:"index;not null;default:0"`
	Namespace      string `json:"namespace" gorm:"type:varchar(128);index"`
	Question       string `json:"question" gorm:"type:text;not null"`
	ClusterChunkID uint   `json:"clusterChunkId,string" gorm:"index;default:0"`
	TotalCount     int    `json:"totalCount" gorm:"not null;default:0"`
	QuotedCount    int    `json:"quotedCount" gorm:"not null;default:0"`
}

func (KnowledgeTypicalQuestion) TableName() string {
	return constants.KNOWLEDGE_TYPICAL_QUESTION_TABLE_NAME
}

// KnowledgeTypicalQuestionStat is daily HF question stats.
type KnowledgeTypicalQuestionStat struct {
	common.BaseModel

	GroupID           uint      `json:"groupId" gorm:"index;not null;default:0"`
	NamespaceID       uint      `json:"namespaceId,string" gorm:"index;not null;default:0"`
	TypicalQuestionID uint      `json:"typicalQuestionId,string" gorm:"index;not null"`
	StatDate          time.Time `json:"statDate" gorm:"type:date;index;not null"`
	Count             int       `json:"count" gorm:"not null;default:0"`
	QuotedCount       int       `json:"quotedCount" gorm:"not null;default:0"`
}

func (KnowledgeTypicalQuestionStat) TableName() string {
	return constants.KNOWLEDGE_TYPICAL_QUESTION_STAT_TABLE
}

// KnowledgeSyncSource configures recurring URL/API ingestion.
type KnowledgeSyncSource struct {
	common.BaseModel

	GroupID         uint           `json:"groupId" gorm:"index;not null;default:0"`
	NamespaceID     uint           `json:"namespaceId,string" gorm:"index;not null;default:0"`
	Namespace       string         `json:"namespace" gorm:"type:varchar(128);index"`
	Name            string         `json:"name" gorm:"type:varchar(255);not null"`
	SourceType      string         `json:"sourceType" gorm:"type:varchar(32);index;not null;default:'url'"`
	SourceURL       string         `json:"sourceUrl" gorm:"type:text"`
	IntervalMinutes int            `json:"intervalMinutes" gorm:"not null;default:1440"`
	ChunkConfig     datatypes.JSON `json:"chunkConfig,omitempty" gorm:"type:json"`
	LastSyncAt      *time.Time     `json:"lastSyncAt,omitempty"`
	LastSyncError   string         `json:"lastSyncError,omitempty" gorm:"type:varchar(1024)"`
	DocumentID      uint           `json:"documentId,string" gorm:"index;default:0"`
	Status          string         `json:"status" gorm:"type:varchar(20);index;not null;default:'active'"`
}

func (KnowledgeSyncSource) TableName() string { return constants.KNOWLEDGE_SYNC_SOURCE_TABLE_NAME }

// KnowledgeEvalDataset stores labeled retrieval evaluation samples.
type KnowledgeEvalDataset struct {
	common.BaseModel

	GroupID     uint           `json:"groupId" gorm:"index;not null;default:0"`
	NamespaceID uint           `json:"namespaceId,string" gorm:"index;not null;default:0"`
	Namespace   string         `json:"namespace" gorm:"type:varchar(128);index"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	SamplesJSON datatypes.JSON `json:"samples" gorm:"type:longtext;not null"`
	SampleCount int            `json:"sampleCount" gorm:"not null;default:0"`
}

func (KnowledgeEvalDataset) TableName() string { return constants.KNOWLEDGE_EVAL_DATASET_TABLE_NAME }

// CountKnowledgeUnansweredByNamespace counts open unanswered questions.
func CountKnowledgeUnansweredByNamespace(db *gorm.DB, namespaceID, groupID uint, status string) (int64, error) {
	var n int64
	q := db.Model(&KnowledgeUnansweredQuestion{}).Where("namespace_id = ? AND group_id = ?", namespaceID, groupID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Count(&n).Error
	return n, err
}

// ListKnowledgeUnansweredPage lists unanswered questions with pagination.
func ListKnowledgeUnansweredPage(db *gorm.DB, namespaceID, groupID uint, status string, page, size int) ([]KnowledgeUnansweredQuestion, int64, error) {
	q := db.Model(&KnowledgeUnansweredQuestion{}).Where("namespace_id = ? AND group_id = ?", namespaceID, groupID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	return utils.FindPage[KnowledgeUnansweredQuestion](q, page, size, "occurrence_count DESC, id DESC", utils.MaxPageSize200)
}

// ListKnowledgeTypicalQuestionsPage lists HF typical questions.
func ListKnowledgeTypicalQuestionsPage(db *gorm.DB, namespaceID, groupID uint, page, size int) ([]KnowledgeTypicalQuestion, int64, error) {
	q := db.Model(&KnowledgeTypicalQuestion{}).Where("namespace_id = ? AND group_id = ?", namespaceID, groupID)
	return utils.FindPage[KnowledgeTypicalQuestion](q, page, size, "total_count DESC, id DESC", utils.MaxPageSize200)
}

// GetKnowledgeTypicalQuestionByID loads one typical question scoped to tenant.
func GetKnowledgeTypicalQuestionByID(db *gorm.DB, id, namespaceID, groupID uint) (KnowledgeTypicalQuestion, error) {
	var row KnowledgeTypicalQuestion
	err := db.Where("id = ? AND namespace_id = ? AND group_id = ?", id, namespaceID, groupID).First(&row).Error
	return row, err
}

// ListKnowledgeSyncSourcesByNamespace lists sync sources for a namespace.
func ListKnowledgeSyncSourcesByNamespace(db *gorm.DB, namespaceID, groupID uint) ([]KnowledgeSyncSource, error) {
	var rows []KnowledgeSyncSource
	err := db.Where("namespace_id = ? AND group_id = ?", namespaceID, groupID).Order("id DESC").Find(&rows).Error
	return rows, err
}

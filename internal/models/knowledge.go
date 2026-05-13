// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package models

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Knowledge namespace / document status constants.
const (
	KnowledgeStatusActive     = "active"
	KnowledgeStatusDeleted    = "deleted"
	KnowledgeStatusProcessing = "processing"
	KnowledgeStatusFailed     = "failed"

	// KnowledgeVectorProviderQdrant maps to a Qdrant collection.
	KnowledgeVectorProviderQdrant = "qdrant"
	// KnowledgeVectorProviderMilvus maps to a Milvus collection.
	KnowledgeVectorProviderMilvus = "milvus"

	// KnowledgeTextURLInline is written to knowledge_documents.text_url when markdown is
	// persisted in stored_markdown (LingStorage upload failed or not configured). It is not an HTTP URL.
	KnowledgeTextURLInline = "inline:db"
)

// IsKnowledgeInlineTextURL reports whether text_url points at DB-backed markdown.
func IsKnowledgeInlineTextURL(s string) bool {
	return strings.TrimSpace(s) == KnowledgeTextURLInline
}

// NormalizeVectorProvider canonicalises the vector backend name; defaults to Qdrant.
func NormalizeVectorProvider(s string) string {
	v := strings.TrimSpace(strings.ToLower(s))
	switch v {
	case "", KnowledgeVectorProviderQdrant:
		return KnowledgeVectorProviderQdrant
	case KnowledgeVectorProviderMilvus:
		return KnowledgeVectorProviderMilvus
	default:
		return v
	}
}

// KnowledgeNamespace 表示一个知识库（在向量后端中对应一个 collection）。
type KnowledgeNamespace struct {
	ID        int64 `json:"id,string" gorm:"primaryKey;autoIncrement"`
	GroupID   uint  `json:"groupId" gorm:"uniqueIndex:idx_kn_group_ns;not null;default:0;comment:tenant group id"`
	CreatedBy uint  `json:"createdBy" gorm:"index;comment:user who created the namespace"`

	Namespace      string `json:"namespace" gorm:"type:varchar(128);uniqueIndex:idx_kn_group_ns;not null;comment:vector backend collection name"`
	Name           string `json:"name" gorm:"type:varchar(255);not null;comment:display name"`
	Description    string `json:"description" gorm:"type:text"`
	VectorProvider string `json:"vectorProvider" gorm:"type:varchar(32);not null;default:'qdrant';index;comment:qdrant|milvus"`
	EmbedModel     string `json:"embedModel" gorm:"type:varchar(64);not null;comment:embedding model"`
	VectorDim      int    `json:"vectorDim" gorm:"not null;comment:embedding dim"`
	Status         string `json:"status" gorm:"type:varchar(20);index;not null;default:'active'"`

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (KnowledgeNamespace) TableName() string { return "knowledge_namespaces" }

func (m *KnowledgeNamespace) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(m.VectorProvider) == "" {
		m.VectorProvider = KnowledgeVectorProviderQdrant
	}
	return nil
}

// KnowledgeNamespaceListResult is the paged list response payload.
type KnowledgeNamespaceListResult struct {
	List      []KnowledgeNamespace `json:"list"`
	Total     int64                `json:"total"`
	Page      int                  `json:"page"`
	PageSize  int                  `json:"pageSize"`
	TotalPage int                  `json:"totalPage"`
}

// ListKnowledgeNamespaces returns all namespaces a user can see, scoped to groupIDs.
// keyword optional: matches name or namespace (SQL LIKE).
func ListKnowledgeNamespaces(db *gorm.DB, groupIDs []uint, status, keyword string, page, pageSize int) (*KnowledgeNamespaceListResult, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	q := db.Model(&KnowledgeNamespace{})
	if len(groupIDs) == 0 {
		return &KnowledgeNamespaceListResult{Page: page, PageSize: pageSize}, nil
	}
	q = q.Where("group_id IN ?", groupIDs)
	if s := strings.TrimSpace(status); s != "" {
		q = q.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(name LIKE ? OR namespace LIKE ?)", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []KnowledgeNamespace
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	totalPage := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPage++
	}
	return &KnowledgeNamespaceListResult{
		List:      list,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}, nil
}

// GetKnowledgeNamespace fetches a namespace by id (no group scope).
func GetKnowledgeNamespace(db *gorm.DB, id int64) (*KnowledgeNamespace, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var row KnowledgeNamespace
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetKnowledgeNamespaceByGroupAndNamespace 通过 (groupID, namespace) 查找。
func GetKnowledgeNamespaceByGroupAndNamespace(db *gorm.DB, groupID uint, namespace string) (*KnowledgeNamespace, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return nil, errors.New("namespace is required")
	}
	var row KnowledgeNamespace
	if err := db.Where("group_id = ? AND namespace = ?", groupID, ns).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// KnowledgeNamespaceCreateUpdate 创建/更新参数。
type KnowledgeNamespaceCreateUpdate struct {
	Namespace      string
	Name           string
	Description    string
	VectorProvider string
	EmbedModel     string
	VectorDim      int
	Status         string
}

// UpsertKnowledgeNamespace creates or updates a namespace within a group.
// id==0 means "create or upsert by (group_id, namespace)".
func UpsertKnowledgeNamespace(db *gorm.DB, groupID uint, createdBy uint, id int64, req *KnowledgeNamespaceCreateUpdate) (*KnowledgeNamespace, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if req == nil {
		return nil, errors.New("nil req")
	}
	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	vp := NormalizeVectorProvider(req.VectorProvider)
	if vp != KnowledgeVectorProviderQdrant && vp != KnowledgeVectorProviderMilvus {
		return nil, errors.New("vector_provider must be qdrant or milvus")
	}
	embedModel := strings.TrimSpace(req.EmbedModel)
	if embedModel == "" {
		return nil, errors.New("embed_model is required")
	}
	if req.VectorDim <= 0 {
		return nil, errors.New("vector_dim must be > 0")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = KnowledgeStatusActive
	}

	if id == 0 {
		var existing KnowledgeNamespace
		err := db.Where("group_id = ? AND namespace = ?", groupID, namespace).First(&existing).Error
		if err == nil {
			existing.Name = name
			existing.Description = strings.TrimSpace(req.Description)
			existing.VectorProvider = vp
			existing.EmbedModel = embedModel
			existing.VectorDim = req.VectorDim
			existing.Status = status
			if err := db.Save(&existing).Error; err != nil {
				return nil, err
			}
			return &existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		row := KnowledgeNamespace{
			GroupID:        groupID,
			CreatedBy:      createdBy,
			Namespace:      namespace,
			Name:           name,
			Description:    strings.TrimSpace(req.Description),
			VectorProvider: vp,
			EmbedModel:     embedModel,
			VectorDim:      req.VectorDim,
			Status:         status,
		}
		if err := db.Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}

	var row KnowledgeNamespace
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Namespace = namespace
	row.Name = name
	row.Description = strings.TrimSpace(req.Description)
	row.VectorProvider = vp
	row.EmbedModel = embedModel
	row.VectorDim = req.VectorDim
	row.Status = status
	if err := db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// SoftDeleteKnowledgeNamespace marks a namespace as deleted.
func SoftDeleteKnowledgeNamespace(db *gorm.DB, id int64) error {
	if db == nil {
		return errors.New("nil db")
	}
	var row KnowledgeNamespace
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	row.Status = KnowledgeStatusDeleted
	return db.Save(&row).Error
}

// KnowledgeDocument 表示用户上传到某个知识库的一份材料，对应一批向量 records。
type KnowledgeDocument struct {
	ID        int64 `json:"id,string" gorm:"primaryKey;autoIncrement"`
	GroupID   uint  `json:"groupId" gorm:"uniqueIndex:idx_kd_group_ns_filehash;not null;default:0;comment:tenant group id"`
	CreatedBy uint  `json:"createdBy" gorm:"index"`

	Namespace string `json:"namespace" gorm:"type:varchar(128);index;not null;uniqueIndex:idx_kd_group_ns_filehash"`

	Title    string `json:"title" gorm:"type:varchar(255);not null"`
	Source   string `json:"source" gorm:"type:varchar(128);comment:upload|url|api|..."`
	FileHash string `json:"fileHash" gorm:"type:varchar(64);index;not null;uniqueIndex:idx_kd_group_ns_filehash"`

	TextURL   string `json:"textUrl,omitempty" gorm:"type:text;comment:markdown text URL in object storage"`
	// StoredMarkdown holds full markdown when LingStorage upload failed or URL is unavailable (GET /text fallback).
	StoredMarkdown string `json:"storedMarkdown,omitempty" gorm:"type:longtext;comment:fallback markdown when text_url empty or fetch fails"`
	RecordIDs      string `json:"recordIds" gorm:"type:text;comment:related vector ids (csv or json)"`
	Status    string `json:"status" gorm:"type:varchar(20);index;not null;default:'active'"`

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"autoUpdateTime"`
}

func (KnowledgeDocument) TableName() string { return "knowledge_documents" }

// KnowledgeDocumentListResult paged list response.
type KnowledgeDocumentListResult struct {
	List      []KnowledgeDocument `json:"list"`
	Total     int64               `json:"total"`
	Page      int                 `json:"page"`
	PageSize  int                 `json:"pageSize"`
	TotalPage int                 `json:"totalPage"`
}

// ListKnowledgeDocuments lists docs scoped by groupIDs (+ optional namespace / status).
// keyword optional: matches title or file_hash (SQL LIKE).
// When excludeDeleted is true, rows with status "deleted" are never returned (non-admin web UI).
func ListKnowledgeDocuments(db *gorm.DB, groupIDs []uint, namespace, status, keyword string, page, pageSize int, excludeDeleted bool) (*KnowledgeDocumentListResult, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if len(groupIDs) == 0 {
		return &KnowledgeDocumentListResult{Page: page, PageSize: pageSize}, nil
	}
	q := db.Model(&KnowledgeDocument{}).Where("group_id IN ?", groupIDs)
	if excludeDeleted {
		q = q.Where("status <> ?", KnowledgeStatusDeleted)
	}
	if ns := strings.TrimSpace(namespace); ns != "" {
		q = q.Where("namespace = ?", ns)
	}
	if s := strings.TrimSpace(status); s != "" {
		q = q.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(title LIKE ? OR file_hash LIKE ?)", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var list []KnowledgeDocument
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}
	totalPage := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPage++
	}
	return &KnowledgeDocumentListResult{
		List:      list,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}, nil
}

// GetKnowledgeDocument fetches one document by id (no group scope).
func GetKnowledgeDocument(db *gorm.DB, id int64) (*KnowledgeDocument, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	var row KnowledgeDocument
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// KnowledgeDocumentUpsertReq 创建/更新文档参数。
type KnowledgeDocumentUpsertReq struct {
	Namespace string
	Title     string
	Source    string
	FileHash  string
	RecordIDs string
	TextURL   string
	Status    string
}

// UpsertKnowledgeDocument creates or updates a document; dedup by (group_id, namespace, file_hash).
func UpsertKnowledgeDocument(db *gorm.DB, groupID uint, createdBy uint, id int64, req *KnowledgeDocumentUpsertReq) (*KnowledgeDocument, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	if req == nil {
		return nil, errors.New("nil req")
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		return nil, errors.New("namespace is required")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, errors.New("title is required")
	}
	fileHash := strings.TrimSpace(req.FileHash)
	if fileHash == "" {
		return nil, errors.New("file_hash is required")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = KnowledgeStatusActive
	}

	if id == 0 {
		var existing KnowledgeDocument
		err := db.Where("group_id = ? AND namespace = ? AND file_hash = ?", groupID, ns, fileHash).First(&existing).Error
		if err == nil {
			existing.Title = title
			existing.Source = strings.TrimSpace(req.Source)
			existing.RecordIDs = strings.TrimSpace(req.RecordIDs)
			existing.TextURL = strings.TrimSpace(req.TextURL)
			existing.Status = status
			if err := db.Save(&existing).Error; err != nil {
				return nil, err
			}
			return &existing, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		row := KnowledgeDocument{
			GroupID:   groupID,
			CreatedBy: createdBy,
			Namespace: ns,
			Title:     title,
			Source:    strings.TrimSpace(req.Source),
			FileHash:  fileHash,
			RecordIDs: strings.TrimSpace(req.RecordIDs),
			TextURL:   strings.TrimSpace(req.TextURL),
			Status:    status,
		}
		if err := db.Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}

	var row KnowledgeDocument
	if err := db.First(&row, id).Error; err != nil {
		return nil, err
	}
	row.Namespace = ns
	row.Title = title
	row.Source = strings.TrimSpace(req.Source)
	row.FileHash = fileHash
	row.RecordIDs = strings.TrimSpace(req.RecordIDs)
	row.TextURL = strings.TrimSpace(req.TextURL)
	row.Status = status
	if err := db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// SoftDeleteKnowledgeDocument marks a document as deleted.
func SoftDeleteKnowledgeDocument(db *gorm.DB, id int64) error {
	if db == nil {
		return errors.New("nil db")
	}
	var row KnowledgeDocument
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	row.Status = KnowledgeStatusDeleted
	return db.Save(&row).Error
}

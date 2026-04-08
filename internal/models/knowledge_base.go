package models

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// KnowledgeBase 通用知识库实体（用于配置第三方知识库连接与索引参数）
type KnowledgeBase struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	CreatedAt   time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	UserID      uint           `json:"userId" gorm:"index;not null"`
	GroupID     *uint          `json:"groupId,omitempty" gorm:"index"`
	Name        string         `json:"name" gorm:"size:128;index;not null"`
	Description string         `json:"description,omitempty" gorm:"size:512"`
	Provider    string         `json:"provider" gorm:"size:64;index;not null"` // qdrant, aliyun, pinecone...
	EndpointURL string         `json:"endpointUrl,omitempty" gorm:"size:512"`
	APIKey      string         `json:"apiKey,omitempty" gorm:"size:512"`       // 预留：后续可改为加密存储
	APISecret   string         `json:"apiSecret,omitempty" gorm:"size:512"`    // 预留：后续可改为加密存储
	IndexName   string         `json:"indexName,omitempty" gorm:"size:256"`    // 集合/索引名
	Namespace   string         `json:"namespace,omitempty" gorm:"size:256"`    // 命名空间
	ExtraConfig datatypes.JSON `json:"extraConfig,omitempty" gorm:"type:json"` // 提供商扩展参数
	IsActive    bool           `json:"isActive" gorm:"default:true;index"`
}

func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}

func CreateKnowledgeBase(db *gorm.DB, kb *KnowledgeBase) error {
	return db.Create(kb).Error
}

func UpdateKnowledgeBase(db *gorm.DB, id uint, userID uint, updates map[string]interface{}) error {
	return db.Model(&KnowledgeBase{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

func DeleteKnowledgeBase(db *gorm.DB, id uint, userID uint) error {
	return db.Where("id = ? AND user_id = ?", id, userID).Delete(&KnowledgeBase{}).Error
}

func GetKnowledgeBaseByID(db *gorm.DB, id uint, userID uint) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&kb).Error; err != nil {
		return nil, err
	}
	return &kb, nil
}

func ListKnowledgeBasesByUser(db *gorm.DB, userID uint) ([]KnowledgeBase, error) {
	var list []KnowledgeBase
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}


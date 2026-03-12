package service

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"gorm.io/gorm"
)

// MCPMarketplaceService MCP 广场服务
type MCPMarketplaceService struct {
	db *gorm.DB
}

// NewMCPMarketplaceService 创建 MCP 广场服务
func NewMCPMarketplaceService(db *gorm.DB) *MCPMarketplaceService {
	return &MCPMarketplaceService{db: db}
}

// ListMarketplaceItems 列出广场项目
func (s *MCPMarketplaceService) ListMarketplaceItems(ctx context.Context, req *models.SearchMCPRequest) (*models.MCPMarketplaceResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := s.db.WithContext(ctx).Where("status = ?", "published")

	// 搜索
	if req.Query != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+req.Query+"%", "%"+req.Query+"%")
	}

	// 分类过滤
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}

	// 标签过滤
	if len(req.Tags) > 0 {
		for _, tag := range req.Tags {
			query = query.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf(`"%s"`, tag))
		}
	}

	// 排序
	sortBy := "created_at"
	if req.SortBy == "downloads" {
		sortBy = "downloads"
	} else if req.SortBy == "rating" {
		sortBy = "rating"
	}

	order := "DESC"
	if req.Order == "asc" {
		order = "ASC"
	}

	query = query.Order(fmt.Sprintf("%s %s", sortBy, order))

	// 计算总数
	var total int64
	if err := query.Model(&models.MCPMarketplaceItem{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	offset := (req.Page - 1) * req.PageSize
	var items []models.MCPMarketplaceItem
	if err := query.Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	// 获取分类
	var categories []models.MCPCategory
	s.db.WithContext(ctx).Order("`order` ASC").Find(&categories)

	return &models.MCPMarketplaceResponse{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		Categories: categories,
	}, nil
}

// GetMarketplaceItem 获取广场项目详情
func (s *MCPMarketplaceService) GetMarketplaceItem(ctx context.Context, id uint) (*models.MCPMarketplaceItem, error) {
	var item models.MCPMarketplaceItem
	if err := s.db.WithContext(ctx).Where("id = ? AND status = ?", id, "published").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// InstallMCP 安装 MCP
func (s *MCPMarketplaceService) InstallMCP(ctx context.Context, userID uint, req *models.InstallMCPRequest) (*models.MCPUserInstallation, error) {
	// 检查是否已安装
	var existing models.MCPUserInstallation
	if err := s.db.WithContext(ctx).Where("user_id = ? AND marketplace_id = ?", userID, req.MarketplaceID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("MCP already installed")
	}

	// 获取广场项目
	item, err := s.GetMarketplaceItem(ctx, req.MarketplaceID)
	if err != nil {
		return nil, fmt.Errorf("marketplace item not found")
	}

	// 转换配置为 JSON
	var configJSON []byte
	if len(req.Config) > 0 {
		configJSON, _ = json.Marshal(req.Config)
	}

	// 创建安装记录
	installation := &models.MCPUserInstallation{
		UserID:           userID,
		MarketplaceID:    req.MarketplaceID,
		InstalledVersion: item.Version,
		Config:           configJSON,
		Enabled:          true,
	}

	if err := s.db.WithContext(ctx).Create(installation).Error; err != nil {
		return nil, err
	}

	// 更新下载次数
	s.db.WithContext(ctx).Model(&models.MCPMarketplaceItem{}).Where("id = ?", req.MarketplaceID).Update("downloads", gorm.Expr("downloads + ?", 1))

	return installation, nil
}

// UninstallMCP 卸载 MCP
func (s *MCPMarketplaceService) UninstallMCP(ctx context.Context, userID uint, installationID uint) error {
	return s.db.WithContext(ctx).Where("id = ? AND user_id = ?", installationID, userID).Delete(&models.MCPUserInstallation{}).Error
}

// GetUserInstalledMCPs 获取用户已安装的 MCP
func (s *MCPMarketplaceService) GetUserInstalledMCPs(ctx context.Context, userID uint) ([]models.MCPUserInstallation, error) {
	var installations []models.MCPUserInstallation
	if err := s.db.WithContext(ctx).
		Preload("Marketplace").
		Where("user_id = ?", userID).
		Find(&installations).Error; err != nil {
		return nil, err
	}
	return installations, nil
}

// UpdateInstallationConfig 更新安装配置
func (s *MCPMarketplaceService) UpdateInstallationConfig(ctx context.Context, userID uint, installationID uint, config map[string]interface{}) error {
	configJSON, _ := json.Marshal(config)
	return s.db.WithContext(ctx).
		Model(&models.MCPUserInstallation{}).
		Where("id = ? AND user_id = ?", installationID, userID).
		Update("config", configJSON).Error
}

// ReviewMCP 评论 MCP
func (s *MCPMarketplaceService) ReviewMCP(ctx context.Context, userID uint, marketplaceID uint, req *models.ReviewMCPRequest) (*models.MCPReview, error) {
	// 检查是否已评论
	var existing models.MCPReview
	if err := s.db.WithContext(ctx).Where("user_id = ? AND marketplace_id = ?", userID, marketplaceID).First(&existing).Error; err == nil {
		// 更新现有评论
		existing.Rating = req.Rating
		existing.Title = req.Title
		existing.Content = req.Content
		if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	// 创建新评论
	review := &models.MCPReview{
		UserID:        userID,
		MarketplaceID: marketplaceID,
		Rating:        req.Rating,
		Title:         req.Title,
		Content:       req.Content,
	}

	if err := s.db.WithContext(ctx).Create(review).Error; err != nil {
		return nil, err
	}

	// 更新 MCP 的平均评分
	s.updateMCPRating(ctx, marketplaceID)

	return review, nil
}

// GetMCPReviews 获取 MCP 评论
func (s *MCPMarketplaceService) GetMCPReviews(ctx context.Context, marketplaceID uint, page, pageSize int) ([]models.MCPReview, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var reviews []models.MCPReview
	var total int64

	query := s.db.WithContext(ctx).Where("marketplace_id = ?", marketplaceID)
	if err := query.Model(&models.MCPReview{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reviews).Error; err != nil {
		return nil, 0, err
	}

	return reviews, total, nil
}

// GetCategories 获取分类列表
func (s *MCPMarketplaceService) GetCategories(ctx context.Context) ([]models.MCPCategory, error) {
	var categories []models.MCPCategory
	if err := s.db.WithContext(ctx).Order("`order` ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// updateMCPRating 更新 MCP 评分
func (s *MCPMarketplaceService) updateMCPRating(ctx context.Context, marketplaceID uint) {
	var reviews []models.MCPReview
	s.db.WithContext(ctx).Where("marketplace_id = ?", marketplaceID).Find(&reviews)

	if len(reviews) == 0 {
		return
	}

	var totalRating float64
	for _, review := range reviews {
		totalRating += float64(review.Rating)
	}

	avgRating := totalRating / float64(len(reviews))

	s.db.WithContext(ctx).Model(&models.MCPMarketplaceItem{}).
		Where("id = ?", marketplaceID).
		Updates(map[string]interface{}{
			"rating":  avgRating,
			"reviews": len(reviews),
		})
}

// GetFeaturedMCPs 获取推荐的 MCP
func (s *MCPMarketplaceService) GetFeaturedMCPs(ctx context.Context, limit int) ([]models.MCPMarketplaceItem, error) {
	var items []models.MCPMarketplaceItem
	if err := s.db.WithContext(ctx).
		Where("status = ? AND is_featured = ?", "published", true).
		Order("rating DESC, downloads DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetTrendingMCPs 获取热门 MCP
func (s *MCPMarketplaceService) GetTrendingMCPs(ctx context.Context, limit int) ([]models.MCPMarketplaceItem, error) {
	var items []models.MCPMarketplaceItem
	if err := s.db.WithContext(ctx).
		Where("status = ?", "published").
		Order("downloads DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// SearchByTag 按标签搜索
func (s *MCPMarketplaceService) SearchByTag(ctx context.Context, tag string, limit int) ([]models.MCPMarketplaceItem, error) {
	var items []models.MCPMarketplaceItem
	if err := s.db.WithContext(ctx).
		Where("status = ? AND JSON_CONTAINS(tags, ?)", "published", fmt.Sprintf(`"%s"`, tag)).
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

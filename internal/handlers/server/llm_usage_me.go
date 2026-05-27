// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 用户自助 LLM 用量查询（/api/me/llm-usage）：仅返回当前用户、开放平台 Relay 产生的记录。
package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

func meLLMUsageUserID(u *auth.User) string {
	if u == nil {
		return ""
	}
	return fmt.Sprintf("%d", u.ID)
}

// handleMeListLLMUsage GET /api/me/llm-usage
func (h *Handlers) handleMeListLLMUsage(c *gin.Context) {
	u := auth.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	page, pageSize := utils.ParsePagination(c)
	uid := meLLMUsageUserID(u)

	q := h.db.Model(&svcmodels.LLMUsage{}).
		Where("user_id = ?", uid).
		Where("request_type LIKE ?", "openapi_%")

	if model := strings.TrimSpace(c.Query("model")); model != "" {
		q = q.Where("model = ?", model)
	}
	if success := strings.TrimSpace(c.Query("success")); success != "" {
		if parsed, err := parseBoolQuery(success); err == nil {
			q = q.Where("success = ?", parsed)
		}
	}
	if from := strings.TrimSpace(c.Query("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("requested_at >= ?", t)
		}
	}
	if to := strings.TrimSpace(c.Query("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("requested_at <= ?", t)
		}
	}
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		q = q.Where("request_id LIKE ? OR model LIKE ? OR provider LIKE ? OR request_type LIKE ?", like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list llm usage failed", err)
		return
	}

	var items []svcmodels.LLMUsage
	if err := q.Order("requested_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error; err != nil {
		response.Fail(c, "list llm usage failed", err)
		return
	}

	response.Success(c, "llm usage fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

type meLLMUsageSummaryRow struct {
	TotalRequests   int64 `json:"total_requests"`
	SuccessRequests int64 `json:"success_requests"`
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
	QuotaDelta      int64 `json:"quota_delta"`
}

// handleMeLLMUsageSummary GET /api/me/llm-usage/summary
func (h *Handlers) handleMeLLMUsageSummary(c *gin.Context) {
	u := auth.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	uid := meLLMUsageUserID(u)

	q := h.db.Model(&svcmodels.LLMUsage{}).
		Where("user_id = ?", uid).
		Where("request_type LIKE ?", "openapi_%")

	if from := strings.TrimSpace(c.Query("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("requested_at >= ?", t)
		}
	}
	if to := strings.TrimSpace(c.Query("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("requested_at <= ?", t)
		}
	}

	var row meLLMUsageSummaryRow
	if err := q.Select(`
		COUNT(*) AS total_requests,
		SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS success_requests,
		COALESCE(SUM(input_tokens), 0) AS input_tokens,
		COALESCE(SUM(output_tokens), 0) AS output_tokens,
		COALESCE(SUM(total_tokens), 0) AS total_tokens,
		COALESCE(SUM(quota_delta), 0) AS quota_delta
	`).Scan(&row).Error; err != nil {
		response.Fail(c, "llm usage summary failed", err)
		return
	}

	// 按模型 Top N
	type modelRow struct {
		Model        string `json:"model"`
		RequestCount int64  `json:"request_count"`
		TotalTokens  int64  `json:"total_tokens"`
	}
	var byModel []modelRow
	_ = h.db.Model(&svcmodels.LLMUsage{}).
		Select("model, COUNT(*) AS request_count, COALESCE(SUM(total_tokens), 0) AS total_tokens").
		Where("user_id = ? AND request_type LIKE ?", uid, "openapi_%").
		Group("model").
		Order("total_tokens DESC").
		Limit(10).
		Scan(&byModel).Error

	response.Success(c, "llm usage summary", gin.H{
		"summary":  row,
		"by_model": byModel,
	})
}

func parseBoolQuery(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool")
	}
}

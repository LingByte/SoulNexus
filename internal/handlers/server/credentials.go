package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// UpdateRateLimiterConfig updates rate limiter configuration
func (h *Handlers) handleCreateCredential(c *gin.Context) {
	var credential auth.UserCredentialRequest
	if err := c.ShouldBindJSON(&credential); err != nil {
		response.Fail(c, "Invalid request", nil)
		return
	}

	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	userCredential, err := h.rpc.Auth.CreateUserCredential(c.Request.Context(), user.ID, &credential)
	if err != nil {
		response.Fail(c, "create user credential failed", err)
		return
	}

	response.Success(c, "create user credential success", gin.H{
		"apiKey":    userCredential.APIKey,
		"apiSecret": userCredential.APISecret,
		"name":      credential.Name,
	})
}

func (h *Handlers) handleGetCredential(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}
	credentials, err := h.rpc.Auth.ListUserCredentials(c.Request.Context(), user.ID)
	if err != nil {
		response.Fail(c, "get user credentials failed", err)
		return
	}
	// 转换为响应格式，不包含敏感信息
	response.Success(c, "get user credentials success", auth.ToResponseList(credentials))
}

// handleGetCredentialByKey 根据 apikey 和 apisecret 获取凭证信息（用于 controlpanel 调用）
func (h *Handlers) handleGetCredentialByKey(c *gin.Context) {
	var req struct {
		APIKey    string `json:"apiKey" binding:"required"`
		APISecret string `json:"apiSecret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "apiKey and apiSecret are required", nil)
		return
	}

	credential, err := h.resolveCredential(c.Request.Context(), req.APIKey, req.APISecret)
	if err != nil {
		response.Fail(c, "invalid credentials", err)
		return
	}

	response.Success(c, "success", gin.H{
		"llmProvider": credential.LLMProvider,
		"llmApiKey":   credential.LLMApiKey,
		"llmApiUrl":   credential.LLMApiURL,
		"asrProvider": credential.GetASRProvider(),
		"ttsProvider": credential.GetTTSProvider(),
	})
}

// handleDeleteCredential 删除用户凭证
func (h *Handlers) handleDeleteCredential(c *gin.Context) {
	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
		return
	}

	// Get credential ID from path parameter
	idStr := c.Param("id")
	var credentialID uint
	_, err := fmt.Sscanf(idStr, "%d", &credentialID)
	if err != nil {
		response.Fail(c, "Invalid credential ID", err)
		return
	}

	// Delete credential
	err = h.rpc.Auth.DeleteUserCredentialForUser(c.Request.Context(), user.ID, credentialID)
	if err != nil {
		response.Fail(c, "Failed to delete credential", err)
		return
	}

	response.Success(c, "Credential deleted successfully", nil)
}

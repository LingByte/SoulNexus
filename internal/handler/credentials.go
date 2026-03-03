package handlers

import (
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// UpdateRateLimiterConfig updates rate limiter configuration
func (h *Handlers) handleCreateCredential(c *gin.Context) {
	var credential models.UserCredentialRequest
	if err := c.ShouldBindJSON(&credential); err != nil {
		response.Fail(c, "Invalid request", nil)
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User is not logged in.", nil)
	}

	userCredential, err := models.CreateUserCredential(h.db, user.ID, &credential)
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
	user := models.CurrentUser(c)
	credentials, err := models.GetUserCredentials(h.db, user.ID)
	if err != nil {
		response.Fail(c, "get user credentials failed", err)
		return
	}
	// 转换为响应格式，不包含敏感信息
	response.Success(c, "get user credentials success", models.ToResponseList(credentials))
}

// handleDeleteCredential 删除用户凭证
func (h *Handlers) handleDeleteCredential(c *gin.Context) {
	user := models.CurrentUser(c)
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
	err = models.DeleteUserCredential(h.db, user.ID, credentialID)
	if err != nil {
		response.Fail(c, "Failed to delete credential", err)
		return
	}

	response.Success(c, "Credential deleted successfully", nil)
}

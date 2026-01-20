package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UploadHandler file upload handler
type UploadHandler struct{}

// NewUploadHandler creates a new upload handler
func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// Register registers routes
func (h *UploadHandler) Register(r *gin.Engine) {
	// Audio file upload route
	r.POST("/api/upload/audio", h.UploadAudio)
}

// UploadAudio uploads audio file
func (h *UploadHandler) UploadAudio(c *gin.Context) {
	// Get uploaded file
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		response.Fail(c, "Failed to get uploaded file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// Check file type
	contentType := header.Header.Get("Content-Type")
	if contentType != "audio/webm" && contentType != "audio/wav" && contentType != "audio/mp3" {
		response.Fail(c, "Unsupported file type: "+contentType, nil)
		return
	}

	// Generate storage key (relative to storage root)
	timestamp := time.Now().Unix()
	randomStr := utils.RandString(8)
	fileName := fmt.Sprintf("audio_%d_%s.webm", timestamp, randomStr)
	storageKey := fmt.Sprintf("audio/%s", fileName)
	reader, err := config.GlobalStore.UploadFromReader(&lingstorage.UploadFromReaderRequest{
		Reader:   file,
		Bucket:   config.GlobalConfig.LingstorageBucket,
		Filename: storageKey,
		Key:      storageKey,
	})
	if err != nil {
		response.Fail(c, "Failed to upload file: "+err.Error(), nil)
		return
	}

	// Record storage usage
	user := models.CurrentUser(c)
	if user != nil {
		// 从middleware获取数据库连接
		db, exists := c.Get("db")
		if exists {
			if gormDB, ok := db.(*gorm.DB); ok {
				// Try to get credential ID (from request parameters or user's default credential)
				var credentialID uint
				if credIDStr := c.Query("credentialId"); credIDStr != "" {
					if id, err := strconv.ParseUint(credIDStr, 10, 32); err == nil {
						credentialID = uint(id)
					}
				}
				// 如果没有提供凭证ID，尝试获取用户的第一个凭证
				if credentialID == 0 {
					credentials, err := models.GetUserCredentials(gormDB, user.ID)
					if err == nil && len(credentials) > 0 {
						credentialID = credentials[0].ID
					}
				}
			}
		}
	}

	// Return success response
	response.Success(c, "音s频文件上传成功", map[string]interface{}{
		"fileName":   fileName,
		"filePath":   reader.URL,
		"fileSize":   reader.Size,
		"uploadTime": time.Now().Format(time.RFC3339),
		"url":        reader.URL,
	})
}

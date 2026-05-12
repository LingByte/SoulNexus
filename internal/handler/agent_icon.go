// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Agent icon 上传：multipart/form-data，字段名 icon。
// 走 LingStorage 默认 bucket，返回可直接落到 agents.icon 字段的 URL。

package handlers

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gin-gonic/gin"
)

const agentIconMaxSize = int64(5 * 1024 * 1024) // 5 MB

var agentIconAllowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"image/svg+xml": true,
}

var agentIconExtToType = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
}

// UploadAgentIcon POST /api/agents/icon/upload
// 表单字段：icon (file, required)
// 返回：{ url: "..." }
func (h *Handlers) UploadAgentIcon(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "User not found", errors.New("user not found"))
		return
	}
	if config.GlobalStore == nil {
		response.Fail(c, "Storage not configured", errors.New("LINGSTORAGE_* not initialized"))
		return
	}

	file, header, err := c.Request.FormFile("icon")
	if err != nil {
		response.Fail(c, "Failed to get uploaded file", err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if mapped, ok := agentIconExtToType[ext]; ok {
			contentType = mapped
		}
	}
	if !agentIconAllowedTypes[contentType] {
		response.Fail(c, "Invalid file type", errors.New("only jpeg/png/gif/webp/svg are allowed"))
		return
	}
	if header.Size > agentIconMaxSize {
		response.Fail(c, "File too large", errors.New("file size must be less than 5MB"))
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".png"
	}
	bucket := ""
	if config.GlobalConfig != nil {
		bucket = config.GlobalConfig.Services.Storage.Bucket
	}
	key := fmt.Sprintf("agents/icons/%d_%d%s", user.ID, time.Now().UnixNano(), ext)

	res, err := config.GlobalStore.UploadFromReader(&lingstorage.UploadFromReaderRequest{
		Reader:   file,
		Bucket:   bucket,
		Filename: filepath.Base(key),
		Key:      key,
	})
	if err != nil {
		response.Fail(c, "Failed to upload icon", err)
		return
	}

	url := res.URL
	if strings.HasPrefix(url, "/") {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		if host == "" {
			host = "localhost:7072"
		}
		url = fmt.Sprintf("%s://%s%s", scheme, host, url)
	}
	response.Success(c, "uploaded", gin.H{"url": url, "key": res.Key})
}

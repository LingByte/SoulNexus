package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voiceprint"
)

// GetVoiceprints 获取声纹列表
func (h *Handlers) GetVoiceprints(c *gin.Context) {
	assistantID := c.Query("agent_id")
	if assistantID == "" {
		response.Fail(c, "agent_id is required", nil)
		return
	}

	var voiceprints []models.Voiceprint
	result := h.db.Where("agent_id = ?", assistantID).Find(&voiceprints)
	if result.Error != nil {
		response.Fail(c, "Failed to get voiceprints: "+result.Error.Error(), nil)
		return
	}

	// 转换为响应格式
	voiceprintResponses := make([]models.VoiceprintResponse, len(voiceprints))
	for i, vp := range voiceprints {
		voiceprintResponses[i] = models.VoiceprintResponse{
			ID:          vp.ID,
			SpeakerID:   vp.SpeakerID,
			AgentID:     vp.AgentID,
			SpeakerName: vp.SpeakerName,
			Description: vp.Description,
			CreatedAt:   vp.CreatedAt,
			UpdatedAt:   vp.UpdatedAt,
		}
	}

	response.Success(c, "Success", models.VoiceprintListResponse{
		Total:       len(voiceprintResponses),
		Voiceprints: voiceprintResponses,
	})
}

// CreateVoiceprint 创建声纹记录
func (h *Handlers) CreateVoiceprint(c *gin.Context) {
	var req models.VoiceprintCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request: "+err.Error(), nil)
		return
	}

	// 检查是否已存在相同的speaker_id和agent_id组合
	var existingVoiceprint models.Voiceprint
	result := h.db.Where("speaker_id = ? AND agent_id = ?", req.SpeakerID, req.AgentID).First(&existingVoiceprint)
	if result.Error == nil {
		response.Fail(c, "Voiceprint already exists for this speaker and assistant", nil)
		return
	}

	// 创建声纹记录
	voiceprint := models.Voiceprint{
		SpeakerID: req.SpeakerID,
		AgentID:   req.AgentID,
		SpeakerName:   req.SpeakerName,
		Description:   req.Description,
		FeatureVector: []byte{}, // 初始为空，等待音频上传后更新
	}

	if err := h.db.Create(&voiceprint).Error; err != nil {
		response.Fail(c, "Failed to create voiceprint: "+err.Error(), nil)
		return
	}

	response.Success(c, "Voiceprint created successfully", models.VoiceprintResponse{
		ID:        voiceprint.ID,
		SpeakerID: voiceprint.SpeakerID,
		AgentID:   voiceprint.AgentID,
		SpeakerName: voiceprint.SpeakerName,
		Description: voiceprint.Description,
		CreatedAt:   voiceprint.CreatedAt,
		UpdatedAt:   voiceprint.UpdatedAt,
	})
}

// RegisterVoiceprint 注册声纹（上传音频并调用voiceprint-api）
func (h *Handlers) RegisterVoiceprint(c *gin.Context) {
	assistantID := c.PostForm("agent_id")
	speakerName := c.PostForm("speaker_name")
	description := c.PostForm("description")

	if assistantID == "" || speakerName == "" {
		response.Fail(c, "agent_id and speaker_name are required", nil)
		return
	}

	// 获取上传的音频文件
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		response.Fail(c, "Failed to get audio file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// 验证文件格式
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".wav") {
		response.Fail(c, "Only WAV format is supported", nil)
		return
	}

	// 读取音频数据
	audioData, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, "Failed to read audio file: "+err.Error(), nil)
		return
	}

	// 自动生成唯一的 speaker_id
	speakerID := fmt.Sprintf("speaker_%s_%d", assistantID, time.Now().UnixNano())

	// 检查是否已存在相同姓名的声纹（在同一助手下）
	var existingVoiceprint models.Voiceprint
	result := h.db.Where("speaker_name = ? AND agent_id = ?", speakerName, assistantID).First(&existingVoiceprint)
	if result.Error == nil {
		response.Fail(c, "A voiceprint with this name already exists for this assistant", nil)
		return
	}

	// 先调用voiceprint-api进行声纹注册（Python服务会创建记录）
	if err := h.callVoiceprintRegisterAPI(speakerID, assistantID, audioData); err != nil {
		response.Fail(c, "Failed to register voiceprint: "+err.Error(), nil)
		return
	}

	// Python服务注册成功后，在Go数据库中更新或创建记录
	voiceprint := models.Voiceprint{
		SpeakerID:     speakerID,
		AgentID:   assistantID,
		SpeakerName:   speakerName,
		Description:   description,
		FeatureVector: []byte{}, // 特征向量由Python服务管理
	}

	// 使用UPSERT操作：如果存在则更新，不存在则创建
	if err := h.db.Where("speaker_id = ? AND agent_id = ?", speakerID, assistantID).
		Assign(models.Voiceprint{SpeakerName: speakerName, Description: description}).
		FirstOrCreate(&voiceprint).Error; err != nil {
		response.Fail(c, "Failed to update voiceprint record: "+err.Error(), nil)
		return
	}

	response.Success(c, "Voiceprint registered successfully", models.VoiceprintResponse{
		ID:          voiceprint.ID,
		SpeakerID:   voiceprint.SpeakerID,
		AgentID: voiceprint.AgentID,
		SpeakerName: voiceprint.SpeakerName,
		Description: voiceprint.Description,
		CreatedAt:   voiceprint.CreatedAt,
		UpdatedAt:   voiceprint.UpdatedAt,
	})
}

// UpdateVoiceprint 更新声纹记录
func (h *Handlers) UpdateVoiceprint(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, "Invalid voiceprint ID", nil)
		return
	}

	var req models.VoiceprintUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid request: "+err.Error(), nil)
		return
	}

	var voiceprint models.Voiceprint
	if err := h.db.First(&voiceprint, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "Voiceprint not found", nil)
		} else {
			response.Fail(c, "Database error: "+err.Error(), nil)
		}
		return
	}

	// 构建更新数据
	updateData := map[string]interface{}{}
	if req.SpeakerName != "" {
		updateData["speaker_name"] = req.SpeakerName
		voiceprint.SpeakerName = req.SpeakerName
	}
	if req.Description != "" {
		updateData["description"] = req.Description
		voiceprint.Description = req.Description
	}

	// 只更新指定的字段
	if len(updateData) > 0 {
		if err := h.db.Model(&voiceprint).Updates(updateData).Error; err != nil {
			response.Fail(c, "Failed to update voiceprint: "+err.Error(), nil)
			return
		}
	}

	response.Success(c, "Voiceprint updated successfully", models.VoiceprintResponse{
		ID:          voiceprint.ID,
		SpeakerID:   voiceprint.SpeakerID,
		AgentID: voiceprint.AgentID,
		SpeakerName: voiceprint.SpeakerName,
		Description: voiceprint.Description,
		CreatedAt:   voiceprint.CreatedAt,
		UpdatedAt:   voiceprint.UpdatedAt,
	})
}

// DeleteVoiceprint 删除声纹记录
func (h *Handlers) DeleteVoiceprint(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Fail(c, "Invalid voiceprint ID", nil)
		return
	}

	var voiceprint models.Voiceprint
	if err := h.db.First(&voiceprint, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "Voiceprint not found", nil)
		} else {
			response.Fail(c, "Database error: "+err.Error(), nil)
		}
		return
	}

	// 调用voiceprint-api删除声纹特征
	if err := h.callVoiceprintDeleteAPI(voiceprint.SpeakerID, voiceprint.AgentID); err != nil {
		// 记录错误但不阻止删除数据库记录
		fmt.Printf("Warning: Failed to delete voiceprint from API: %v\n", err)
	}

	// 删除数据库记录
	if err := h.db.Delete(&voiceprint).Error; err != nil {
		response.Fail(c, "Failed to delete voiceprint: "+err.Error(), nil)
		return
	}

	response.Success(c, "Voiceprint deleted successfully", nil)
}

// VerifyVoiceprint 验证特定声纹
func (h *Handlers) VerifyVoiceprint(c *gin.Context) {
	assistantID := c.PostForm("agent_id")
	speakerID := c.PostForm("speaker_id")

	if assistantID == "" || speakerID == "" {
		response.Fail(c, "agent_id and speaker_id are required", nil)
		return
	}

	// 获取上传的音频文件
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		response.Fail(c, "Failed to get audio file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// 验证文件格式
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".wav") {
		response.Fail(c, "Only WAV format is supported", nil)
		return
	}

	// 读取音频数据
	audioData, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, "Failed to read audio file: "+err.Error(), nil)
		return
	}

	// 验证目标声纹是否存在
	var targetVoiceprint models.Voiceprint
	if err := h.db.Where("speaker_id = ? AND agent_id = ?", speakerID, assistantID).First(&targetVoiceprint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Fail(c, "Target voiceprint not found", nil)
		} else {
			response.Fail(c, "Database error: "+err.Error(), nil)
		}
		return
	}

	// 获取该助手下的所有声纹记录（用于识别）
	var voiceprints []models.Voiceprint
	if err := h.db.Where("agent_id = ?", assistantID).Find(&voiceprints).Error; err != nil {
		response.Fail(c, "Failed to get voiceprints: "+err.Error(), nil)
		return
	}

	if len(voiceprints) == 0 {
		response.Fail(c, "No voiceprints found for this assistant", nil)
		return
	}

	// 构建候选ID列表
	candidateIDs := make([]string, len(voiceprints))
	for i, vp := range voiceprints {
		candidateIDs[i] = vp.SpeakerID
	}

	// 调用voiceprint-api进行识别
	identifiedSpeakerID, score, err := h.callVoiceprintIdentifyAPI(candidateIDs, assistantID, audioData)
	if err != nil {
		response.Fail(c, "Failed to verify voiceprint: "+err.Error(), nil)
		return
	}

	// 判断是否为目标说话人
	isTargetSpeaker := identifiedSpeakerID == speakerID

	// 获取置信度等级
	confidence := "low"
	isMatch := false
	if score >= 0.8 {
		confidence = "very_high"
		isMatch = true
	} else if score >= 0.6 {
		confidence = "high"
		isMatch = true
	} else if score >= 0.4 {
		confidence = "medium"
	} else if score >= 0.2 {
		confidence = "low"
	} else {
		confidence = "very_low"
	}

	// 验证结果：需要同时满足识别为目标说话人且置信度足够高
	verificationPassed := isTargetSpeaker && isMatch

	response.Success(c, "Verification completed", models.VoiceprintVerifyResponse{
		TargetSpeakerID:     speakerID,
		IdentifiedSpeakerID: identifiedSpeakerID,
		Score:               score,
		Confidence:          confidence,
		IsMatch:             isMatch,
		IsTargetSpeaker:     isTargetSpeaker,
		VerificationPassed:  verificationPassed,
	})
}

// IdentifyVoiceprint 声纹识别
func (h *Handlers) IdentifyVoiceprint(c *gin.Context) {
	assistantID := c.PostForm("agent_id")
	if assistantID == "" {
		response.Fail(c, "agent_id is required", nil)
		return
	}

	// 获取上传的音频文件
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		response.Fail(c, "Failed to get audio file: "+err.Error(), nil)
		return
	}
	defer file.Close()

	// 验证文件格式
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".wav") {
		response.Fail(c, "Only WAV format is supported", nil)
		return
	}

	// 读取音频数据
	audioData, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, "Failed to read audio file: "+err.Error(), nil)
		return
	}

	// 获取该助手下的所有声纹记录
	var voiceprints []models.Voiceprint
	if err := h.db.Where("agent_id = ?", assistantID).Find(&voiceprints).Error; err != nil {
		response.Fail(c, "Failed to get voiceprints: "+err.Error(), nil)
		return
	}

	if len(voiceprints) == 0 {
		response.Fail(c, "No voiceprints found for this assistant", nil)
		return
	}

	// 构建候选ID列表
	candidateIDs := make([]string, len(voiceprints))
	for i, vp := range voiceprints {
		candidateIDs[i] = vp.SpeakerID
	}

	// 调用voiceprint-api进行识别
	speakerID, score, err := h.callVoiceprintIdentifyAPI(candidateIDs, assistantID, audioData)
	if err != nil {
		response.Fail(c, "Failed to identify voiceprint: "+err.Error(), nil)
		return
	}

	// 获取置信度等级
	confidence := "low"
	isMatch := false
	if score >= 0.8 {
		confidence = "very_high"
		isMatch = true
	} else if score >= 0.6 {
		confidence = "high"
		isMatch = true
	} else if score >= 0.4 {
		confidence = "medium"
	} else if score >= 0.2 {
		confidence = "low"
	} else {
		confidence = "very_low"
	}

	response.Success(c, "Identification completed", models.VoiceprintIdentifyResponse{
		SpeakerID:  speakerID,
		Score:      score,
		Confidence: confidence,
		IsMatch:    isMatch,
	})
}

// callVoiceprintRegisterAPI 调用voiceprint-api注册接口
func (h *Handlers) callVoiceprintRegisterAPI(speakerID, assistantID string, audioData []byte) error {
	cfg := voiceprint.DefaultConfig()
	serviceURL := cfg.BaseURL
	apiKey := cfg.APIKey

	// 创建multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加表单字段
	writer.WriteField("speaker_id", speakerID)
	writer.WriteField("agent_id", assistantID)

	// 添加文件
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return err
	}
	part.Write(audioData)
	writer.Close()

	// 创建请求
	req, err := http.NewRequest("POST", serviceURL+"/voiceprint/register", &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	return nil
}

// callVoiceprintIdentifyAPI 调用voiceprint-api识别接口
func (h *Handlers) callVoiceprintIdentifyAPI(candidateIDs []string, assistantID string, audioData []byte) (string, float64, error) {
	cfg := voiceprint.DefaultConfig()
	serviceURL := cfg.BaseURL
	apiKey := cfg.APIKey

	// 创建multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加表单字段
	writer.WriteField("speaker_ids", strings.Join(candidateIDs, ","))
	writer.WriteField("agent_id", assistantID)

	// 添加文件
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", 0, err
	}
	part.Write(audioData)
	writer.Close()

	// 创建请求
	req, err := http.NewRequest("POST", serviceURL+"/voiceprint/identify", &buf)
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("API error: %s", string(body))
	}

	// 解析响应
	var result struct {
		SpeakerID string  `json:"speaker_id"`
		Score     float64 `json:"score"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}

	return result.SpeakerID, result.Score, nil
}

// callVoiceprintDeleteAPI 调用voiceprint-api删除接口
func (h *Handlers) callVoiceprintDeleteAPI(speakerID, assistantID string) error {
	cfg := voiceprint.DefaultConfig()
	serviceURL := cfg.BaseURL
	apiKey := cfg.APIKey

	// 创建form data
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("agent_id", assistantID)
	writer.Close()

	// 创建请求
	req, err := http.NewRequest("DELETE", serviceURL+"/voiceprint/"+speakerID, &buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	return nil
}

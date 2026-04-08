package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/graph"
	v2 "github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceclone"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Audio processing status cache
type AudioProcessResult struct {
	Status   string // processing | text_ready | completed | failed
	Text     string // Recognized and generated text
	AudioURL string // Audio URL after synthesis is completed
}

var (
	audioProcessingCache = make(map[string]AudioProcessResult) // requestID -> result
	audioCacheMutex      sync.RWMutex
)

// getLanguageConfigKey Get language configuration field name based on provider
// Load from language configuration file, use default value if no configuration file exists
func getLanguageConfigKey(provider string) string {
	// Try to load language options from JSON configuration file
	languages, err := loadLanguageOptionsFromJSON(provider)
	if err == nil && len(languages) > 0 {
		// 使用第一个语言选项的 configKey（所有语言选项的 configKey 应该相同）
		return languages[0].ConfigKey
	}

	// If no configuration file exists, return default configuration field name based on provider
	normalizedProvider := strings.ToLower(provider)
	switch normalizedProvider {
	case "minimax":
		return "languageBoost"
	case "google", "elevenlabs":
		return "languageCode"
	case "baidu":
		return "lan"
	default:
		return "language"
	}
}

// cleanTextForTTS Clean text, remove Markdown format symbols to make it suitable for TTS playback
func cleanTextForTTS(text string) string {
	// 移除Markdown粗体标记 **text**
	text = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(text, "$1")

	// 移除Markdown斜体标记 *text*
	text = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(text, "$1")

	// 移除Markdown代码标记 `text`
	text = regexp.MustCompile("`(.*?)`").ReplaceAllString(text, "$1")

	// 移除Markdown链接 [text](url)
	text = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(text, "$1")

	// 移除Markdown标题标记 # ## ###
	text = regexp.MustCompile(`^#{1,6}\s*`).ReplaceAllString(text, "")

	// 移除Markdown列表标记 - * +
	text = regexp.MustCompile(`^[\s]*[-*+]\s*`).ReplaceAllString(text, "")

	// 移除Markdown引用标记 >
	text = regexp.MustCompile(`^>\s*`).ReplaceAllString(text, "")

	// 移除多余的空行（连续的空行）
	text = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(text, "\n")

	// 移除行首行尾的空白字符
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	// 移除开头和结尾的空白字符
	text = strings.TrimSpace(text)

	return text
}

// CreateTrainingTaskRequest Create training task request
type CreateTrainingTaskRequest struct {
	TaskName string `json:"taskName" binding:"required"`
	Sex      int    `json:"sex"`      // 1:Male 2:Female
	AgeGroup int    `json:"ageGroup"` // 1:Child 2:Youth 3:Middle-aged 4:Middle-aged and elderly
	Language string `json:"language"` // zh, en, ja, ko, ru
}

// SubmitAudioRequest Submit audio request
type SubmitAudioRequest struct {
	TaskID    string `form:"taskId" binding:"required"`
	TextSegID int64  `form:"textSegId" binding:"required"`
}

// QueryTaskStatusRequest Query task status request
type QueryTaskStatusRequest struct {
	TaskID string `json:"taskId" binding:"required"`
}

// SynthesizeRequest Synthesis request
type SynthesizeRequest struct {
	VoiceCloneID uint   `json:"voiceCloneId" binding:"required"`
	Text         string `json:"text" binding:"required"`
	Language     string `json:"language"`
	StorageKey   string `json:"storageKey"`
}

// UpdateVoiceCloneRequest Update voice clone request
type UpdateVoiceCloneRequest struct {
	ID               uint   `json:"id" binding:"required"`
	VoiceName        string `json:"voiceName" binding:"required"`
	VoiceDescription string `json:"voiceDescription"`
}

// CreateTrainingTask 创建训练任务
func (h *Handlers) CreateTrainingTask(c *gin.Context) {
	var req CreateTrainingTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 设置默认值
	if req.Sex == 0 {
		req.Sex = models.SexMale
	}
	if req.AgeGroup == 0 {
		req.AgeGroup = models.AgeGroupYouth
	}
	if req.Language == "" {
		req.Language = models.LanguageChinese
	}

	// 1) 调用讯飞创建任务（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	createReq := &voiceclone.CreateTaskRequest{
		TaskName: req.TaskName,
		Sex:      req.Sex,
		AgeGroup: req.AgeGroup,
		Language: req.Language,
	}
	createResp, err := service.CreateTask(c.Request.Context(), createReq)
	if err != nil {
		// 记录详细错误信息用于调试
		fmt.Printf("讯飞创建任务失败: %v\n", err)

		// 检查是否是训练次数不足的错误
		errMsg := err.Error()
		fmt.Printf("错误信息: %s\n", errMsg)

		if strings.Contains(errMsg, "未分配训练次数") ||
			strings.Contains(errMsg, "训练次数") ||
			strings.Contains(errMsg, "quota") ||
			strings.Contains(errMsg, "次数") {
			fmt.Printf("检测到训练次数不足错误\n")
			response.Fail(c, "训练次数不足", "您的讯飞TTS账户训练次数已用完，请联系管理员或升级账户")
		} else {
			fmt.Printf("其他类型错误\n")
			response.Fail(c, "创建训练任务失败", errMsg)
		}
		return
	}

	taskID := createResp.TaskID

	// 2) 保存配置到数据库（如果配置了）
	h.saveVoiceCloneConfig("xunfei")

	// 3) 保存到数据库
	task := &models.VoiceTrainingTask{
		UserID:   user.ID,
		TaskID:   taskID,
		TaskName: req.TaskName,
		Sex:      req.Sex,
		AgeGroup: req.AgeGroup,
		Language: req.Language,
		Status:   models.TrainingStatusQueued,
		TextID:   5001,
	}
	if err := h.db.Create(task).Error; err != nil {
		response.Fail(c, "保存训练任务失败", err.Error())
		return
	}

	response.Success(c, "创建训练任务成功", task)
}

// saveVoiceCloneConfig 保存音色克隆配置到数据库
func (h *Handlers) saveVoiceCloneConfig(provider string) {
	var configKey string
	var config map[string]interface{}

	switch provider {
	case "xunfei":
		configKey = constants.KEY_VOICE_CLONE_XUNFEI_CONFIG
		config = map[string]interface{}{
			"app_id":        utils.GetEnv("XUNFEI_APP_ID"),
			"api_key":       utils.GetEnv("XUNFEI_API_KEY"),
			"base_url":      utils.GetEnv("XUNFEI_BASE_URL"),
			"ws_app_id":     utils.GetEnv("XUNFEI_WS_APP_ID"),
			"ws_api_key":    utils.GetEnv("XUNFEI_WS_API_KEY"),
			"ws_api_secret": utils.GetEnv("XUNFEI_WS_API_SECRET"),
		}
		if config["base_url"] == "" {
			config["base_url"] = "http://opentrain.xfyousheng.com"
		}
	case "volcengine":
		configKey = constants.KEY_VOICE_CLONE_VOLCENGINE_CONFIG
		config = map[string]interface{}{
			"app_id":         utils.GetEnv("VOLCENGINE_CLONE_APP_ID"),
			"token":          utils.GetEnv("VOLCENGINE_CLONE_TOKEN"),
			"cluster":        utils.GetEnv("VOLCENGINE_CLONE_CLUSTER"),
			"voice_type":     utils.GetEnv("VOLCENGINE_CLONE_VOICE_TYPE"),
			"encoding":       utils.GetEnv("VOLCENGINE_CLONE_ENCODING"),
			"frame_duration": utils.GetEnv("VOLCENGINE_CLONE_FRAME_DURATION"),
		}
		if config["cluster"] == "" {
			config["cluster"] = "volcano_icl"
		}
		if sampleRate := utils.GetIntEnv("VOLCENGINE_CLONE_SAMPLE_RATE"); sampleRate > 0 {
			config["sample_rate"] = sampleRate
		}
		if bitDepth := utils.GetIntEnv("VOLCENGINE_CLONE_BIT_DEPTH"); bitDepth > 0 {
			config["bit_depth"] = bitDepth
		}
		if channels := utils.GetIntEnv("VOLCENGINE_CLONE_CHANNELS"); channels > 0 {
			config["channels"] = channels
		}
		if speedRatio := utils.GetFloatEnv("VOLCENGINE_CLONE_SPEED_RATIO"); speedRatio > 0 {
			config["speed_ratio"] = speedRatio
		}
		if trainingTimes := utils.GetIntEnv("VOLCENGINE_CLONE_TRAINING_TIMES"); trainingTimes > 0 {
			config["training_times"] = trainingTimes
		}
	default:
		return
	}

	// 检查配置是否有效
	if !h.isConfigValid(provider, config) {
		return
	}

	// 序列化为 JSON
	configJSON, err := json.Marshal(config)
	if err != nil {
		logrus.WithError(err).Warn("Failed to marshal voice clone config")
		return
	}

	// 保存到数据库
	utils.SetValue(h.db, configKey, string(configJSON), "json", true, true)
}

// SubmitAudio 提交音频文件
func (h *Handlers) SubmitAudio(c *gin.Context) {
	var req SubmitAudioRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 获取上传的音频文件
	file, err := c.FormFile("audio")
	if err != nil {
		response.Fail(c, "获取音频文件失败", err.Error())
		return
	}

	// 调试信息
	fmt.Printf("接收到的文件: %s, 大小: %d bytes\n", file.Filename, file.Size)

	// 打开文件
	src, err := file.Open()
	if err != nil {
		response.Fail(c, "打开音频文件失败", err.Error())
		return
	}
	defer src.Close()

	// 读取音频数据
	audioData, err := io.ReadAll(src)
	if err != nil {
		response.Fail(c, "读取音频文件失败", err.Error())
		return
	}

	// 调试信息
	fmt.Printf("音频文件大小: %d bytes\n", len(audioData))
	if len(audioData) == 0 {
		response.Fail(c, "音频文件为空", "上传的音频文件没有内容")
		return
	}

	// 1) 查找训练任务
	var task models.VoiceTrainingTask
	if err := h.db.Where("user_id = ? AND task_id = ?", user.ID, req.TaskID).First(&task).Error; err != nil {
		response.Fail(c, "训练任务不存在", err.Error())
		return
	}

	// 2) 调用讯飞提交音频（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	submitReq := &voiceclone.SubmitAudioRequest{
		TaskID:    task.TaskID,
		TextID:    task.TextID,
		TextSegID: req.TextSegID,
		AudioFile: bytes.NewReader(audioData),
		Language:  task.Language,
	}
	if err := service.SubmitAudio(c.Request.Context(), submitReq); err != nil {
		response.Fail(c, "提交音频失败", err.Error())
		return
	}

	// 3) 更新任务状态
	task.Status = models.TrainingStatusInProgress
	task.TextSegID = req.TextSegID
	if err := h.db.Save(&task).Error; err != nil {
		response.Fail(c, "更新任务状态失败", err.Error())
		return
	}

	response.Success(c, "提交音频成功", nil)
}

// QueryTaskStatus 查询任务状态
func (h *Handlers) QueryTaskStatus(c *gin.Context) {
	var req QueryTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 1) 查找训练任务
	var task models.VoiceTrainingTask
	if err := h.db.Where("user_id = ? AND task_id = ?", user.ID, req.TaskID).First(&task).Error; err != nil {
		response.Fail(c, "训练任务不存在", err.Error())
		return
	}

	// 2) 查询讯飞状态（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	status, err := service.QueryTaskStatus(c.Request.Context(), task.TaskID)
	if err != nil {
		response.Fail(c, "查询任务状态失败", err.Error())
		return
	}

	// 3) 更新本地状态
	// 转换状态码
	var trainStatus int
	switch status.Status {
	case voiceclone.TrainingStatusInProgress:
		trainStatus = models.TrainingStatusInProgress
	case voiceclone.TrainingStatusSuccess:
		trainStatus = models.TrainingStatusSuccess
	case voiceclone.TrainingStatusFailed:
		trainStatus = models.TrainingStatusFailed
	case voiceclone.TrainingStatusQueued:
		trainStatus = models.TrainingStatusQueued
	default:
		trainStatus = models.TrainingStatusInProgress
	}

	task.Status = trainStatus
	task.TrainVID = status.TrainVID
	task.AssetID = status.AssetID // xunfei 返回的音色ID
	task.FailedReason = status.FailedDesc
	if err := h.db.Save(&task).Error; err != nil {
		response.Fail(c, "更新任务状态失败", err.Error())
		return
	}

	// 4) 如果训练成功，落表VoiceClone
	if trainStatus == models.TrainingStatusSuccess && status.AssetID != "" {
		if err := h.upsertVoiceClone(c.Request.Context(), user.ID, &task, status.AssetID, status.TrainVID, "xunfei"); err != nil {
			response.Fail(c, "创建音色记录失败", err.Error())
			return
		}
	}

	response.Success(c, "查询任务状态成功", task)
}

// GetUserVoiceClones 获取用户的音色列表
func (h *Handlers) GetUserVoiceClones(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	var clones []models.VoiceClone
	query := h.db.Where("user_id = ? AND is_active = ?", user.ID, true)

	// 支持按 provider 过滤
	if provider := c.Query("provider"); provider != "" {
		query = query.Where("provider = ?", provider)
	}

	if err := query.Order("created_at DESC").
		Find(&clones).Error; err != nil {
		response.Fail(c, "获取音色列表失败", err.Error())
		return
	}
	response.Success(c, "获取音色列表成功", clones)
}

// GetVoiceClone 获取指定音色
func (h *Handlers) GetVoiceClone(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 获取音色ID
	cloneIDStr := c.Param("id")
	cloneID, err := strconv.ParseUint(cloneIDStr, 10, 32)
	if err != nil {
		response.Fail(c, "音色ID格式错误", err.Error())
		return
	}

	var clone models.VoiceClone
	if err := h.db.Where("user_id = ? AND id = ? AND is_active = ?", user.ID, uint(cloneID), true).
		First(&clone).Error; err != nil {
		response.Fail(c, "音色不存在", err.Error())
		return
	}
	response.Success(c, "获取音色信息成功", clone)
}

// SynthesizeWithVoice 使用音色合成语音
func (h *Handlers) SynthesizeWithVoice(c *gin.Context) {
	var req SynthesizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 设置默认值
	if req.Language == "" {
		req.Language = models.LanguageChinese
	}
	if req.StorageKey == "" {
		// 添加时间戳避免文件名冲突
		timestamp := time.Now().Format("20060102_150405")
		req.StorageKey = "voice_synthesis/" + strconv.FormatUint(uint64(req.VoiceCloneID), 10) + "_" + timestamp + "_" + strconv.FormatInt(int64(len(req.Text)), 10) + ".mp3"
	}

	// 1) 获取音色
	var clone models.VoiceClone
	if err := h.db.Where("user_id = ? AND id = ? AND is_active = ?", user.ID, req.VoiceCloneID, true).
		First(&clone).Error; err != nil {
		response.Fail(c, "音色不存在", err.Error())
		return
	}

	// 验证 AssetID 是否存在
	if clone.AssetID == "" {
		response.Fail(c, "音色未训练完成", "该音色尚未训练完成，无法使用")
		return
	}

	// 2) 调用讯飞合成（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	// 使用正确的 AssetID（从数据库查询到的音色ID）
	synthesizeReq := &voiceclone.SynthesizeRequest{
		AssetID:  clone.AssetID, // 使用训练得到的音色ID（从数据库查询）
		Text:     req.Text,
		Language: req.Language,
	}

	// 添加调试日志
	fmt.Printf("[SynthesizeWithVoice] VoiceCloneID=%d, AssetID=%s, VoiceName=%s\n",
		req.VoiceCloneID, clone.AssetID, clone.VoiceName)
	audioURL, err := service.SynthesizeToStorage(c.Request.Context(), synthesizeReq, req.StorageKey)
	if err != nil {
		response.Fail(c, "语音合成失败", err.Error())
		return
	}
	audioSize := int64(0)
	audioDuration := float64(0)
	// 计算音频时长
	// MP3 文件：使用近似估算，128kbps 比特率
	// 时长 = (文件大小 * 8) / (比特率 * 1000) 秒
	if strings.HasSuffix(req.StorageKey, ".mp3") {
		// MP3 128kbps 比特率估算
		bitrate := 128000 // 128 kbps = 128000 bps
		audioDuration = float64(audioSize) * 8.0 / float64(bitrate)
	} else {
		// WAV/PCM 文件：PCM数据大小 / (采样率 * 声道数 * 位深度/8)
		// 讯飞默认：24000Hz, 1声道, 16bit
		// WAV文件有44字节头部，需要减去
		pcmDataSize := audioSize - 44
		if pcmDataSize > 0 {
			sampleRate := 24000
			channels := 1
			bitDepth := 16
			bytesPerSecond := sampleRate * channels * (bitDepth / 8)
			audioDuration = float64(pcmDataSize) / float64(bytesPerSecond)
		}
	}

	// 4) 记录合成历史
	synthesis := &models.VoiceSynthesis{
		UserID:        user.ID,
		VoiceCloneID:  clone.ID,
		Text:          req.Text,
		Language:      req.Language,
		AudioURL:      audioURL,
		AudioDuration: audioDuration,
		AudioSize:     audioSize,
		Status:        "success",
	}
	if err := h.db.Create(synthesis).Error; err != nil {
		response.Fail(c, "保存合成记录失败", err.Error())
		return
	}

	// 4) 更新音色使用统计
	clone.IncrementUsage()
	if err := h.db.Save(&clone).Error; err != nil {
		fmt.Printf("更新音色使用统计失败: %v\n", err)
	}

	response.Success(c, "语音合成成功", synthesis)
}

// SynthesisHistoryItem 合成历史项（只返回必要字段）
type SynthesisHistoryItem struct {
	ID           uint   `json:"id"`
	VoiceCloneID uint   `json:"voice_clone_id"`
	Text         string `json:"text"`
	Language     string `json:"language"`
	AudioURL     string `json:"audio_url"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	Provider     string `json:"provider"` // 从 VoiceClone 获取
}

// GetSynthesisHistory 获取合成历史
func (h *Handlers) GetSynthesisHistory(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 获取限制参数
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	// 支持按 provider 过滤
	provider := c.Query("provider")

	var history []models.VoiceSynthesis
	query := h.db.Model(&models.VoiceSynthesis{}).Where("voice_syntheses.user_id = ?", user.ID)

	// 如果指定了 provider，需要 join VoiceClone 表过滤
	if provider != "" {
		query = query.Joins("JOIN voice_clones ON voice_syntheses.voice_clone_id = voice_clones.id").
			Where("voice_clones.provider = ? AND voice_clones.deleted_at IS NULL", provider)
	}

	query = query.Order("voice_syntheses.created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&history).Error; err != nil {
		response.Fail(c, "获取合成历史失败", err.Error())
		return
	}

	// 获取所有相关的 VoiceClone ID
	voiceCloneIDs := make([]uint, 0, len(history))
	for _, item := range history {
		voiceCloneIDs = append(voiceCloneIDs, item.VoiceCloneID)
	}

	// 批量查询 VoiceClone 获取 provider
	voiceClones := make(map[uint]string)
	if len(voiceCloneIDs) > 0 {
		var clones []models.VoiceClone
		if err := h.db.Select("id, provider").Where("id IN ?", voiceCloneIDs).Find(&clones).Error; err == nil {
			for _, clone := range clones {
				voiceClones[clone.ID] = clone.Provider
			}
		}
	}

	// 只返回必要字段
	result := make([]SynthesisHistoryItem, 0, len(history))
	for _, item := range history {
		providerStr := voiceClones[item.VoiceCloneID]

		result = append(result, SynthesisHistoryItem{
			ID:           item.ID,
			VoiceCloneID: item.VoiceCloneID,
			Text:         item.Text,
			Language:     item.Language,
			AudioURL:     item.AudioURL,
			Status:       item.Status,
			CreatedAt:    item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Provider:     providerStr,
		})
	}

	response.Success(c, "获取合成历史成功", result)
}

// DeleteSynthesisRecord 删除合成历史记录
func (h *Handlers) DeleteSynthesisRecord(c *gin.Context) {
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 检查记录是否存在且属于当前用户
	var record models.VoiceSynthesis
	if err := h.db.Where("user_id = ? AND id = ?", user.ID, req.ID).First(&record).Error; err != nil {
		response.Fail(c, "合成记录不存在", err.Error())
		return
	}

	// 删除记录
	if err := h.db.Where("user_id = ? AND id = ?", user.ID, req.ID).
		Delete(&models.VoiceSynthesis{}).Error; err != nil {
		response.Fail(c, "删除合成记录失败", err.Error())
		return
	}

	response.Success(c, "删除合成记录成功", nil)
}

// UpdateVoiceClone 更新音色信息
func (h *Handlers) UpdateVoiceClone(c *gin.Context) {
	var req UpdateVoiceCloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	if err := h.db.Model(&models.VoiceClone{}).
		Where("user_id = ? AND id = ?", user.ID, req.ID).
		Updates(map[string]any{
			"voice_name":        req.VoiceName,
			"voice_description": req.VoiceDescription,
			"updated_at":        time.Now(),
		}).Error; err != nil {
		response.Fail(c, "更新音色信息失败", err.Error())
		return
	}
	response.Success(c, "更新音色信息成功", nil)
}

// VoiceOption 音色选项结构
type VoiceOption struct {
	ID          string `json:"id"`                   // 音色编码
	Name        string `json:"name"`                 // 音色名称
	Description string `json:"description"`          // 音色描述
	Type        string `json:"type"`                 // 音色类型（男声/女声/童声等）
	Language    string `json:"language"`             // 支持的语言
	SampleRate  string `json:"sampleRate,omitempty"` // 音色采样率
	Emotion     string `json:"emotion,omitempty"`    // 音色情感
	Scene       string `json:"scene,omitempty"`      // 推荐场景
}

// LanguageOption 语言选项结构
type LanguageOption struct {
	Code        string `json:"code"`        // 语言代码，如 zh-CN, en-US
	Name        string `json:"name"`        // 语言名称，如 中文、English
	NativeName  string `json:"nativeName"`  // 本地名称，如 中文、English
	ConfigKey   string `json:"configKey"`   // 配置字段名（不同平台可能不同），如 language, languageCode, lan
	Description string `json:"description"` // 语言描述
}

// GetVoiceOptions 根据TTS Provider获取音色列表
func (h *Handlers) GetVoiceOptions(c *gin.Context) {
	provider := c.Query("provider")
	if provider == "" {
		response.Fail(c, "缺少provider参数", nil)
		return
	}

	// 将provider标准化（qcloud和tencent都映射到tencent）
	normalizedProvider := strings.ToLower(provider)
	if normalizedProvider == "qcloud" {
		normalizedProvider = "tencent"
	}

	// 特殊处理 FishSpeech - 从 API 动态获取发音人列表
	if normalizedProvider == "fishspeech" {
		h.getFishSpeechVoices(c)
		return
	}

	// 特殊处理 Fish Audio - 从 API 动态获取发音人列表
	if normalizedProvider == "fishaudio" {
		h.getFishAudioVoices(c)
		return
	}

	// 从JSON文件读取音色列表
	voices, err := loadVoiceOptionsFromJSON(normalizedProvider)
	if err != nil {
		logrus.WithError(err).Errorf("加载音色列表失败: provider=%s", normalizedProvider)
		response.Fail(c, fmt.Sprintf("加载音色列表失败: %v", err), nil)
		return
	}

	if voices == nil || len(voices) == 0 {
		response.Fail(c, fmt.Sprintf("不支持的TTS Provider: %s 或音色列表为空", provider), nil)
		return
	}

	response.Success(c, "获取音色列表成功", gin.H{
		"provider": normalizedProvider,
		"voices":   voices,
	})
}

// loadVoiceOptionsFromJSON 从JSON文件加载音色列表
func loadVoiceOptionsFromJSON(provider string) ([]VoiceOption, error) {
	// 获取项目根目录（从当前工作目录向上查找）
	jsonPath := getVoiceJSONPath(provider)
	if jsonPath == "" {
		return nil, fmt.Errorf("无法找到音色配置文件: provider=%s", provider)
	}

	// 读取JSON文件
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("读取音色配置文件失败: %w", err)
	}

	// 解析JSON
	var voices []VoiceOption
	if err := json.Unmarshal(data, &voices); err != nil {
		return nil, fmt.Errorf("解析音色配置文件失败: %w", err)
	}

	return voices, nil
}

// getVoiceJSONPath 获取音色JSON文件路径
func getVoiceJSONPath(provider string) string {
	// 尝试多个可能的路径（相对于项目根目录）
	possiblePaths := []string{
		filepath.Join("scripts", "voices", provider+".json"),
		filepath.Join("..", "scripts", "voices", provider+".json"),
		filepath.Join("../../scripts", "voices", provider+".json"),
		filepath.Join(".", "scripts", "voices", provider+".json"),
	}

	// 从当前工作目录开始查找
	wd, err := os.Getwd()
	if err == nil {
		for _, p := range possiblePaths {
			fullPath := filepath.Join(wd, p)
			// 清理路径（处理相对路径）
			fullPath = filepath.Clean(fullPath)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	// 如果工作目录查找失败，尝试从执行文件所在目录查找
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		for _, p := range possiblePaths {
			fullPath := filepath.Join(execDir, p)
			fullPath = filepath.Clean(fullPath)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	// 尝试从环境变量获取项目根目录
	if projectRoot := os.Getenv("PROJECT_ROOT"); projectRoot != "" {
		fullPath := filepath.Join(projectRoot, "scripts", "voices", provider+".json")
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

// GetLanguageOptions 根据TTS Provider获取支持的语言列表
func (h *Handlers) GetLanguageOptions(c *gin.Context) {
	provider := c.Query("provider")
	if provider == "" {
		response.Fail(c, "缺少provider参数", nil)
		return
	}

	// 将provider标准化（qcloud和tencent都映射到tencent）
	normalizedProvider := strings.ToLower(provider)
	if normalizedProvider == "qcloud" {
		normalizedProvider = "tencent"
	}

	// 从JSON文件读取语言列表
	languages, err := loadLanguageOptionsFromJSON(normalizedProvider)
	if err != nil {
		logrus.WithError(err).Errorf("加载语言列表失败: provider=%s", normalizedProvider)
		response.Fail(c, fmt.Sprintf("加载语言列表失败: %v", err), nil)
		return
	}

	if languages == nil || len(languages) == 0 {
		// 如果没有配置文件，返回默认支持的语言
		languages = getDefaultLanguageOptions(normalizedProvider)
	}

	response.Success(c, "获取语言列表成功", gin.H{
		"provider":  normalizedProvider,
		"languages": languages,
	})
}

// loadLanguageOptionsFromJSON 从JSON文件加载语言列表
func loadLanguageOptionsFromJSON(provider string) ([]LanguageOption, error) {
	jsonPath := getLanguageJSONPath(provider)
	if jsonPath == "" {
		return nil, fmt.Errorf("无法找到语言配置文件: provider=%s", provider)
	}

	// 读取JSON文件
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("读取语言配置文件失败: %w", err)
	}

	// 解析JSON
	var languages []LanguageOption
	if err := json.Unmarshal(data, &languages); err != nil {
		return nil, fmt.Errorf("解析语言配置文件失败: %w", err)
	}

	return languages, nil
}

// getLanguageJSONPath 获取语言JSON文件路径
func getLanguageJSONPath(provider string) string {
	possiblePaths := []string{
		filepath.Join("scripts", "languages", provider+".json"),
		filepath.Join("..", "scripts", "languages", provider+".json"),
		filepath.Join("../../scripts", "languages", provider+".json"),
		filepath.Join(".", "scripts", "languages", provider+".json"),
	}

	// 从当前工作目录开始查找
	wd, err := os.Getwd()
	if err == nil {
		for _, p := range possiblePaths {
			fullPath := filepath.Join(wd, p)
			fullPath = filepath.Clean(fullPath)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	// 如果工作目录查找失败，尝试从执行文件所在目录查找
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		for _, p := range possiblePaths {
			fullPath := filepath.Join(execDir, p)
			fullPath = filepath.Clean(fullPath)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath
			}
		}
	}

	// 尝试从环境变量获取项目根目录
	if projectRoot := os.Getenv("PROJECT_ROOT"); projectRoot != "" {
		fullPath := filepath.Join(projectRoot, "scripts", "languages", provider+".json")
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

// getDefaultLanguageOptions 获取默认支持的语言列表（当没有配置文件时）
func getDefaultLanguageOptions(provider string) []LanguageOption {
	// 根据不同的提供商返回默认支持的语言
	switch provider {
	case "tencent", "qcloud":
		return []LanguageOption{
			{Code: "zh-CN", Name: "中文", NativeName: "中文", ConfigKey: "language", Description: "中文（普通话）"},
			{Code: "en-US", Name: "English", NativeName: "English", ConfigKey: "language", Description: "英语（美式）"},
		}
	case "volcengine":
		return []LanguageOption{
			{Code: "zh", Name: "中文", NativeName: "中文", ConfigKey: "language", Description: "中文"},
			{Code: "en", Name: "English", NativeName: "English", ConfigKey: "language", Description: "英语"},
		}
	case "azure":
		return []LanguageOption{
			{Code: "zh-CN", Name: "中文", NativeName: "中文", ConfigKey: "language", Description: "中文（简体）"},
			{Code: "en-US", Name: "English", NativeName: "English", ConfigKey: "language", Description: "英语（美式）"},
			{Code: "ja-JP", Name: "Japanese", NativeName: "日本語", ConfigKey: "language", Description: "日语"},
		}
	case "elevenlabs":
		return []LanguageOption{
			{Code: "en", Name: "English", NativeName: "English", ConfigKey: "languageCode", Description: "英语"},
			{Code: "zh", Name: "Chinese", NativeName: "中文", ConfigKey: "languageCode", Description: "中文"},
		}
	case "google":
		return []LanguageOption{
			{Code: "en-US", Name: "English", NativeName: "English", ConfigKey: "languageCode", Description: "英语（美式）"},
			{Code: "zh-CN", Name: "Chinese", NativeName: "中文", ConfigKey: "languageCode", Description: "中文（简体）"},
		}
	case "baidu":
		return []LanguageOption{
			{Code: "zh", Name: "中文", NativeName: "中文", ConfigKey: "lan", Description: "中文"},
			{Code: "en", Name: "English", NativeName: "English", ConfigKey: "lan", Description: "英语"},
		}
	case "minimax":
		return []LanguageOption{
			{Code: "zh", Name: "中文", NativeName: "中文", ConfigKey: "languageBoost", Description: "中文"},
			{Code: "en", Name: "English", NativeName: "English", ConfigKey: "languageBoost", Description: "英语"},
		}
	default:
		// 默认返回通用语言列表
		return []LanguageOption{
			{Code: "zh-CN", Name: "中文", NativeName: "中文", ConfigKey: "language", Description: "中文（简体）"},
			{Code: "en-US", Name: "English", NativeName: "English", ConfigKey: "language", Description: "英语（美式）"},
		}
	}
}

// getFishSpeechVoices 从 FishSpeech API 获取音色列表
func (h *Handlers) getFishSpeechVoices(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 从查询参数获取 API Key（优先级最高）
	apiKey := c.Query("apiKey")

	// 如果没有从查询参数获取，尝试从 user_credential 中获取
	if apiKey == "" {
		credentials, err := models.GetUserCredentials(h.db, user.ID)
		if err != nil {
			logrus.WithError(err).Errorf("获取用户凭证失败")
			response.Fail(c, "获取用户凭证失败", nil)
			return
		}

		// 查找 FishSpeech 的 TTS 配置
		for _, cred := range credentials {
			if cred.TtsConfig != nil {
				if provider, ok := cred.TtsConfig["provider"].(string); ok && provider == "fishspeech" {
					if key, ok := cred.TtsConfig["apiKey"].(string); ok && key != "" {
						apiKey = key
						break
					}
				}
			}
		}
	}

	if apiKey == "" {
		response.Fail(c, "缺少 FishSpeech API Key", "请在用户凭证中配置 FishSpeech API Key")
		return
	}

	// 调用 FishSpeech API 获取音色列表
	voices, err := synthesizer.GetFishSpeechVoices(apiKey)
	if err != nil {
		logrus.WithError(err).Errorf("获取 FishSpeech 音色列表失败")
		response.Fail(c, fmt.Sprintf("获取 FishSpeech 音色列表失败: %v", err), nil)
		return
	}

	// 将 FishSpeech 音色转换为 VoiceOption 格式
	voiceOptions := make([]VoiceOption, len(voices))
	for i, voice := range voices {
		voiceOptions[i] = VoiceOption{
			ID:          voice.ModelID,
			Name:        voice.Title,
			Description: voice.Description,
		}
	}

	response.Success(c, "获取 FishSpeech 音色列表成功", gin.H{
		"provider": "fishspeech",
		"voices":   voiceOptions,
	})
}

// getFishAudioVoices 获取 Fish Audio 音色列表
func (h *Handlers) getFishAudioVoices(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	// 从查询参数获取 API Key（优先级最高）
	apiKey := c.Query("apiKey")

	// 如果没有从查询参数获取，尝试从 user_credential 中获取
	if apiKey == "" {
		credentials, err := models.GetUserCredentials(h.db, user.ID)
		if err != nil {
			logrus.WithError(err).Errorf("获取用户凭证失败")
			response.Fail(c, "获取用户凭证失败", nil)
			return
		}

		// 查找 Fish Audio 的 TTS 配置
		for _, cred := range credentials {
			if cred.TtsConfig != nil {
				if provider, ok := cred.TtsConfig["provider"].(string); ok && provider == "fishaudio" {
					if key, ok := cred.TtsConfig["apiKey"].(string); ok && key != "" {
						apiKey = key
						break
					}
				}
			}
		}
	}

	if apiKey == "" {
		response.Fail(c, "缺少 Fish Audio API Key", "请在用户凭证中配置 Fish Audio API Key")
		return
	}

	// 调用 Fish Audio API 获取音色列表
	voices, err := synthesizer.GetFishAudioVoices(apiKey)
	if err != nil {
		logrus.WithError(err).Errorf("获取 Fish Audio 音色列表失败")
		response.Fail(c, fmt.Sprintf("获取 Fish Audio 音色列表失败: %v", err), nil)
		return
	}

	// 将 Fish Audio 音色转换为 VoiceOption 格式
	voiceOptions := make([]VoiceOption, len(voices))
	for i, voice := range voices {
		voiceOptions[i] = VoiceOption{
			ID:          voice.ID,
			Name:        voice.Title,
			Description: voice.Description,
		}
	}

	response.Success(c, "获取 Fish Audio 音色列表成功", gin.H{
		"provider": "fishaudio",
		"voices":   voiceOptions,
	})
}

// DeleteVoiceClone 删除音色
func (h *Handlers) DeleteVoiceClone(c *gin.Context) {
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "未授权", "用户未登录")
		return
	}

	if err := h.db.Where("user_id = ? AND id = ?", user.ID, req.ID).
		Delete(&models.VoiceClone{}).Error; err != nil {
		response.Fail(c, "删除音色失败", err.Error())
		return
	}
	response.Success(c, "删除音色成功", nil)
}

// GetTrainingTexts 获取训练文本
func (h *Handlers) GetTrainingTexts(c *gin.Context) {
	// 获取文本ID参数
	textIDStr := c.DefaultQuery("textId", "5001")
	textID, err := strconv.ParseInt(textIDStr, 10, 64)
	if err != nil {
		response.Fail(c, "文本ID格式错误", err.Error())
		return
	}

	// 1) 先查库
	var text models.VoiceTrainingText
	if err := h.db.Where("text_id = ? AND is_active = ?", textID, true).
		First(&text).Error; err == nil {
		// 查询关联的段落
		var segments []models.VoiceTrainingTextSegment
		h.db.Where("text_id = ?", text.ID).Find(&segments)
		text.TextSegments = segments
		response.Success(c, "获取训练文本成功", text)
		return
	}

	// 2) 查讯飞（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	xfText, err := service.GetTrainingTexts(c.Request.Context(), textID)
	if err != nil {
		response.Fail(c, "获取训练文本失败", err.Error())
		return
	}

	// 3) 保存主表
	text = models.VoiceTrainingText{
		TextID:   xfText.TextID,
		TextName: xfText.TextName,
		Language: models.LanguageChinese,
		IsActive: true,
	}
	if err := h.db.Create(&text).Error; err != nil {
		response.Fail(c, "保存训练文本失败", err.Error())
		return
	}

	// 4) 保存段落
	for _, seg := range xfText.Segments {
		segRow := models.VoiceTrainingTextSegment{
			TextID:  text.ID,
			SegID:   fmt.Sprintf("%v", seg.SegID),
			SegText: seg.SegText,
		}
		if err := h.db.Create(&segRow).Error; err != nil {
			response.Fail(c, "保存文本段落失败", err.Error())
			return
		}
	}

	// 5) 重新加载返回
	if err := h.db.First(&text, text.ID).Error; err != nil {
		response.Fail(c, "加载训练文本失败", err.Error())
		return
	}
	// 加载关联的段落
	var segments []models.VoiceTrainingTextSegment
	h.db.Where("text_id = ?", text.ID).Find(&segments)
	text.TextSegments = segments
	response.Success(c, "获取训练文本成功", text)
}

// upsertVoiceClone 如果不存在则创建，存在则更新
func (h *Handlers) upsertVoiceClone(ctx context.Context, userID uint, task *models.VoiceTrainingTask, assetID, trainVID, provider string) error {
	var existing models.VoiceClone
	if err := h.db.Where("user_id = ? AND asset_id = ? AND provider = ?", userID, assetID, provider).First(&existing).Error; err == nil {
		existing.TrainVID = trainVID
		existing.IsActive = true
		return h.db.Save(&existing).Error
	}
	clone := &models.VoiceClone{
		UserID:           userID,
		TrainingTaskID:   task.ID,
		Provider:         provider,
		AssetID:          assetID,
		TrainVID:         trainVID,
		VoiceName:        task.TaskName,
		VoiceDescription: fmt.Sprintf("基于任务 %s 训练的音色", task.TaskName),
		IsActive:         true,
		UsageCount:       0,
	}
	return h.db.Create(clone).Error
}

// GetAudioStatus 获取音频处理状态
func (h *Handlers) GetAudioStatus(c *gin.Context) {
	requestID := c.Query("requestId")
	if requestID == "" {
		response.Fail(c, "缺少requestId参数", nil)
		return
	}

	audioCacheMutex.RLock()
	result, exists := audioProcessingCache[requestID]
	audioCacheMutex.RUnlock()

	if !exists {
		response.Success(c, "处理中", gin.H{
			"status":   "processing",
			"audioUrl": "",
			"text":     "",
		})
		return
	}

	if result.Status == "completed" {
		// 返回后删除
		audioCacheMutex.Lock()
		delete(audioProcessingCache, requestID)
		audioCacheMutex.Unlock()
	}

	response.Success(c, "状态", gin.H{
		"status":   result.Status,
		"audioUrl": result.AudioURL,
		"text":     result.Text,
	})
}

// OneShotTextRequest 请求结构
type OneShotTextRequest struct {
	APIKey          string  `json:"apiKey" binding:"required"`
	APISecret       string  `json:"apiSecret" binding:"required"`
	Text            string  `json:"text" binding:"required"`
	AssistantID     int     `json:"assistantId"`
	Language        string  `json:"language"`
	SessionID       string  `json:"sessionId"`
	SystemPrompt    string  `json:"systemPrompt"`
	Speaker         string  `json:"speaker"`         // 音色编码
	VoiceCloneID    int     `json:"voiceCloneId"`    // 训练音色ID（优先级高于speaker）
	Temperature     float32 `json:"temperature"`     // 生成多样性 (0-2)
	MaxTokens       int     `json:"maxTokens"`       // 最大回复长度
}

// OneShotText 处理一句话模式的文本输入（V2版本，使用用户凭证配置）
func (h *Handlers) OneShotText(c *gin.Context) {
	var req OneShotTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 1. 查询用户凭证配置
	credential, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, req.APIKey, req.APISecret)
	if err != nil {
		response.Fail(c, "查询凭证失败", err.Error())
		return
	}
	if credential == nil {
		response.Fail(c, "凭证不存在", "无效的 apiKey 或 apiSecret")
		return
	}

	// 获取用户信息
	var user models.User
	if err := h.db.First(&user, credential.UserID).Error; err != nil {
		response.Fail(c, "用户不存在", err.Error())
		return
	}

	// 设置默认值
	if req.Language == "" {
		req.Language = models.LanguageChinese
		if asrLanguage := credential.GetASRConfigString("language"); asrLanguage != "" {
			req.Language = asrLanguage
		}
	}

	// 2. 调用LLM处理文本
	var llmResponse string
	var errLLM error
	if credential.LLMProvider != "" && credential.LLMApiURL != "" {
		llmBaseURL := credential.LLMApiURL
		if llmBaseURL == "" {
			llmBaseURL = utils.GetEnv("LLM_BASE_URL")
		}
		// 获取模型和参数，优先级：Assistant配置 > 环境变量 > 默认值
		var assistant models.Assistant
		llmModel := ""
		if req.AssistantID > 0 {
			if err := h.db.First(&assistant, req.AssistantID).Error; err == nil {
				if assistant.LLMModel != "" {
					llmModel = assistant.LLMModel
				}
			}
		}

		// 如果Assistant没有配置模型，使用环境变量
		if llmModel == "" {
			llmModel = utils.GetEnv("LLM_MODEL")
		}

		// 如果环境变量也没有，使用默认值
		if llmModel == "" {
			llmModel = "deepseek-v3.1" // 最终默认值
		}

		// 优先使用请求中的 temperature 和 maxTokens，如果没有则从 assistant 中读取
		var temp *float32
		var maxTokens *int

		if req.Temperature > 0 {
			temp = &req.Temperature
		} else if req.AssistantID > 0 {
			// 从 assistant 中读取
			if assistant.Temperature > 0 {
				temp = &assistant.Temperature
			}
		}
		// 如果还是没有，使用默认值
		if temp == nil {
			defaultTemp := float32(0.7)
			temp = &defaultTemp
		}

		if req.MaxTokens > 0 {
			maxTokens = &req.MaxTokens
		} else if req.AssistantID > 0 {
			// 从 assistant 中读取
			if assistant.MaxTokens > 0 {
				maxTokens = &assistant.MaxTokens
			}
		}

		// 构建系统提示词，如果设置了 maxTokens，添加回复长度指导
		systemPrompt := req.SystemPrompt
		if systemPrompt == "" {
			if req.AssistantID > 0 && assistant.SystemPrompt != "" {
				systemPrompt = assistant.SystemPrompt
			} else {
				systemPrompt = "请用中文回复用户的问题。"
			}
		}

		// 如果开启了图记忆功能，则尝试从 Neo4j 中获取该用户的长期偏好主题，并拼接到系统提示词中
		if config.GlobalConfig.Services.GraphMemory.Neo4j.Enabled && assistant.EnableGraphMemory {
			if store := graph.GetDefaultStore(); store != nil {
				ctx := c.Request.Context()
				if userCtx, err := store.GetUserContext(ctx, user.ID, int64(req.AssistantID)); err == nil {
					if len(userCtx.Topics) > 0 {
						preferenceText := fmt.Sprintf("该用户在历史对话中经常讨论这些主题：%s。请在回答时优先从这些兴趣和习惯的角度来组织内容，让风格尽量贴近他的偏好。",
							strings.Join(userCtx.Topics, "、"))
						systemPrompt = systemPrompt + "\n\n" + preferenceText
					}
				}
			}
		}

		// 如果设置了 maxTokens，在系统提示词中添加回复长度指导
		// 让 AI 知道要在限制内完整回答，避免被截断
		if maxTokens != nil && *maxTokens > 0 {
			// 估算 maxTokens 对应的中文字数（大约 1 token = 1.5 个中文字符）
			estimatedChars := *maxTokens * 3 / 2
			lengthGuidance := fmt.Sprintf("\n\n重要提示：你的回复有长度限制（约 %d 个字符），请确保在限制内完整回答。如果回答较长，请优先保证回答的完整性和逻辑性，可以适当精简表述，但不要被截断。", estimatedChars)
			systemPrompt = systemPrompt + lengthGuidance
		}

		llmHandler, err := v2.NewLLMProvider(c.Request.Context(), credential.LLMProvider, credential.LLMApiKey, credential.LLMApiURL, systemPrompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  fmt.Sprintf("初始化LLM失败: %v", err),
			})
			return
		}

		queryText := req.Text

		userID := user.ID
		assistantID := int64(req.AssistantID)
		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = fmt.Sprintf("text_v2_%d_%d", user.ID, time.Now().Unix())
		}
		credentialID := credential.ID
		llmResponse, errLLM = llmHandler.QueryWithOptions(queryText, v2.QueryOptions{
			Model:            llmModel,
			Temperature:      temp,
			MaxTokens:        maxTokens,
			UserID:           &userID,
			AssistantID:      &assistantID,
			CredentialID:     &credentialID,
			SessionID:        sessionID,
			ChatType:         models.ChatTypeText,
			EnableJSONOutput: assistant.EnableJSONOutput,
		})
		if errLLM != nil {
			// 提取更友好的错误信息
			errMsg := errLLM.Error()

			// 检查是否是模型不可用的错误
			if strings.Contains(errMsg, "no available channels") || strings.Contains(errMsg, "model") {
				response.Fail(c, "模型不可用", fmt.Sprintf("模型 %s 当前不可用，请检查模型配置或尝试其他模型。错误详情：%s", llmModel, errMsg))
			} else {
				response.Fail(c, "LLM处理失败", errMsg)
			}
			return
		}
	} else {
		// 如果没有配置LLM，直接返回原文本
		llmResponse = req.Text
	}

	// 3. 立即返回文本，异步处理音频
	requestId := fmt.Sprintf("%d_%d", user.ID, time.Now().Unix())
	response.Success(c, "处理完成", gin.H{
		"text":      llmResponse,
		"audioUrl":  "",        // 先返回空，后续通过轮询获取
		"requestId": requestId, // 用于轮询
	})

	// 4. 聊天记录已通过 LLMListener 自动保存（如果提供了 UserID 和 AssistantID）
	// 这里不再需要手动保存，避免重复记录

	// 5. 异步处理音频合成（使用pkg/synthesis）
	go h.processAudioAsyncV2(context.Background(), credential, user.ID, llmResponse, req.Language, req.Speaker, req.VoiceCloneID, requestId)
}

// SimpleTextChatRequest 简单文本对话请求结构（无需token）
type SimpleTextChatRequest struct {
	APIKey       string  `json:"apiKey" binding:"required"`
	APISecret    string  `json:"apiSecret" binding:"required"`
	Text         string  `json:"text" binding:"required"`
	AssistantID  int     `json:"assistantId" binding:"required"`
	SessionID    string  `json:"sessionId"`
	Temperature  float32 `json:"temperature"`
	MaxTokens    int     `json:"maxTokens"`
	SystemPrompt string  `json:"systemPrompt"`
}

// SimpleTextChatResponse 简单文本对话响应
type SimpleTextChatResponse struct {
	Text      string `json:"text"`
	SessionID string `json:"sessionId"`
}

// SimpleTextChat 处理简单文本对话（无需token验证，仅返回文本）
func (h *Handlers) SimpleTextChat(c *gin.Context) {
	var req SimpleTextChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 1. 验证凭证
	credential, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, req.APIKey, req.APISecret)
	if err != nil {
		response.Fail(c, "查询凭证失败", err.Error())
		return
	}
	if credential == nil {
		response.Fail(c, "凭证不存在", "无效的 apiKey 或 apiSecret")
		return
	}

	// 2. 获取用户信息
	var user models.User
	if err := h.db.First(&user, credential.UserID).Error; err != nil {
		response.Fail(c, "用户不存在", err.Error())
		return
	}

	// 3. 获取助手配置
	var assistant models.Assistant
	if err := h.db.First(&assistant, req.AssistantID).Error; err != nil {
		response.Fail(c, "助手不存在", err.Error())
		return
	}

	// 验证助手是否属于该用户
	if assistant.UserID != user.ID {
		response.Fail(c, "无权访问该助手", "助手不属于当前用户")
		return
	}

	// 4. 检查LLM配置
	if credential.LLMProvider == "" || credential.LLMApiKey == "" {
		response.Fail(c, "LLM未配置", "请先配置LLM服务")
		return
	}

	// 5. 构建系统提示词
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		if assistant.SystemPrompt != "" {
			systemPrompt = assistant.SystemPrompt
		} else {
			systemPrompt = "请用中文回复用户的问题。"
		}
	}

	// 6. 设置参数（优先使用请求参数，其次使用助手配置）
	var temp *float32
	var maxTokens *int

	if req.Temperature > 0 {
		temp = &req.Temperature
	} else if assistant.Temperature > 0 {
		temp = &assistant.Temperature
	} else {
		defaultTemp := float32(0.7)
		temp = &defaultTemp
	}

	if req.MaxTokens > 0 {
		maxTokens = &req.MaxTokens
	} else if assistant.MaxTokens > 0 {
		maxTokens = &assistant.MaxTokens
	}

	// 7. 获取LLM模型
	llmModel := assistant.LLMModel
	if llmModel == "" {
		llmModel = utils.GetEnv("LLM_MODEL")
	}
	if llmModel == "" {
		llmModel = "deepseek-v3.1"
	}

	// 8. 如果开启了图记忆功能，添加用户偏好
	if config.GlobalConfig.Services.GraphMemory.Neo4j.Enabled && assistant.EnableGraphMemory {
		if store := graph.GetDefaultStore(); store != nil {
			ctx := c.Request.Context()
			if userCtx, err := store.GetUserContext(ctx, user.ID, int64(req.AssistantID)); err == nil {
				if len(userCtx.Topics) > 0 {
					preferenceText := fmt.Sprintf("该用户在历史对话中经常讨论这些主题：%s。请在回答时优先从这些兴趣和习惯的角度来组织内容，让风格尽量贴近他的偏好。",
						strings.Join(userCtx.Topics, "、"))
					systemPrompt = systemPrompt + "\n\n" + preferenceText
				}
			}
		}
	}

	// 9. 添加长度限制提示
	if maxTokens != nil && *maxTokens > 0 {
		estimatedChars := *maxTokens * 3 / 2
		lengthGuidance := fmt.Sprintf("\n\n重要提示：你的回复有长度限制（约 %d 个字符），请确保在限制内完整回答。", estimatedChars)
		systemPrompt = systemPrompt + lengthGuidance
	}

	// 10. 初始化LLM处理器
	llmHandler, err := v2.NewLLMProvider(c.Request.Context(), credential.LLMProvider, credential.LLMApiKey, credential.LLMApiURL, systemPrompt)
	if err != nil {
		response.Fail(c, "初始化LLM失败", err.Error())
		return
	}

	// 11. 构建查询文本
	queryText := req.Text

	// 12. 生成会话ID
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("simple_text_%d_%d", user.ID, time.Now().Unix())
	}

	// 13. 调用LLM
	userID := user.ID
	assistantID := int64(req.AssistantID)
	credentialID := credential.ID
	llmResponse, err := llmHandler.QueryWithOptions(queryText, v2.QueryOptions{
		Model:            llmModel,
		Temperature:      temp,
		MaxTokens:        maxTokens,
		UserID:           &userID,
		AssistantID:      &assistantID,
		CredentialID:     &credentialID,
		SessionID:        sessionID,
		ChatType:         models.ChatTypeText,
		EnableJSONOutput: assistant.EnableJSONOutput,
	})
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no available channels") || strings.Contains(errMsg, "model") {
			response.Fail(c, "模型不可用", fmt.Sprintf("模型 %s 当前不可用，请检查模型配置。", llmModel))
		} else {
			response.Fail(c, "LLM处理失败", errMsg)
		}
		return
	}

	// 14. 返回结果
	response.Success(c, "对话成功", SimpleTextChatResponse{
		Text:      llmResponse,
		SessionID: sessionID,
	})
}

// PlainText 处理纯文本对话（不进行TTS合成，用于调试）
func (h *Handlers) PlainText(c *gin.Context) {
	var req OneShotTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 1. 检查助手是否存在
	var assistant models.Assistant
	if err := h.db.First(&assistant, req.AssistantID).Error; err != nil {
		response.Fail(c, "助手不存在", "请检查助手ID是否正确")
		return
	}

	// 2. 查询用户凭证配置
	credential, err := models.GetUserCredentialByApiSecretAndApiKey(h.db, req.APIKey, req.APISecret)
	if err != nil {
		response.Fail(c, "查询凭证失败", err.Error())
		return
	}
	if credential == nil {
		response.Fail(c, "凭证不存在", "无效的 apiKey 或 apiSecret")
		return
	}

	// 3. 获取用户信息
	var user models.User
	if err := h.db.First(&user, credential.UserID).Error; err != nil {
		response.Fail(c, "用户不存在", err.Error())
		return
	}

	// 4. 判断是否是该用户的助手
	if assistant.UserID != user.ID {
		response.Fail(c, "无权限", "请检查助手ID是否正确")
		return
	}

	// 5. 调用LLM处理文本
	var llmResponse string
	var errLLM error
	if credential.LLMProvider != "" && credential.LLMApiURL != "" {
		llmBaseURL := credential.LLMApiURL
		if llmBaseURL == "" {
			llmBaseURL = utils.GetEnv("LLM_BASE_URL")
		}
		systemPrompt := req.SystemPrompt
		if systemPrompt == "" {
			systemPrompt = assistant.SystemPrompt
			if systemPrompt == "" {
				systemPrompt = "请用中文回复用户。"
			}
		}

		// 如果开启了图记忆功能，则尝试从 Neo4j 中获取该用户的长期偏好主题，并拼接到系统提示词中
		if config.GlobalConfig.Services.GraphMemory.Neo4j.Enabled && assistant.EnableGraphMemory {
			if store := graph.GetDefaultStore(); store != nil {
				ctx := c.Request.Context()
				if userCtx, err := store.GetUserContext(ctx, user.ID, int64(req.AssistantID)); err == nil {
					if len(userCtx.Topics) > 0 {
						preferenceText := fmt.Sprintf("该用户在历史对话中经常讨论这些主题：%s。请在回答时优先从这些兴趣和习惯的角度来组织内容，让风格尽量贴近他的偏好。",
							strings.Join(userCtx.Topics, "、"))
						systemPrompt = systemPrompt + "\n\n" + preferenceText
					}
				}
			}
		}

		llmHandler, err := v2.NewLLMProvider(c.Request.Context(), credential.LLMProvider, credential.LLMApiKey, credential.LLMApiURL, systemPrompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  fmt.Sprintf("初始化LLM失败: %v", err),
			})
			return
		}

		// 获取模型，优先级：Assistant配置 > 环境变量 > 默认值
		llmModel := assistant.LLMModel
		if llmModel == "" {
			llmModel = utils.GetEnv("LLM_MODEL")
		}
		if llmModel == "" {
			llmModel = "deepseek-v3.1" // 默认值
		}

		queryText := req.Text

		// 优先使用请求中的 temperature 和 maxTokens，如果没有则从 assistant 中读取
		var temp *float32
		var maxTokens *int

		if req.Temperature > 0 {
			temp = &req.Temperature
		} else if req.AssistantID > 0 {
			// 从 assistant 中读取
			if assistant.Temperature > 0 {
				temp = &assistant.Temperature
			}
		}
		// 如果还是没有，使用默认值
		if temp == nil {
			defaultTemp := float32(0.7)
			temp = &defaultTemp
		}

		if req.MaxTokens > 0 {
			maxTokens = &req.MaxTokens
		} else if req.AssistantID > 0 {
			// 从 assistant 中读取
			if assistant.MaxTokens > 0 {
				maxTokens = &assistant.MaxTokens
			}
		}

		if maxTokens != nil && *maxTokens > 0 {
			estimatedChars := *maxTokens * 3 / 2
			lengthGuidance := fmt.Sprintf("\n\n重要提示：你的回复有长度限制（约 %d 个字符），请确保在限制内完整回答。如果回答较长，请优先保证回答的完整性和逻辑性，可以适当精简表述，但不要被截断。", estimatedChars)
			enhancedSystemPrompt := systemPrompt + lengthGuidance
			llmHandler.SetSystemPrompt(enhancedSystemPrompt)
		}

		userID := user.ID
		assistantID := int64(req.AssistantID)
		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = fmt.Sprintf("plain_text_%d_%d", user.ID, time.Now().Unix())
		}
		credentialID := credential.ID
		llmResponse, errLLM = llmHandler.QueryWithOptions(queryText, v2.QueryOptions{
			Model:            llmModel,
			Temperature:      temp,
			MaxTokens:        maxTokens,
			UserID:           &userID,
			AssistantID:      &assistantID,
			CredentialID:     &credentialID,
			SessionID:        sessionID,
			ChatType:         models.ChatTypeText,
			EnableJSONOutput: assistant.EnableJSONOutput,
		})
		if errLLM != nil {
			errMsg := errLLM.Error()
			if strings.Contains(errMsg, "no available channels") || strings.Contains(errMsg, "model") {
				response.Fail(c, "模型不可用", fmt.Sprintf("模型 %s 当前不可用，请检查模型配置或尝试其他模型。错误详情：%s", llmModel, errMsg))
			} else {
				response.Fail(c, "LLM处理失败", errMsg)
			}
			return
		}
	} else {
		// 如果没有配置LLM，直接返回原文本
		llmResponse = req.Text
	}

	// 返回成功响应
	response.Success(c, "处理成功", map[string]string{
		"text": llmResponse,
	})
}

// processAudioAsyncV2 异步处理音频合成（V2版本，使用用户凭证配置）
func (h *Handlers) processAudioAsyncV2(ctx context.Context, credential *models.UserCredential, userID uint, text, language, speaker string, voiceCloneID int, requestID string) {
	// 清理文本，移除Markdown格式符号
	text = cleanTextForTTS(text)

	// 确定使用的音色
	voiceType := speaker
	if voiceType == "" {
		// 如果未指定speaker，使用默认音色（根据provider选择）
		ttsProvider := credential.GetTTSProvider()
		switch strings.ToLower(ttsProvider) {
		case "qcloud", "tencent":
			voiceType = "601002" // 爱小辰 - 默认男声
		case "qiniu":
			voiceType = "qiniu_zh_female_tmjxxy"
		case "volcengine":
			voiceType = "BV700_streaming" // 火山引擎默认音色
		default:
			voiceType = "601002"
		}
	}

	// 如果使用了训练音色，优先使用训练音色（通过 voiceclone 服务）
	if voiceCloneID > 0 {
		// 1) 从数据库查询音色克隆信息
		var clone models.VoiceClone
		if err := h.db.Where("user_id = ? AND id = ? AND is_active = ?", userID, voiceCloneID, true).
			First(&clone).Error; err != nil {
			fmt.Printf("[V2] 音色克隆不存在或未激活: VoiceCloneID=%d, Error=%v\n", voiceCloneID, err)
			// 如果音色不存在，继续使用普通TTS合成
		} else if clone.AssetID == "" {
			fmt.Printf("[V2] 音色克隆未训练完成: VoiceCloneID=%d\n", voiceCloneID)
			// 如果音色未训练完成，继续使用普通TTS合成
		} else {
			// 2) 根据 provider 创建相应的 voiceclone 服务
			factory := voiceclone.NewFactory()
			var voiceCloneService voiceclone.VoiceCloneService
			var err error

			switch strings.ToLower(clone.Provider) {
			case "xunfei":
				voiceCloneService, err = factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
			case "volcengine":
				voiceCloneService, err = factory.CreateServiceFromEnv(voiceclone.ProviderVolcengine)
			default:
				fmt.Printf("[V2] 不支持的音色克隆提供商: %s\n", clone.Provider)
				err = fmt.Errorf("unsupported voice clone provider: %s", clone.Provider)
			}

			if err != nil {
				fmt.Printf("[V2] 创建音色克隆服务失败: %v\n", err)
				// 如果创建服务失败，继续使用普通TTS合成
			} else {
				// 3) 调用音色克隆流式合成接口
				synthesizeReq := &voiceclone.SynthesizeRequest{
					AssetID:  clone.AssetID,
					Text:     text,
					Language: language,
				}

				fmt.Printf("[V2] 使用音色克隆流式合成: VoiceCloneID=%d, AssetID=%s, Provider=%s\n",
					voiceCloneID, clone.AssetID, clone.Provider)

				// 确定采样率（用于后续处理）
				var sampleRate int
				if strings.ToLower(clone.Provider) == "xunfei" {
					sampleRate = 24000 // 讯飞默认 24000Hz
				} else {
					sampleRate = 8000 // 火山引擎默认 8000Hz
				}

				// 创建音频收集器（流式处理）
				var audioData []byte
				audioMu := sync.Mutex{}

				handler := &voiceCloneAudioCollector{
					onMessage: func(data []byte) {
						audioMu.Lock()
						audioData = append(audioData, data...)
						audioMu.Unlock()
					},
				}

				// 使用流式合成
				err := voiceCloneService.SynthesizeStream(ctx, synthesizeReq, handler)
				if err != nil {
					fmt.Printf("[V2] 音色克隆流式合成失败: %v\n", err)
					// 如果合成失败，继续使用普通TTS合成
				} else {
					audioMu.Lock()
					collectedAudio := audioData
					audioMu.Unlock()

					if len(collectedAudio) == 0 {
						fmt.Printf("[V2] 音色克隆音频数据为空\n")
						audioCacheMutex.Lock()
						audioProcessingCache[requestID] = AudioProcessResult{
							Status:   "failed",
							Text:     text,
							AudioURL: "",
						}
						audioCacheMutex.Unlock()
						return
					}

					// 4) 处理音频格式（注意采样率等参数）
					// 默认声道数和位深度
					channels := 1
					bitDepth := 16

					// 创建带WAV头的音频数据
					wavData, err := h.createWAVFile(collectedAudio, sampleRate, channels, bitDepth)
					if err != nil {
						fmt.Printf("[V2] 创建WAV文件失败: %v\n", err)
						audioCacheMutex.Lock()
						audioProcessingCache[requestID] = AudioProcessResult{
							Status:   "failed",
							Text:     text,
							AudioURL: "",
						}
						audioCacheMutex.Unlock()
						return
					}

					ttsKey := fmt.Sprintf("oneshot/v2_voiceclone_%d_%d.wav", userID, time.Now().Unix())
					reader, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
						Bucket:   config.GlobalConfig.Services.Storage.Bucket,
						Data:     wavData,
						Filename: ttsKey,
						Key:      ttsKey,
					})
					if err != nil {
						fmt.Printf("[V2] 保存音频失败: %v\n", err)
						audioCacheMutex.Lock()
						audioProcessingCache[requestID] = AudioProcessResult{
							Status:   "failed",
							Text:     text,
							AudioURL: "",
						}
						audioCacheMutex.Unlock()
						return
					}

					// 获取音频URL
					ttsAudioURL := reader.URL

					// 更新音色使用统计
					clone.UsageCount++
					now := time.Now()
					clone.LastUsedAt = &now
					h.db.Save(&clone)

					// 将音频URL存储到缓存中
					audioCacheMutex.Lock()
					audioProcessingCache[requestID] = AudioProcessResult{
						Status:   "completed",
						Text:     text,
						AudioURL: ttsAudioURL,
					}
					audioCacheMutex.Unlock()

					fmt.Printf("[V2] 音色克隆流式音频合成完成: %s (SampleRate=%d, AudioSize=%d)\n", ttsAudioURL, sampleRate, len(collectedAudio))
					return
				}
			}
		}
	}

	// 使用工厂方法创建TTS服务（普通TTS合成，作为fallback或默认方式）
	ttsProvider := credential.GetTTSProvider()
	if ttsProvider == "" {
		fmt.Printf("[V2] TTS provider 未配置\n")
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	// 构建灵活的配置 map
	ttsConfig := make(synthesizer.TTSCredentialConfig)
	ttsConfig["provider"] = ttsProvider

	// 从凭证配置中获取所有字段
	if credential.TtsConfig != nil {
		for key, value := range credential.TtsConfig {
			// 跳过 provider 字段，因为我们已经设置了
			if key != "provider" {
				ttsConfig[key] = value
			}
		}
	}

	// 设置音色类型（如果未在配置中设置）
	if _, exists := ttsConfig["voiceType"]; !exists && voiceType != "" {
		ttsConfig["voiceType"] = voiceType
	}
	// 兼容字段名
	if _, exists := ttsConfig["voice_type"]; !exists && voiceType != "" {
		ttsConfig["voice_type"] = voiceType
	}

	// 设置语言配置（如果提供了语言参数）
	// 语言代码应该已经是平台特定的格式（从配置文件中的code字段获取）
	if language != "" {
		// 如果配置中已有语言设置，优先使用配置中的（允许用户通过配置覆盖）
		// 从配置文件获取该平台使用的配置字段名
		configKey := getLanguageConfigKey(ttsProvider)

		// 检查是否已设置（支持多种字段名格式）
		exists := false
		switch configKey {
		case "languageBoost":
			_, exists = ttsConfig["languageBoost"]
			if !exists {
				_, exists = ttsConfig["language_boost"]
			}
		case "languageCode":
			_, exists = ttsConfig["languageCode"]
			if !exists {
				_, exists = ttsConfig["language_code"]
			}
		case "lan":
			_, exists = ttsConfig["lan"]
			if !exists {
				_, exists = ttsConfig["language"]
			}
		default:
			_, exists = ttsConfig["language"]
		}

		// 如果未设置，使用从配置文件获取的字段名设置
		if !exists {
			ttsConfig[configKey] = language
			// 同时设置兼容格式（下划线格式）
			if configKey == "languageCode" {
				ttsConfig["language_code"] = language
			} else if configKey == "languageBoost" {
				ttsConfig["language_boost"] = language
			}
		}
	}

	ttsService, err := synthesizer.NewSynthesisServiceFromCredential(ttsConfig)
	if err != nil {
		fmt.Printf("[V2] 无法创建TTS服务: %v\n", err)
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	if ttsService == nil {
		fmt.Printf("[V2] 无法创建TTS服务，配置不完整\n")
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	// 创建音频收集器
	var audioData []byte
	audioMu := sync.Mutex{}

	handler := &audioCollector{
		onMessage: func(data []byte) {
			audioMu.Lock()
			audioData = append(audioData, data...)
			audioMu.Unlock()
		},
	}

	// 调用TTS合成
	err = ttsService.Synthesize(ctx, handler, text)
	if err != nil {
		fmt.Printf("[V2] TTS合成失败: %v\n", err)
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	// 保存音频到存储
	if len(audioData) == 0 {
		fmt.Printf("[V2] 音频数据为空\n")
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	// 获取音频格式信息
	format := ttsService.Format()

	// 创建带WAV头的音频数据
	wavData, err := h.createWAVFile(audioData, format.SampleRate, format.Channels, format.BitDepth)
	if err != nil {
		fmt.Printf("[V2] 创建WAV文件失败: %v\n", err)
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	ttsKey := fmt.Sprintf("oneshot/v2_tts_%d_%d.wav", userID, time.Now().Unix())
	reader, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
		Bucket:   config.GlobalConfig.Services.Storage.Bucket,
		Data:     wavData,
		Filename: ttsKey,
		Key:      ttsKey,
	})
	if err != nil {
		fmt.Printf("[V2] 保存音频失败: %v\n", err)
		audioCacheMutex.Lock()
		audioProcessingCache[requestID] = AudioProcessResult{
			Status:   "failed",
			Text:     text,
			AudioURL: "",
		}
		audioCacheMutex.Unlock()
		return
	}

	// 获取音频URL
	ttsAudioURL := reader.URL

	// 将音频URL存储到缓存中
	audioCacheMutex.Lock()
	audioProcessingCache[requestID] = AudioProcessResult{
		Status:   "completed",
		Text:     text,
		AudioURL: ttsAudioURL,
	}
	audioCacheMutex.Unlock()

	fmt.Printf("[V2] 音频合成完成: %s\n", ttsAudioURL)
	ttsService.Close()
}

// voiceCloneAudioCollector 音色克隆音频收集器，实现 voiceclone.SynthesisHandler 接口
type voiceCloneAudioCollector struct {
	onMessage func([]byte)
}

func (a *voiceCloneAudioCollector) OnMessage(data []byte) {
	if a.onMessage != nil {
		a.onMessage(data)
	}
}

func (a *voiceCloneAudioCollector) OnTimestamp(timestamp voiceclone.SentenceTimestamp) {
	// 时间戳信息暂时不处理
}

// audioCollector 音频收集器，实现 SynthesisHandler 接口
type audioCollector struct {
	onMessage func([]byte)
}

func (a *audioCollector) OnMessage(data []byte) {
	if a.onMessage != nil {
		a.onMessage(data)
	}
}

func (a *audioCollector) OnTimestamp(timestamp synthesizer.SentenceTimestamp) {
	// 暂不处理时间戳
}

// createWAVFile 将PCM音频数据转换为WAV格式（添加WAV文件头）
func (h *Handlers) createWAVFile(pcmData []byte, sampleRate int, channels int, bitDepth int) ([]byte, error) {
	// 创建44字节的WAV头部
	header := make([]byte, 44)
	dataSize := len(pcmData)

	// RIFF header
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize)) // File size
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)                                     // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)                                      // Audio format (PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))                       // Number of channels
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))                     // Sample rate
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*channels*bitDepth/8)) // Byte rate
	binary.LittleEndian.PutUint16(header[32:34], uint16(channels*bitDepth/8))            // Block align
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitDepth))                       // Bits per sample

	// data chunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize)) // Data size

	// 合并头部和音频数据
	wavData := append(header, pcmData...)
	return wavData, nil
}

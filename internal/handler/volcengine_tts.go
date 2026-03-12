package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"strconv"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voiceclone"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// VolcengineTTSRequest represents Volcengine TTS request
type VolcengineTTSRequest struct {
	AssetID  string `json:"assetId" binding:"required"` // speaker_id
	Text     string `json:"text" binding:"required"`
	Language string `json:"language" binding:"required"`
	Key      string `json:"key,omitempty"` // Optional, specify storage path
}

// VolcengineTTSResponse represents Volcengine TTS response
type VolcengineTTSResponse struct {
	URL string `json:"url"`
}

// VolcengineSubmitAudioRequest represents submit audio request
type VolcengineSubmitAudioRequest struct {
	SpeakerID string `form:"speakerId" binding:"required"` // speaker_id from console
	Language  string `form:"language" binding:"required"`
}

// VolcengineQueryTaskRequest represents query task request
type VolcengineQueryTaskRequest struct {
	SpeakerID string `json:"speakerId" binding:"required"` // speaker_id
}

// VolcengineQueryTaskResponse represents query task response
type VolcengineQueryTaskResponse struct {
	SpeakerID  string `json:"speakerId"`
	Status     int    `json:"status"`     // 0=NotFound, 1=Training, 2=Success, 3=Failed, 4=Active
	TrainVID   string `json:"trainVid"`   // Training version
	AssetID    string `json:"assetId"`    // Voice ID (same as speaker_id)
	FailedDesc string `json:"failedDesc"` // Failure reason
	CreateTime int64  `json:"createTime"` // Creation time
}

// VolcengineSynthesize handles Volcengine voice synthesis
func (h *Handlers) VolcengineSynthesize(c *gin.Context) {
	var req VolcengineTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	// Find corresponding voice clone by assetId
	var clone models.VoiceClone
	if err := h.db.Where("user_id = ? AND asset_id = ? AND provider = ? AND is_active = ?",
		user.ID, req.AssetID, "volcengine", true).First(&clone).Error; err != nil {
		// If voice clone not found, still allow synthesis but don't save history
		logrus.WithError(err).Warn("volcengine: voice clone not found, synthesis will proceed without history")
	}

	// Generate storage path (use .wav format for browser playback)
	key := req.Key
	if key == "" {
		// Generate relative storage key, let storage layer decide external prefix
		key = "volcengine/" + req.AssetID + "_" + strconv.FormatInt(int64(len(req.Text)), 10)
	}

	// Call Volcengine synthesis (using voiceclone)
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderVolcengine)
	if err != nil {
		response.Fail(c, "Failed to initialize Volcengine service", err.Error())
		return
	}

	synthesizeReq := &voiceclone.SynthesizeRequest{
		AssetID:  req.AssetID, // Use speaker_id
		Text:     req.Text,
		Language: req.Language,
	}
	url, err := service.SynthesizeToStorage(c.Request.Context(), synthesizeReq, key)
	if err != nil {
		response.Fail(c, "Voice synthesis failed", err.Error())
		return
	}

	// If voice clone found, save synthesis history
	if clone.ID > 0 {
		// Record synthesis history
		synthesis := &models.VoiceSynthesis{
			UserID:       user.ID,
			VoiceCloneID: clone.ID,
			Text:         req.Text,
			Language:     req.Language,
			AudioURL:     url,
			Status:       "success",
		}
		if err := h.db.Create(synthesis).Error; err != nil {
			logrus.WithError(err).Error("volcengine: failed to save synthesis history")
			// Don't return error for history save failure, synthesis already succeeded
		} else {
			// Update voice clone usage statistics
			clone.IncrementUsage()
			if err := h.db.Save(&clone).Error; err != nil {
				logrus.WithError(err).Error("volcengine: failed to update voice clone usage")
			}
		}
	}

	response.Success(c, "Voice synthesis successful", VolcengineTTSResponse{URL: url})
}

// VolcengineSubmitAudio submits audio file for training
// Note: speaker_id needs to be obtained from Volcengine console
func (h *Handlers) VolcengineSubmitAudio(c *gin.Context) {
	var req VolcengineSubmitAudioRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	// Get uploaded file
	file, err := c.FormFile("audio")
	if err != nil {
		response.Fail(c, "Failed to get audio file", err.Error())
		return
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		response.Fail(c, "Failed to open audio file", err.Error())
		return
	}
	defer src.Close()

	// Submit audio (using voiceclone)
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderVolcengine)
	if err != nil {
		response.Fail(c, "Failed to initialize Volcengine service", err.Error())
		return
	}

	submitReq := &voiceclone.SubmitAudioRequest{
		TaskID:    req.SpeakerID, // Use speaker_id as TaskID
		TextID:    0,             // Not needed for Volcengine
		TextSegID: 0,             // Not needed for Volcengine
		AudioFile: src,
		Language:  req.Language,
	}
	err = service.SubmitAudio(c.Request.Context(), submitReq)
	if err != nil {
		response.Fail(c, "Failed to submit audio", err.Error())
		return
	}

	// Save configuration to database (if configured)
	h.saveVoiceCloneConfig("volcengine")

	response.Success(c, "Audio submitted successfully", map[string]interface{}{
		"speakerId": req.SpeakerID,
		"message":   "Audio submitted, please use speaker_id to query training status",
	})
}

// VolcengineQueryTask queries task status
func (h *Handlers) VolcengineQueryTask(c *gin.Context) {
	var req VolcengineQueryTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	user := models.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	// Query task status (using voiceclone)
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderVolcengine)
	if err != nil {
		response.Fail(c, "Failed to initialize Volcengine service", err.Error())
		return
	}

	status, err := service.QueryTaskStatus(c.Request.Context(), req.SpeakerID)
	if err != nil {
		response.Fail(c, "Failed to query task status", err.Error())
		return
	}

	// Convert status code
	var trainStatus int
	switch status.Status {
	case voiceclone.TrainingStatusInProgress:
		trainStatus = 1 // Training
	case voiceclone.TrainingStatusSuccess:
		trainStatus = 2 // Success
	case voiceclone.TrainingStatusFailed:
		trainStatus = 3 // Failed
	case voiceclone.TrainingStatusQueued:
		trainStatus = 0 // NotFound/Queued
	default:
		trainStatus = 0
	}

	// If training succeeded, save to VoiceClone table
	if trainStatus == 2 && status.AssetID != "" {
		// Volcengine doesn't have VoiceTrainingTask, need to create or find a virtual task first
		// Find or create virtual training task (use speaker_id as task_id)
		var task models.VoiceTrainingTask
		if err := h.db.Where("user_id = ? AND task_id = ?", user.ID, req.SpeakerID).First(&task).Error; err != nil {
			// Doesn't exist, create virtual task
			task = models.VoiceTrainingTask{
				UserID:   user.ID,
				TaskID:   req.SpeakerID,
				TaskName: fmt.Sprintf("Volcengine Voice %s", req.SpeakerID),
				Status:   models.TrainingStatusSuccess,
				AssetID:  status.AssetID,
				TrainVID: status.TrainVID,
			}
			if err := h.db.Create(&task).Error; err != nil {
				response.Fail(c, "Failed to create training task record", err.Error())
				return
			}
		} else {
			// Already exists, update status
			task.Status = models.TrainingStatusSuccess
			task.AssetID = status.AssetID
			task.TrainVID = status.TrainVID
			if err := h.db.Save(&task).Error; err != nil {
				response.Fail(c, "Failed to update training task record", err.Error())
				return
			}
		}

		// Use upsertVoiceClone to save voice record
		if err := h.upsertVoiceClone(c.Request.Context(), user.ID, &task, status.AssetID, status.TrainVID, "volcengine"); err != nil {
			response.Fail(c, "Failed to create voice record", err.Error())
			return
		}
	}

	response.Success(c, "Query task status successful", VolcengineQueryTaskResponse{
		SpeakerID:  status.TaskID,
		Status:     trainStatus,
		TrainVID:   status.TrainVID,
		AssetID:    status.AssetID, // speaker_id is asset_id
		FailedDesc: status.FailedDesc,
		CreateTime: status.CreatedAt.UnixMilli(),
	})
}

package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models/auth"
	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/synthesizer"
	"github.com/LingByte/lingllm/voiceclone"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type VolcengineTTSRequest struct {
	AssetID  string `json:"assetId" binding:"required"`
	Text     string `json:"text" binding:"required"`
	Language string `json:"language" binding:"required"`
	Key      string `json:"key,omitempty"`
}

type VolcengineTTSResponse struct {
	URL string `json:"url"`
}

type VolcengineSubmitAudioRequest struct {
	SpeakerID string `form:"speakerId" binding:"required"`
	Language  string `form:"language" binding:"required"`
}

type VolcengineQueryTaskRequest struct {
	SpeakerID string `json:"speakerId" binding:"required"`
}

type VolcengineQueryTaskResponse struct {
	SpeakerID  string `json:"speakerId"`
	Status     int    `json:"status"`
	TrainVID   string `json:"trainVid"`
	AssetID    string `json:"assetId"`
	FailedDesc string `json:"failedDesc"`
	CreateTime int64  `json:"createTime"`
}

func (h *Handlers) getVolcengineCloneConfig() map[string]interface{} {
	cfg := map[string]interface{}{
		"app_id":  utils.GetEnv("VOLCENGINE_CLONE_APP_ID"),
		"token":   utils.GetEnv("VOLCENGINE_CLONE_TOKEN"),
		"cluster": utils.GetEnv("VOLCENGINE_CLONE_CLUSTER"),
	}
	if cfg["cluster"] == "" {
		cfg["cluster"] = "volcano_icl"
	}
	if v := utils.GetEnv("VOLCENGINE_CLONE_VOICE_TYPE"); v != "" {
		cfg["voice_type"] = v
	}
	if v := utils.GetEnv("VOLCENGINE_CLONE_ENCODING"); v != "" {
		cfg["encoding"] = v
	}
	if v := utils.GetEnv("VOLCENGINE_CLONE_FRAME_DURATION"); v != "" {
		cfg["frame_duration"] = v
	}
	if v := utils.GetIntEnv("VOLCENGINE_CLONE_SAMPLE_RATE"); v > 0 {
		cfg["sample_rate"] = v
	}
	if v := utils.GetIntEnv("VOLCENGINE_CLONE_BIT_DEPTH"); v > 0 {
		cfg["bit_depth"] = v
	}
	if v := utils.GetIntEnv("VOLCENGINE_CLONE_CHANNELS"); v > 0 {
		cfg["channels"] = v
	}
	if v := utils.GetFloatEnv("VOLCENGINE_CLONE_SPEED_RATIO"); v > 0 {
		cfg["speed_ratio"] = v
	}
	if v := utils.GetIntEnv("VOLCENGINE_CLONE_TRAINING_TIMES"); v > 0 {
		cfg["training_times"] = v
	}
	return cfg
}

func (h *Handlers) VolcengineSynthesize(c *gin.Context) {
	var req VolcengineTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	groupIDs, gerr := svcmodels.MemberGroupIDs(h.db, user.ID)
	var clone svcmodels.VoiceClone
	if gerr != nil || len(groupIDs) == 0 {
		logrus.WithError(gerr).Warn("volcengine: no groups for user")
	} else if err := h.db.Where("group_id IN ? AND asset_id = ? AND provider = ? AND is_active = ?",
		groupIDs, req.AssetID, "volcengine", true).First(&clone).Error; err != nil {
		logrus.WithError(err).Warn("volcengine: voice clone not found")
	}

	key := req.Key
	if key == "" {
		key = "volcengine/" + req.AssetID + "_" + strconv.FormatInt(int64(len(req.Text)), 10)
	}

	cfg := h.getVolcengineCloneConfig()
	appID, _ := cfg["app_id"].(string)
	token, _ := cfg["token"].(string)
	cluster, _ := cfg["cluster"].(string)
	if cluster == "" {
		cluster = "volcano_icl"
	}

	engine, err := synthesizer.NewVolcengineCloneEngine(synthesizer.VolcengineCloneOption{
		AppID:       appID,
		AccessToken: token,
		Cluster:     cluster,
		AssetID:     req.AssetID,
		Rate:        16000,
		SourceRate:  24000,
		Streaming:   true,
	})
	if err != nil {
		response.Fail(c, "Failed to initialize Volcengine clone engine", err.Error())
		return
	}
	defer engine.Close()

	var audioBuf []byte
	collector := &audioCollector{onMessage: func(data []byte) { audioBuf = append(audioBuf, data...) }}
	if err := engine.Synthesize(c.Request.Context(), collector, req.Text); err != nil {
		response.Fail(c, "Voice synthesis failed", err.Error())
		return
	}

	wavData := pcmToWAV(audioBuf, 16000, 1, 16)
	if strings.HasSuffix(key, ".pcm") {
		key = strings.TrimSuffix(key, ".pcm") + ".wav"
	} else if !strings.HasSuffix(key, ".wav") {
		key = key + ".wav"
	}
	st := stores.Default()
	if err := st.Write(key, bytes.NewReader(wavData)); err != nil {
		response.Fail(c, "Failed to save audio", err.Error())
		return
	}
	url := strings.TrimSpace(st.PublicURL(key))

	if clone.ID > 0 {
		synthesis := &svcmodels.VoiceSynthesis{
			GroupID:      clone.GroupID,
			CreatedBy:    user.ID,
			VoiceCloneID: clone.ID,
			Text:         req.Text,
			Language:     req.Language,
			AudioURL:     url,
			Status:       "success",
		}
		if err := h.db.Create(synthesis).Error; err != nil {
			logrus.WithError(err).Error("volcengine: failed to save synthesis history")
		} else {
			clone.IncrementUsage()
			h.db.Save(&clone)
		}
	}

	response.Success(c, "Voice synthesis successful", VolcengineTTSResponse{URL: url})
}

func (h *Handlers) VolcengineSubmitAudio(c *gin.Context) {
	var req VolcengineSubmitAudioRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	file, err := c.FormFile("audio")
	if err != nil {
		response.Fail(c, "Failed to get audio file", err.Error())
		return
	}
	src, err := file.Open()
	if err != nil {
		response.Fail(c, "Failed to open audio file", err.Error())
		return
	}
	defer src.Close()

	cfg := h.getVolcengineCloneConfig()
	f := voiceclone.NewFactory()
	service, err := f.CreateService(&voiceclone.Config{
		Provider: voiceclone.ProviderVolcengine,
		Options:  cfg,
	})
	if err != nil {
		response.Fail(c, "Failed to initialize Volcengine service", err.Error())
		return
	}

	err = service.SubmitAudio(c.Request.Context(), &voiceclone.SubmitAudioRequest{
		TaskID:    req.SpeakerID,
		AudioFile: src,
		Language:  req.Language,
	})
	if err != nil {
		response.Fail(c, "Failed to submit audio", err.Error())
		return
	}

	response.Success(c, "Audio submitted successfully", map[string]interface{}{
		"speakerId": req.SpeakerID,
	})
}

func (h *Handlers) VolcengineQueryTask(c *gin.Context) {
	var req VolcengineQueryTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "Invalid parameters", err.Error())
		return
	}

	user := auth.CurrentUser(c)
	if user == nil {
		response.Fail(c, "Unauthorized", "User not logged in")
		return
	}

	cfg := h.getVolcengineCloneConfig()
	f := voiceclone.NewFactory()
	service, err := f.CreateService(&voiceclone.Config{
		Provider: voiceclone.ProviderVolcengine,
		Options:  cfg,
	})
	if err != nil {
		response.Fail(c, "Failed to initialize Volcengine service", err.Error())
		return
	}

	status, err := service.QueryTaskStatus(c.Request.Context(), req.SpeakerID)
	if err != nil {
		response.Fail(c, "Failed to query task status", err.Error())
		return
	}

	var trainStatus int
	switch status.Status {
	case voiceclone.TrainingStatusInProgress:
		trainStatus = 1
	case voiceclone.TrainingStatusSuccess:
		trainStatus = 2
	case voiceclone.TrainingStatusFailed:
		trainStatus = 3
	case voiceclone.TrainingStatusQueued:
		trainStatus = 0
	default:
		trainStatus = 0
	}

	if trainStatus == 2 && status.AssetID != "" {
		pg, perr := svcmodels.EnsurePersonalGroupForUser(h.db, user.ID)
		if perr != nil {
			response.Fail(c, "Failed to create training task record", perr.Error())
			return
		}
		var task svcmodels.VoiceTrainingTask
		if err := h.db.Where("group_id = ? AND task_id = ?", pg.ID, req.SpeakerID).First(&task).Error; err != nil {
			task = svcmodels.VoiceTrainingTask{
				GroupID:   pg.ID,
				CreatedBy: user.ID,
				TaskID:    req.SpeakerID,
				TaskName:  fmt.Sprintf("Volcengine Voice %s", req.SpeakerID),
				Status:    svcmodels.TrainingStatusSuccess,
				AssetID:   status.AssetID,
				TrainVID:  status.TrainVID,
			}
			h.db.Create(&task)
		} else {
			task.Status = svcmodels.TrainingStatusSuccess
			task.AssetID = status.AssetID
			task.TrainVID = status.TrainVID
			h.db.Save(&task)
		}
		h.upsertVoiceClone(c.Request.Context(), user.ID, &task, status.AssetID, status.TrainVID, "volcengine")
	}

	response.Success(c, "Query task status successful", VolcengineQueryTaskResponse{
		SpeakerID:  status.TaskID,
		Status:     trainStatus,
		TrainVID:   status.TrainVID,
		AssetID:    status.AssetID,
		FailedDesc: status.FailedDesc,
		CreateTime: status.CreatedAt.UnixMilli(),
	})
}

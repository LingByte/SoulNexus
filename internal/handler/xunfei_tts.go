package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/voiceclone"
	"github.com/gin-gonic/gin"
)

// XunfeiTTSRequest 讯飞TTS请求
type XunfeiTTSRequest struct {
	AssetID  string `json:"assetId" binding:"required"`
	Text     string `json:"text" binding:"required"`
	Language string `json:"language" binding:"required"`
	Key      string `json:"key,omitempty"` // 可选，指定存储路径
}

// XunfeiTTSResponse 讯飞TTS响应
type XunfeiTTSResponse struct {
	URL string `json:"url"`
}

// XunfeiCreateTaskRequest 创建训练任务请求
type XunfeiCreateTaskRequest struct {
	TaskName string `json:"taskName" binding:"required"`
	Sex      int    `json:"sex"`      // 1:男 2:女
	AgeGroup int    `json:"ageGroup"` // 1:儿童 2:青年 3:中年 4:中老年
	Language string `json:"language"` // zh, en, ja, ko, ru
}

// XunfeiCreateTaskResponse 创建训练任务响应
type XunfeiCreateTaskResponse struct {
	TaskID string `json:"taskId"`
}

// XunfeiSubmitAudioRequest 提交音频请求
type XunfeiSubmitAudioRequest struct {
	TaskID    string `form:"taskId" binding:"required"`
	TextID    int64  `form:"textId" binding:"required"`
	TextSegID int64  `form:"textSegId" binding:"required"`
	Language  string `form:"language" binding:"required"`
}

// XunfeiQueryTaskRequest 查询任务请求
type XunfeiQueryTaskRequest struct {
	TaskID string `json:"taskId" binding:"required"`
}

// XunfeiQueryTaskResponse 查询任务响应
type XunfeiQueryTaskResponse struct {
	TaskName     string `json:"taskName"`
	ResourceName string `json:"resourceName"`
	Sex          int    `json:"sex"`
	AgeGroup     int    `json:"ageGroup"`
	TrainVID     string `json:"trainVid"`
	AssetID      string `json:"assetId"`
	TrainID      string `json:"trainId"`
	AppID        string `json:"appId"`
	TrainStatus  int    `json:"trainStatus"`
	FailedDesc   string `json:"failedDesc"`
}

// XunfeiSynthesize 讯飞语音合成
func (h *Handlers) XunfeiSynthesize(c *gin.Context) {
	var req XunfeiTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 生成存储路径
	key := req.Key
	if key == "" {
		// 添加时间戳避免文件名冲突
		timestamp := time.Now().Format("20060102_150405")
		// 只生成相对存储 key，统一由存储层决定对外前缀
		key = "xunfei/" + req.AssetID + "_" + timestamp + "_" + strconv.FormatInt(int64(len(req.Text)), 10) + ".wav"
	}

	// 调用讯飞合成（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	synthesizeReq := &voiceclone.SynthesizeRequest{
		AssetID:  req.AssetID,
		Text:     req.Text,
		Language: req.Language,
	}
	url, err := service.SynthesizeToStorage(c.Request.Context(), synthesizeReq, key)
	if err != nil {
		response.Fail(c, "语音合成失败", err.Error())
		return
	}

	response.Success(c, "语音合成成功", XunfeiTTSResponse{URL: url})
}

// XunfeiCreateTask 创建训练任务
func (h *Handlers) XunfeiCreateTask(c *gin.Context) {
	var req XunfeiCreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 设置默认值
	if req.Sex == 0 {
		req.Sex = 1 // 默认男性
	}
	if req.AgeGroup == 0 {
		req.AgeGroup = 2 // 默认青年
	}
	if req.Language == "" {
		req.Language = "zh" // 默认中文
	}

	// 创建任务（使用 voiceclone）
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
		response.Fail(c, "创建训练任务失败", err.Error())
		return
	}

	taskID := createResp.TaskID

	response.Success(c, "创建训练任务成功", XunfeiCreateTaskResponse{TaskID: taskID})
}

// XunfeiSubmitAudio 提交音频文件
func (h *Handlers) XunfeiSubmitAudio(c *gin.Context) {
	var req XunfeiSubmitAudioRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("audio")
	if err != nil {
		response.Fail(c, "获取音频文件失败", err.Error())
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		response.Fail(c, "打开音频文件失败", err.Error())
		return
	}
	defer src.Close()

	// 提交音频（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	submitReq := &voiceclone.SubmitAudioRequest{
		TaskID:    req.TaskID,
		TextID:    req.TextID,
		TextSegID: req.TextSegID,
		AudioFile: src,
		Language:  req.Language,
	}
	err = service.SubmitAudio(c.Request.Context(), submitReq)
	if err != nil {
		response.Fail(c, "提交音频失败", err.Error())
		return
	}

	response.Success(c, "提交音频成功", nil)
}

// XunfeiQueryTask 查询任务状态
func (h *Handlers) XunfeiQueryTask(c *gin.Context) {
	var req XunfeiQueryTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	// 查询任务状态（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	status, err := service.QueryTaskStatus(c.Request.Context(), req.TaskID)
	if err != nil {
		response.Fail(c, "查询任务状态失败", err.Error())
		return
	}

	// 转换状态码
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

	response.Success(c, "查询任务状态成功", XunfeiQueryTaskResponse{
		TaskName:     status.TaskName,
		ResourceName: status.TaskName,
		Sex:          0, // 旧接口兼容
		AgeGroup:     0, // 旧接口兼容
		TrainVID:     status.TrainVID,
		AssetID:      status.AssetID, // xunfei 返回的音色ID
		TrainID:      status.TaskID,
		AppID:        "",
		TrainStatus:  trainStatus,
		FailedDesc:   status.FailedDesc,
	})
}

// XunfeiGetTrainingTexts 获取训练文本
func (h *Handlers) XunfeiGetTrainingTexts(c *gin.Context) {
	textIDStr := c.Query("textId")
	if textIDStr == "" {
		textIDStr = "5001" // 默认通用训练文本
	}

	textID, err := strconv.ParseInt(textIDStr, 10, 64)
	if err != nil {
		response.Fail(c, "文本ID格式错误", err.Error())
		return
	}

	// 获取训练文本（使用 voiceclone）
	factory := voiceclone.NewFactory()
	service, err := factory.CreateServiceFromEnv(voiceclone.ProviderXunfei)
	if err != nil {
		response.Fail(c, "初始化讯飞服务失败", err.Error())
		return
	}

	trainingText, err := service.GetTrainingTexts(c.Request.Context(), textID)
	if err != nil {
		response.Fail(c, "获取训练文本失败", err.Error())
		return
	}

	// 转换格式以兼容旧接口
	textSegs := make([]struct {
		SegID   interface{} `json:"segId"`
		SegText string      `json:"segText"`
	}, len(trainingText.Segments))
	for i, seg := range trainingText.Segments {
		textSegs[i] = struct {
			SegID   interface{} `json:"segId"`
			SegText string      `json:"segText"`
		}{
			SegID:   seg.SegID,
			SegText: seg.SegText,
		}
	}

	responseTexts := struct {
		TextID   int64  `json:"textId"`
		TextName string `json:"textName"`
		TextSegs []struct {
			SegID   interface{} `json:"segId"`
			SegText string      `json:"segText"`
		} `json:"textSegs"`
	}{
		TextID:   trainingText.TextID,
		TextName: trainingText.TextName,
		TextSegs: textSegs,
	}

	response.Success(c, "获取训练文本成功", responseTexts)
}

package handlers

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/mediagen"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	mediaGenOnce    sync.Once
	mediaGenService *mediagen.Service
	mediaGenWorker  *mediagen.Worker
)

func initMediaGen(db *gorm.DB) {
	mediaGenOnce.Do(func() {
		mediaGenService = mediagen.NewService(mediagen.LoadConfig())
		mediaGenWorker = mediagen.NewWorker(db, mediaGenService, 2)
		if mediaGenWorker != nil {
			logger.Info("mediagen worker started", zap.Int("workers", 2))
		}
	})
}

func mediaGenSvc() *mediagen.Service {
	if mediaGenService == nil {
		mediaGenService = mediagen.NewService(mediagen.LoadConfig())
	}
	return mediaGenService
}

func mediaWorker() *mediagen.Worker {
	return mediaGenWorker
}

func (h *Handlers) registerMediaGenerateRoutes(r *humax.Group) {
	initMediaGen(h.db)
	g := r.Group("media")
	{
		g.GET("/config", h.getMediaConfig)
		g.GET("/jobs", h.listMediaJobs)
		g.GET("/jobs/:jobId", h.getMediaJob)
		g.POST("/image/generate", h.createMediaImage)
		g.POST("/image/image-to-image", h.createMediaImageToImage)
		g.POST("/video/generate", h.createMediaVideo)
		g.POST("/video/image-to-video", h.createMediaImageToVideo)
		g.GET("/video/tasks/:taskId", h.getMediaVideoTask)
	}
}

type mediaImageGenerateReq struct {
	Prompt   string `json:"prompt"`
	Negative string `json:"negative"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Style    string `json:"style"`
	Category string `json:"category"`
}

type mediaVideoGenerateReq struct {
	Prompt        string `json:"prompt"`
	Ratio         string `json:"ratio"`
	Duration      int    `json:"duration"`
	Resolution    string `json:"resolution"`
	GenerateAudio *bool  `json:"generateAudio"`
	Watermark     bool   `json:"watermark"`
	Category      string `json:"category"`
	Motion        string `json:"motion"`
	FPS           string `json:"fps"`
}

func (h *Handlers) getMediaConfig(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, mediaGenSvc().AppConfig())
}

func (h *Handlers) listMediaJobs(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	rows, total, err := mediagen.ListMediaGenerateJobsPage(h.db, tid, c.Query("kind"), c.Query("status"), page, size)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]*mediagen.VideoTaskResult, 0, len(rows))
	for i := range rows {
		item := mediaGenSvc().JobToVideoResult(&rows[i])
		if item != nil {
			item.WithPublicURL(func(key string) string { return ginutil.UploadURL(c, key) })
			out = append(out, item)
		}
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

func (h *Handlers) getMediaJob(c *gin.Context) {
	tid := middleware.CurrentTenantID(c)
	jobID := strings.TrimSpace(c.Param("jobId"))
	row, err := mediagen.GetMediaGenerateJobForTenant(h.db, tid, jobID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Render(c, response.New(response.CodeNotFound, i18n.TGin(c, i18n.KeyNotFound)))
		return
	}
	if writeMediaGenError(c, err) {
		return
	}
	result := mediaGenSvc().JobToVideoResult(row)
	response.SuccessI18n(c, i18n.KeySuccess, result.WithPublicURL(func(key string) string {
		return ginutil.UploadURL(c, key)
	}))
}

func (h *Handlers) createMediaImage(c *gin.Context) {
	var req mediaImageGenerateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		response.Render(c, response.New(response.CodeBadRequest, mediagen.ErrEmptyPrompt.Error()))
		return
	}
	width, height := normalizeMediaImageSize(req.Width, req.Height)
	prompt := mediagen.EnrichPrompt(req.Prompt, req.Style, req.Negative)
	row, err := mediaGenSvc().CreateImageJobRecord(h.db, mediagen.CreateImageJobInput{
		TenantID:       middleware.CurrentTenantID(c),
		UserID:         middleware.AuthUserID(c),
		Kind:           mediagen.JobKindTextToImage,
		Prompt:         prompt,
		NegativePrompt: req.Negative,
		Width:          width,
		Height:         height,
		Style:          req.Style,
		Category:       req.Category,
	})
	if writeMediaGenError(c, err) {
		return
	}
	if mediaWorker() == nil {
		response.Render(c, response.New(response.CodeInternal, "mediagen worker unavailable"))
		return
	}
	if _, err := mediaWorker().EnqueueAsync(mediagen.WorkerJob{PublicID: row.PublicID}); writeMediaGenError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, &mediagen.VideoTaskCreateResult{
		TaskID: row.PublicID,
		JobID:  row.PublicID,
		Status: "queued",
	})
}

func (h *Handlers) createMediaImageToImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil || file == nil {
		response.Render(c, response.New(response.CodeBadRequest, mediagen.ErrInvalidImage.Error()))
		return
	}
	prompt := strings.TrimSpace(c.PostForm("prompt"))
	if prompt == "" {
		response.Render(c, response.New(response.CodeBadRequest, mediagen.ErrEmptyPrompt.Error()))
		return
	}
	width, _ := strconv.Atoi(c.PostForm("width"))
	height, _ := strconv.Atoi(c.PostForm("height"))
	width, height = normalizeMediaImageSize(width, height)
	style := c.PostForm("style")
	negative := c.PostForm("negative")
	prompt = mediagen.EnrichPrompt(prompt, style, negative)

	ref, err := mediagen.PrepareReferenceImage(mediaGenSvc().Storage(), file)
	if writeMediaGenError(c, err) {
		return
	}
	row, err := mediaGenSvc().CreateImageJobRecord(h.db, mediagen.CreateImageJobInput{
		TenantID:       middleware.CurrentTenantID(c),
		UserID:         middleware.AuthUserID(c),
		Kind:           mediagen.JobKindImageToImage,
		Prompt:         prompt,
		NegativePrompt: negative,
		Width:          width,
		Height:         height,
		Style:          style,
		Category:       c.PostForm("category"),
		Reference:      ref,
	})
	if writeMediaGenError(c, err) {
		return
	}
	if mediaWorker() == nil {
		response.Render(c, response.New(response.CodeInternal, "mediagen worker unavailable"))
		return
	}
	if _, err := mediaWorker().EnqueueAsync(mediagen.WorkerJob{PublicID: row.PublicID}); writeMediaGenError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, &mediagen.VideoTaskCreateResult{
		TaskID: row.PublicID,
		JobID:  row.PublicID,
		Status: "queued",
	})
}

func (h *Handlers) createMediaVideo(c *gin.Context) {
	var req mediaVideoGenerateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		response.Render(c, response.New(response.CodeBadRequest, mediagen.ErrEmptyPrompt.Error()))
		return
	}
	params, prompt := buildMediaVideoParams(req)
	row, err := mediaGenSvc().CreateVideoJobRecord(h.db, mediagen.CreateVideoJobInput{
		TenantID:      middleware.CurrentTenantID(c),
		UserID:        middleware.AuthUserID(c),
		Kind:          mediagen.JobKindTextToVideo,
		Prompt:        prompt,
		Ratio:         params.Ratio,
		Duration:      params.Duration,
		Resolution:    params.Resolution,
		GenerateAudio: params.GenerateAudio,
		Watermark:     params.Watermark,
		Category:      params.Category,
		Motion:        params.Motion,
		FPS:           params.FPS,
	})
	if writeMediaGenError(c, err) {
		return
	}
	if mediaWorker() == nil {
		response.Render(c, response.New(response.CodeInternal, "mediagen worker unavailable"))
		return
	}
	if _, err := mediaWorker().EnqueueAsync(mediagen.WorkerJob{PublicID: row.PublicID}); writeMediaGenError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, &mediagen.VideoTaskCreateResult{
		TaskID: row.PublicID,
		JobID:  row.PublicID,
		Status: "queued",
	})
}

func (h *Handlers) createMediaImageToVideo(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil || file == nil {
		response.Render(c, response.New(response.CodeBadRequest, mediagen.ErrInvalidImage.Error()))
		return
	}
	prompt := strings.TrimSpace(c.PostForm("prompt"))
	if prompt == "" {
		response.Render(c, response.New(response.CodeBadRequest, mediagen.ErrEmptyPrompt.Error()))
		return
	}
	req := mediaVideoGenerateReq{
		Prompt:     prompt,
		Ratio:      c.DefaultPostForm("ratio", "16:9"),
		Duration:   mediagen.ParseDurationSeconds(c.PostForm("duration")),
		Resolution: c.DefaultPostForm("resolution", "1080p"),
		Watermark:  c.PostForm("watermark") == "true",
		Motion:     c.PostForm("motion"),
		FPS:        c.PostForm("fps"),
		Category:   c.PostForm("category"),
	}
	if v := c.PostForm("generateAudio"); v != "" {
		b := v == "true" || v == "1"
		req.GenerateAudio = &b
	}
	params, enrichedPrompt := buildMediaVideoParams(req)

	firstRef, err := mediagen.PrepareReferenceImage(mediaGenSvc().Storage(), file)
	if writeMediaGenError(c, err) {
		return
	}
	var lastRef *mediagen.PreparedReference
	if lastFile, err := c.FormFile("lastImage"); err == nil && lastFile != nil {
		lastRef, err = mediagen.PrepareReferenceImage(mediaGenSvc().Storage(), lastFile)
		if writeMediaGenError(c, err) {
			return
		}
	}

	row, err := mediaGenSvc().CreateVideoJobRecord(h.db, mediagen.CreateVideoJobInput{
		TenantID:      middleware.CurrentTenantID(c),
		UserID:        middleware.AuthUserID(c),
		Kind:          mediagen.JobKindImageToVideo,
		Prompt:        enrichedPrompt,
		Ratio:         params.Ratio,
		Duration:      params.Duration,
		Resolution:    params.Resolution,
		GenerateAudio: params.GenerateAudio,
		Watermark:     params.Watermark,
		Category:      params.Category,
		Motion:        params.Motion,
		FPS:           params.FPS,
		FirstFrame:    firstRef,
		LastFrame:     lastRef,
	})
	if writeMediaGenError(c, err) {
		return
	}
	if mediaWorker() == nil {
		response.Render(c, response.New(response.CodeInternal, "mediagen worker unavailable"))
		return
	}
	if _, err := mediaWorker().EnqueueAsync(mediagen.WorkerJob{PublicID: row.PublicID}); writeMediaGenError(c, err) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, &mediagen.VideoTaskCreateResult{
		TaskID: row.PublicID,
		JobID:  row.PublicID,
		Status: "queued",
	})
}

func (h *Handlers) getMediaVideoTask(c *gin.Context) {
	taskID := strings.TrimSpace(c.Param("taskId"))
	if taskID == "" {
		response.Render(c, response.New(response.CodeBadRequest, i18n.TGin(c, i18n.KeyInvalidParams)))
		return
	}
	tid := middleware.CurrentTenantID(c)
	row, err := mediagen.GetMediaGenerateJobForTenant(h.db, tid, taskID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Render(c, response.New(response.CodeNotFound, "任务不存在"))
		return
	}
	if writeMediaGenError(c, err) {
		return
	}
	result := mediaGenSvc().JobToVideoResult(row)
	response.SuccessI18n(c, i18n.KeySuccess, result.WithPublicURL(func(key string) string {
		return ginutil.UploadURL(c, key)
	}))
}

func buildMediaVideoParams(req mediaVideoGenerateReq) (mediaVideoGenerateReq, string) {
	if req.Ratio == "" {
		req.Ratio = "16:9"
	}
	if req.Duration <= 0 {
		req.Duration = 5
	}
	if req.Duration < 5 {
		req.Duration = 5
	}
	if req.Duration > 10 {
		req.Duration = 10
	}
	if req.Resolution == "" {
		req.Resolution = "1080p"
	}
	prompt := mediagen.EnrichVideoPrompt(req.Prompt, req.Motion, req.FPS)
	return req, prompt
}

func normalizeMediaImageSize(width, height int) (int, int) {
	if width <= 0 {
		width = 1024
	}
	if height <= 0 {
		height = 1024
	}
	if width < 64 {
		width = 64
	}
	if height < 64 {
		height = 64
	}
	if width > 2048 {
		width = 2048
	}
	if height > 2048 {
		height = 2048
	}
	return width, height
}

func writeMediaGenError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, mediagen.ErrAPIKeyMissing),
		errors.Is(err, mediagen.ErrEmptyPrompt),
		errors.Is(err, mediagen.ErrInvalidImage):
		response.Render(c, response.New(response.CodeBadRequest, err.Error()))
	case errors.Is(err, mediagen.ErrNoImageData),
		errors.Is(err, mediagen.ErrNoTaskID):
		response.Render(c, response.New(response.CodeInternal, err.Error()))
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		response.Render(c, response.New(response.CodeProviderError, err.Error()))
	default:
		var upstream *mediagen.UpstreamError
		if errors.As(err, &upstream) {
			response.Render(c, response.New(response.CodeProviderError, upstream.Error()))
			return true
		}
		response.Render(c, response.Wrap(response.CodeInternal, "internal error", err))
	}
	return true
}

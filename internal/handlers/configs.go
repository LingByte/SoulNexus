package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	dto "github.com/LingByte/SoulNexus/internal/request"
	internalResp "github.com/LingByte/SoulNexus/internal/response"
	vo "github.com/LingByte/SoulNexus/internal/response"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/audit"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerSystemConfigRoutes(r *humax.Group) {
	g := r.Group("system-configs")
	g.Use(middleware.RequirePlatformAdmin())
	{
		g.GET("/route-policy/catalog", h.getRouteCatalog)
		g.GET("/route-policy", h.getRoutePolicy)
		g.PUT("/route-policy", h.updateRoutePolicy)
		g.GET("", h.listSystemConfigs)
		g.GET("/:id", h.getSystemConfig)
		g.POST("", h.createSystemConfig)
		g.PUT("/:id", h.updateSystemConfig)
		g.DELETE("/:id", h.deleteSystemConfig)
	}
}

// listSystemConfigs list system config
func (h *Handlers) listSystemConfigs(c *gin.Context) {
	page, size := ginutil.QueryPage(c, 100)
	list, total, err := models.ListPage(h.db, page, size, c.Query("search"))
	if ginutil.WriteInternalError(c, err) {
		return
	}
	out := make([]internalResp.ConfigResponse, 0, len(list))
	for _, row := range list {
		out = append(out, vo.NewConfigResponse(row))
	}
	ginutil.PageSuccess(c, out, total, page, size)
}

// getSystemConfig get system config by id
func (h *Handlers) getSystemConfig(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	row, err := models.GetByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, vo.NewConfigResponse(row))
}

// createSystemConfig create system config
func (h *Handlers) createSystemConfig(c *gin.Context) {
	var req dto.SystemConfigCreateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	row, err := models.Create(h.db, models.CreateInput{
		Key:      req.Key,
		Desc:     req.Desc,
		Value:    req.Value,
		Format:   req.Format,
		Autoload: req.Autoload,
		Public:   req.Public,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrConfigKeyExists):
			response.Render(c, response.NewI18n(response.CodeConflict, i18n.KeyConfigKeyExists))
		case errors.Is(err, apperror.ErrConfigKeyRequired):
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyConfigKeyRequired))
		case errors.Is(err, apperror.ErrConfigFormatInvalid):
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidFormat))
		default:
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		}
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action: constants.OpActionCreate, Resource: constants.OpResourceSystemConfig,
		ResourceID: row.ID, ResourceName: row.Key,
		Summary: fmt.Sprintf("Created system config %s", row.Key), Detail: audit.Redact(req),
	}, nil, row)
	response.SuccessI18n(c, i18n.KeySuccess, vo.NewConfigResponse(*row))
}

// updateSystemConfig update system config
func (h *Handlers) updateSystemConfig(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := models.GetByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	var req dto.SystemConfigUpdateReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	after, err := models.Update(h.db, id, models.UpdateInput{
		Desc: req.Desc, Value: req.Value, Format: req.Format,
		Autoload: req.Autoload, Public: req.Public,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrConfigNotFound):
			response.Render(c, response.Err(response.CodeNotFound))
		case errors.Is(err, apperror.ErrConfigFormatInvalid):
			response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyInvalidFormat))
		default:
			response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		}
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action: constants.OpActionUpdate, Resource: constants.OpResourceSystemConfig,
		ResourceID: id, ResourceName: before.Key,
		Summary: fmt.Sprintf("Updated system config %s", before.Key), Detail: audit.Redact(req),
	}, before, after)
	response.SuccessI18n(c, i18n.KeySuccess, vo.NewConfigResponse(*after))
}

// deleteSystemConfig delete system config
func (h *Handlers) deleteSystemConfig(c *gin.Context) {
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	before, err := models.GetByID(h.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	if err := models.Delete(h.db, id); err != nil {
		if errors.Is(err, apperror.ErrConfigNotFound) {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	h.recordOpChange(c, OpLogEntry{
		Action: constants.OpActionDelete, Resource: constants.OpResourceSystemConfig,
		ResourceID: id, ResourceName: before.Key,
		Summary: fmt.Sprintf("Deleted system config %s", before.Key),
	}, before, nil)
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"id": id})
}

// getRouteCatalog returns the full API Key route catalog for platform admins.
func (h *Handlers) getRouteCatalog(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"groups": models.AKSKCatalogGrouped(),
		"total":  len(models.AKSKRouteCatalog),
	})
}

// getRoutePolicy returns the platform-wide API Key HTTP route allowlist.
func (h *Handlers) getRoutePolicy(c *gin.Context) {
	policy := models.CurrentAKSKRoutePolicy()
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"enabled":  policy.Enabled,
		"routeIds": policy.RouteIDs,
	})
}

// updateRoutePolicy persists the API Key route allowlist to system config.
func (h *Handlers) updateRoutePolicy(c *gin.Context) {
	var req dto.RoutePolicyReq
	if !ginutil.BindJSON(c, &req) {
		return
	}
	policy := models.AKSKRoutePolicy{
		Enabled:  req.Enabled,
		RouteIDs: models.NormalizeAKSKRouteIDs(req.RouteIDs),
	}
	if policy.Enabled && len(policy.RouteIDs) == 0 {
		response.Render(c, response.NewI18n(response.CodeBadRequest, i18n.KeyRoutePolicyRequiresRoute))
		return
	}
	value, err := models.MarshalAKSKRoutePolicy(policy)
	if ginutil.WriteInternalError(c, err) {
		return
	}
	var row utils.Config
	err = h.db.Where("`key` = ?", constants.KEY_API_AKSK_ROUTE_POLICY).First(&row).Error
	if err != nil {
		if _, cerr := models.Create(h.db, models.CreateInput{
			Key:      constants.KEY_API_AKSK_ROUTE_POLICY,
			Desc:     "API Key 路由白名单（JSON）。enabled=true 时 routeIds 列出平台开放的接口 catalog id；默认全部关闭。",
			Value:    value,
			Format:   "json",
			Autoload: true,
			Public:   false,
		}); ginutil.WriteInternalError(c, cerr) {
			return
		}
	} else if _, uerr := models.Update(h.db, row.ID, models.UpdateInput{Value: &value}); ginutil.WriteInternalError(c, uerr) {
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"enabled":  policy.Enabled,
		"routeIds": policy.RouteIDs,
	})
}

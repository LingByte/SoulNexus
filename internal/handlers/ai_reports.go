package handlers

import (
	"net/http"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// registerAIReportRoutes mounts tenant-scoped AI report endpoints.
// Call analytics were removed; handlers return empty placeholders.
func (h *Handlers) registerAIReportRoutes(r *humax.Group) {
	g := r.Group("reports/ai")
	g.Use(middleware.RequireTenantPermissionAll(constants.PermAPIReportsRead))
	{
		g.GET("/analytics", h.getAICallAnalytics)
		g.GET("/analytics/caller-attributes", h.getAICallerAttributes)
		g.GET("/analytics/caller-export", h.exportAICallerAttributes)
		g.GET("/overview", h.getAIReportOverview)
		g.GET("/callin-analysis", h.getAIReportCallinAnalysis)
		g.GET("/transfer-analysis", h.getAIReportTransferAnalysis)
		g.GET("/assistant-analysis", h.getAIReportAssistantAnalysis)
		g.GET("/business-analysis", h.getAIReportBusinessAnalysis)
		g.GET("/callout-analysis", h.getAIReportCalloutAnalysis)
		g.GET("/key-findings", h.getAIReportKeyFindings)
		g.GET("", h.listAIReports)
		g.GET("/:id", h.getAIReport)
	}
}

func (h *Handlers) listAIReports(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": []any{}, "total": 0, "page": 1, "size": 20})
}

func (h *Handlers) getAIReport(c *gin.Context) {
	response.FailWithCode(c, http.StatusNotFound, "ai reports removed with telephony", nil)
}

func (h *Handlers) getAICallAnalytics(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{})
}

func (h *Handlers) getAICallerAttributes(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"items": []any{}, "total": 0, "page": 1, "size": 50})
}

func (h *Handlers) exportAICallerAttributes(c *gin.Context) {
	response.FailWithCode(c, http.StatusGone, "caller export removed with telephony", nil)
}

func (h *Handlers) getAIReportOverview(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"totalCalls": 0, "connectedRate": 0, "transferRate": 0, "totalMinutes": 0, "callTrend": []any{},
	})
}

func (h *Handlers) getAIReportCallinAnalysis(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{})
}

func (h *Handlers) getAIReportTransferAnalysis(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{})
}

func (h *Handlers) getAIReportAssistantAnalysis(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{})
}

func (h *Handlers) getAIReportBusinessAnalysis(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{})
}

func (h *Handlers) getAIReportKeyFindings(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"userCareAbout": []any{}, "knowledgeTopUnanswered": []any{}, "transferToHumanIssues": []any{},
	})
}

func (h *Handlers) getAIReportCalloutAnalysis(c *gin.Context) {
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{})
}

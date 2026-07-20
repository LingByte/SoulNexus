package handlers

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/humax"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) registerEmbedRoutes(r *humax.Group) {
	g := r.Group("lingecho/embed/v1")
	{
		g.GET("/embed.js", h.serveLingEchoEmbedJS)
		g.GET("/t/:jsSourceId/embed.js", h.serveJSTemplateEmbedJS)
		g.GET("/assets/sprite_idle.png", h.serveLingEchoSpriteIdle)
		g.GET("/assets/sprite_hello.png", h.serveLingEchoSpriteHello)
		g.GET("/assets/icon-lingyu.png", h.serveLingEchoIconLingyu)
	}
}

func (h *Handlers) serveLingEchoEmbedJS(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", SoulNexus.LingEchoEmbedJS)
}

func (h *Handlers) serveLingEchoSpriteIdle(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(http.StatusOK, "image/png", SoulNexus.LingEchoSpriteIdle)
}

func (h *Handlers) serveLingEchoSpriteHello(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(http.StatusOK, "image/png", SoulNexus.LingEchoSpriteHello)
}

func (h *Handlers) serveLingEchoIconLingyu(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(http.StatusOK, "image/png", SoulNexus.LingEchoIconLingyu)
}

func (h *Handlers) serveJSTemplateEmbedJS(c *gin.Context) {
	jsSourceID := strings.TrimSpace(c.Param("jsSourceId"))
	if jsSourceID == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	row, err := models.GetActiveJSTemplateByJsSourceID(h.db, jsSourceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	// Best-effort load audit for mini-program / APP / H5 inject.
	_ = models.RecordJSTemplateUsage(h.db, row.TenantID, jsSourceID, models.JSTemplateUsageLoad, "", 0, 0)
	// Inject source id so stock embed.js (or custom scripts) can attribute sessions.
	boot := fmt.Sprintf(
		"(function(g){try{g.__LINGECHO_JS_SOURCE_ID__=%q;}catch(e){}})(typeof globalThis!=='undefined'?globalThis:window);\n",
		jsSourceID,
	)
	payload := append([]byte(boot), []byte(row.Content)...)
	c.Header("Cache-Control", "public, max-age=60")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", payload)
}

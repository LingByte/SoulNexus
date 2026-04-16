package handlers

import (
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/sip/webseat"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registerLingechoWebSeatRoutes(r *gin.RouterGroup) {
	g := r.Group(constants.LingechoWebSeatPathPrefix)
	{
		g.POST("/join", gin.WrapF(webseat.JoinHTTP))
		g.POST("/hangup", gin.WrapF(webseat.HangupHTTP))
		g.POST("/reject", gin.WrapF(webseat.RejectHTTP))
		g.GET("/ws", gin.WrapF(webseat.WebSocketHTTP))
		g.GET("/status/:callId", h.lingechoWebSeatStatus)
	}
}

func (h *Handlers) lingechoWebSeatStatus(c *gin.Context) {
	callID := strings.TrimSpace(c.Param("callId"))
	if callID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"call_id":           callID,
		"pending_or_active": webseat.IsPendingOrActive(callID),
	})
}

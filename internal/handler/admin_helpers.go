package handlers

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gin-gonic/gin"
)

func operatorEmail(c *gin.Context) string {
	if cur := models.CurrentUser(c); cur != nil && cur.Email != "" {
		return cur.Email
	}
	return "system"
}

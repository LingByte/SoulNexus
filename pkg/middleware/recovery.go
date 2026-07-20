package middleware

import (
	"net/http"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PanicRecovery logs panics, increments the ops counter, and returns 500.
func PanicRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		system.IncPanic()
		logger.Error("panic recovered",
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}

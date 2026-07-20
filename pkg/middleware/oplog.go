package middleware

import "github.com/gin-gonic/gin"

const ctxKeyOperationLogged = "middleware.operation_logged"

// MarkOperationLogged records that this HTTP request already wrote an operation log.
func MarkOperationLogged(c *gin.Context) {
	if c == nil {
		return
	}
	c.Set(ctxKeyOperationLogged, true)
}

// OperationAlreadyLogged reports whether an operation log was written for this request.
func OperationAlreadyLogged(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(ctxKeyOperationLogged)
	return ok && v == true
}

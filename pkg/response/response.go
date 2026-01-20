package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"` // 状态码，通常为 200 表示成功，非 200 为错误码
	Message string      `json:"msg"`  // 响应的消息描述
	Data    interface{} `json:"data"` // 返回的数据，可以是任意类型
}

func Success(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  msg,
		"data": data,
	})
}

func Fail(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": 500,
		"msg":  msg,
		"data": data,
	})
}

func Result(context *gin.Context, httpStatus int, code int, msg string, data gin.H) {
	context.JSON(httpStatus, gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
}

func AbortWithStatus(c *gin.Context, httpStatus int) {
	c.AbortWithStatus(httpStatus)
}

func AbortWithStatusJSON(c *gin.Context, httpStatus int, err error) {
	c.AbortWithStatusJSON(httpStatus, gin.H{"error": err.Error()})
}

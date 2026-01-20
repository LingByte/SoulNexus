package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/gin-gonic/gin"
)

// 生成 HMAC 签名
func generateSignature(data, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// API 签名验证中间件
func SignVerifyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := config.GlobalConfig.APISecretKey
		if secret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured"})
			c.Abort()
			return
		}

		// 从请求头中获取签名
		signature := c.GetHeader("X-Signature")
		if signature == "" {
			signature = c.GetHeader("Signature")
			if signature == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Signature is missing"})
				c.Abort()
				return
			}
		}

		// 获取请求的时间戳
		timestampStr := c.GetHeader("X-Timestamp")
		if timestampStr == "" {
			timestampStr = c.DefaultQuery("timestamp", "")
			if timestampStr == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Timestamp is missing"})
				c.Abort()
				return
			}
		}

		// 验证时间戳有效性（防止重放攻击）
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format"})
			c.Abort()
			return
		}

		// 检查时间戳是否在合理范围内（例如15分钟内）
		currentTime := time.Now().Unix()
		if abs(currentTime-timestamp) > 15*60 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Request expired"})
			c.Abort()
			return
		}

		// 获取nonce防止重放攻击
		nonce := c.GetHeader("X-Nonce")
		if nonce == "" {
			nonce = c.DefaultQuery("nonce", "")
		}

		// 获取请求体内容
		var requestBody string
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			// 读取原始请求体
			bodyBytes, err := c.GetRawData()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
				c.Abort()
				return
			}

			// 将读取的请求体重新写回上下文，以便后续处理
			c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			requestBody = string(bodyBytes)
		}

		// 构建签名数据
		// 包含: HTTP方法 + 请求路径 + 请求体 + 时间戳 + nonce
		var signatureData strings.Builder
		signatureData.WriteString(c.Request.Method)
		signatureData.WriteString(c.Request.URL.Path)
		signatureData.WriteString(requestBody)
		signatureData.WriteString(timestampStr)
		if nonce != "" {
			signatureData.WriteString(nonce)
		}

		// 添加关键的自定义头部信息到签名数据中
		for key, values := range c.Request.Header {
			// 只包含以X-开头的自定义头部
			if strings.HasPrefix(strings.ToLower(key), "x-") &&
				!strings.EqualFold(key, "X-Signature") {
				for _, value := range values {
					signatureData.WriteString(key)
					signatureData.WriteString(":")
					signatureData.WriteString(value)
				}
			}
		}

		// 生成期望的签名
		expectedSignature := generateSignature(signatureData.String(), config.GlobalConfig.APISecretKey)

		// 使用时间常数比较防止时序攻击
		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			c.Abort()
			return
		}

		// 签名验证通过，继续处理请求
		c.Next()
	}
}

// abs 返回绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

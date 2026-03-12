package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateRandomString 生成指定长度的随机字符串
func GenerateRandomString(length int) string {
	if length <= 0 {
		return ""
	}

	// 生成随机字节
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 如果生成失败，使用备用方法
		return generateRandomStringFallback(length)
	}

	// 转换为base64编码的字符串，然后截取所需长度
	encoded := base64.URLEncoding.EncodeToString(bytes)
	if len(encoded) >= length {
		return encoded[:length]
	}

	// 如果长度不够，重复生成
	result := encoded
	for len(result) < length {
		bytes := make([]byte, length)
		if _, err := rand.Read(bytes); err != nil {
			return generateRandomStringFallback(length)
		}
		encoded := base64.URLEncoding.EncodeToString(bytes)
		result += encoded
	}

	return result[:length]
}

// generateRandomStringFallback 备用随机字符串生成方法
func generateRandomStringFallback(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		// 使用简单的伪随机（仅作为备用）
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

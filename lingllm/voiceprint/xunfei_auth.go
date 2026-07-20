// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"
)

// XunfeiAuth 讯飞认证
type XunfeiAuth struct {
	APIKey    string
	APISecret string
	Host      string
}

// NewXunfeiAuth 创建讯飞认证
func NewXunfeiAuth(apiKey, apiSecret, host string) *XunfeiAuth {
	return &XunfeiAuth{
		APIKey:    apiKey,
		APISecret: apiSecret,
		Host:      host,
	}
}

// GenerateAuthHeader 生成认证头
func (a *XunfeiAuth) GenerateAuthHeader(method, path string) (string, string, error) {
	// 生成RFC1123格式的日期
	date := time.Now().UTC().Format(time.RFC1123)

	// 构建签名原始字段
	requestLine := fmt.Sprintf("%s %s HTTP/1.1", method, path)
	signatureOrigin := fmt.Sprintf("host: %s\ndate: %s\n%s", a.Host, date, requestLine)

	// 使用HMAC-SHA256签名
	h := hmac.New(sha256.New, []byte(a.APISecret))
	h.Write([]byte(signatureOrigin))
	signatureSha := h.Sum(nil)

	// Base64编码签名
	signature := base64.StdEncoding.EncodeToString(signatureSha)

	// 构建authorization字符串
	authorizationOrigin := fmt.Sprintf(
		`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		a.APIKey,
		signature,
	)

	// Base64编码authorization
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	return authorization, date, nil
}

// BuildAuthURL 构建带认证参数的URL
func (a *XunfeiAuth) BuildAuthURL(baseURL, path string) (string, error) {
	authorization, date, err := a.GenerateAuthHeader("POST", path)
	if err != nil {
		return "", err
	}

	// 构建完整URL
	params := url.Values{}
	params.Set("authorization", authorization)
	params.Set("host", a.Host)
	params.Set("date", date)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	return fullURL, nil
}

// VerifyResponse 验证响应（可选，用于验证服务器响应的完整性）
func (a *XunfeiAuth) VerifyResponse(responseDate string) error {
	// 解析响应中的日期
	respTime, err := time.Parse(time.RFC1123, responseDate)
	if err != nil {
		return fmt.Errorf("invalid response date format: %w", err)
	}

	// 检查时间偏差（最大允许300秒）
	now := time.Now().UTC()
	diff := now.Sub(respTime)
	if diff < 0 {
		diff = -diff
	}

	if diff > 300*time.Second {
		return fmt.Errorf("response date too old or in future: %v", diff)
	}

	return nil
}

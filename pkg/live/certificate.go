package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

// UploadCertificateRequest 上传域名证书请求
type UploadCertificateRequest struct {
	CertificateID string `json:"certificateID"` // 证书 ID
	Domain        string `json:"domain"`        // 域名
	Cert          string `json:"cert"`          // 证书内容
	PriKey        string `json:"priKey"`        // 私钥内容
}

// CertificateResponse 证书操作响应（上传、删除、更新）
type CertificateResponse struct {
	Message   string `json:"message"`
	ConnectID string `json:"connectId"`
}

// CertificateInfo 证书信息
type CertificateInfo struct {
	CertificateID string `json:"certificateID"` // 证书 ID
	Domain        string `json:"domain"`        // 域名
	Cert          string `json:"cert"`          // 证书内容
	PriKey        string `json:"priKey"`        // 私钥内容
	NotBefore     int64  `json:"notBefore"`     // 证书生效时间，Unix timestamp 格式，单位 s
	NotAfter      int64  `json:"notAfter"`      // 证书过期时间，Unix timestamp 格式，单位 s
}

// UpdateCertificateRequest 更新证书请求
type UpdateCertificateRequest struct {
	Cert              string `json:"cert,omitempty"`              // 证书内容
	PriKey            string `json:"priKey,omitempty"`            // 私钥内容
	ExpireCheckEnable *bool  `json:"expireCheckEnable,omitempty"` // 是否启用过期检查
}

// UploadCertificate 上传域名证书
// bucketName: 空间名称
// req: 证书信息
func (c *BucketClient) UploadCertificate(bucketName string, req *UploadCertificateRequest) (*CertificateResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if req.CertificateID == "" {
		return nil, fmt.Errorf("certificateID cannot be empty")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}
	if req.Cert == "" {
		return nil, fmt.Errorf("cert cannot be empty")
	}
	if req.PriKey == "" {
		return nil, fmt.Errorf("priKey cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "POST"
	rawQuery := "domainCertificate"

	// 构建请求 body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:      method,
		Path:        path,
		RawQuery:    rawQuery,
		Host:        host,
		ContentType: "application/json",
		Body:        bodyBytes,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result CertificateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// DeleteCertificate 删除域名证书
// bucketName: 空间名称
// domain: 域名
// certName: 证书名称
func (c *BucketClient) DeleteCertificate(bucketName, domain, certName string) (*CertificateResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}
	if certName == "" {
		return nil, fmt.Errorf("certName cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "DELETE"
	rawQuery := fmt.Sprintf("domainCertificate&domain=%s&certName=%s", url.QueryEscape(domain), url.QueryEscape(certName))

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:   method,
		Path:     path,
		RawQuery: rawQuery,
		Host:     host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result CertificateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListCertificates 列举域名证书
// bucketName: 空间名称
// domain: 域名
func (c *BucketClient) ListCertificates(bucketName, domain string) ([]CertificateInfo, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := fmt.Sprintf("domainCertificate&domain=%s", url.QueryEscape(domain))

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:   method,
		Path:     path,
		RawQuery: rawQuery,
		Host:     host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应（返回的是数组）
	var result []CertificateInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return result, nil
}

// UpdateCertificate 更新证书
// bucketName: 空间名称
// domain: 域名
// certName: 证书名称
// req: 更新请求
func (c *BucketClient) UpdateCertificate(bucketName, domain, certName string, req *UpdateCertificateRequest) (*CertificateResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}
	if certName == "" {
		return nil, fmt.Errorf("certName cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "PATCH"
	rawQuery := fmt.Sprintf("domainCertificate&domain=%s&certName=%s", url.QueryEscape(domain), url.QueryEscape(certName))

	// 构建请求 body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:      method,
		Path:        path,
		RawQuery:    rawQuery,
		Host:        host,
		ContentType: "application/json",
		Body:        bodyBytes,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result CertificateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

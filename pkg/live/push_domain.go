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

// BindPushDomainRequest 绑定上行域名请求
type BindPushDomainRequest struct {
	Domain string `json:"domain"` // 域名
	Type   string `json:"type"`   // 域名类型：pushRtmp, whip, pushSrt
}

// BindPushDomainResponse 绑定上行域名响应
type BindPushDomainResponse struct {
	Domain       string `json:"domain"`
	CNAME        string `json:"cname"`
	CreationDate string `json:"creationDate"`
	LastModified string `json:"lastModified"`
	ConnectID    string `json:"connectId"`
	Type         string `json:"type"`
}

// UnbindPushDomainResponse 解绑上行域名响应
type UnbindPushDomainResponse struct {
	Message   string `json:"message"`
	ConnectID string `json:"connectId"`
}

// PushDomainAuthConfig 推流域名防盗链配置
type PushDomainAuthConfig struct {
	Type          string `json:"type,omitempty"`          // 防盗链类型
	Enable        bool   `json:"enable,omitempty"`        // 是否启用防盗链
	PrimaryKey    string `json:"primaryKey,omitempty"`    // 主密钥
	SecondaryKey  string `json:"secondaryKey,omitempty"`  // 从密钥
	ExpireSeconds int    `json:"expireSeconds,omitempty"` // 过期时间（秒）
}

// URLRewriteRule URL 重写规则
type URLRewriteRule struct {
	Pattern string `json:"pattern"` // 匹配规则
	Replace string `json:"replace"` // 替换规则
}

// IPLimitConfig IP 黑白名单配置
type IPLimitConfig struct {
	Whitelist []string `json:"whitelist,omitempty"` // 白名单
	Blacklist []string `json:"blacklist,omitempty"` // 黑名单
}

// PushDomainInfo 上行域名信息
type PushDomainInfo struct {
	Enable        bool                  `json:"enable"`
	Domain        string                `json:"domain"`
	CNAME         string                `json:"cname"`
	Type          string                `json:"type"`
	Auth          *PushDomainAuthConfig `json:"auth"`
	CertificateID string                `json:"certificateID"`
	CreationDate  string                `json:"creationDate"`
	LastModified  string                `json:"lastModified"`
	HTTPSEnable   bool                  `json:"httpsEnable"`
}

// ListPushDomainsResponse 列举上行域名响应
type ListPushDomainsResponse struct {
	ConnectID string           `json:"connectId"`
	Domains   []PushDomainInfo `json:"domains"`
}

// UpdatePushDomainConfigRequest 修改上行域名配置请求
type UpdatePushDomainConfigRequest struct {
	Enable        *bool                 `json:"enable,omitempty"`
	Type          string                `json:"type,omitempty"`
	Auth          *PushDomainAuthConfig `json:"auth,omitempty"`
	CertificateID string                `json:"certificateID,omitempty"`
	CNAME         string                `json:"cname,omitempty"`
	IPLimit       *IPLimitConfig        `json:"ipLimit,omitempty"`
	URLRewrites   []URLRewriteRule      `json:"urlRewrites,omitempty"`
	HTTPSEnable   *bool                 `json:"httpsEnable,omitempty"`
}

// PushDomainConfigResponse 上行域名配置响应
type PushDomainConfigResponse struct {
	Enable        bool                  `json:"enable"`
	BucketID      string                `json:"bucketId"`
	Region        string                `json:"region"`
	Domain        string                `json:"domain"`
	CNAME         string                `json:"cname"`
	Type          string                `json:"type"`
	Auth          *PushDomainAuthConfig `json:"auth"`
	CertificateID string                `json:"certificateID"`
	CreationDate  string                `json:"creationDate"`
	LastModified  string                `json:"lastModified"`
	IPLimit       *IPLimitConfig        `json:"ipLimit"`
	HTTPSEnable   bool                  `json:"httpsEnable"`
}

// BindPushDomain 绑定上行域名（推流域名）
func (c *BucketClient) BindPushDomain(bucketName string, req *BindPushDomainRequest) (*BindPushDomainResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "POST"
	rawQuery := "pushDomain"

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
	var result BindPushDomainResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// UnbindPushDomain 解绑上行域名（推流域名）
func (c *BucketClient) UnbindPushDomain(bucketName, domain string) (*UnbindPushDomainResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "DELETE"
	rawQuery := fmt.Sprintf("pushDomain&name=%s", url.QueryEscape(domain))

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
	var result UnbindPushDomainResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListPushDomains 列举上行域名（推流域名）
func (c *BucketClient) ListPushDomains(bucketName string) (*ListPushDomainsResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := "pushDomain"

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
	var result ListPushDomainsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// UpdatePushDomainConfig 修改上行域名配置
func (c *BucketClient) UpdatePushDomainConfig(bucketName, domain string, req *UpdatePushDomainConfigRequest) (*PushDomainConfigResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "PATCH"
	rawQuery := fmt.Sprintf("pushDomainConfig&name=%s", url.QueryEscape(domain))

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
	var result PushDomainConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// GetPushDomainConfig 获取上行域名配置
func (c *BucketClient) GetPushDomainConfig(bucketName, domain string) (*PushDomainConfigResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := fmt.Sprintf("pushDomainConfig&name=%s", url.QueryEscape(domain))

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
	var result PushDomainConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

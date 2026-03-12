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

// BindPlayDomainRequest 绑定下行域名请求
type BindPlayDomainRequest struct {
	Domain string `json:"domain"`         // 域名
	Type   string `json:"type,omitempty"` // 域名类型：liveRtmp, liveHls, liveDash, liveFlv, whep, live, liveSrt
}

// BindPlayDomainResponse 绑定下行域名响应
type BindPlayDomainResponse struct {
	Domain       string `json:"domain"`
	CNAME        string `json:"cname"`
	CreationDate string `json:"creationDate"`
	LastModified string `json:"lastModified"`
	ConnectID    string `json:"connectId"`
	Type         string `json:"type"`
}

// UnbindPlayDomainResponse 解绑下行域名响应
type UnbindPlayDomainResponse struct {
	Message   string `json:"message"`
	ConnectID string `json:"connectId"`
}

// PlayDomainAuthConfig 播放域名防盗链配置
type PlayDomainAuthConfig struct {
	Type          string `json:"type,omitempty"`          // 防盗链类型
	Enable        bool   `json:"enable,omitempty"`        // 是否启用防盗链
	PrimaryKey    string `json:"primaryKey,omitempty"`    // 主密钥
	SecondaryKey  string `json:"secondaryKey,omitempty"`  // 从密钥
	ExpireSeconds int    `json:"expireSeconds,omitempty"` // 过期时间（秒）
}

// PlayDomainInfo 下行域名信息
type PlayDomainInfo struct {
	Enable        bool                  `json:"enable"`
	Domain        string                `json:"domain"`
	CNAME         string                `json:"cname"`
	Type          string                `json:"type"`
	Auth          *PlayDomainAuthConfig `json:"auth"`
	CertificateID string                `json:"certificateID"`
	CreationDate  string                `json:"creationDate"`
	LastModified  string                `json:"lastModified"`
	HTTPSEnable   bool                  `json:"httpsEnable"`
}

// ListPlayDomainsResponse 列举下行域名响应
type ListPlayDomainsResponse struct {
	ConnectID string           `json:"connectId"`
	Domains   []PlayDomainInfo `json:"domains"`
}

// UpdatePlayDomainConfigRequest 修改下行域名配置请求
type UpdatePlayDomainConfigRequest struct {
	Type          string                `json:"type,omitempty"`
	Auth          *PlayDomainAuthConfig `json:"auth,omitempty"`
	CertificateID string                `json:"certificateID,omitempty"`
	HTTPSEnable   *bool                 `json:"httpsEnable,omitempty"`
}

// PlayDomainConfigResponse 下行域名配置响应
type PlayDomainConfigResponse struct {
	Auth            *PlayDomainAuthConfig  `json:"auth"`
	CertificateID   string                 `json:"certificateID"`
	CNAME           string                 `json:"cname"`
	ConnectID       string                 `json:"connectId"`
	CreationDate    string                 `json:"creationDate"`
	Domain          string                 `json:"domain"`
	Enable          bool                   `json:"enable"`
	HTTPSEnable     bool                   `json:"httpsEnable"`
	IPLimit         *IPLimitConfig         `json:"ipLimit"`
	LastModified    string                 `json:"lastModified"`
	RefererSecurity *RefererSecurityConfig `json:"refererSecurity"`
	Type            string                 `json:"type"`
}

// BindPlayDomain 绑定下行域名（播放域名）
func (c *BucketClient) BindPlayDomain(bucketName string, req *BindPlayDomainRequest) (*BindPlayDomainResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if req.Domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "POST"
	rawQuery := "domain"

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
	var result BindPlayDomainResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// UnbindPlayDomain 解绑下行域名（播放域名）
func (c *BucketClient) UnbindPlayDomain(bucketName, domain string) (*UnbindPlayDomainResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "DELETE"
	rawQuery := fmt.Sprintf("domain&name=%s", url.QueryEscape(domain))

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
	var result UnbindPlayDomainResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListPlayDomains 列举下行域名（播放域名）
func (c *BucketClient) ListPlayDomains(bucketName string) (*ListPlayDomainsResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := "domain"

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
	var result ListPlayDomainsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// UpdatePlayDomainConfig 修改下行域名配置
func (c *BucketClient) UpdatePlayDomainConfig(bucketName, domain string, req *UpdatePlayDomainConfigRequest) (*PlayDomainConfigResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "PATCH"
	rawQuery := fmt.Sprintf("domainConfig&name=%s", url.QueryEscape(domain))

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
	var result PlayDomainConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// GetPlayDomainConfig 获取下行域名配置
func (c *BucketClient) GetPlayDomainConfig(bucketName, domain string) (*PlayDomainConfigResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := fmt.Sprintf("domainConfig&name=%s", url.QueryEscape(domain))

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
	var result PlayDomainConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

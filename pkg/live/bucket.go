package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

const (
	// DefaultRegion 默认区域
	DefaultRegion = "cn-east-1"
	// DefaultBaseHost 默认基础域名
	DefaultBaseHost = "mls.cn-east-1.qiniumiku.com"
)

// CreateBucketResponse 创建空间响应
type CreateBucketResponse struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	Status       string `json:"status"`
	CreationDate string `json:"creationDate"`
	ConnectID    string `json:"connectId"`
}

// DeleteBucketResponse 删除空间响应
type DeleteBucketResponse struct {
	Message   string `json:"message"`
	ConnectID string `json:"connectId"`
}

// ListBucketsResponse 列举空间响应
type ListBucketsResponse struct {
	Buckets   []BucketInfo `json:"buckets"`
	ConnectID string       `json:"connectId"`
}

// BucketInfo 空间信息
type BucketInfo struct {
	Name         string `json:"name"`
	Region       string `json:"region"`
	Status       string `json:"status"` // Enabled, Disabled, Banned
	CreationDate string `json:"creationDate"`
	ConnectID    string `json:"connectId,omitempty"`
}

// UpdateBucketConfigRequest 修改空间配置请求
type UpdateBucketConfigRequest struct {
	Status          string                 `json:"status,omitempty"` // Enabled, Disabled, Banned
	Recording       *RecordingConfig       `json:"recording,omitempty"`
	CallbackURL     string                 `json:"callbackUrl,omitempty"`
	CallbackEnable  *bool                  `json:"callbackEnable,omitempty"`
	RefererSecurity *RefererSecurityConfig `json:"refererSecurity,omitempty"`
}

// BucketConfigResponse 空间配置响应
type BucketConfigResponse struct {
	CallbackEnable  bool                   `json:"callbackEnable"`
	CallbackURL     string                 `json:"callbackUrl"`
	Codec           interface{}            `json:"codec"`
	ConnectID       string                 `json:"connectId"`
	CreationDate    string                 `json:"creationDate"`
	LastModified    string                 `json:"lastModified"`
	Name            string                 `json:"name"`
	Recording       *RecordingConfig       `json:"recording"`
	RefererSecurity *RefererSecurityConfig `json:"refererSecurity"`
	Region          string                 `json:"region"`
	Status          string                 `json:"status"`
}

// RefererSecurityConfig 防盗链配置
type RefererSecurityConfig struct {
	Type         string   `json:"type,omitempty"` // "black", "white", ""
	List         []string `json:"list,omitempty"`
	IncludeEmpty bool     `json:"includeEmpty,omitempty"`
}

// RecordingConfig 录制配置
type RecordingConfig struct {
	Enable          bool            `json:"enable,omitempty"`
	ObjectBucket    string          `json:"objectBucket,omitempty"`
	ObjectZone      string          `json:"objectZone,omitempty"`
	ExpireDays      int             `json:"expireDays,omitempty"`
	SegmentDuration int             `json:"segmentDuration,omitempty"`
	Snapshot        *SnapshotConfig `json:"snapshot,omitempty"`
	Format          []string        `json:"format,omitempty"`
	PlaybackDomain  string          `json:"playbackDomain,omitempty"`
}

// SnapshotConfig 截图配置
type SnapshotConfig struct {
	Enable       bool   `json:"enable,omitempty"`
	NotifyURL    string `json:"notifyUrl,omitempty"`
	Pipeline     string `json:"pipeline,omitempty"`
	Bucket       string `json:"bucket,omitempty"`
	Interval     int    `json:"interval,omitempty"`
	Filename     string `json:"filename,omitempty"`
	Format       string `json:"format,omitempty"`
	RoundingGaps int    `json:"roundingGaps,omitempty"`
	ExpireDays   int    `json:"expireDays,omitempty"`
}

// BucketClient 七牛云直播空间管理客户端
type BucketClient struct {
	accessKey  string
	secretKey  string
	region     string
	baseHost   string
	httpClient *http.Client
}

// NewBucketClient 创建新的客户端
func NewBucketClient() (*BucketClient, error) {
	accessKey := utils.GetEnv("QINIU_ACCESS_KEY")
	secretKey := utils.GetEnv("QINIU_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("please set QINIU_ACCESS_KEY and QINIU_SECRET_KEY 环境变量")
	}

	return &BucketClient{
		accessKey:  accessKey,
		secretKey:  secretKey,
		region:     DefaultRegion,
		baseHost:   DefaultBaseHost,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// SetRegion 设置区域
func (c *BucketClient) SetRegion(region string) {
	c.region = region
	if region != "" {
		c.baseHost = fmt.Sprintf("mls.%s.qiniumiku.com", region)
	}
}

// SetHTTPClient 设置自定义 HTTP 客户端
func (c *BucketClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// CreateBucket 创建空间
// bucketName: 空间名称
func (c *BucketClient) CreateBucket(bucketName string) (*CreateBucketResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "PUT"

	// 构建请求 body
	bodyBytes, _ := json.Marshal(map[string]interface{}{})

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:      method,
		Path:        path,
		Host:        host,
		ContentType: "application/json",
		Body:        bodyBytes,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
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
	var result CreateBucketResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// DeleteBucket 删除空间
// bucketName: 空间名称
func (c *BucketClient) DeleteBucket(bucketName string) (*DeleteBucketResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "DELETE"

	// 构建请求 body
	bodyBytes, _ := json.Marshal(map[string]interface{}{})

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:      method,
		Path:        path,
		Host:        host,
		ContentType: "application/json",
		Body:        bodyBytes,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
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
	var result DeleteBucketResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListBuckets 列举空间
func (c *BucketClient) ListBuckets() (*ListBucketsResponse, error) {
	host := c.baseHost
	path := "/"
	method := "GET"

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method: method,
		Path:   path,
		Host:   host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return nil, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Authorization", token)

	// 发送请求
	resp, err := c.httpClient.Do(req)
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
	var result ListBucketsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// UpdateBucketConfig 修改空间配置
// bucketName: 空间名称
// config: 配置信息
func (c *BucketClient) UpdateBucketConfig(bucketName string, config *UpdateBucketConfigRequest) (*BucketConfigResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "PATCH"
	rawQuery := "config"

	// 构建请求 body
	bodyBytes, err := json.Marshal(config)
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
	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
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
	var result BucketConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// GetBucketConfig 获取空间配置
// bucketName: 空间名称
func (c *BucketClient) GetBucketConfig(bucketName string) (*BucketConfigResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := "config"

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
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
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
	var result BucketConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// BucketExists 检查空间是否存在
// bucketName: 空间名称
// 返回: true 表示存在，false 表示不存在，error 表示请求过程中发生的错误
func (c *BucketClient) BucketExists(bucketName string) (bool, error) {
	if bucketName == "" {
		return false, fmt.Errorf("bucket name cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := "/"
	method := "GET"
	rawQuery := "config"

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method:   method,
		Path:     path,
		RawQuery: rawQuery,
		Host:     host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return false, fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)

	// 创建 HTTP 请求
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return false, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取响应失败: %w", err)
	}

	// 如果状态码是 200-299，说明 bucket 存在
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	// 如果状态码是 404，说明 bucket 不存在
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// 其他状态码表示请求过程中发生了错误
	return false, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
}

package live

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

// CreateStreamResponse 创建流响应
type CreateStreamResponse struct {
	Key          string `json:"key"`
	CreationDate string `json:"creationDate"`
	ConnectID    string `json:"connectId"`
}

// StreamProfile 流信息
type StreamProfile struct {
	VideoCodec      string `json:"videoCodec"`
	VideoProfile    string `json:"videoProfile"`
	VideoRate       string `json:"videoRate"`
	VideoFps        string `json:"videoFps"`
	VideoResolution string `json:"videoResolution"`
	AudioCodec      string `json:"audioCodec"`
	AudioProfile    string `json:"audioProfile"`
	AudioRate       string `json:"audioRate"`
	AudioChannel    string `json:"audioChannel"`
	AudioSample     string `json:"audioSample"`
}

// StreamInfo 流详细信息
type StreamInfo struct {
	BucketID               string         `json:"bucketId"`
	Region                 string         `json:"region"`
	Key                    string         `json:"key"`
	RealKey                string         `json:"realKey"`
	CreationDate           string         `json:"creationDate"`
	Forbidden              bool           `json:"forbidden"`
	Status                 string         `json:"status"` // online / offline
	DisablePublisherBattle bool           `json:"disablePublisherBattle"`
	EnableBackupPublish    bool           `json:"enableBackupPublish"`
	IsConverted            bool           `json:"isConverted"`
	LastStartAt            *int64         `json:"lastStartAt,omitempty"`
	Domain                 string         `json:"domain,omitempty"`
	URL                    string         `json:"url,omitempty"`
	LocalAddr              string         `json:"localAddr,omitempty"`
	RemoteAddr             string         `json:"remoteAddr,omitempty"`
	StreamProfile          *StreamProfile `json:"streamProfile,omitempty"`
}

// ForbidStreamRequest 封禁流请求
type ForbidStreamRequest struct {
	ForbiddenTill int64 `json:"forbiddenTill"` // 禁播结束时间，Unix 秒级时间戳，0 永久禁播，-1 解除封禁
}

// ForbidStreamResponse 封禁流响应
type ForbidStreamResponse struct {
	Message   string `json:"message"`
	ConnectID string `json:"connectId"`
}

// ReleaseStreamResponse 解封流响应
type ReleaseStreamResponse struct {
	Message   string `json:"message"`
	ConnectID string `json:"connectId"`
}

// StreamListItem 流列表项
type StreamListItem struct {
	Key          string `json:"key"`
	Bucket       string `json:"bucket,omitempty"`
	Forbidden    bool   `json:"forbidden"`
	CreationDate int64  `json:"creationDate"`
	Status       string `json:"status"` // online / offline
	LastStartAt  int64  `json:"lastStartAt"`
	PublishURL   string `json:"publishUrl,omitempty"`
}

// ListStreamsResponse 列举流列表响应
type ListStreamsResponse struct {
	Items     []StreamListItem `json:"items"`
	Total     int              `json:"total"`
	TotalPage string           `json:"totalPage,omitempty"`
	PageIndex string           `json:"pageIndex,omitempty"`
	PageSize  string           `json:"pageSize,omitempty"`
}

// ListStreamsRequest 列举流列表请求参数
type ListStreamsRequest struct {
	Prefix   string // 流名前缀
	Offset   string // 游标，默认为 0
	Limit    string // 返回个数，默认为 10，最大 500
	Domain   string // 推流域名，bucketId 和 domain 必须二传一
	BucketID string // bucketId，bucketId 和 domain 必须二传一
	Start    *int64 // 最近一次推流开始时间（Unix 秒级时间戳）
	End      *int64 // 最近一次推流结束时间（Unix 秒级时间戳）
	IsForbid *bool  // 是否只查询被封禁的流
}

// CreateStream 创建流
// bucketName: 空间名称
// streamKey: 流名称
func (c *BucketClient) CreateStream(bucketName, streamKey string) (*CreateStreamResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if streamKey == "" {
		return nil, fmt.Errorf("stream key cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := fmt.Sprintf("/%s", url.PathEscape(streamKey))
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
	var result CreateStreamResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// GetStreamInfo 获取流信息
// bucketName: 空间名称
// streamKey: 流名称
func (c *BucketClient) GetStreamInfo(bucketName, streamKey string) (*StreamInfo, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if streamKey == "" {
		return nil, fmt.Errorf("stream key cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := fmt.Sprintf("/%s", url.PathEscape(streamKey))
	method := "GET"
	rawQuery := "info"

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

	// 解析响应
	var result StreamInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ForbidStream 封禁流
// bucketName: 空间名称
// streamKey: 流名称
// req: 封禁请求
func (c *BucketClient) ForbidStream(bucketName, streamKey string, req *ForbidStreamRequest) (*ForbidStreamResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if streamKey == "" {
		return nil, fmt.Errorf("stream key cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := fmt.Sprintf("/%s", url.PathEscape(streamKey))
	method := "POST"
	rawQuery := fmt.Sprintf("forbid&bucket=%s&streamKey=%s", url.QueryEscape(bucketName), url.QueryEscape(streamKey))

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
	var result ForbidStreamResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ReleaseStream 解封流
// bucketName: 空间名称
// streamKey: 流名称
func (c *BucketClient) ReleaseStream(bucketName, streamKey string) (*ReleaseStreamResponse, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if streamKey == "" {
		return nil, fmt.Errorf("stream key cannot be empty")
	}

	host := fmt.Sprintf("%s.%s", bucketName, c.baseHost)
	path := fmt.Sprintf("/%s", url.PathEscape(streamKey))
	method := "POST"
	rawQuery := "release"

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
	var result ReleaseStreamResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListStreams 列举流列表
// req: 请求参数
func (c *BucketClient) ListStreams(req *ListStreamsRequest) (*ListStreamsResponse, error) {
	host := c.baseHost
	path := "/"
	method := "GET"

	// 构建查询参数
	queryParams := url.Values{}
	queryParams.Set("streamlist", "")
	if req.Prefix != "" {
		queryParams.Set("prefix", req.Prefix)
	}
	if req.Offset != "" {
		queryParams.Set("offset", req.Offset)
	}
	if req.Limit != "" {
		queryParams.Set("limit", req.Limit)
	}
	if req.Domain != "" {
		queryParams.Set("domain", req.Domain)
	}
	if req.BucketID != "" {
		queryParams.Set("bucketId", req.BucketID)
	}
	if req.Start != nil {
		queryParams.Set("start", fmt.Sprintf("%d", *req.Start))
	}
	if req.End != nil {
		queryParams.Set("end", fmt.Sprintf("%d", *req.End))
	}
	if req.IsForbid != nil {
		queryParams.Set("isForbid", fmt.Sprintf("%t", *req.IsForbid))
	}

	rawQuery := queryParams.Encode()

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

	// 解析响应
	var result ListStreamsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// StreamExists 检查流是否存在
// bucketName: 空间名称
// streamKey: 流名称
// 返回: true 表示存在，false 表示不存在，error 表示请求过程中发生的错误
func (c *BucketClient) StreamExists(bucketName, streamKey string) (bool, error) {
	if bucketName == "" {
		return false, fmt.Errorf("bucket name cannot be empty")
	}
	if streamKey == "" {
		return false, fmt.Errorf("stream key cannot be empty")
	}

	// 尝试获取流信息
	_, err := c.GetStreamInfo(bucketName, streamKey)
	if err != nil {
		// 如果返回 404，说明流不存在
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		// 其他错误返回错误信息
		return false, err
	}

	// 成功获取到流信息，说明流存在
	return true, nil
}

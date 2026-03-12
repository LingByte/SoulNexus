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

	"github.com/LingByte/SoulNexus/pkg/utils/qiniu/auth"
)

const (
	// PubManagerHost Pub 转推服务域名
	PubManagerHost = "pub-manager.mikudns.com"
)

// SourceURL 源地址配置
type SourceURL struct {
	URL       string `json:"url"`                 // 源地址
	ISP       string `json:"isp,omitempty"`       // 运营商：电信，联通，移动
	Seek      int    `json:"seek,omitempty"`      // 起播偏移（秒）
	VideoType int    `json:"videoType,omitempty"` // 0: 直播 1: 点播
	RTSPType  int    `json:"rtspType,omitempty"`  // 0: rtsp udp, 1: rtsp tcp
}

// ForwardURL 转推地址配置
type ForwardURL struct {
	URL string `json:"url"`           // 转推地址
	ISP string `json:"isp,omitempty"` // 运营商：电信，联通，移动
}

// PubFilter 转推机器筛选配置
type PubFilter struct {
	Area string   `json:"area,omitempty"` // 地域：东北, 华北, 华中, 华东, 华南, 西北, 西南, 其它
	ISP  string   `json:"isp,omitempty"`  // 运营商：电信，联通，移动
	IPs  []string `json:"ips,omitempty"`  // IP 地址列表
}

// PreloadConfig 预加载配置
type PreloadConfig struct {
	Enable      bool  `json:"enable,omitempty"`      // 是否开启预加载
	PreloadTime int64 `json:"preloadTime,omitempty"` // 预加载开始时间，ms
}

// StatusCallbackConfig 任务状态回调配置
type StatusCallbackConfig struct {
	Type          string                 `json:"type,omitempty"`          // 请求类型：GET、FORM、JSON
	URL           string                 `json:"url,omitempty"`           // 回调通知地址
	Vars          map[string]interface{} `json:"vars,omitempty"`          // 自定义配置回调参数
	Timeout       int                    `json:"timeout,omitempty"`       // 超时时间
	RetryTimes    int                    `json:"retryTimes,omitempty"`    // 重试次数
	RetryInterval int                    `json:"retryInterval,omitempty"` // 重试间隔
}

// CreatePubTaskRequest 创建 Pub 转推任务请求
type CreatePubTaskRequest struct {
	Name             string                `json:"name"`                       // 任务名称，1-20 个数字或字母，单用户内唯一
	Desc             string                `json:"desc,omitempty"`             // 任务描述
	SourceURLs       []SourceURL           `json:"sourceUrls"`                 // 源地址列表
	RunType          string                `json:"runType"`                    // 任务类型：normal 普通转推，seek seek 转推
	ForwardURLs      []ForwardURL          `json:"forwardUrls"`                // 转推地址列表
	Filter           *PubFilter            `json:"filter,omitempty"`           // 转推机器筛选
	LoopTimes        int                   `json:"loopTimes,omitempty"`        // 循环次数，默认 0 不循环，-1 无限循环
	RetryTime        int                   `json:"retryTime,omitempty"`        // 总断流重试时间，单位：s，必须大于等于60
	DeliverStartTime *int64                `json:"deliverStartTime,omitempty"` // 定时开始时间（毫秒）
	DeliverStopTime  *int64                `json:"deliverStopTime,omitempty"`  // 定时结束时间（毫秒）
	Sustain          *bool                 `json:"sustain,omitempty"`          // 循环转推不中断
	Preload          *PreloadConfig        `json:"preload,omitempty"`          // 预加载配置
	StatusCallback   *StatusCallbackConfig `json:"statusCallback,omitempty"`   // 任务状态回调配置
}

// CreatePubTaskResponse 创建 Pub 转推任务响应
type CreatePubTaskResponse struct {
	TaskID string `json:"taskID"` // 任务 ID
}

// UpdatePubTaskRequest 编辑 Pub 转推任务请求
type UpdatePubTaskRequest struct {
	RunType          string                `json:"runType"`                    // 任务类型：normal 普通转推，seek seek 转推
	Desc             string                `json:"desc,omitempty"`             // 任务描述
	SourceURLs       []SourceURL           `json:"sourceUrls"`                 // 源地址列表
	ForwardURLs      []ForwardURL          `json:"forwardUrls"`                // 转推地址列表
	Filter           *PubFilter            `json:"filter,omitempty"`           // 转推机器筛选
	LoopTimes        int                   `json:"loopTimes,omitempty"`        // 循环次数
	RetryTime        int                   `json:"retryTime,omitempty"`        // 总断流重试时间
	DeliverStartTime *int64                `json:"deliverStartTime,omitempty"` // 定时开始时间（毫秒）
	DeliverStopTime  *int64                `json:"deliverStopTime,omitempty"`  // 定时结束时间（毫秒）
	Sustain          *bool                 `json:"sustain,omitempty"`          // 循环转推不中断
	Preload          *PreloadConfig        `json:"preload,omitempty"`          // 预加载配置
	StatusCallback   *StatusCallbackConfig `json:"statusCallback,omitempty"`   // 任务状态回调配置
}

// PubTaskInfo Pub 转推任务详细信息
type PubTaskInfo struct {
	TaskID           string                `json:"taskID"`
	UID              int                   `json:"uid"`
	Name             string                `json:"name"`
	Desc             string                `json:"desc"`
	RunType          string                `json:"runType"`
	SourceURLs       []SourceURL           `json:"sourceUrls"`
	ForwardURLs      []ForwardURL          `json:"forwardUrls"`
	Filter           *PubFilter            `json:"filter"`
	Status           string                `json:"status"` // pending/running/stopped/failed/finished
	DeliverStartTime int64                 `json:"deliverStartTime"`
	DeliverStopTime  int64                 `json:"deliverStopTime"`
	StartTime        int64                 `json:"startTime"`
	StopTime         int64                 `json:"stopTime"`
	CreateTime       int64                 `json:"createTime"`
	LoopTimes        int                   `json:"loopTimes"`
	RetryTime        int                   `json:"retryTime"`
	Retry            int                   `json:"retry"`
	Preload          *PreloadConfig        `json:"preload"`
	PreloadStatus    string                `json:"preloadStatus"` // running/failed/finished
	StatusCallback   *StatusCallbackConfig `json:"statusCallback"`
}

// ListPubTasksRequest 列举 Pub 转推任务列表请求参数
type ListPubTasksRequest struct {
	Marker string // 翻页游标
	Limit  int    // 返回数量上限，最大 1000
	Name   string // 任务名称/描述模糊匹配
}

// ListPubTasksResponse 列举 Pub 转推任务列表响应
type ListPubTasksResponse struct {
	Marker string        `json:"marker"`
	IsEnd  bool          `json:"isEnd"`
	List   []PubTaskInfo `json:"list"`
}

// PubTaskHistoryItem 任务历史记录项
type PubTaskHistoryItem struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	StopTime  int64  `json:"stopTime"`
	Message   string `json:"message"`
}

// ListPubTaskHistoryRequest 查询任务历史记录请求参数
type ListPubTaskHistoryRequest struct {
	Marker string // 翻页游标
	Limit  int    // 返回数量上限，最大 1000
	Name   string // 任务名称前缀匹配
	Start  *int64 // 开始时间，毫秒
	End    *int64 // 结束时间，毫秒
}

// ListPubTaskHistoryResponse 查询任务历史记录响应
type ListPubTaskHistoryResponse struct {
	Marker    string               `json:"marker"`
	IsEnd     bool                 `json:"isEnd"`
	Histories []PubTaskHistoryItem `json:"histories"`
}

// PubTaskRunInfoResponse 任务运行日志响应
type PubTaskRunInfoResponse struct {
	Out string `json:"out"` // 运行日志
}

// CreatePubTask 创建 Pub 转推任务
func (c *BucketClient) CreatePubTask(req *CreatePubTaskRequest) (*CreatePubTaskResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("task name cannot be empty")
	}
	if len(req.SourceURLs) == 0 {
		return nil, fmt.Errorf("sourceUrls cannot be empty")
	}
	if len(req.ForwardURLs) == 0 {
		return nil, fmt.Errorf("forwardUrls cannot be empty")
	}
	if req.RunType == "" {
		return nil, fmt.Errorf("runType cannot be empty")
	}

	host := PubManagerHost
	path := "/tasks"
	method := "POST"

	// 构建请求 body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

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
	var result CreatePubTaskResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// UpdatePubTask 编辑 Pub 转推任务
func (c *BucketClient) UpdatePubTask(taskID string, req *UpdatePubTaskRequest) error {
	if taskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}
	if len(req.SourceURLs) == 0 {
		return fmt.Errorf("sourceUrls cannot be empty")
	}
	if len(req.ForwardURLs) == 0 {
		return fmt.Errorf("forwardUrls cannot be empty")
	}
	if req.RunType == "" {
		return fmt.Errorf("runType cannot be empty")
	}

	host := PubManagerHost
	path := fmt.Sprintf("/tasks/%s", url.PathEscape(taskID))
	method := "POST"

	// 构建请求 body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

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
		return fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// StartPubTask 开始 Pub 转推任务
func (c *BucketClient) StartPubTask(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}

	host := PubManagerHost
	path := fmt.Sprintf("/tasks/%s/start", url.PathEscape(taskID))
	method := "POST"

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method: method,
		Path:   path,
		Host:   host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// StopPubTask 停止 Pub 转推任务
func (c *BucketClient) StopPubTask(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}

	host := PubManagerHost
	path := fmt.Sprintf("/tasks/%s/stop", url.PathEscape(taskID))
	method := "POST"

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method: method,
		Path:   path,
		Host:   host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeletePubTask 删除 Pub 转推任务
func (c *BucketClient) DeletePubTask(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("taskID cannot be empty")
	}

	host := PubManagerHost
	path := fmt.Sprintf("/tasks/%s", url.PathEscape(taskID))
	method := "DELETE"

	// 生成鉴权 token
	authReq := auth.QiniuAuthRequest{
		Method: method,
		Path:   path,
		Host:   host,
	}

	token, err := auth.GenerateQiniuToken(c.accessKey, c.secretKey, authReq)
	if err != nil {
		return fmt.Errorf("生成鉴权 token 失败: %w", err)
	}

	// 构建请求 URL
	url := fmt.Sprintf("https://%s%s", host, path)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("Authorization", token)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 检查状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetPubTask 获取 Pub 转推任务详情
func (c *BucketClient) GetPubTask(taskID string) (*PubTaskInfo, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	host := PubManagerHost
	path := fmt.Sprintf("/tasks/%s", url.PathEscape(taskID))
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
	var result PubTaskInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListPubTasks 列举 Pub 转推任务列表
func (c *BucketClient) ListPubTasks(req *ListPubTasksRequest) (*ListPubTasksResponse, error) {
	host := PubManagerHost
	path := "/tasks"
	method := "GET"

	// 构建查询参数
	queryParams := url.Values{}
	if req.Marker != "" {
		queryParams.Set("marker", req.Marker)
	}
	if req.Limit > 0 {
		queryParams.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Name != "" {
		queryParams.Set("name", req.Name)
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
	url := fmt.Sprintf("https://%s%s", host, path)
	if rawQuery != "" {
		url = fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)
	}

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
	var result ListPubTasksResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// GetPubTaskRunInfo 获取 Pub 转推任务运行日志
func (c *BucketClient) GetPubTaskRunInfo(taskID string) (*PubTaskRunInfoResponse, error) {
	if taskID == "" {
		return nil, fmt.Errorf("taskID cannot be empty")
	}

	host := PubManagerHost
	path := fmt.Sprintf("/tasks/%s/runinfo", url.PathEscape(taskID))
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
	var result PubTaskRunInfoResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

// ListPubTaskHistory 查询 Pub 转推任务历史记录
func (c *BucketClient) ListPubTaskHistory(req *ListPubTaskHistoryRequest) (*ListPubTaskHistoryResponse, error) {
	host := PubManagerHost
	path := "/history"
	method := "GET"

	// 构建查询参数
	queryParams := url.Values{}
	if req.Marker != "" {
		queryParams.Set("marker", req.Marker)
	}
	if req.Limit > 0 {
		queryParams.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Name != "" {
		queryParams.Set("name", req.Name)
	}
	if req.Start != nil {
		queryParams.Set("start", fmt.Sprintf("%d", *req.Start))
	}
	if req.End != nil {
		queryParams.Set("end", fmt.Sprintf("%d", *req.End))
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
	url := fmt.Sprintf("https://%s%s", host, path)
	if rawQuery != "" {
		url = fmt.Sprintf("https://%s%s?%s", host, path, rawQuery)
	}

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
	var result ListPubTaskHistoryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(respBody))
	}

	return &result, nil
}

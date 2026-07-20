package voiceclone

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// XunfeiCloneConfig 讯飞一句话复刻配置（标准版 / 多风格版 omni_v1）
type XunfeiCloneConfig struct {
	AppID              string `json:"app_id"`
	APIKey             string `json:"api_key"`
	BaseURL            string `json:"base_url"`
	Timeout            int    `json:"timeout"`
	EngineVersion      string `json:"engine_version"` // 多风格版: omni_v1
	VCN                string `json:"vcn"`            // 合成 vcn：多风格 x6_clone，标准 x5_clone
	WebSocketAppID     string `json:"ws_app_id"`
	WebSocketAPIKey    string `json:"ws_api_key"`
	WebSocketAPISecret string `json:"ws_api_secret"`
}

// XunfeiCloneService 讯飞语音克隆服务
type XunfeiCloneService struct {
	config      *XunfeiCloneConfig
	httpClient  *http.Client
	token       *AuthToken
	tokenExpiry time.Time
}

// AuthToken 鉴权token
type AuthToken struct {
	AccessToken string `json:"accesstoken"`
	ExpiresIn   string `json:"expiresin"`
	RetCode     string `json:"retcode"`
}

// NewXunfeiCloneService 创建讯飞克隆服务
// 讯飞语音克隆服务实现
func NewXunfeiCloneService(config XunfeiCloneConfig) *XunfeiCloneService {
	config.AppID = strings.TrimSpace(config.AppID)
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.BaseURL = strings.TrimSpace(config.BaseURL)
	if config.BaseURL == "" {
		config.BaseURL = "http://opentrain.xfyousheng.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}
	if config.EngineVersion == "" {
		config.EngineVersion = "omni_v1"
	}
	if config.VCN == "" {
		if config.EngineVersion == "omni_v1" {
			config.VCN = "x6_clone"
		} else {
			config.VCN = "x5_clone"
		}
	}
	if config.WebSocketAppID == "" {
		config.WebSocketAppID = config.AppID
	}

	return &XunfeiCloneService{
		config: &config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// xunfeiAuthBody must keep field order appid → version → timestamp (讯飞签名校验依赖 JSON 键序).
type xunfeiAuthBase struct {
	AppID     string `json:"appid"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

type xunfeiAuthBody struct {
	Base  xunfeiAuthBase `json:"base"`
	Model string         `json:"model"`
}

func marshalXunfeiAuthBody(appID string, timestamp int64) ([]byte, error) {
	return json.Marshal(xunfeiAuthBody{
		Base: xunfeiAuthBase{
			AppID:     appID,
			Version:   "v1",
			Timestamp: strconv.FormatInt(timestamp, 10),
		},
		Model: "remote",
	})
}

// Provider 返回服务提供商
func (s *XunfeiCloneService) Provider() Provider {
	return ProviderXunfei
}

// getAuthToken 获取认证token
// 使用与旧实现完全相同的认证方式
func (s *XunfeiCloneService) getAuthToken(ctx context.Context) error {
	// 如果token未过期，直接返回
	if s.token != nil && time.Now().Before(s.tokenExpiry) {
		return nil
	}

	// 使用旧的认证URL：http://avatar-hci.xfyousheng.com/aiauth/v1/token
	url := "http://avatar-hci.xfyousheng.com/aiauth/v1/token"

	// 构建请求体（键序必须与讯飞官方示例一致，否则 sign 校验失败 retcode=000007）
	timestamp := time.Now().UnixMilli()
	bodyBytes, err := marshalXunfeiAuthBody(s.config.AppID, timestamp)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// 生成签名（与官方 Python 示例一致）
	keySign := fmt.Sprintf("%x", md5.Sum([]byte(s.config.APIKey+strconv.FormatInt(timestamp, 10))))
	sign := fmt.Sprintf("%x", md5.Sum([]byte(keySign+string(bodyBytes))))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", sign)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("auth request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp AuthToken
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	if tokenResp.RetCode != "000000" {
		switch tokenResp.RetCode {
		case "000007":
			return fmt.Errorf("xunfei auth sign failed (retcode=000007): use 一句话复刻应用的 APIKey（不是 API Secret）；确认 XUNFEI_APP_ID / XUNFEI_API_KEY 正确，且勿将 APIKey 误填到 XUNFEI_TRAIN_APP_ID")
		case "000004":
			return fmt.Errorf("xunfei auth failed (retcode=000004): missing XUNFEI_APP_ID or XUNFEI_API_KEY")
		default:
			return fmt.Errorf("xunfei auth failed (retcode=%s): check XUNFEI_APP_ID and XUNFEI_API_KEY for voice clone", tokenResp.RetCode)
		}
	}

	s.token = &tokenResp
	// 解析过期时间
	expiresIn, err := strconv.Atoi(tokenResp.ExpiresIn)
	if err != nil {
		expiresIn = 7200 // 默认2小时
	}
	s.tokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return nil
}

// makeRequest 发送HTTP请求
// 使用与旧实现完全相同的请求方式（包含签名）
func (s *XunfeiCloneService) makeRequest(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	if err := s.getAuthToken(ctx); err != nil {
		return nil, err
	}

	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置通用请求头（与旧实现完全一致）
	timestamp := time.Now().UnixMilli()
	bodyMD5 := fmt.Sprintf("%x", md5.Sum(bodyBytes))
	sign := fmt.Sprintf("%x", md5.Sum([]byte(s.config.APIKey+strconv.FormatInt(timestamp, 10)+bodyMD5)))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sign", sign)
	req.Header.Set("X-Token", s.token.AccessToken)
	req.Header.Set("X-AppId", s.config.AppID)
	req.Header.Set("X-Time", strconv.FormatInt(timestamp, 10))

	return s.httpClient.Do(req)
}

// DefaultTrainingText returns built-in segments for known text IDs when upstream API is unavailable.
func DefaultTrainingText(textID int64) *TrainingText {
	if textID != 5001 {
		return nil
	}
	return &TrainingText{
		TextID:   5001,
		TextName: "通用训练文本",
		Segments: []TextSegment{
			{SegID: "1", SegText: "您好，欢迎使用科大讯飞一句话复刻服务，请朗读以下文本。"},
			{SegID: "2", SegText: "科大讯飞是中国领先的智能语音技术提供商，愿为更多用户带来更优质的声音体验。"},
			{SegID: "3", SegText: "请问有什么可以帮到您？我将竭诚为您服务。"},
		},
	}
}

// GetTrainingTexts 获取训练文本
// 讯飞语音克隆服务实现
func (s *XunfeiCloneService) GetTrainingTexts(ctx context.Context, textID int64) (*TrainingText, error) {
	if textID <= 0 {
		textID = 5001
	}
	url := s.config.BaseURL + "/voice_train/task/traintext"
	body := map[string]interface{}{
		"textId": textID,
	}

	resp, err := s.makeRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get training texts failed with status %d: %s", resp.StatusCode, string(rawBody))
	}

	var textResp struct {
		Code int    `json:"code"`
		Desc string `json:"desc"`
		Data struct {
			TextID   int64  `json:"textId"`
			TextName string `json:"textName"`
			TextSegs []struct {
				SegID   interface{} `json:"segId"`
				SegText string      `json:"segText"`
			} `json:"textSegs"`
		} `json:"data"`
		Flag bool `json:"flag"`
	}

	if err := json.Unmarshal(rawBody, &textResp); err != nil {
		return nil, fmt.Errorf("failed to decode training texts response: %w", err)
	}

	if textResp.Code != 0 {
		return nil, fmt.Errorf("get training texts failed: %s (code=%d)", textResp.Desc, textResp.Code)
	}

	segments := make([]TextSegment, len(textResp.Data.TextSegs))
	for i, seg := range textResp.Data.TextSegs {
		segments[i] = TextSegment{
			SegID:   seg.SegID,
			SegText: seg.SegText,
		}
	}

	out := &TrainingText{
		TextID:   textResp.Data.TextID,
		TextName: textResp.Data.TextName,
		Segments: segments,
	}
	if out.TextID <= 0 {
		out.TextID = textID
	}
	if len(out.Segments) == 0 {
		if fb := DefaultTrainingText(textID); fb != nil {
			return fb, nil
		}
		return nil, fmt.Errorf("training texts empty for textId=%d", textID)
	}
	return out, nil
}

// CreateTask 创建训练任务（task/add，resourceType=12 一句话复刻）
func (s *XunfeiCloneService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	url := s.config.BaseURL + "/voice_train/task/add"

	resourceType := req.ResourceType
	if resourceType == 0 {
		resourceType = 12
	}
	body := map[string]interface{}{
		"taskName":     req.TaskName,
		"sex":          req.Sex,
		"ageGroup":     req.AgeGroup,
		"resourceType": resourceType,
	}
	engineVersion := req.EngineVersion
	if engineVersion == "" {
		engineVersion = s.config.EngineVersion
	}
	if engineVersion != "" {
		body["engineVersion"] = engineVersion
	}
	if req.Denoise > 0 {
		body["denoise"] = req.Denoise
	}
	if req.MosRatio > 0 {
		body["mosRatio"] = req.MosRatio
	}

	resp, err := s.makeRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create task failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp struct {
		Code int    `json:"code"`
		Desc string `json:"desc"`
		Data string `json:"data"`
		Flag bool   `json:"flag"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode create task response: %w", err)
	}

	if taskResp.Code != 0 {
		return nil, fmt.Errorf("create task failed: %s", taskResp.Desc)
	}

	return &CreateTaskResponse{
		TaskID: taskResp.Data,
	}, nil
}

// SubmitAudio 提交音频文件
func (s *XunfeiCloneService) SubmitAudio(ctx context.Context, req *SubmitAudioRequest) error {
	if err := s.getAuthToken(ctx); err != nil {
		return fmt.Errorf("failed to get auth token: %w", err)
	}

	url := s.config.BaseURL + "/voice_train/task/submitWithAudio"

	// 创建multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	fileWriter, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(fileWriter, req.AudioFile); err != nil {
		return fmt.Errorf("failed to copy audio file: %w", err)
	}

	// 添加其他字段
	writer.WriteField("taskId", req.TaskID)
	writer.WriteField("textId", strconv.FormatInt(req.TextID, 10))
	writer.WriteField("textSegId", strconv.FormatInt(req.TextSegID, 10))
	if req.MosRatio > 0 {
		writer.WriteField("mosRatio", strconv.FormatFloat(req.MosRatio, 'f', -1, 32))
	}

	writer.Close()

	// 生成签名
	timestamp := time.Now().UnixMilli()
	bodyMD5 := fmt.Sprintf("%x", md5.Sum(buf.Bytes()))
	sign := fmt.Sprintf("%x", md5.Sum([]byte(s.config.APIKey+strconv.FormatInt(timestamp, 10)+bodyMD5)))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create submit request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-Sign", sign)
	httpReq.Header.Set("X-Token", s.token.AccessToken)
	httpReq.Header.Set("X-AppId", s.config.AppID)
	httpReq.Header.Set("X-Time", strconv.FormatInt(timestamp, 10))

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send submit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("submit with audio failed with status %d: %s", resp.StatusCode, string(body))
	}

	var submitResp struct {
		Code int         `json:"code"`
		Desc string      `json:"desc"`
		Data interface{} `json:"data"`
		Flag bool        `json:"flag"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return fmt.Errorf("failed to decode submit response: %w", err)
	}

	if submitResp.Code != 0 {
		return fmt.Errorf("submit with audio failed: %s", submitResp.Desc)
	}

	return nil
}

// QueryTaskStatus 查询任务状态
// 讯飞语音克隆服务实现
func (s *XunfeiCloneService) QueryTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	url := s.config.BaseURL + "/voice_train/task/result"
	body := map[string]interface{}{
		"taskId": taskID,
	}

	resp, err := s.makeRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query task status failed with status %d: %s", resp.StatusCode, string(body))
	}

	var statusResp struct {
		Code int    `json:"code"`
		Desc string `json:"desc"`
		Data struct {
			TaskID      string `json:"taskId"`
			TaskName    string `json:"taskName"`
			TrainStatus int    `json:"trainStatus"`
			AssetID     string `json:"assetId"`
			TrainVID    string `json:"trainVid"`
			FailedDesc  string `json:"failedDesc"`
		} `json:"data"`
		Flag bool `json:"flag"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode task status response: %w", err)
	}

	if statusResp.Code != 0 {
		return nil, fmt.Errorf("query task status failed: %s", statusResp.Desc)
	}

	status := TrainingStatus(statusResp.Data.TrainStatus)

	return &TaskStatus{
		TaskID:     statusResp.Data.TaskID,
		TaskName:   statusResp.Data.TaskName,
		Status:     status,
		AssetID:    statusResp.Data.AssetID,
		TrainVID:   statusResp.Data.TrainVID,
		FailedDesc: statusResp.Data.FailedDesc,
		UpdatedAt:  time.Now(),
	}, nil
}

// xunfeiLanguageID 将语言代码映射为讯飞合成 languageID
func xunfeiLanguageID(lang string) int {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "en", "english":
		return 1
	case "ja", "jp", "japanese":
		return 2
	case "ko", "korean":
		return 3
	case "ru", "russian":
		return 4
	case "fr", "french":
		return 5
	case "ar", "arabic":
		return 6
	case "es", "spanish":
		return 7
	case "yue", "cantonese", "zh-yue":
		return 8
	default:
		return 0
	}
}

func (s *XunfeiCloneService) buildVoiceCloneWSRequest(req *SynthesizeRequest) map[string]interface{} {
	tts := map[string]interface{}{
		"vcn":      s.config.VCN,
		"volume":   50,
		"rhy":      0,
		"pybuffer": 1,
		"speed":    50,
		"pitch":    50,
		"bgs":      0,
		"reg":      0,
		"rdn":      0,
		"audio": map[string]interface{}{
			"encoding":    "lame",
			"sample_rate": 24000,
		},
	}
	langID := 0
	if req.LanguageID != nil {
		langID = *req.LanguageID
	} else if req.Language != "" {
		langID = xunfeiLanguageID(req.Language)
	}
	if langID != 0 {
		tts["languageID"] = langID
	}
	if req.Style != "" {
		tts["style"] = req.Style
	}
	return map[string]interface{}{
		"header": map[string]interface{}{
			"app_id": s.config.WebSocketAppID,
			"status": 2,
			"res_id": req.AssetID,
		},
		"parameter": map[string]interface{}{
			"tts": tts,
		},
		"payload": map[string]interface{}{
			"text": map[string]interface{}{
				"encoding": "utf8",
				"compress": "raw",
				"format":   "plain",
				"status":   2,
				"seq":      0,
				"text":     base64.StdEncoding.EncodeToString([]byte(req.Text)),
			},
		},
	}
}

func wsSynthError(respData map[string]interface{}) error {
	if header, ok := respData["header"].(map[string]interface{}); ok {
		if code, ok := header["code"].(float64); ok && code != 0 {
			msg, _ := header["message"].(string)
			return fmt.Errorf("synthesis failed (code %d): %s", int(code), msg)
		}
	}
	if code, ok := respData["code"].(float64); ok && code != 0 {
		desc, _ := respData["desc"].(string)
		return fmt.Errorf("synthesis failed: %s", desc)
	}
	return nil
}

func appendWSAudioChunk(respData map[string]interface{}, dest []byte) ([]byte, bool, error) {
	done := false
	if payload, ok := respData["payload"].(map[string]interface{}); ok {
		if audio, ok := payload["audio"].(map[string]interface{}); ok {
			if audioBase64, ok := audio["audio"].(string); ok && audioBase64 != "" {
				decoded, err := base64.StdEncoding.DecodeString(audioBase64)
				if err != nil {
					return dest, false, fmt.Errorf("failed to decode audio: %w", err)
				}
				dest = append(dest, decoded...)
			}
			if status, ok := audio["status"].(float64); ok && status == 2 {
				done = true
			}
		}
	}
	return dest, done, nil
}

// generateWebSocketAuthURL 生成WebSocket鉴权URL
func (s *XunfeiCloneService) generateWebSocketAuthURL(host, path string) (string, error) {
	if s.config.WebSocketAPIKey == "" || s.config.WebSocketAPISecret == "" {
		return "", fmt.Errorf("WebSocket API credentials not configured")
	}

	date := time.Now().UTC().Format(time.RFC1123)
	tmp := fmt.Sprintf("host: %s\ndate: %s\nGET %s HTTP/1.1", host, date, path)

	hmacSha256 := hmac.New(sha256.New, []byte(s.config.WebSocketAPISecret))
	hmacSha256.Write([]byte(tmp))
	signature := base64.StdEncoding.EncodeToString(hmacSha256.Sum(nil))

	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		s.config.WebSocketAPIKey, signature)
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	params := url.Values{}
	params.Add("authorization", authorization)
	params.Add("date", date)
	params.Add("host", host)

	return fmt.Sprintf("wss://%s%s?%s", host, path, params.Encode()), nil
}

// Synthesize 使用训练好的音色合成语音
func (s *XunfeiCloneService) Synthesize(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error) {
	if s.config.WebSocketAppID == "" || s.config.WebSocketAPIKey == "" || s.config.WebSocketAPISecret == "" {
		return nil, fmt.Errorf("WebSocket credentials not configured")
	}

	host := "cn-huabei-1.xf-yun.com"
	path := "/v1/private/voice_clone"

	wsURL, err := s.generateWebSocketAuthURL(host, path)
	if err != nil {
		return nil, fmt.Errorf("failed to generate WebSocket auth URL: %w", err)
	}

	// 连接到WebSocket服务
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}
	defer conn.Close()

	// 验证 AssetID
	if req.AssetID == "" {
		return nil, fmt.Errorf("AssetID is required for synthesis")
	}

	wsReq := s.buildVoiceCloneWSRequest(req)
	message, err := json.Marshal(wsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	var allAudioData []byte
	for {
		_, response, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("failed to read message: %w", err)
		}

		var respData map[string]interface{}
		if err := json.Unmarshal(response, &respData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		if err := wsSynthError(respData); err != nil {
			return nil, err
		}
		var done bool
		allAudioData, done, err = appendWSAudioChunk(respData, allAudioData)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
	}

	return &SynthesizeResponse{
		AudioData:  allAudioData,
		Format:     "mp3",
		SampleRate: 24000,
	}, nil
}

// SynthesizeStream 流式合成语音
func (s *XunfeiCloneService) SynthesizeStream(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error {
	if s.config.WebSocketAppID == "" || s.config.WebSocketAPIKey == "" || s.config.WebSocketAPISecret == "" {
		return fmt.Errorf("WebSocket credentials not configured")
	}

	host := "cn-huabei-1.xf-yun.com"
	path := "/v1/private/voice_clone"

	wsURL, err := s.generateWebSocketAuthURL(host, path)
	if err != nil {
		return fmt.Errorf("failed to generate WebSocket auth URL: %w", err)
	}

	// 连接到WebSocket服务
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}
	defer conn.Close()

	// 验证 AssetID
	if req.AssetID == "" {
		return fmt.Errorf("AssetID is required for synthesis")
	}

	wsReq := s.buildVoiceCloneWSRequest(req)
	message, err := json.Marshal(wsReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, response, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return fmt.Errorf("failed to read message: %w", err)
			}
			break
		}

		var respData map[string]interface{}
		if err := json.Unmarshal(response, &respData); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
		if err := wsSynthError(respData); err != nil {
			return err
		}

		if payload, ok := respData["payload"].(map[string]interface{}); ok {
			if audio, ok := payload["audio"].(map[string]interface{}); ok {
				if audioBase64, ok := audio["audio"].(string); ok && audioBase64 != "" {
					decodedAudio, err := base64.StdEncoding.DecodeString(audioBase64)
					if err != nil {
						return fmt.Errorf("failed to decode audio: %w", err)
					}
					if handler != nil && len(decodedAudio) > 0 {
						handler.OnMessage(decodedAudio)
					}
				}
				if status, ok := audio["status"].(float64); ok && status == 2 {
					break
				}
			}
		}
	}

	return nil
}

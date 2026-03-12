package voiceclone

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gorilla/websocket"
)

// XunfeiCloneConfig 讯飞克隆配置
type XunfeiCloneConfig struct {
	AppID              string `json:"app_id"`
	APIKey             string `json:"api_key"`
	BaseURL            string `json:"base_url"`
	Timeout            int    `json:"timeout"`
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
	if config.BaseURL == "" {
		// 使用与已有实现相同的默认 BaseURL
		config.BaseURL = "http://opentrain.xfyousheng.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}

	return &XunfeiCloneService{
		config: &config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
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

	// 构建请求体（与旧实现完全一致）
	timestamp := time.Now().UnixMilli()
	body := map[string]interface{}{
		"base": map[string]interface{}{
			"appid":     s.config.AppID,
			"version":   "v1",
			"timestamp": strconv.FormatInt(timestamp, 10),
		},
		"model": "remote",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	// 生成签名（与旧实现完全一致）
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
		return fmt.Errorf("auth failed: %s", tokenResp.RetCode)
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

// GetTrainingTexts 获取训练文本
// 讯飞语音克隆服务实现
func (s *XunfeiCloneService) GetTrainingTexts(ctx context.Context, textID int64) (*TrainingText, error) {
	url := s.config.BaseURL + "/voice_train/task/traintext"
	body := map[string]interface{}{
		"textId": textID,
	}

	resp, err := s.makeRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get training texts failed with status %d: %s", resp.StatusCode, string(body))
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

	if err := json.NewDecoder(resp.Body).Decode(&textResp); err != nil {
		return nil, fmt.Errorf("failed to decode training texts response: %w", err)
	}

	if textResp.Code != 0 {
		return nil, fmt.Errorf("get training texts failed: %s", textResp.Desc)
	}

	segments := make([]TextSegment, len(textResp.Data.TextSegs))
	for i, seg := range textResp.Data.TextSegs {
		segments[i] = TextSegment{
			SegID:   seg.SegID,
			SegText: seg.SegText,
		}
	}

	return &TrainingText{
		TextID:   textResp.Data.TextID,
		TextName: textResp.Data.TextName,
		Segments: segments,
	}, nil
}

// CreateTask 创建训练任务
// 讯飞语音克隆服务实现
func (s *XunfeiCloneService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	url := s.config.BaseURL + "/voice_train/task/add"

	// 构建请求体，确保所有必需参数都包含
	body := map[string]interface{}{
		"taskName":      req.TaskName,
		"sex":           req.Sex,
		"ageGroup":      req.AgeGroup,
		"resourceType":  12,  // 高质量音色合成
		"denoiseSwitch": 1,   // 开启降噪
		"mosRatio":      0.8, // 提高音色质量
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
	writer.WriteField("denoiseSwitch", "0")
	writer.WriteField("mosRatio", "0.0")

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

	// 构建WebSocket请求
	wsReq := map[string]interface{}{
		"header": map[string]interface{}{
			"app_id": s.config.WebSocketAppID,
			"status": 2,
			"res_id": req.AssetID, // 使用传入的 AssetID（音色ID）
		},
		"parameter": map[string]interface{}{
			"tts": map[string]interface{}{
				"vcn":      "x5_clone",
				"volume":   8,
				"rhy":      1,
				"pybuffer": 1,
				"speed":    50,
				"pitch":    50,
				"bgs":      0,
				"reg":      2,
				"rdn":      2,
				"audio": map[string]interface{}{
					"encoding":    "raw",
					"sample_rate": 24000,
				},
			},
		},
		"payload": map[string]interface{}{
			"text": map[string]interface{}{
				"encoding": "utf8",
				"compress": "raw",
				"format":   "plain",
				"status":   2,
				"seq":      1,
				"text":     base64.StdEncoding.EncodeToString([]byte(req.Text)),
			},
		},
	}

	message, err := json.Marshal(wsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// 接收响应
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

		// 检查是否有错误
		if code, ok := respData["code"].(float64); ok && code != 0 {
			desc, _ := respData["desc"].(string)
			return nil, fmt.Errorf("synthesis failed: %s", desc)
		}

		// 提取音频数据
		if payload, ok := respData["payload"].(map[string]interface{}); ok {
			if audio, ok := payload["audio"].(map[string]interface{}); ok {
				if audioBase64, ok := audio["audio"].(string); ok && audioBase64 != "" {
					decodedAudio, err := base64.StdEncoding.DecodeString(audioBase64)
					if err != nil {
						return nil, fmt.Errorf("failed to decode audio: %w", err)
					}
					allAudioData = append(allAudioData, decodedAudio...)
				}

				// 检查是否完成
				if status, ok := audio["status"].(float64); ok && status == 2 {
					break
				}
			}
		}
	}

	return &SynthesizeResponse{
		AudioData:  allAudioData,
		Format:     "pcm",
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

	// 构建WebSocket请求
	wsReq := map[string]interface{}{
		"header": map[string]interface{}{
			"app_id": s.config.WebSocketAppID,
			"status": 2,
			"res_id": req.AssetID, // 使用传入的 AssetID（音色ID）
		},
		"parameter": map[string]interface{}{
			"tts": map[string]interface{}{
				"vcn":      "x5_clone",
				"volume":   8,
				"rhy":      1,
				"pybuffer": 1,
				"speed":    50,
				"pitch":    50,
				"bgs":      0,
				"reg":      2,
				"rdn":      2,
				"audio": map[string]interface{}{
					"encoding":    "raw",
					"sample_rate": 24000,
				},
			},
		},
		"payload": map[string]interface{}{
			"text": map[string]interface{}{
				"encoding": "utf8",
				"compress": "raw",
				"format":   "plain",
				"status":   2,
				"seq":      1,
				"text":     base64.StdEncoding.EncodeToString([]byte(req.Text)),
			},
		},
	}

	message, err := json.Marshal(wsReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// 流式接收响应
	firstAudio := true
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

		// 检查是否有错误
		if code, ok := respData["code"].(float64); ok && code != 0 {
			desc, _ := respData["desc"].(string)
			return fmt.Errorf("synthesis failed: %s", desc)
		}

		// 提取音频数据并流式回调
		if payload, ok := respData["payload"].(map[string]interface{}); ok {
			if audio, ok := payload["audio"].(map[string]interface{}); ok {
				if audioBase64, ok := audio["audio"].(string); ok && audioBase64 != "" {
					decodedAudio, err := base64.StdEncoding.DecodeString(audioBase64)
					if err != nil {
						return fmt.Errorf("failed to decode audio: %w", err)
					}

					// 流式回调音频数据
					if handler != nil && len(decodedAudio) > 0 {
						if firstAudio {
							firstAudio = false
						}
						handler.OnMessage(decodedAudio)
					}
				}

				// 检查是否完成
				if status, ok := audio["status"].(float64); ok && status == 2 {
					break
				}
			}
		}
	}

	return nil
}

// SynthesizeToStorage 合成并保存到存储
func (s *XunfeiCloneService) SynthesizeToStorage(ctx context.Context, req *SynthesizeRequest, storageKey string) (string, error) {
	resp, err := s.Synthesize(ctx, req)
	if err != nil {
		return "", err
	}

	// 将 PCM 转换为 WAV 格式（先转WAV，再转MP3）
	// 讯飞默认参数：24000Hz, 16bit, 单声道
	sampleRate := 24000
	channels := 1
	bitDepth := 16
	wavData, err := s.convertPCMToWAV(resp.AudioData, sampleRate, channels, bitDepth)
	if err != nil {
		return "", fmt.Errorf("failed to convert PCM to WAV: %w", err)
	}

	// 将 WAV 转换为 MP3 格式（浏览器兼容性更好）
	mp3Data, err := s.convertWAVToMP3(wavData)
	if err != nil {
		// 如果转换失败，降级使用 WAV
		fmt.Printf("Warning: failed to convert WAV to MP3, using WAV instead: %v\n", err)
		mp3Data = wavData
		// 如果存储路径是 .mp3 或 .pcm，改为 .wav
		if strings.HasSuffix(storageKey, ".mp3") {
			storageKey = strings.TrimSuffix(storageKey, ".mp3") + ".wav"
		} else if strings.HasSuffix(storageKey, ".pcm") {
			storageKey = strings.TrimSuffix(storageKey, ".pcm") + ".wav"
		} else if !strings.HasSuffix(storageKey, ".wav") {
			storageKey = storageKey + ".wav"
		}
	} else {
		// 转换成功，使用 MP3
		if strings.HasSuffix(storageKey, ".wav") {
			storageKey = strings.TrimSuffix(storageKey, ".wav") + ".mp3"
		} else if strings.HasSuffix(storageKey, ".pcm") {
			storageKey = strings.TrimSuffix(storageKey, ".pcm") + ".mp3"
		} else if !strings.HasSuffix(storageKey, ".mp3") {
			storageKey = storageKey + ".mp3"
		}
	}

	// 测试上传
	result, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
		Data:     mp3Data,
		Filename: storageKey,
		Bucket:   config.GlobalConfig.Services.Storage.Bucket,
		Key:      storageKey,
	})

	// 获取URL
	return result.URL, nil
}

// convertPCMToWAV 将 PCM 音频数据转换为 WAV 格式（添加 WAV 文件头）
func (s *XunfeiCloneService) convertPCMToWAV(pcmData []byte, sampleRate int, channels int, bitDepth int) ([]byte, error) {
	// 创建44字节的WAV头部
	header := make([]byte, 44)
	dataSize := len(pcmData)

	// RIFF header
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize)) // File size
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)                                     // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)                                      // Audio format (PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))                       // Number of channels
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))                     // Sample rate
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*channels*bitDepth/8)) // Byte rate
	binary.LittleEndian.PutUint16(header[32:34], uint16(channels*bitDepth/8))            // Block align
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitDepth))                       // Bits per sample

	// data chunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize)) // Data size

	// 合并头部和音频数据
	wavData := append(header, pcmData...)
	return wavData, nil
}

// convertWAVToMP3 使用 ffmpeg 将 WAV 音频数据转换为 MP3 格式
func (s *XunfeiCloneService) convertWAVToMP3(wavData []byte) ([]byte, error) {
	// 创建临时文件
	tmpWavFile, err := os.CreateTemp("", "voice_synthesis_*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp WAV file: %w", err)
	}
	defer os.Remove(tmpWavFile.Name())
	defer tmpWavFile.Close()

	// 写入 WAV 数据
	if _, err := tmpWavFile.Write(wavData); err != nil {
		return nil, fmt.Errorf("failed to write WAV data: %w", err)
	}
	tmpWavFile.Close()

	// 使用 ffmpeg 转换为 MP3
	cmd := exec.Command("ffmpeg",
		"-v", "quiet", // 安静模式
		"-y",                    // 覆盖输出文件
		"-i", tmpWavFile.Name(), // 输入文件
		"-acodec", "libmp3lame", // MP3 编码器
		"-ab", "128k", // 音频比特率
		"-ar", "24000", // 采样率
		"-ac", "1", // 单声道
		"-f", "mp3", // 输出格式
		"-", // 输出到标准输出
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{} // 忽略错误输出（已使用 quiet 模式）

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	if out.Len() == 0 {
		return nil, fmt.Errorf("ffmpeg produced empty output")
	}

	return out.Bytes(), nil
}

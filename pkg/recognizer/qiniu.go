package recognizer

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/sirupsen/logrus"
)

// QiniuASR 七牛云ASR实现
type QiniuASR struct {
	conn             *websocket.Conn
	config           QiniuASROption
	transcribeResult TranscribeResult
	processError     ProcessError
	dialogID         string
	handler          media.MediaHandler
	seq              int
	sendReqTime      *time.Time
	endReqTime       *time.Time
	sentence         string
	activity         bool
	mu               sync.Mutex
}

// QiniuASROption 七牛云ASR选项
type QiniuASROption struct {
	APIKey         string    `json:"api_key" yaml:"api_key" env:"QINIU_ASR_API_KEY"`
	BaseURL        string    `json:"base_url" yaml:"base_url" env:"QINIU_ASR_BASE_URL"`
	SampleRate     int       `json:"sample_rate" yaml:"sample_rate" default:"16000"`
	Channels       int       `json:"channels" yaml:"channels" default:"1"`
	Bits           int       `json:"bits" yaml:"bits" default:"16"`
	SegDuration    int       `json:"seg_duration" yaml:"seg_duration" default:"300"`
	SilenceTimeout int       `json:"silence_timeout" yaml:"silence_timeout" default:"1500"`
	EnablePunc     bool      `json:"enable_punc" yaml:"enable_punc" default:"true"`
	HotWords       []HotWord `json:"hot_words" yaml:"hot_words"`
	ReqChanSize    int       `json:"req_chan_size" yaml:"req_chan_size" default:"128"`
}

// QiniuASRRequest 七牛云ASR请求结构
type QiniuASRRequest struct {
	User    QiniuASRUser          `json:"user"`
	Audio   QiniuASRAudio         `json:"audio"`
	Request QiniuASRRequestConfig `json:"request"`
}

// QiniuASRUser 七牛云ASR用户信息
type QiniuASRUser struct {
	UID string `json:"uid"`
}

// QiniuASRAudio 七牛云ASR音频配置
type QiniuASRAudio struct {
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
	Bits       int    `json:"bits"`
	Channel    int    `json:"channel"`
	Codec      string `json:"codec"`
}

// QiniuASRRequestConfig 七牛云ASR请求配置
type QiniuASRRequestConfig struct {
	ModelName  string `json:"model_name"`
	EnablePunc bool   `json:"enable_punc"`
}

// QiniuASRResponse 七牛云ASR响应结构
type QiniuASRResponse struct {
	Result     *QiniuASRSegment `json:"result,omitempty"`
	PayloadMsg *QiniuASRSegment `json:"payload_msg,omitempty"`
}

// QiniuASRSegment 七牛云ASR片段
type QiniuASRSegment struct {
	Text      string `json:"text"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
}

// NewQiniuASROption 创建七牛云ASR选项
func NewQiniuASROption(apiKey string) QiniuASROption {
	return QiniuASROption{
		APIKey:         apiKey,
		BaseURL:        "",
		SampleRate:     16000,
		Channels:       1,
		Bits:           16,
		SegDuration:    300,
		SilenceTimeout: 1500,
		EnablePunc:     true,
		ReqChanSize:    128,
	}
}

// NewQiniuASR 创建七牛云ASR实例
func NewQiniuASR(opt QiniuASROption) *QiniuASR {
	// 从环境变量获取默认值
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("QINIU_ASR_API_KEY")
	}
	if opt.BaseURL == "" {
		opt.BaseURL = utils.GetEnv("QINIU_ASR_BASE_URL")
		if opt.BaseURL == "" {
			opt.BaseURL = "wss://openai.qiniu.com/v1/media/asr"
		}
	}
	if opt.SampleRate == 0 {
		opt.SampleRate = int(utils.GetIntEnv("QINIU_ASR_SAMPLE_RATE"))
		if opt.SampleRate == 0 {
			opt.SampleRate = 16000
		}
	}
	if opt.Channels == 0 {
		opt.Channels = int(utils.GetIntEnv("QINIU_ASR_CHANNELS"))
		if opt.Channels == 0 {
			opt.Channels = 1
		}
	}
	if opt.Bits == 0 {
		opt.Bits = int(utils.GetIntEnv("QINIU_ASR_BITS"))
		if opt.Bits == 0 {
			opt.Bits = 16
		}
	}
	if opt.EnablePunc == false && !utils.GetBoolEnv("QINIU_ASR_ENABLE_PUNC") {
		opt.EnablePunc = utils.GetBoolEnv("QINIU_ASR_ENABLE_PUNC")
	}

	return &QiniuASR{
		config:   opt,
		seq:      1,
		activity: false,
	}
}

// Init 初始化
func (q *QiniuASR) Init(tr TranscribeResult, er ProcessError) {
	q.transcribeResult = tr
	q.processError = er
}

// Vendor 返回提供商名称
func (q *QiniuASR) Vendor() string {
	return "qiniu"
}

// ConnAndReceive 连接并接收
func (q *QiniuASR) ConnAndReceive(dialogID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if dialogID == "" {
		dialogID, _ = gonanoid.Nanoid()
	}
	q.dialogID = dialogID

	// 关闭旧连接
	if q.conn != nil {
		q.conn.Close()
		q.conn = nil
	}

	// 建立WebSocket连接
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+q.config.APIKey)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(q.config.BaseURL, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	q.conn = conn

	// 发送配置信息
	if err := q.sendConfig(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send config: %w", err)
	}

	// 启动消息处理协程
	go q.handleMessages()

	now := time.Now()
	q.sendReqTime = &now
	q.endReqTime = &now
	q.activity = true

	logrus.WithFields(logrus.Fields{
		"sessionID": q.handler.GetSession().ID,
		"dialogID":  dialogID,
		"vendor":    "qiniu",
	}).Info("qiniu asr: connected")

	return nil
}

// Activity 检查是否活跃
func (q *QiniuASR) Activity() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.activity && q.conn != nil
}

// RestartClient 重启客户端
func (q *QiniuASR) RestartClient() {
	_ = q.StopConn()
	dialogID, _ := gonanoid.Nanoid()
	_ = q.ConnAndReceive(dialogID)
}

// SendAudioBytes 发送音频数据
func (q *QiniuASR) SendAudioBytes(data []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.conn == nil {
		return fmt.Errorf("WebSocket connection is not established")
	}

	if len(data) == 0 {
		return nil
	}

	q.seq++
	compressed := qiniuGzipCompress(data)
	msg := q.buildMessage(2, 1, 1, 1, compressed)

	return q.conn.WriteMessage(websocket.BinaryMessage, msg)
}

// SendEnd 发送结束信号
func (q *QiniuASR) SendEnd() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.conn == nil {
		return nil
	}

	q.activity = false

	if q.transcribeResult != nil && q.sentence != "" {
		var duration time.Duration
		if q.sendReqTime != nil && q.endReqTime != nil {
			duration = q.endReqTime.Sub(*q.sendReqTime)
		}
		q.transcribeResult(q.sentence, true, duration, q.dialogID)
		q.sentence = ""
	}

	return nil
}

// StopConn 停止连接
func (q *QiniuASR) StopConn() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.activity = false

	if q.conn != nil {
		err := q.conn.Close()
		q.conn = nil
		return err
	}

	return nil
}

// SetHandler 设置会话处理器
func (q *QiniuASR) SetHandler(h media.MediaHandler) {
	q.handler = h
}

// sendConfig 发送配置信息
func (q *QiniuASR) sendConfig() error {
	req := QiniuASRRequest{
		User: QiniuASRUser{
			UID: q.dialogID,
		},
		Audio: QiniuASRAudio{
			Format:     "pcm",
			SampleRate: q.config.SampleRate,
			Bits:       q.config.Bits,
			Channel:    q.config.Channels,
			Codec:      "raw",
		},
		Request: QiniuASRRequestConfig{
			ModelName:  "asr",
			EnablePunc: q.config.EnablePunc,
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	compressed := qiniuGzipCompress(payload)
	msg := q.buildMessage(1, 1, 1, 1, compressed)

	return q.conn.WriteMessage(websocket.BinaryMessage, msg)
}

// buildMessage 构建消息
func (q *QiniuASR) buildMessage(messageType, flags, serial, compress int, payload []byte) []byte {
	header := make([]byte, 4)
	header[0] = byte((1 << 4) | 1) // protocol version and header size
	header[1] = byte((messageType << 4) | flags)
	header[2] = byte((serial << 4) | compress)
	header[3] = 0

	seqBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(seqBytes, uint32(q.seq))

	payloadSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadSizeBytes, uint32(len(payload)))

	return bytes.Join([][]byte{header, seqBytes, payloadSizeBytes, payload}, nil)
}

// handleMessages 处理WebSocket消息
func (q *QiniuASR) handleMessages() {
	defer func() {
		q.mu.Lock()
		q.activity = false
		q.mu.Unlock()
	}()

	for {
		_, message, err := q.conn.ReadMessage()
		if err != nil {
			if q.processError != nil {
				q.processError(err, false)
			}
			return
		}

		text := q.parseTextFromResponse(message)
		if text != "" && q.transcribeResult != nil {
			var duration time.Duration
			if q.sendReqTime != nil {
				duration = time.Since(*q.sendReqTime)
			}
			q.transcribeResult(text, false, duration, q.dialogID)
			q.sentence = text
		}
	}
}

// parseTextFromResponse 解析响应中的文本
func (q *QiniuASR) parseTextFromResponse(data []byte) string {
	if len(data) < 4 {
		return ""
	}

	headerSize := data[0] & 0x0f
	messageType := data[1] >> 4
	messageTypeSpecificFlags := data[1] & 0x0f
	serializationMethod := data[2] >> 4
	messageCompression := data[2] & 0x0f

	payload := data[headerSize*4:]

	if messageTypeSpecificFlags&0x01 != 0 {
		if len(payload) >= 4 {
			payload = payload[4:]
		}
	}

	if messageType == 0b1001 && len(payload) >= 4 {
		payloadSize := binary.BigEndian.Uint32(payload[:4])
		if len(payload) >= int(4+payloadSize) {
			payload = payload[4 : 4+payloadSize]
		}
	}

	if messageCompression == 0b0001 {
		decompressed, err := qiniuGzipDecompress(payload)
		if err == nil {
			payload = decompressed
		}
	}

	var obj QiniuASRResponse
	if serializationMethod == 0b0001 {
		if err := json.Unmarshal(payload, &obj); err == nil {
			if obj.Result != nil && obj.Result.Text != "" {
				return obj.Result.Text
			}
			if obj.PayloadMsg != nil && obj.PayloadMsg.Text != "" {
				return obj.PayloadMsg.Text
			}
		}
	}

	return ""
}

// gzipCompress GZIP压缩
func qiniuGzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(data)
	gz.Close()
	return buf.Bytes()
}

// gzipDecompress GZIP解压
func qiniuGzipDecompress(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var buf bytes.Buffer
	_, err = buf.ReadFrom(gz)
	return buf.Bytes(), err
}

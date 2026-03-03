package recognizer

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	SuccessCode = 1000

	ServerFullResponse  = MessageType(0b1001)
	ServerAck           = MessageType(0b1011)
	ServerErrorResponse = MessageType(0b1111)
)

var DefaultFullClientWsHeader = []byte{0x11, 0x10, 0x11, 0x00}
var DefaultAudioOnlyWsHeader = []byte{0x11, 0x20, 0x11, 0x00}
var DefaultLastAudioWsHeader = []byte{0x11, 0x22, 0x11, 0x00}

func gzipCompress(input []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(input)
	if err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func gzipDecompress(input []byte) ([]byte, error) {
	b := bytes.NewBuffer(input)
	r, _ := gzip.NewReader(b)
	out, _ := io.ReadAll(r)
	if err := r.Close(); err != nil {
		return nil, err
	}
	return out, nil
}

type VolcEngineResponse struct {
	Reqid    string             `json:"reqid"`
	Code     int                `json:"code"`
	Message  string             `json:"message"`
	Sequence int                `json:"sequence"`
	Results  []VolcengineResult `json:"result,omitempty"`
}

type VolcengineResult struct {
	Text       string      `json:"text"`
	Confidence int         `json:"confidence"`
	Language   string      `json:"language,omitempty"`
	Utterances []Utterance `json:"utterances,omitempty"`
}

type Utterance struct {
	Text      string `json:"text"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
	Definite  bool   `json:"definite"`
	Words     []Word `json:"words"`
	Language  string `json:"language"`
}

type Word struct {
	Text          string `json:"text"`
	StartTime     int    `json:"start_time"`
	EndTime       int    `json:"end_time"`
	Pronounce     string `json:"pronounce"`
	BlankDuration int    `json:"blank_duration"`
}

type Volcengine struct {
	handler        media.MediaHandler
	client         *VolcengineClient
	opt            VolcengineOption
	ttfbDone       bool
	sendReqTime    *time.Time
	endReqTime     *time.Time
	Sentence       string
	isTranscribing bool

	audioChan chan []byte
	closeChan chan struct{}
}

type VolcengineOption struct {
	Url         string `json:"url" yaml:"url" default:"wss://openspeech.bytedance.com/api/v2/asr"`
	AppID       string `json:"appId" yaml:"app_id" env:"VOLC_APPID"`
	Token       string `json:"token" yaml:"token" env:"VOLC_TOKEN"`
	Cluster     string `json:"cluster" yaml:"cluster" env:"VOLC_CLUSTER"`
	WorkFlow    string `json:"workFlow" yaml:"work_flow" default:"audio_in,resample,partition,vad,fe,decode"`
	Format      string `json:"format" yaml:"format" default:"raw" env:"VOLC_FORMAT"`
	Codec       string `json:"codec" yaml:"codec" default:"raw"`
	ReqChanSize int    `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
}

type VolcengineClient struct {
	uuid          string
	conn          *websocket.Conn
	ctx           context.Context
	cancel        context.CancelFunc
	sendLastAudio bool
}

func (c *VolcengineClient) String() string {
	return fmt.Sprintf("volc client{uuid: %s, sendLastAudio: %t}", c.uuid, c.sendLastAudio)
}

func NewVolcengineOption(appId string, token string, cluster string, format string) VolcengineOption {
	return VolcengineOption{
		Url:         "wss://openspeech.bytedance.com/api/v2/asr",
		AppID:       appId,
		Token:       token,
		Cluster:     cluster,
		ReqChanSize: 128,
		WorkFlow:    "audio_in,resample,partition,vad,fe,decode",
		Codec:       "raw",
		Format:      format,
	}
}

func WithVolcengineASR(opt VolcengineOption) media.MediaHandlerFunc {
	if opt.ReqChanSize <= 0 {
		opt.ReqChanSize = 128
	}
	executor := media.NewAsyncTaskRunner[[]byte](opt.ReqChanSize)

	volc := &Volcengine{opt: opt, audioChan: make(chan []byte, 1024), closeChan: make(chan struct{}, 24)}

	executor.ConcurrentMode = false
	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[[]byte], error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			h.EmitPacket(volc, packet)
			return nil, nil
		}
		if volc.handler == nil {
			volc.handler = h
		}
		audioPacket.Payload, _ = media.ResamplePCM(audioPacket.Payload, h.GetSession().Codec().SampleRate, 16000)
		req := media.PacketRequest[[]byte]{
			Req:       audioPacket.Payload,
			Interrupt: true,
		}
		return &req, nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		return volc.buildClient()
	}

	executor.TerminateCallback = func(h media.MediaHandler) error {
		if volc.client == nil {
			return nil
		}
		return volc.client.conn.Close()
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.StartSilence:
			volc.closeChan <- struct{}{}
			n := time.Now()
			volc.endReqTime = &n
			return nil
		case media.StartSpeaking:
			n := time.Now()
			volc.sendReqTime = &n
			return nil
		case media.Hangup:
			return volc.closeClient()
		}
		return nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[[]byte]) error {
		volc.audioChan <- req.Req
		return nil
	}

	return executor.HandleMediaData
}

func (volc *Volcengine) closeClient() error {
	if volc.client != nil {
		return volc.client.conn.Close()
	}
	return nil
}

func (volc *Volcengine) buildClient() error {
	var err error
	var tokenHeader = http.Header{"Authorization": []string{fmt.Sprintf("Bearer;%s", volc.opt.Token)}}
	conn, _, err := websocket.DefaultDialer.Dial(volc.opt.Url, tokenHeader)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).WithError(err).Error("volcengine asr: fail to dial")
		return err
	}
	if err = volc.sendFullClientMsg(conn); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).WithError(err).Error("volcengine asr: fail to send full client msg")
		return err
	}

	ctx, clientCancel := context.WithCancel(context.Background())
	client := VolcengineClient{uuid: uuid.New().String(), conn: conn, sendLastAudio: false, ctx: ctx, cancel: clientCancel}
	logrus.WithFields(logrus.Fields{
		"client": client.String(),
	}).Info("volcengine asr: build client")
	volc.client = &client

	go volc.recvFrames(&client)
	go volc.sendFrames(&client)
	return err
}

func (volc *Volcengine) restartClient() {
	logrus.Info("volcengine asr: restart client")
	if volc.client != nil && volc.client.cancel != nil {
		volc.client.cancel()
	}
	err := volc.buildClient()
	if err != nil {
		volc.handler.CauseError(volc, err)
	}
}

func (volc *Volcengine) sendFrames(client *VolcengineClient) {
	// 速率限制：1秒内最多发送3秒音频数据
	// 16000 Hz 采样率，16-bit PCM，单声道 = 32000 bytes/秒
	// 3秒音频 = 96000 bytes，所以每秒最多发送 96000 bytes
	const maxBytesPerSecond = 96000
	const sendInterval = 100 * time.Millisecond        // 每100ms发送一次
	const maxBytesPerInterval = maxBytesPerSecond / 10 // 每100ms最多发送9600字节

	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	var pendingData []byte

	for {
		select {
		case data := <-volc.audioChan:
			// 累积待发送的数据
			pendingData = append(pendingData, data...)

		case <-ticker.C:
			// 定时发送数据，控制速率
			if len(pendingData) > 0 {
				// 每次最多发送 maxBytesPerInterval 字节
				sendSize := len(pendingData)
				if sendSize > maxBytesPerInterval {
					sendSize = maxBytesPerInterval
				}

				toSend := pendingData[:sendSize]
				pendingData = pendingData[sendSize:]

				if err := volc.sendAudioMsg(client, toSend, false); err != nil {
					logrus.WithFields(logrus.Fields{
						"error": err.Error(),
					}).WithError(err).Error("volcengine asr: fail to send audio msg")
					volc.restartClient()
					return
				}
			}

		case <-volc.closeChan:
			// 发送剩余的数据
			if len(pendingData) > 0 {
				if err := volc.sendAudioMsg(client, pendingData, false); err != nil {
					logrus.WithFields(logrus.Fields{
						"error": err.Error(),
					}).WithError(err).Error("volcengine asr: fail to send remaining audio")
				}
			}
			// 发送结束标记
			client.sendLastAudio = true
			if err := volc.sendAudioMsg(client, nil, true); err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
				}).WithError(err).Error("volcengine asr: fail to send audio msg")
				volc.restartClient()
			}
			return

		case <-client.ctx.Done():
			return
		}
	}
}

// volcengine requires marking the final audio frame
func (volc *Volcengine) sendAudioMsg(client *VolcengineClient, audio []byte, isLast bool) error {
	var err error
	audioMsg := make([]byte, len(DefaultAudioOnlyWsHeader))

	if isLast {
		copy(audioMsg, DefaultLastAudioWsHeader)
	} else {
		copy(audioMsg, DefaultAudioOnlyWsHeader)
	}
	payload, _ := gzipCompress(audio)
	payloadSize := len(payload)
	payloadSizeArr := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadSizeArr, uint32(payloadSize))
	audioMsg = append(audioMsg, payloadSizeArr...)
	audioMsg = append(audioMsg, payload...)

	err = client.conn.WriteMessage(websocket.BinaryMessage, audioMsg)

	return err
}

func (volc *Volcengine) recvFrames(client *VolcengineClient) {
	for {
		select {
		case <-client.ctx.Done():
			return
		default:
			conn := client.conn
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					logrus.Info("volcengine asr: recv close message, connection closed")
				} else {
					logrus.WithFields(logrus.Fields{
						"sessionID":   volc.handler.GetSession().ID,
						"err":         err,
						"message":     string(message),
						"messageType": messageType,
						"client":      client,
					}).WithError(err).Error("volcengine asr: recv error, connection closed")
				}
				if volc.Sentence != "" {
					volc.handler.EmitPacket(volc, &media.TextPacket{Text: volc.Sentence, IsTranscribed: true})
					volc.handler.EmitState(volc, media.Completed, &media.CompletedData{
						SenderName: "asr.volcengine",
						Result:     volc.Sentence,
						Duration:   time.Since(*volc.sendReqTime),
					})
					if volc.sendReqTime != nil {
						volc.handler.AddMetric("asr.volcengine", time.Since(*volc.sendReqTime))
					}
				}
				volc.restartClient()
				return
			}

			response, err := volc.parseResponse(message)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"sessionID": volc.handler.GetSession().ID,
					"error":     err,
					"client":    client,
				}).Error("volcengine asr: fail to parse response")
				volc.restartClient()
				return
			}
			if response.Code != SuccessCode {
				logrus.WithFields(logrus.Fields{
					"code":    response.Code,
					"message": response.Message,
					"client":  client,
				}).WithError(err).Error("volcengine asr: receive error message")
				volc.restartClient()
				return
			}
			if len(response.Results) > 0 {
				latestResult := response.Results[0]
				if latestResult.Text != "" {
					if !volc.ttfbDone {
						volc.ttfbDone = true
						if volc.sendReqTime != nil {
							volc.handler.AddMetric("asr.volcengine.ttfb", time.Since(*volc.sendReqTime))
						}
					}

					result := response.Results[0]
					text := result.Text

					volc.Sentence = text
					logrus.WithFields(logrus.Fields{
						"sessionID": volc.handler.GetSession().ID,
						"Sentence":  volc.Sentence,
						"client":    client,
					}).Info("volcengine asr: recv frame")

					volc.handler.EmitState(volc, media.Transcribing, &media.TranscribingData{
						SenderName: "asr.volcengine",
						Result:     volc.Sentence,
					})
				}
				if len(latestResult.Utterances) > 0 && latestResult.Utterances[0].Definite && client.sendLastAudio {
					if volc.Sentence != "" {
						volc.handler.EmitPacket(volc, &media.TextPacket{Text: volc.Sentence, IsTranscribed: true})
					}
					volc.handler.EmitState(volc, media.Completed, &media.CompletedData{
						SenderName: "asr.volcengine",
						Result:     volc.Sentence,
						Duration:   time.Since(*volc.sendReqTime),
					})
					if volc.sendReqTime != nil {
						volc.handler.AddMetric("asr.volcengine", time.Since(*volc.sendReqTime))
					}
					if volc.endReqTime != nil {
						volc.handler.AddMetric("asr.volcengine.complete", time.Since(*volc.endReqTime))
					}
					err = volc.buildClient()
					if err != nil {
						volc.handler.CauseError(volc, err)
					}
					return
				}
			}
		}
	}
}

func (volc *Volcengine) sendFullClientMsg(conn *websocket.Conn) error {
	req := volc.constructRequest()
	payload, _ := gzipCompress(req)
	payloadSize := len(payload)
	payloadSizeArr := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadSizeArr, uint32(payloadSize))

	fullClientMsg := make([]byte, len(DefaultFullClientWsHeader))
	copy(fullClientMsg, DefaultFullClientWsHeader)
	fullClientMsg = append(fullClientMsg, payloadSizeArr...)
	fullClientMsg = append(fullClientMsg, payload...)

	err := conn.WriteMessage(websocket.BinaryMessage, fullClientMsg)
	if err != nil {
		return err
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		logrus.WithError(err).Error("volcengine asr: fail to read message")
		return err
	}
	_, err = volc.parseResponse(msg)
	if err != nil {
		return err
	}
	return nil
}

func (volc *Volcengine) constructRequest() []byte {
	uid := strings.ReplaceAll(uuid.New().String(), "-", "")

	req := make(map[string]map[string]interface{})
	req["app"] = make(map[string]interface{})
	req["app"]["appid"] = volc.opt.AppID
	req["app"]["cluster"] = volc.opt.Cluster
	req["app"]["token"] = volc.opt.Token
	req["user"] = make(map[string]interface{})
	req["user"]["uid"] = uid
	req["request"] = make(map[string]interface{})
	req["request"]["reqid"] = uuid.New().String()
	req["request"]["nbest"] = 1
	req["request"]["workflow"] = volc.opt.WorkFlow
	req["request"]["show_utterances"] = true
	req["request"]["result_type"] = "signle"
	req["request"]["sequence"] = 1
	req["audio"] = make(map[string]interface{})
	req["audio"]["format"] = volc.opt.Format
	req["audio"]["codec"] = volc.opt.Codec
	reqStr, _ := json.Marshal(req)
	return reqStr
}

func (volc *Volcengine) parseResponse(msg []byte) (VolcEngineResponse, error) {
	var err error

	headerSize := msg[0] & 0x0f
	messageType := msg[1] >> 4
	serializationMethod := msg[2] >> 4
	messageCompression := msg[2] & 0x0f
	payload := msg[headerSize*4:]
	payloadMsg := make([]byte, 0)
	payloadSize := 0

	if messageType == byte(ServerFullResponse) {
		payloadSize = int(int32(binary.BigEndian.Uint32(payload[0:4])))
		payloadMsg = payload[4:]
	} else if messageType == byte(ServerAck) {
		seq := int32(binary.BigEndian.Uint32(payload[:4]))
		if len(payload) >= 8 {
			payloadSize = int(binary.BigEndian.Uint32(payload[4:8]))
			payloadMsg = payload[8:]
		}
		logrus.Debug("volcengine asr: server ack seq: ", seq)
	} else if messageType == byte(ServerErrorResponse) {
		code := int32(binary.BigEndian.Uint32(payload[:4]))
		payloadSize = int(binary.BigEndian.Uint32(payload[4:8]))
		payloadMsg = payload[8:]
		var errResponse = VolcEngineResponse{}
		payloadMsg, _ = gzipDecompress(payloadMsg)
		_ = json.Unmarshal(payloadMsg, &errResponse)
		return VolcEngineResponse{}, errors.New(fmt.Sprintf("volcengine asr: server response error code: %d msg: %s", code, errResponse.Message))
	}
	if payloadSize == 0 {
		return VolcEngineResponse{}, errors.New("volcengine asr: payload size is 0")
	}
	if messageCompression == byte(CompressionGZIP) {
		payloadMsg, _ = gzipDecompress(payloadMsg)
	}

	var asrResponse = VolcEngineResponse{}
	if serializationMethod == byte(SerializationJSON) {
		err = json.Unmarshal(payloadMsg, &asrResponse)
		if err != nil {
			logrus.Error("volcengine asr: fail to unmarshal response, ", err.Error())
			return VolcEngineResponse{}, err
		}
	}
	return asrResponse, nil
}

package recognizer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var finishMessage = []byte("{\"type\":\"FINISH\"}")

var cancelMessage = []byte("{\"type\":\"CANCEL\"}")

const (
	finText   = "FIN_TEXT"
	midText   = "MID_TEXT"
	heartbeat = "HEARTBEAT"
)

type BaiduASR struct {
	handler     media.MediaHandler
	conn        *websocket.Conn
	opt         BaiduASROption
	ttfbDone    bool
	sendReqTime *time.Time
	Sentence    string
}

type BaiduASROption struct {
	Url         string `json:"url" yaml:"url" default:"wss://vop.baidu.com/realtime_asr"`
	AppID       int    `json:"appId" yaml:"app_id"`
	AppKey      string `json:"appKey" yaml:"app_key"`
	DevPid      int    `json:"devPid" yaml:"dev_pid"`
	LmId        int    `json:"lmId" yaml:"lm_id"`
	CuId        string `json:"cuId" yaml:"cu_id"`
	Format      string `json:"format" yaml:"format"`
	Sample      int    `json:"sample" yaml:"sample"`
	ReqChanSize int    `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
}

type BaiduASRBeginParam struct {
	Type string `json:"type"`
	Data Data   `json:"data"`
}

type Data struct {
	AppId  int    `json:"appid"`
	AppKey string `json:"appkey"`
	DevPid int    `json:"dev_pid"`
	LmId   int    `json:"lm_id"`
	CuId   string `json:"cuid"`
	Format string `json:"format"`
	Sample int    `json:"sample"`
}

type BaiduASRWSResponse struct {
	ErrNo     int    `json:"err_no"`
	ErrMsg    string `json:"err_msg"`
	Type      string `json:"type"`
	Result    string `json:"result"`
	StartTime int    `json:"start_time"`
	EndTime   int    `json:"end_time"`
	LogId     int    `json:"log_id"`
	Sn        string `json:"sn"`
}

func NewBaiduASROption(appId int, appKey string, devPid int, format string, sample int) BaiduASROption {
	return BaiduASROption{
		Url:         "wss://vop.baidu.com/realtime_asr",
		AppID:       appId,
		AppKey:      appKey,
		DevPid:      devPid,
		CuId:        "cuid-1",
		Format:      format,
		Sample:      sample,
		ReqChanSize: 128,
	}
}

func NewBeginParam(opt BaiduASROption) BaiduASRBeginParam {
	return BaiduASRBeginParam{
		Type: "START",
		Data: Data{
			AppId:  opt.AppID,
			AppKey: opt.AppKey,
			DevPid: opt.DevPid,
			LmId:   opt.LmId,
			CuId:   opt.CuId,
			Format: opt.Format,
			Sample: opt.Sample,
		},
	}
}

func WithBaiduASR(opt BaiduASROption) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[*media.AudioPacket](opt.ReqChanSize)

	baidu := &BaiduASR{}

	executor.ConcurrentMode = true
	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[*media.AudioPacket], error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			h.EmitPacket(baidu, packet)
			return nil, nil
		}
		if baidu.handler == nil {
			baidu.handler = h
		}
		audioPacket.Payload, _ = media.ResamplePCM(audioPacket.Payload, h.GetSession().Codec().SampleRate, 16000)
		req := media.PacketRequest[*media.AudioPacket]{
			Req:       audioPacket,
			Interrupt: true,
		}
		return &req, nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		var err error
		url := fmt.Sprintf("%s?sn=%s", opt.Url, opt.CuId)
		baidu.conn, _, err = websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			logrus.WithError(err).Error("baidu asr: websocket connect failed")
			return err
		}

		beginParam := NewBeginParam(opt)
		bytes, err := json.Marshal(beginParam)
		if err != nil {
			logrus.WithError(err).Error("baidu asr: marshal begin param failed")
			return err
		}
		err = baidu.conn.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			return err
		}
		go baidu.recvFrames()
		return nil
	}

	executor.TerminateCallback = func(h media.MediaHandler) error {
		return baidu.close()
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.Hangup:
			return baidu.conn.WriteMessage(websocket.TextMessage, finishMessage)
		}
		return nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[*media.AudioPacket]) error {
		if baidu.sendReqTime == nil {
			n := time.Now()
			baidu.sendReqTime = &n
			logrus.Info("baidu asr start send request")
		}
		return baidu.conn.WriteMessage(websocket.BinaryMessage, req.Req.Payload[:])
	}

	return executor.HandleMediaData
}

func (baidu *BaiduASR) close() (err error) {
	err = baidu.conn.WriteMessage(websocket.TextMessage, cancelMessage)
	if err != nil {
		logrus.WithError(err).Error("baidu asr: cancel message send failed")
		return err
	}
	return baidu.conn.Close()
}

func (baidu *BaiduASR) recvFrames() {
	for {
		messageType, message, err := baidu.conn.ReadMessage()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"sessionID":   baidu.handler.GetSession().ID,
				"error":       err,
				"messageType": messageType,
				"message":     string(message),
			}).Error("baidu asr: recv message failed")
			baidu.handler.CauseError(baidu, err)
			if messageType != -1 {
				break
			}
		}
		if messageType == -1 {
			if baidu.sendReqTime != nil {
				baidu.handler.AddMetric("asr.baidu", time.Since(*baidu.sendReqTime))
			}
			break
		}

		var resp BaiduASRWSResponse
		err = json.Unmarshal(message, &resp)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"sessionID": baidu.handler.GetSession().ID,
				"error":     err,
			}).WithError(err).Error("baidu asr: failed to unmarshal message.")
			continue
		}

		if resp.ErrNo != 0 {
			logrus.WithFields(logrus.Fields{
				"sessionID": baidu.handler.GetSession().ID,
				"resp":      resp,
			}).Error("baidu asr: result error.")
			continue
		}

		if resp.Type == heartbeat {
			continue
		}
		if !baidu.ttfbDone {
			baidu.ttfbDone = true
			if baidu.sendReqTime != nil {
				baidu.handler.AddMetric("asr.baidu.ttfb", time.Since(*baidu.sendReqTime))
			}
		}

		baidu.Sentence = resp.Result
		logrus.WithFields(logrus.Fields{
			"sessionID": baidu.handler.GetSession().ID,
			"resp":      resp,
		}).Info("baidu asr: recv frame.")

		if resp.Type == midText {
			baidu.handler.EmitState(baidu, media.Transcribing, baidu.Sentence)
		}

		if resp.Type == finText {
			packet := &media.TextPacket{
				Text:          baidu.Sentence,
				IsTranscribed: true,
			}
			baidu.handler.EmitState(baidu, media.Completed, baidu.Sentence)
			baidu.handler.EmitPacket(baidu, packet)
		}
	}
}

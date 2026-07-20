package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/lingllm/media"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/sirupsen/logrus"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/asr"
	"github.com/tencentcloud/tencentcloud-speech-sdk-go/common"
)

type QCloudASR struct {
	Handler     media.MediaHandler
	sentence    string
	sliceType   uint32
	startTime   uint32
	endTime     uint32
	sendReqTime *time.Time
	endReqTime  *time.Time

	opt              QCloudASROption
	recognizer       *asr.SpeechRecognizer
	transcribeResult SpeechRecognitionResult
	processError     RecognitionError
	dialogID         string
}

type QCloudASROption struct {
	AppID       string    `json:"appId" yaml:"app_id" env:"QCLOUD_APP_ID"`
	SecretID    string    `json:"secretId" yaml:"secret_id" env:"QCLOUD_SECRET_ID"`
	SecretKey   string    `json:"secret" yaml:"secret" env:"QCLOUD_SECRET"`
	Format      int       `json:"format" yaml:"format" default:"1"`
	ModelType   string    `json:"modelType" yaml:"model_type" env:"QCLOUD_MODEL_TYPE" default:"16k_zh"`
	ReqChanSize int       `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
	HotWords    []HotWord `json:"hotWords" yaml:"hot_words"`
	// VadSilenceTime is cloud endpointing silence (ms) before OnSentenceEnd.
	// Vendor default is 1000 when omitted from the WS query; we apply
	// defaultQCloudVadSilenceMs when unset so contact-center turns cut faster.
	// Range 240–2000; requires NeedVad=1 (SDK default).
	VadSilenceTime int `json:"vadSilenceTime" yaml:"vad_silence_time"`
	// NeedVad mirrors needvad. Zero = default on (1). Set to -1 to force off.
	NeedVad int `json:"needVad" yaml:"need_vad"`
}

// defaultQCloudVadSilenceMs is the contact-center default when the tenant
// does not set vadSilenceTime. Cloud default without the query param is 1000ms.
const defaultQCloudVadSilenceMs = 300

func NewQcloudASROption(appId string, secretId string, secretKey string) QCloudASROption {
	return QCloudASROption{
		AppID:          appId,
		SecretID:       secretId,
		SecretKey:      secretKey,
		Format:         asr.AudioFormatPCM,
		ModelType:      "16k_zh",
		ReqChanSize:    128,
		VadSilenceTime: defaultQCloudVadSilenceMs,
	}
}

func (opt QCloudASROption) effectiveVadSilenceTime() int {
	v := opt.VadSilenceTime
	if v <= 0 {
		v = defaultQCloudVadSilenceMs
	}
	if v < 240 {
		return 240
	}
	if v > 2000 {
		return 2000
	}
	return v
}

func applyQCloudRecognizerParams(recognizer *asr.SpeechRecognizer, opt QCloudASROption) {
	if recognizer == nil {
		return
	}
	if opt.NeedVad < 0 {
		recognizer.NeedVad = 0
		return
	}
	// Default / explicit on: cloud VAD required for vad_silence_time.
	recognizer.NeedVad = 1
	recognizer.VadSilenceTime = opt.effectiveVadSilenceTime()
}

func WithQCloudASR(opt QCloudASROption) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[[]byte](opt.ReqChanSize)
	credential := common.NewCredential(opt.SecretID, opt.SecretKey)

	asq := &QCloudASR{opt: opt}
	recognizer := asr.NewSpeechRecognizer(opt.AppID, credential, opt.ModelType, asq)
	recognizer.VoiceFormat = opt.Format
	applyQCloudRecognizerParams(recognizer, opt)

	executor.ConcurrentMode = false // QCloud ASR write is not blocking so we need to set this to false
	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[[]byte], error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			h.EmitPacket(asq, packet)
			return nil, nil
		}
		if asq.Handler == nil {
			asq.Handler = h
		}
		req := media.PacketRequest[[]byte]{
			Req:       audioPacket.Payload,
			Interrupt: true,
		}
		return &req, nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		asq.Handler = h
		return recognizer.Start()
	}

	executor.TerminateCallback = func(h media.MediaHandler) error {
		err := recognizer.Stop()
		if err != nil && err.Error() == "recognizer is not running" {
			return nil
		}
		return err
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.Hangup:
			err := recognizer.Stop()
			if err != nil && err.Error() == "recognizer is not running" {
				return nil
			}
			return err
		case media.StartSilence:
			n := time.Now()
			asq.endReqTime = &n
		case media.StartSpeaking:
			n := time.Now()
			asq.sendReqTime = &n
		}
		return nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[[]byte]) error {
		if asq.sendReqTime == nil {
			n := time.Now()
			asq.sendReqTime = &n
			logrus.Info("qcloud asr: start send request")
		}
		return recognizer.Write(req.Req)
	}
	return executor.HandleMediaData
}

func (opt QCloudASROption) String() string {
	return fmt.Sprintf("QCloudASROption{AppID: %s, Format: %d, ModelType: %s, ReqChanSize: %d, VadSilenceTime: %d}",
		opt.AppID, opt.Format, opt.ModelType, opt.ReqChanSize, opt.effectiveVadSilenceTime())
}

// OnRecognitionStart implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnRecognitionStart(response *asr.SpeechRecognitionResponse) {
	logrus.WithField("voice_id", response.VoiceID).Info("OnRecognitionStart")
}

// OnSentenceBegin implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnSentenceBegin(response *asr.SpeechRecognitionResponse) {
	sendReqTime := time.Now()
	asq.sendReqTime = &sendReqTime
}

// OnRecognitionResultChange implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnRecognitionResultChange(response *asr.SpeechRecognitionResponse) {
	if asq.transcribeResult != nil {
		asq.transcribeResult(response.Result.VoiceTextStr, false, time.Since(*asq.sendReqTime), asq.dialogID)
		return
	}
}
// OnSentenceEnd — 一句说完，isLast 应为 true
func (asq *QCloudASR) OnSentenceEnd(response *asr.SpeechRecognitionResponse) {
	logFields := logrus.Fields{
		"voiceTextStr": response.Result.VoiceTextStr,
	}
	if asq.Handler != nil {
		logFields["sessionID"] = asq.Handler.GetSession().ID
	}
	logrus.WithFields(logFields).Info("qcloud: on sentence end")

	asq.sentence += response.Result.VoiceTextStr
	asq.sliceType = response.Result.SliceType
	asq.startTime = response.Result.StartTime
	asq.endTime = response.Result.EndTime

	completed := strings.TrimSpace(asq.sentence)
	if completed == "" {
		return
	}

	duration := time.Duration(0)
	if asq.sendReqTime != nil {
		duration = time.Since(*asq.sendReqTime)
	}

	if asq.transcribeResult != nil {
		asq.transcribeResult(completed, true, duration, asq.dialogID)
		asq.sentence = ""
		return
	}

	if asq.Handler != nil {
		packet := &media.TextPacket{
			Text:          completed,
			IsTranscribed: true,
		}
		asq.Handler.EmitPacket(asq.Handler, packet)
		asq.Handler.EmitState(asq, media.Completed, &media.CompletedData{
			SenderName: "asr.qcloud",
			Result:     completed,
			Duration:   duration,
		})
		asq.sentence = ""
	}
}

// OnRecognitionComplete — 会话结束，只 flush 没碰到 OnSentenceEnd 的尾巴
func (asq *QCloudASR) OnRecognitionComplete(response *asr.SpeechRecognitionResponse) {
	finalSentence := strings.TrimSpace(asq.sentence)
	asq.sentence = ""
	asq.sliceType = 0

	logFields := logrus.Fields{
		"voiceTextStr":  response.Result.VoiceTextStr,
		"finalSentence": finalSentence,
	}
	if asq.Handler != nil {
		logFields["sessionID"] = asq.Handler.GetSession().ID
	}
	logrus.WithFields(logFields).Info("qcloud: on recognition complete")

	if finalSentence != "" {
		duration := time.Duration(0)
		if asq.sendReqTime != nil {
			duration = time.Since(*asq.sendReqTime)
		}

		if asq.transcribeResult != nil {
			asq.transcribeResult(finalSentence, true, duration, asq.dialogID)
		} else if asq.Handler != nil {
			packet := &media.TextPacket{
				Text:          finalSentence,
				IsTranscribed: true,
			}
			asq.Handler.EmitPacket(asq.Handler, packet)
			asq.Handler.EmitState(asq, media.Completed, &media.CompletedData{
				SenderName: "asr.qcloud",
				Result:     finalSentence,
				Duration:   duration,
			})
		}
	}

	// Vendor closed the stream (idle / max duration). Clear handle so the
	// next SendAudioBytes path restarts; do not Start here to avoid races
	// with concurrent Write.
	asq.recognizer = nil
}

// OnFail implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnFail(response *asr.SpeechRecognitionResponse, err error) {
	if response.Code == 4008 {
		// no audio data send error
		return
	}
	if strings.Contains(err.Error(), "EOF") {
		logrus.WithFields(logrus.Fields{
			"voice_id": response.VoiceID,
			"error":    err,
		}).Warn("qcloud: eof onfail")
		return
	}
	logrus.WithFields(logrus.Fields{
		"voice_id": response.VoiceID,
		"error":    err,
	}).Error("OnFail")

	// 优先使用 processError 回调
	if asq.processError != nil {
		asq.processError(err, true)
		return
	}

	// 如果没有 processError 回调，尝试使用 Handler
	if asq.Handler != nil {
		asq.Handler.CauseError(asq, err)
	}
}

func NewQcloudASR(opt QCloudASROption) *QCloudASR {
	asq := &QCloudASR{opt: opt}
	return asq
}

func (asq *QCloudASR) Init(tr SpeechRecognitionResult, er RecognitionError) {
	asq.transcribeResult = tr
	asq.processError = er
}

func (asq *QCloudASR) Vendor() string {
	return "qcloud"
}

func (asq *QCloudASR) ConnAndReceive(dialogID string) error {
	asq.dialogID = dialogID
	credential := common.NewCredential(asq.opt.SecretID, asq.opt.SecretKey)
	recognizer := asr.NewSpeechRecognizer(asq.opt.AppID, credential, asq.opt.ModelType, asq)
	recognizer.VoiceFormat = asq.opt.Format
	applyQCloudRecognizerParams(recognizer, asq.opt)
	hotWords := asq.opt.HotWords

	var hotWordsStr string
	for _, hotWord := range hotWords {
		var weight string
		if hotWord.Weight > 0 {
			weight = fmt.Sprintf("%d", hotWord.Weight)
		} else {
			weight = "10"
		}
		wordStr := hotWord.Word + "|" + weight
		hotWordsStr += wordStr + ","
	}
	recognizer.HotwordList = strings.TrimSuffix(hotWordsStr, ",")
	if len(hotWordsStr) > 0 {
		logFields := logrus.Fields{
			"hotwords": recognizer.HotwordList,
		}
		if asq.Handler != nil {
			logFields["sessionID"] = asq.Handler.GetSession().ID
		}
		logrus.WithFields(logFields).Info("qcloud: hotwords")
	}
	err := recognizer.Start()
	if err != nil {
		logrus.WithError(err).Error("qcloud: recognizer.Start")
		return err
	}
	asq.recognizer = recognizer
	now := time.Now()
	asq.sendReqTime = &now
	asq.endReqTime = &now
	return nil
}

func (asq *QCloudASR) Activity() bool {
	return asq.recognizer != nil
}

func (asq *QCloudASR) RestartClient() {
	_ = asq.StopConn()
	dialogID, _ := gonanoid.Nanoid()
	_ = asq.ConnAndReceive(dialogID)
}

func (asq *QCloudASR) SendAudioBytes(data []byte) error {
	if data == nil {
		return nil
	}
	if asq.recognizer == nil {
		if len(data) == 0 {
			return nil
		}
		asq.RestartClient()
		if asq.recognizer == nil {
			return fmt.Errorf("recognizer is not running")
		}
	}
	err := asq.recognizer.Write(data)
	if err == nil {
		return nil
	}
	msg := err.Error()
	if !strings.Contains(msg, "not running") {
		return err
	}
	asq.RestartClient()
	if asq.recognizer == nil {
		return err
	}
	return asq.recognizer.Write(data)
}

func (asq *QCloudASR) SendEnd() error {
	if asq.recognizer != nil {
		_ = asq.recognizer.Stop()
		asq.recognizer = nil
	}
	return nil
}

func (asq *QCloudASR) StopConn() error {
	if asq.recognizer != nil {
		_ = asq.recognizer.Stop()
		asq.recognizer = nil
	}
	return nil
}

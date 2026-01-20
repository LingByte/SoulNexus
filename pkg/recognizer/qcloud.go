package recognizer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
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
	transcribeResult TranscribeResult
	processError     ProcessError
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
}

func NewQcloudASROption(appId string, secretId string, secretKey string) QCloudASROption {
	return QCloudASROption{
		AppID:       appId,
		SecretID:    secretId,
		SecretKey:   secretKey,
		Format:      asr.AudioFormatPCM,
		ModelType:   "16k_zh",
		ReqChanSize: 128,
	}
}

func WithQCloudASR(opt QCloudASROption) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[[]byte](opt.ReqChanSize)
	credential := common.NewCredential(opt.SecretID, opt.SecretKey)

	asq := &QCloudASR{opt: opt}
	recognizer := asr.NewSpeechRecognizer(opt.AppID, credential, opt.ModelType, asq)
	recognizer.VoiceFormat = opt.Format

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
	return fmt.Sprintf("QCloudASROption{AppID: %s, Format: %d, ModelType: %s, ReqChanSize: %d}",
		opt.AppID, opt.Format, opt.ModelType, opt.ReqChanSize)
}

// OnRecognitionStart implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnRecognitionStart(response *asr.SpeechRecognitionResponse) {
	logFields := logrus.Fields{
		"voice_id": response.VoiceID,
	}
	if asq.Handler != nil {
		logFields["handler"] = asq.Handler
	}
	logrus.WithFields(logFields).Info("OnRecognitionStart")
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

// OnSentenceEnd implementation of SpeechRecognitionListener
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
	if asq.transcribeResult != nil {
		asq.transcribeResult(asq.sentence, false, time.Since(*asq.sendReqTime), asq.dialogID)
		return
	}
}

// OnRecognitionComplete implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnRecognitionComplete(response *asr.SpeechRecognitionResponse) {
	finalSentence := asq.sentence
	asq.sentence = ""
	asq.sliceType = 0
	logFields := logrus.Fields{
		"voiceTextStr":  response.Result.VoiceTextStr,
		"finalSentence": finalSentence,
	}
	if asq.Handler != nil {
		logFields["sessionID"] = asq.Handler.GetSession().ID
	}
	logrus.WithFields(logFields).Info("qcloud: on sentence complete")

	// 优先使用 transcribeResult 回调
	if asq.transcribeResult != nil {
		asq.transcribeResult(finalSentence, true, time.Since(*asq.sendReqTime), asq.dialogID)
		return
	}

	// 如果没有 transcribeResult 回调，尝试使用 Handler
	if asq.Handler != nil {
		packet := &media.TextPacket{
			Text:          finalSentence,
			IsTranscribed: true,
		}
		asq.Handler.EmitPacket(asq.Handler, packet)
		asq.Handler.EmitState(asq, media.Completed, &media.CompletedData{
			SenderName: "asr.qcloud",
			Result:     finalSentence,
			Duration:   time.Since(*asq.sendReqTime),
		})
	}
}

// OnFail implementation of SpeechRecognitionListener
func (asq *QCloudASR) OnFail(response *asr.SpeechRecognitionResponse, err error) {
	if response.Code == 4008 {
		// no audio data send error
		return
	}
	if strings.Contains(err.Error(), "EOF") {
		logFields := logrus.Fields{
			"voice_id": response.VoiceID,
			"error":    err,
		}
		if asq.Handler != nil {
			logFields["handler"] = asq.Handler
		}
		logrus.WithFields(logFields).Warn("qcloud: eof onfail")
		return
	}
	logFields := logrus.Fields{
		"voice_id": response.VoiceID,
		"error":    err,
	}
	if asq.Handler != nil {
		logFields["handler"] = asq.Handler
	}
	logrus.WithFields(logFields).Error("OnFail")

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

func (asq *QCloudASR) Init(tr TranscribeResult, er ProcessError) {
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
	if asq.recognizer == nil || data == nil {
		return nil
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

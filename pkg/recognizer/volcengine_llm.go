package recognizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	gonanoid "github.com/matoous/go-nanoid"

	"github.com/sirupsen/logrus"
)

type VolcengineLLMASR struct {
	handler      media.MediaHandler
	opt          VolcengineLLMOption
	sendReqTime  time.Time
	endReqTime   *time.Time
	dialogID     string
	ttfbDone     bool
	audioDataLen int
	recognizer   *Recognizer
	tr           TranscribeResult
	er           ProcessError
}

type VolcengineLLMOption struct {
	Url         string    `json:"url" yaml:"url" default:"wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async"`
	ResourceId  string    `json:"resourceId" yaml:"resource_id" default:"volc.bigasr.sauc.duration"`
	AppID       string    `json:"appId" yaml:"app_id" env:"ASR_VOLC_LLM_APPID"`
	AccessToken string    `json:"accessToken" yaml:"access_token" env:"ASR_VOLC_LLM_ACCESS_TOKEN"`
	Format      string    `json:"format" yaml:"format" default:"pcm"`
	SampleRate  int       `json:"sampleRate" yaml:"sample_rate" default:"16000"`
	BitDepth    int       `json:"bitDepth" yaml:"bit_depth" default:"16"`
	Channel     int       `json:"channel" yaml:"channel" default:"1"`
	Codec       string    `json:"codec" yaml:"codec" default:"raw"`
	ReqChanSize int       `json:"reqChanSize" yaml:"req_chan_size" default:"128"`
	HotWords    []HotWord `json:"hotWords" yaml:"hot_words"`
}

func NewVolcengineLLMOption(token, appID string) VolcengineLLMOption {
	return VolcengineLLMOption{
		Url:         "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async",
		ResourceId:  "volc.bigasr.sauc.duration",
		AccessToken: token,
		AppID:       appID,
		Format:      "pcm",
		SampleRate:  16000,
		BitDepth:    16,
		Channel:     1,
		Codec:       "raw",
		ReqChanSize: 128,
	}
}

func NewVolcengineLLM(opt VolcengineLLMOption) VolcengineLLMASR {
	return VolcengineLLMASR{opt: opt}
}

func (v *VolcengineLLMASR) Init(tr TranscribeResult, er ProcessError) {
	v.tr = tr
	v.er = er
}

func (v *VolcengineLLMASR) Vendor() string {
	return "volcllmasr"
}

func (v *VolcengineLLMASR) ConnAndReceive(dialogID string) error {
	v.dialogID = dialogID

	config := DefaultConfig().
		WithURL(v.opt.Url).
		WithAuth(AuthConfig{
			ResourceId: v.opt.ResourceId,
			AccessKey:  v.opt.AccessToken,
			AppKey:     v.opt.AppID,
		}).
		WithAudio(AudioConfig{
			Format:  v.opt.Format,
			Codec:   v.opt.Codec,
			Rate:    v.opt.SampleRate,
			Bits:    v.opt.BitDepth,
			Channel: v.opt.Channel,
		}).
		WithBuffer(BufferConfig{
			SegmentDurationMs: 100,
		})

	// 设置热词上下文
	if len(v.opt.HotWords) > 0 {
		config.Request.Corpus.Context = GenerateCorpusContext(v.opt.HotWords)
	}

	v.recognizer = NewRecognizer(config)
	v.sendReqTime = time.Now()

	err := v.recognizer.Start()
	if err != nil {
		v.er(errors.New("failed to start recognizer"), true)
		return err
	}

	v.recognizer.SetResultCallback(func(result *Result) {
		v.handleRecognitionResult(result)
	})

	v.recognizer.SetErrorCallback(func(err error) {
		v.er(fmt.Errorf("recognizer error: %s", err), true)
	})

	logrus.WithFields(logrus.Fields{
		"dialogId": v.dialogID,
		"traceId":  v.recognizer.GetTraceID(),
	}).Infof("volcenginellm asr: start recognize")

	return nil
}

func (v *VolcengineLLMASR) handleRecognitionResult(result *Result) {
	duration := time.Since(v.sendReqTime)
	v.tr(result.Text, result.IsFinal, duration, v.dialogID)
	if result.IsFinal {
		logrus.WithFields(logrus.Fields{
			"dialogId": v.dialogID,
			"traceId":  v.recognizer.GetTraceID(),
		}).Infof("volcenginellm asr: recv last result: %s", result.Text)

		if v.recognizer != nil {
			logrus.WithFields(logrus.Fields{
				"dialogId": v.dialogID,
				"traceId":  v.recognizer.GetTraceID(),
			}).Infof("volcenginellm asr: stop recognize")
			v.recognizer.Stop()
			v.recognizer = nil
		}

	}
}

func (v *VolcengineLLMASR) Activity() bool {
	return v.recognizer != nil
}

func (v *VolcengineLLMASR) RestartClient() {
	_ = v.StopConn()
	dialogID, _ := gonanoid.Nanoid()
	if err := v.ConnAndReceive(dialogID); err != nil {
		v.er(err, true)
	}
}

func (v *VolcengineLLMASR) SendAudioBytes(data []byte) error {
	if v.recognizer != nil {
		err := v.recognizer.SendAudioFrame(data, false)
		if errors.Is(err, ErrClientClosed) {
			return nil
		}
		return err
	}
	return nil
}

func (v *VolcengineLLMASR) SendEnd() error {
	if v.recognizer != nil {
		logrus.WithFields(logrus.Fields{
			"dialogId": v.dialogID,
			"traceId":  v.recognizer.GetTraceID(),
		}).Infof("volcenginellm asr: end recognize")
		return v.recognizer.SendAudioFrame(nil, true)
	}
	return nil
}

func (v *VolcengineLLMASR) StopConn() error {
	if v.recognizer != nil {
		logrus.WithFields(logrus.Fields{
			"dialogId": v.dialogID,
			"traceId":  v.recognizer.GetTraceID(),
		}).Infof("volcenginellm asr: stop recognize")
		v.recognizer.Stop()
		v.recognizer = nil
	}

	return nil
}

func GenerateCorpusContext(hotwords []HotWord) string {
	type Hotword struct {
		Word string `json:"word"`
	}
	type Context struct {
		Hotwords []Hotword `json:"hotwords"`
	}

	var ctx Context
	for _, w := range hotwords {
		ctx.Hotwords = append(ctx.Hotwords, Hotword{Word: w.Word})
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		return ""
	}
	return string(data)
}

package synthesizer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/voiceclone"
	"github.com/sirupsen/logrus"
)

// VolcengineCloneOption configures Volcengine voice-clone TTS (volcano_icl).
type VolcengineCloneOption struct {
	AppID       string  `json:"appId"`
	AccessToken string  `json:"accessToken"`
	Cluster     string  `json:"cluster"`
	ResourceID  string  `json:"resourceId"`
	ModelType   int     `json:"modelType"`
	AssetID     string  `json:"assetId"`
	Encoding    string  `json:"encoding"`
	Rate        int     `json:"rate"`       // pipeline output sample rate (default 16000)
	SourceRate  int     `json:"sourceRate"` // API PCM rate when it differs from Rate
	SpeedRatio  float64 `json:"speedRatio"`
	Streaming   bool    `json:"streaming"`
}

func (o *VolcengineCloneOption) GetProvider() TTSProvider {
	return ProviderVolcengineClone
}

// VolcengineCloneEngine wraps voiceclone.VoiceCloneService as AudioSynthesisEngine.
type VolcengineCloneEngine struct {
	svc voiceclone.VoiceCloneService
	opt VolcengineCloneOption
	mu  sync.Mutex
}

// NewVolcengineCloneEngine builds a clone TTS engine from option.
func NewVolcengineCloneEngine(opt VolcengineCloneOption) (*VolcengineCloneEngine, error) {
	if opt.AppID == "" || opt.AccessToken == "" {
		return nil, fmt.Errorf("volcengine_clone: appId and accessToken are required")
	}
	if opt.AssetID == "" {
		return nil, fmt.Errorf("volcengine_clone: assetId is required")
	}
	cluster := opt.Cluster
	if cluster == "" || cluster == VolcengineLLMCluster {
		cluster = VolcengineCloneCluster
	}
	encoding := opt.Encoding
	if encoding == "" {
		encoding = "pcm"
	}
	sourceRate := opt.SourceRate
	if sourceRate <= 0 {
		sourceRate = opt.Rate
	}
	if sourceRate <= 0 {
		sourceRate = 24000
	}
	outRate := opt.Rate
	if outRate <= 0 {
		outRate = 16000
	}
	speed := opt.SpeedRatio
	if speed <= 0 {
		speed = 1.0
	}

	resourceID := opt.ResourceID
	if resourceID == "" {
		resourceID = "seed-icl-2.0"
	}
	modelType := opt.ModelType
	if modelType == 0 {
		modelType = 4
	}
	svc, err := voiceclone.NewFactory().CreateService(&voiceclone.Config{
		Provider: voiceclone.ProviderVolcengine,
		Options: map[string]interface{}{
			"app_id":      opt.AppID,
			"token":       opt.AccessToken,
			"cluster":     cluster,
			"resource_id": resourceID,
			"model_type":  modelType,
			"encoding":    encoding,
			"sample_rate": sourceRate,
			"speed_ratio": speed,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("volcengine_clone: %w", err)
	}
	opt.Cluster = cluster
	opt.Encoding = encoding
	opt.SourceRate = sourceRate
	opt.Rate = outRate
	opt.SpeedRatio = speed
	return &VolcengineCloneEngine{svc: svc, opt: opt}, nil
}

func (e *VolcengineCloneEngine) Provider() TTSProvider {
	return ProviderVolcengineClone
}

func (e *VolcengineCloneEngine) Format() media.StreamFormat {
	e.mu.Lock()
	rate := e.opt.Rate
	e.mu.Unlock()
	if rate <= 0 {
		rate = 16000
	}
	return media.StreamFormat{
		SampleRate:    rate,
		BitDepth:      16,
		Channels:      1,
		FrameDuration: 20 * time.Millisecond,
	}
}

func (e *VolcengineCloneEngine) Capabilities() TTSCapabilities {
	return TTSCapabilities{
		StreamingTTFB:          true,
		SuggestedFirstMaxRunes: 24,
	}
}

func (e *VolcengineCloneEngine) CacheKey(text string) string {
	e.mu.Lock()
	asset := e.opt.AssetID
	e.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("volcengine_clone-%s-%s.pcm", asset, digest)
}

func (e *VolcengineCloneEngine) Synthesize(ctx context.Context, handler AudioSynthesisHandler, text string) error {
	if e == nil || e.svc == nil {
		return fmt.Errorf("volcengine_clone: nil engine")
	}
	if text == "" {
		return nil
	}
	e.mu.Lock()
	opt := e.opt
	e.mu.Unlock()

	pcmHandler := &clonePCMHandler{
		out:     handler,
		srcRate: opt.SourceRate,
		outRate: opt.Rate,
	}
	req := &voiceclone.SynthesizeRequest{
		AssetID:  opt.AssetID,
		Text:     text,
		Language: "zh",
	}
	err := e.svc.SynthesizeStream(ctx, req, &voicecloneStreamAdapter{inner: pcmHandler})
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	logrus.WithError(err).WithFields(logrus.Fields{
		"assetId": opt.AssetID,
		"cluster": opt.Cluster,
		"text":    text,
	}).Warn("volcengine_clone: icl ws failed, trying standard volcano_tts ws")

	errStd := e.synthesizeViaStandardWS(ctx, pcmHandler, opt, text)
	if errStd == nil {
		return nil
	}
	if !errors.Is(errStd, context.Canceled) {
		logrus.WithError(errStd).WithFields(logrus.Fields{
			"assetId": opt.AssetID,
			"text":    text,
		}).Error("volcengine_clone: all synthesis paths failed")
		return fmt.Errorf("volcengine_clone: icl=%v; standard=%w", err, errStd)
	}
	return errStd
}

// synthesizeViaStandardWS uses the standard TTS websocket (no Resource-Id header).
// Many S_ speaker IDs trained via mega_tts work on volcano_tts this way.
func (e *VolcengineCloneEngine) synthesizeViaStandardWS(ctx context.Context, handler *clonePCMHandler, opt VolcengineCloneOption, text string) error {
	listener := &volcengineSpeechSynthesisListener{handler: handler}
	ttsOpt := VolcengineTTSOption{
		AppID:       opt.AppID,
		AccessToken: opt.AccessToken,
		Cluster:     VolcengineLLMCluster,
		VoiceType:   opt.AssetID,
		Rate:        opt.SourceRate,
		Encoding:    opt.Encoding,
		SpeedRatio:  float32(opt.SpeedRatio),
		Streaming:   true,
	}
	_, err := listener.sendStreamRequest(ctx, ttsOpt, text)
	return err
}

func (e *VolcengineCloneEngine) Close() error {
	return nil
}

type clonePCMHandler struct {
	out     AudioSynthesisHandler
	srcRate int
	outRate int
}

func (h *clonePCMHandler) OnMessage(data []byte) {
	if h.out == nil || len(data) == 0 {
		return
	}
	pcm := data
	if h.srcRate > 0 && h.outRate > 0 && h.srcRate != h.outRate {
		if resampled, err := media.ResamplePCM(data, h.srcRate, h.outRate); err == nil && len(resampled) > 0 {
			pcm = resampled
		}
	}
	h.out.OnMessage(pcm)
}

func (h *clonePCMHandler) OnTimestamp(ts SentenceTimestamp) {
	if h.out != nil {
		h.out.OnTimestamp(ts)
	}
}

// voicecloneStreamAdapter bridges voiceclone.SynthesisHandler to clonePCMHandler.
type voicecloneStreamAdapter struct {
	inner *clonePCMHandler
}

func (a *voicecloneStreamAdapter) OnMessage(data []byte) {
	if a.inner != nil {
		a.inner.OnMessage(data)
	}
}

func (a *voicecloneStreamAdapter) OnTimestamp(ts voiceclone.SentenceTimestamp) {
	if a.inner == nil || a.inner.out == nil {
		return
	}
	a.inner.out.OnTimestamp(SentenceTimestamp{
		Words: []Word{{
			StartTime: int(ts.StartTime),
			EndTime:   int(ts.EndTime),
		}},
	})
}

package synthesizer

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/sirupsen/logrus"
)

var emojiRegex = regexp.MustCompile(`[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{2199}\x{21A9}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F004}\x{1F0CF}\x{1F170}-\x{1F251}\x{1F300}-\x{1F5FF}\x{1F600}-\x{1F64F}\x{1F680}-\x{1F6FF}\x{1F910}-\x{1F93E}\x{1F940}-\x{1F94C}\x{1F950}-\x{1F96B}\x{1F980}-\x{1F997}\x{1F9C0}-\x{1F9E6}\x{1FA70}-\x{1FA74}\x{1FA78}-\x{1FA7A}\x{1FA80}-\x{1FA86}\x{1FA90}-\x{1FAA8}\x{1FAB0}-\x{1FAB6}\x{1FAC0}-\x{1FAC2}\x{1FAD0}-\x{1FAD6}\x{1F1E6}-\x{1F1FF}\x{200D}\x{FE0F}]`)

type Word struct {
	Confidence float64 `json:"confidence"`
	EndTime    int     `json:"end_time"`
	StartTime  int     `json:"start_time"`
	Word       string  `json:"word"`
}

type SentenceTimestamp struct {
	Words []Word `json:"words"`
}

type SynthesisHandler interface {
	OnMessage([]byte)
	OnTimestamp(timestamp SentenceTimestamp)
}

type SynthesisService interface {
	Provider() TTSProvider
	Format() media.StreamFormat
	CacheKey(text string) string
	Synthesize(ctx context.Context, handler SynthesisHandler, text string) error
	Close() error
}

type SynthesisRequest struct {
	handler       media.MediaHandler
	player        *SynthesisPlayer
	result        []byte
	waitTTFB      bool
	startTime     time.Time
	packet        *media.TextPacket
	svc           SynthesisService
	sequence      int
	PlayID        string
	dialogStartAt time.Time
}

func (req *SynthesisRequest) OnTimestamp(timestamp SentenceTimestamp) {
}

type SynthesisPlayer struct {
	SenderName  string
	Format      media.StreamFormat
	reqChan     chan *SynthesisPlayerRequest
	txqueue     []*SynthesisPlayerRequest
	playRecords map[string]*PlayRecord
	lock        sync.RWMutex
}

type SynthesisPlayerRequest struct {
	h             media.MediaHandler
	packet        *media.AudioPacket
	sent          int
	interruptPlay string
}

type PlayRecord struct {
	interruptReason string
	sequences       map[int]string
}

func (req *SynthesisRequest) OnMessage(data []byte) {
	firstFrame := false
	if req.waitTTFB {
		req.waitTTFB = false
		req.handler.AddMetric("tts"+req.svc.Provider().ToString()+".completion.ttfb", time.Since(req.startTime))
		firstFrame = true

		if !req.dialogStartAt.IsZero() {
			milliseconds := time.Since(req.dialogStartAt).Milliseconds()
			if milliseconds < 30000 {
				logrus.WithFields(logrus.Fields{
					"sessionID":  req.handler.GetSession().ID,
					"ttfb":       milliseconds,
					"ttfbType":   "dialog",
					"dialogID":   req.PlayID,
					"dialogType": "segment",
				}).Info("ttfb")
			}
		}
	}

	if firstFrame {
		data = encoder.StripWavHeader(data)
	}
	packet := &media.AudioPacket{
		Payload:       data,
		IsSynthesized: true,
		IsFirstPacket: firstFrame,
		PlayID:        req.PlayID,
		Sequence:      req.sequence,
		SourceText:    req.packet.Text,
	}
	if req.sequence > 0 {
		packet.IsFirstPacket = false
	}
	req.result = append(req.result, data...)
	req.player.Emit(req.handler, packet, req.svc.Format().SampleRate)
}

func (player *SynthesisPlayer) Emit(h media.MediaHandler, audioPacket *media.AudioPacket, inputRate int) {
	if audioPacket != nil && audioPacket.Payload != nil {
		var err error
		audioPacket.Payload, err = media.ResamplePCM(audioPacket.Payload, inputRate, player.Format.SampleRate)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"inputRate":  inputRate,
				"outputRate": player.Format.SampleRate,
			}).WithError(err).Error("synthesis: resample error")
			return
		}
	}

	if player.Format.FrameDuration <= 0 {
		h.EmitPacket(player, audioPacket)
		return
	}
	player.reqChan <- &SynthesisPlayerRequest{
		h:      h,
		packet: audioPacket,
	}
}

func StripEmoji(text string) string {
	return emojiRegex.ReplaceAllString(text, "")
}

func WithSynthesis(svc SynthesisService) media.MediaHandlerFunc {
	executor := media.NewAsyncTaskRunner[*SynthesisRequest](1)
	executor.ConcurrentMode = true

	player := NewSynthesisPlayer("tts."+svc.Provider().ToString(), svc.Format())

	executor.RequestBuilder = func(h media.MediaHandler, packet media.MediaPacket) (*media.PacketRequest[*SynthesisRequest], error) {
		textPacket, ok := packet.(*media.TextPacket)
		if !ok {
			h.EmitPacket(h, packet)
			return nil, nil
		}
		req := &SynthesisRequest{
			handler:       h,
			player:        player,
			packet:        textPacket,
			waitTTFB:      true,
			startTime:     time.Now(),
			svc:           svc,
			sequence:      textPacket.Sequence,
			PlayID:        textPacket.PlayID,
			dialogStartAt: textPacket.StartAt,
		}
		return &media.PacketRequest[*SynthesisRequest]{
			Req:       req,
			Interrupt: true,
		}, nil
	}

	executor.TaskExecutor = func(ctx context.Context, h media.MediaHandler, req media.PacketRequest[*SynthesisRequest]) error {
		if req.Req.sequence == 0 {
			logrus.WithFields(logrus.Fields{
				"handler":  h,
				"playID":   req.Req.PlayID,
				"sequence": req.Req.sequence,
			}).Info("synthesis: interrupting current play by new req")
			player.Interrupt(h, "nextFirstFrameComing")
		}
		text := StripEmoji(req.Req.packet.Text)
		if strings.TrimSpace(text) == "" {
			logrus.WithFields(logrus.Fields{
				"handler":  h,
				"playID":   req.Req.PlayID,
				"sequence": req.Req.sequence,
				"source":   req.Req.packet.Text,
				"vendor":   svc.Provider().ToString(),
			}).Warn("synthesis: empty text may cause synthesize trouble")
			req.Req.OnMessage(make([]byte, 0))
		} else {
			cacheKey := req.Req.svc.CacheKey(req.Req.packet.Text)
			data, err := media.MediaCache().Get(cacheKey)
			if err == nil {
				logrus.WithFields(logrus.Fields{
					"handler":  h,
					"playID":   req.Req.PlayID,
					"cachekey": cacheKey,
					"text":     req.Req.packet.Text,
					"vendor":   svc.Provider().ToString(),
					"playId":   req.Req.PlayID,
				}).Info("synthesis: cache hit")
				req.Req.OnMessage(data)
			} else {
				h.EmitState(req.Req, media.Synthesizing, text)
				err = svc.Synthesize(ctx, req.Req, text)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"handler":     h,
						"playID":      req.Req.PlayID,
						"vendor":      svc.Provider().ToString(),
						"text":        req.Req.packet.Text,
						"stripedText": text,
					}).WithError(err).Error("synthesis: synthesize error")
					return err
				}
				if len(req.Req.result) > 0 && cacheKey != "" {
					media.MediaCache().Store(cacheKey, req.Req.result)
				}
			}
		}

		packet := &media.AudioPacket{ // finish play
			Payload:       nil,
			IsSynthesized: true,
			IsFirstPacket: false,
			IsEndPacket:   true,
			PlayID:        req.Req.PlayID,
			Sequence:      req.Req.sequence,
			SourceText:    req.Req.packet.Text,
		}
		player.Emit(h, packet, req.Req.svc.Format().SampleRate)

		if (req.Req.packet.IsPartial && req.Req.packet.IsEnd) || !req.Req.packet.IsPartial {
			completedData := &media.CompletedData{
				SenderName: "tts." + svc.Provider().ToString(),
				Source:     req.Req.packet,
			}
			h.EmitState(req.Req, media.Completed, completedData)
		}
		return nil
	}

	executor.StateCallback = func(h media.MediaHandler, event media.StateChange) error {
		switch event.State {
		case media.Interruption:
			logrus.WithFields(logrus.Fields{
				"handler": h,
			}).Info("synthesis: interrupting current play")
			player.Interrupt(h, "interrupt")
		}
		return nil
	}

	executor.InitCallback = func(h media.MediaHandler) error {
		format := svc.Format()
		format.SampleRate = h.GetSession().SampleRate // TODO: player must as same as session sample rate
		player.Format = format
		if format.FrameDuration > 0 {
			go player.Run(h, h.GetContext())
		}
		return nil
	}
	executor.TerminateCallback = func(h media.MediaHandler) error {
		player.Close()
		return nil
	}
	return executor.HandleMediaData
}

func NewSynthesisPlayer(vendor string, format media.StreamFormat) *SynthesisPlayer {
	return &SynthesisPlayer{
		SenderName:  vendor,
		Format:      format,
		reqChan:     make(chan *SynthesisPlayerRequest, 1),
		playRecords: make(map[string]*PlayRecord),
	}
}

func (player *SynthesisPlayer) Close() {
	logrus.WithFields(logrus.Fields{
		"vendor": player.SenderName,
	}).Info("synthesis: closed")
}

func (player *SynthesisPlayer) Interrupt(h media.MediaHandler, reason string) {
	player.reqChan <- &SynthesisPlayerRequest{
		interruptPlay: reason,
	}
}

func (player *SynthesisPlayer) Run(handler media.MediaHandler, ctx context.Context) {
	if player.Format.FrameDuration <= 0 {
		return
	}
	t := time.NewTicker(player.Format.FrameDuration)
	frameSize := utils.ComputeSampleByteCount(player.Format.SampleRate, player.Format.BitDepth, player.Format.Channels) * int(player.Format.FrameDuration.Milliseconds())
	st := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			packet := player.streamFrame(&st, frameSize)
			if packet != nil {
				if packet.Payload != nil {
					handler.EmitPacket(player, packet)
				}
				if packet.IsEndPacket {
					player.EmitStopPlayState(handler, time.Since(st).String(), packet.PlayID, packet.Sequence, "finished", packet.SourceText)
				}
			}
		case req := <-player.reqChan:
			player.handleRequest(st, req)
		}
	}
}

func (player *SynthesisPlayer) streamFrame(st *time.Time, frameSize int) *media.AudioPacket {
	if len(player.txqueue) <= 0 {
		return nil
	}
	isFirstFrame := false
	current := player.txqueue[0]
	if current.packet.IsFirstPacket && current.sent == 0 {
		isFirstFrame = true
		*st = time.Now()
		current.h.EmitState(player.SenderName, media.StartPlay, "", current.packet.PlayID, current.packet.Sequence, "", current.packet.SourceText)
	}

	if current.packet.Payload == nil {
		player.txqueue = player.txqueue[1:]
		return &media.AudioPacket{
			Payload:       nil,
			IsSynthesized: true,
			IsFirstPacket: isFirstFrame,
			IsEndPacket:   true,
			PlayID:        current.packet.PlayID,
			Sequence:      current.packet.Sequence,
			SourceText:    current.packet.SourceText,
		}
	}

	playId := current.packet.PlayID
	sequence := current.packet.Sequence
	isEndPacket := current.packet.IsEndPacket
	sourceText := current.packet.SourceText

	var buf = make([]byte, frameSize)

	remaining := frameSize
	for len(player.txqueue) > 0 && remaining > 0 {
		req := player.txqueue[0]
		isEndPacket = req.packet.IsEndPacket
		if len(req.packet.Payload) == 0 {
			player.txqueue = player.txqueue[1:]
			break
		}
		available := len(req.packet.Payload) - req.sent
		if available > 0 {
			n := copy(buf[frameSize-remaining:], req.packet.Payload[req.sent:])
			remaining -= n
			req.sent += n
		}
		if req.sent >= len(req.packet.Payload) {
			player.txqueue = player.txqueue[1:]
		}
	}
	if remaining > 0 {
		logrus.WithFields(logrus.Fields{
			"playId":    playId,
			"sequence":  sequence,
			"remaining": remaining,
			"frameSize": frameSize,
		}).Info("synthesis: remaining")
		buf = buf[:frameSize-remaining]
	}

	return &media.AudioPacket{
		Payload:       buf,
		Sequence:      sequence,
		IsSynthesized: true,
		IsFirstPacket: isFirstFrame,
		IsEndPacket:   isEndPacket,
		PlayID:        playId,
		SourceText:    sourceText,
	}
}

func (player *SynthesisPlayer) handleRequest(st time.Time, req *SynthesisPlayerRequest) {
	if req.interruptPlay == "" {
		if player.isInterrupted(req.packet.PlayID) {
			player.EmitStopPlayState(req.h, time.Since(st).String(), req.packet.PlayID, req.packet.Sequence,
				player.playRecords[req.packet.PlayID].interruptReason, req.packet.SourceText)
			return
		}
		player.txqueue = append(player.txqueue, req)
		return
	}
	if len(player.txqueue) <= 0 {
		return
	}
	//
	logrus.WithFields(logrus.Fields{
		"count": len(player.txqueue),
	}).Info("synthesis: discard wait play list")

	prev := player.txqueue
	for _, prevReq := range prev {
		if prevReq.packet == nil || prevReq.packet.IsEndPacket {
			continue
		}
		player.EmitStopPlayState(prevReq.h, time.Since(st).String(), prevReq.packet.PlayID, prevReq.packet.Sequence, req.interruptPlay, prevReq.packet.SourceText)
	}
	player.txqueue = nil
}

func (player *SynthesisPlayer) isPlayStop(playId string, sequence int) bool {
	player.lock.RLock()
	defer player.lock.RUnlock()
	val, ok := player.playRecords[playId]
	if !ok {
		return false
	}
	// already play stop
	_, ok = val.sequences[sequence]
	if !ok {
		return false
	}
	return true
}

func (player *SynthesisPlayer) isInterrupted(playId string) bool {
	player.lock.RLock()
	defer player.lock.RUnlock()
	val, ok := player.playRecords[playId]
	if !ok {
		return false
	}
	return val.interruptReason == "interrupt"
}

func (player *SynthesisPlayer) playStop(playId string, sequence int, reason string) {
	player.lock.Lock()
	defer player.lock.Unlock()

	val, ok := player.playRecords[playId]
	if !ok {
		val = &PlayRecord{
			interruptReason: reason,
			sequences:       make(map[int]string),
		}
		player.playRecords[playId] = val
	}
	val.sequences[sequence] = reason
}

func (player *SynthesisPlayer) EmitStopPlayState(h media.MediaHandler, duration string, playId string, sequence int, reason string, sourceText string) {
	if player.isPlayStop(playId, sequence) {
		return
	}
	player.playStop(playId, sequence, reason)

	h.EmitState(player.SenderName, media.StopPlay, duration, playId, sequence, reason, sourceText)
}

type SynthesisBuffer struct {
	Data      []byte
	Timestamp SentenceTimestamp
}

func (s *SynthesisBuffer) OnMessage(data []byte) {
	s.Data = append(s.Data, data...)
}

func (s *SynthesisBuffer) OnTimestamp(timestamp SentenceTimestamp) {
	s.Timestamp = timestamp
}

func NewSynthesisService(name string, options map[string]any) (SynthesisService, error) {
	switch name {
	case TTS_QCLOUD:
		opt := media.CastOption[QCloudTTSConfig](options)
		return NewQCloudService(opt), nil
	case TTS_XUNFEI:
		opt := media.CastOption[XunfeiTTSConfig](options)
		return NewXunfeiService(opt), nil
	case TTS_QINIU:
		opt := media.CastOption[QiniuTTSConfig](options)
		return NewQiniuService(opt), nil
	case TTS_AWS:
		opt := media.CastOption[AmazonTTSConfig](options)
		return NewAmazonService(opt), nil
	case TTS_BAIDU:
		opt := media.CastOption[BaiduTTSConfig](options)
		return NewBaiduService(opt), nil
	case TTS_GOOGLE:
		opt := media.CastOption[GoogleTTSOption](options)
		return NewGoogleService(opt), nil
	case TTS_AZURE:
		opt := media.CastOption[AzureConfig](options)
		return NewAzureService(opt), nil
	case TTS_OPENAI:
		opt := media.CastOption[OpenAIConfig](options)
		return NewOpenAIService(opt), nil
	case TTS_ELEVENLABS:
		opt := media.CastOption[ElevenLabsConfig](options)
		return NewElevenLabsService(opt), nil
	case TTS_LOCAL:
		opt := media.CastOption[LocalTTSConfig](options)
		return NewLocalService(opt), nil
	case TTS_FISHSPEECH:
		opt := media.CastOption[FishSpeechConfig](options)
		return NewFishSpeechService(opt), nil
	case TTS_COQUI:
		opt := media.CastOption[CoquiTTSOption](options)
		return NewCoquiService(opt), nil
	case TTS_VOLCENGINE:
		opt := media.CastOption[VolcengineTTSOption](options)
		return NewVolcengineService(opt), nil
	case TTS_MINIMAX:
		opt := media.CastOption[MinimaxOption](options)
		return NewMinimaxService(opt), nil
	default:
		return nil, fmt.Errorf("synthesis: unknown synthesis: %s", name)
	}
}

// TTSCredentialConfig TTS凭证配置结构（灵活的键值对配置）
type TTSCredentialConfig map[string]interface{}

// getString 从配置中获取字符串值
func (c TTSCredentialConfig) getString(key string) string {
	if val, ok := c[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		// 尝试转换为字符串
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// getInt64 从配置中获取 int64 值
func (c TTSCredentialConfig) getInt64(key string) int64 {
	if val, ok := c[key]; ok {
		switch v := val.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// NewSynthesisServiceFromCredential 根据凭证配置创建TTS服务
func NewSynthesisServiceFromCredential(config TTSCredentialConfig) (SynthesisService, error) {
	if config == nil || len(config) == 0 {
		return nil, fmt.Errorf("TTS配置为空")
	}

	provider := strings.ToLower(strings.TrimSpace(config.getString("provider")))
	if provider == "" {
		return nil, fmt.Errorf("TTS provider 未配置")
	}

	var providerName string
	var options map[string]any

	switch provider {
	case "qiniu":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("七牛云TTS配置不完整：缺少apiKey")
		}
		providerName = TTS_QINIU
		baseURL := config.getString("baseUrl")
		if baseURL == "" {
			baseURL = config.getString("base_url") // 兼容下划线格式
		}
		ttsConfig := NewQiniuTTSConfig(apiKey, baseURL)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(ttsConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化七牛云配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化七牛云配置失败: %w", err)
		}

	case "qcloud", "tencent":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id") // 兼容下划线格式
		}
		secretID := config.getString("secretId")
		if secretID == "" {
			secretID = config.getString("secret_id") // 兼容下划线格式
		}
		secretKey := config.getString("secretKey")
		if secretKey == "" {
			secretKey = config.getString("secret_key") // 兼容下划线格式
		}

		if appID == "" || secretID == "" || secretKey == "" {
			return nil, fmt.Errorf("腾讯云TTS配置不完整：缺少appId、secretId或secretKey")
		}

		providerName = TTS_QCLOUD
		voiceTypeStr := config.getString("voiceType")
		if voiceTypeStr == "" {
			voiceTypeStr = config.getString("voice_type") // 兼容下划线格式
		}
		if voiceTypeStr == "" {
			voiceTypeStr = "601002" // 默认值
		}
		voiceType := config.getInt64("voiceType")
		if voiceType == 0 {
			if v, err := strconv.ParseInt(voiceTypeStr, 10, 64); err == nil {
				voiceType = v
			} else {
				voiceType = 601002 // 默认值
			}
		}
		codec := config.getString("codec")
		if codec == "" {
			codec = "pcm"
		}
		// 支持配置采样率（默认16000）
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate")
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认16kHz
		}
		qcloudConfig := NewQcloudTTSConfig(appID, secretID, secretKey, voiceType, codec, int(sampleRate))
		// 读取语言配置（腾讯云通过音色类型区分语言，但保留此字段用于配置和缓存）
		language := config.getString("language")
		if language != "" {
			qcloudConfig.Language = language
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(qcloudConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化腾讯云配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化腾讯云配置失败: %w", err)
		}

	case "baidu":
		token := config.getString("token")
		if token == "" {
			// 兼容旧字段名
			token = config.getString("secretKey")
			if token == "" {
				token = config.getString("tok")
			}
		}
		if token == "" {
			return nil, fmt.Errorf("百度TTS配置不完整：缺少token")
		}
		providerName = TTS_BAIDU
		baiduConfig := NewBaiduTTSOption(token)
		// 读取语言配置（语言代码应该已经是平台特定的格式，如 zh, en, jp, kr）
		language := config.getString("language")
		if language == "" {
			language = config.getString("lan") // 兼容 lan 字段
		}
		if language != "" {
			// 直接使用传入的语言代码（应该已经是百度格式，如 zh, en, jp, kr）
			baiduConfig.Lan = language
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(baiduConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化百度配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化百度配置失败: %w", err)
		}

	case "azure":
		subscriptionKey := config.getString("subscriptionKey")
		if subscriptionKey == "" {
			subscriptionKey = config.getString("subscription_key") // 兼容下划线格式
		}
		region := config.getString("region")
		if subscriptionKey == "" || region == "" {
			return nil, fmt.Errorf("Azure TTS配置不完整：缺少subscriptionKey或region")
		}
		providerName = TTS_AZURE
		voice := config.getString("voice")
		if voice == "" {
			voice = "zh-CN-XiaoxiaoNeural" // 默认值
		}
		azureConfig := NewAzureConfig(subscriptionKey, region)
		azureConfig.Voice = voice
		// 读取语言配置（用于 SSML 中的 xml:lang）
		language := config.getString("language")
		if language == "" {
			// 如果没有指定语言，尝试从 voice 名称中提取（例如：zh-CN-XiaoxiaoNeural -> zh-CN）
			if parts := strings.Split(voice, "-"); len(parts) >= 2 {
				language = parts[0] + "-" + parts[1]
			} else {
				language = "zh-CN" // 默认值
			}
		}
		// 将语言信息存储到配置中（Azure 服务会使用它来设置 SSML 的 xml:lang）
		azureConfig.Language = language
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(azureConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Azure配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Azure配置失败: %w", err)
		}

	case "xunfei":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id") // 兼容下划线格式
		}
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		apiSecret := config.getString("apiSecret")
		if apiSecret == "" {
			apiSecret = config.getString("api_secret") // 兼容下划线格式
		}
		if appID == "" || apiKey == "" || apiSecret == "" {
			return nil, fmt.Errorf("科大讯飞TTS配置不完整：缺少appId、apiKey或apiSecret")
		}
		providerName = TTS_XUNFEI
		xunfeiConfig := NewXunfeiTTSConfig(appID, apiKey, apiSecret)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(xunfeiConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化科大讯飞配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化科大讯飞配置失败: %w", err)
		}

	case "openai":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI TTS配置不完整：缺少apiKey")
		}
		providerName = TTS_OPENAI
		baseURL := config.getString("baseUrl")
		if baseURL == "" {
			baseURL = config.getString("base_url") // 兼容下划线格式
		}
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		// OpenAI 配置
		options = map[string]any{
			"api_key":  apiKey,
			"base_url": baseURL,
		}

	case "google":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Google TTS配置不完整：缺少apiKey")
		}
		providerName = TTS_GOOGLE
		projectID := config.getString("projectId")
		if projectID == "" {
			projectID = config.getString("project_id") // 兼容下划线格式
		}
		languageCode := config.getString("languageCode")
		if languageCode == "" {
			languageCode = config.getString("language_code") // 兼容下划线格式
		}
		if languageCode == "" {
			languageCode = config.getString("language") // 也支持 language 字段
		}
		if languageCode == "" {
			languageCode = "en-US" // 默认值
		}
		googleConfig := NewGoogleTTSOption(languageCode)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(googleConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Google配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Google配置失败: %w", err)
		}
		// 保留 api_key 和 project_id（如果需要）
		if apiKey != "" {
			options["api_key"] = apiKey
		}
		if projectID != "" {
			options["project_id"] = projectID
		}

	case "aws":
		accessKeyID := config.getString("accessKeyId")
		if accessKeyID == "" {
			accessKeyID = config.getString("access_key_id") // 兼容下划线格式
		}
		secretAccessKey := config.getString("secretAccessKey")
		if secretAccessKey == "" {
			secretAccessKey = config.getString("secret_access_key") // 兼容下划线格式
		}
		if accessKeyID == "" || secretAccessKey == "" {
			return nil, fmt.Errorf("AWS TTS配置不完整：缺少accessKeyId或secretAccessKey")
		}
		providerName = TTS_AWS
		region := config.getString("region")
		if region == "" {
			region = "us-east-1" // 默认值
		}
		options = map[string]any{
			"access_key_id":     accessKeyID,
			"secret_access_key": secretAccessKey,
			"region":            region,
		}

	case "volcengine":
		appID := config.getString("appId")
		if appID == "" {
			appID = config.getString("app_id") // 兼容下划线格式
		}
		accessToken := config.getString("accessToken")
		if accessToken == "" {
			accessToken = config.getString("access_token") // 兼容下划线格式
		}
		cluster := config.getString("cluster")
		if cluster == "" {
			cluster = "volcano_tts" // 默认集群
		}
		if appID == "" || accessToken == "" {
			return nil, fmt.Errorf("火山引擎TTS配置不完整：缺少appId或accessToken")
		}
		providerName = TTS_VOLCENGINE
		voiceType := config.getString("voiceType")
		if voiceType == "" {
			voiceType = config.getString("voice_type") // 兼容下划线格式
		}
		if voiceType == "" {
			voiceType = "BV700_streaming" // 默认值
		}
		language := config.getString("language")
		rate := config.getInt64("rate")
		if rate == 0 {
			rate = 8000 // 默认采样率
		}
		encoding := config.getString("encoding")
		if encoding == "" {
			encoding = "pcm"
		}
		speedRatio := float32(1.0)
		if speedStr := config.getString("speedRatio"); speedStr != "" {
			if f, err := strconv.ParseFloat(speedStr, 32); err == nil {
				speedRatio = float32(f)
			}
		}
		volcengineConfig := NewVolcengineTTSOption(appID, accessToken, cluster)
		volcengineConfig.VoiceType = voiceType
		volcengineConfig.Language = language
		volcengineConfig.Rate = int(rate)
		volcengineConfig.Encoding = encoding
		volcengineConfig.SpeedRatio = speedRatio
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(volcengineConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化火山引擎配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化火山引擎配置失败: %w", err)
		}

	case "minimax":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("Minimax TTS配置不完整：缺少apiKey")
		}
		providerName = TTS_MINIMAX
		model := config.getString("model")
		if model == "" {
			model = "speech-2.5-turbo-preview" // 默认值
		}
		voiceID := config.getString("voiceId")
		if voiceID == "" {
			voiceID = config.getString("voice_id") // 兼容下划线格式
		}
		if voiceID == "" {
			voiceID = "male-qn-qingse" // 默认值
		}
		speedRatio := float64(1.0)
		if speedStr := config.getString("speedRatio"); speedStr != "" {
			if f, err := strconv.ParseFloat(speedStr, 64); err == nil {
				speedRatio = f
			}
		}
		volume := float64(1.0)
		if volStr := config.getString("volume"); volStr != "" {
			if f, err := strconv.ParseFloat(volStr, 64); err == nil {
				volume = f
			}
		}
		pitch := float64(0.0)
		if pitchStr := config.getString("pitch"); pitchStr != "" {
			if f, err := strconv.ParseFloat(pitchStr, 64); err == nil {
				pitch = f
			}
		}
		emotion := config.getString("emotion")
		if emotion == "" {
			emotion = "neutral" // 默认值
		}
		languageBoost := config.getString("languageBoost")
		if languageBoost == "" {
			languageBoost = config.getString("language_boost") // 兼容下划线格式
		}
		if languageBoost == "" {
			languageBoost = "auto" // 默认值
		}
		trainingTimes := int(1)
		if tt := config.getInt64("trainingTimes"); tt > 0 {
			trainingTimes = int(tt)
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 8000 // 默认值
		}
		bitrate := config.getInt64("bitrate")
		if bitrate == 0 {
			bitrate = 16 // 默认值
		}
		format := config.getString("format")
		if format == "" {
			format = "pcm" // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		frameDuration := config.getString("frameDuration")
		if frameDuration == "" {
			frameDuration = config.getString("frame_duration") // 兼容下划线格式
		}
		if frameDuration == "" {
			frameDuration = "20ms" // 默认值
		}
		minimaxConfig := NewMinimaxOption(apiKey)
		minimaxConfig.Model = model
		minimaxConfig.VoiceID = voiceID
		minimaxConfig.SpeedRatio = speedRatio
		minimaxConfig.Volume = volume
		minimaxConfig.Pitch = pitch
		minimaxConfig.Emotion = emotion
		minimaxConfig.LanguageBoost = languageBoost
		minimaxConfig.TrainingTimes = trainingTimes
		minimaxConfig.SampleRate = int(sampleRate)
		minimaxConfig.Bitrate = int(bitrate)
		minimaxConfig.Format = format
		minimaxConfig.Channels = int(channels)
		minimaxConfig.FrameDuration = frameDuration
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(minimaxConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Minimax配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Minimax配置失败: %w", err)
		}

	case "elevenlabs":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("ElevenLabs TTS配置不完整：缺少apiKey")
		}
		providerName = TTS_ELEVENLABS
		voiceID := config.getString("voiceId")
		if voiceID == "" {
			voiceID = config.getString("voice_id") // 兼容下划线格式
		}
		if voiceID == "" {
			voiceID = "21m00Tcm4TlvDq8ikWAM" // 默认值
		}
		modelID := config.getString("modelId")
		if modelID == "" {
			modelID = config.getString("model_id") // 兼容下划线格式
		}
		if modelID == "" {
			modelID = "eleven_monolingual_v1" // 默认值
		}
		languageCode := config.getString("languageCode")
		if languageCode == "" {
			languageCode = config.getString("language_code") // 兼容下划线格式
		}
		if languageCode == "" {
			languageCode = config.getString("language") // 也支持 language 字段
		}
		stability := float64(0.5)
		if stabStr := config.getString("stability"); stabStr != "" {
			if f, err := strconv.ParseFloat(stabStr, 64); err == nil {
				stability = f
			}
		}
		similarityBoost := float64(0.75)
		if simStr := config.getString("similarityBoost"); simStr != "" {
			if f, err := strconv.ParseFloat(simStr, 64); err == nil {
				similarityBoost = f
			}
		}
		style := float64(0.0)
		if styleStr := config.getString("style"); styleStr != "" {
			if f, err := strconv.ParseFloat(styleStr, 64); err == nil {
				style = f
			}
		}
		useSpeakerBoost := true
		if boostStr := config.getString("useSpeakerBoost"); boostStr != "" {
			useSpeakerBoost = boostStr == "true" || boostStr == "1"
		}
		elevenlabsConfig := NewElevenLabsConfig(apiKey, voiceID)
		elevenlabsConfig.ModelID = modelID
		elevenlabsConfig.Stability = stability
		elevenlabsConfig.SimilarityBoost = similarityBoost
		elevenlabsConfig.Style = style
		elevenlabsConfig.UseSpeakerBoost = useSpeakerBoost
		// 设置语言代码（如果提供）
		if languageCode != "" {
			elevenlabsConfig.LanguageCode = languageCode
		}
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(elevenlabsConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化ElevenLabs配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化ElevenLabs配置失败: %w", err)
		}

	case "local":
		command := config.getString("command")
		if command == "" {
			command = "say" // 默认值
		}
		providerName = TTS_LOCAL
		voice := config.getString("voice")
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		bitDepth := config.getInt64("bitDepth")
		if bitDepth == 0 {
			bitDepth = config.getInt64("bit_depth") // 兼容下划线格式
		}
		if bitDepth == 0 {
			bitDepth = 16 // 默认值
		}
		codec := config.getString("codec")
		if codec == "" {
			codec = "wav" // 默认值
		}
		outputDir := config.getString("outputDir")
		if outputDir == "" {
			outputDir = config.getString("output_dir") // 兼容下划线格式
		}
		if outputDir == "" {
			outputDir = "/tmp" // 默认值
		}
		localConfig := NewLocalTTSConfig(command)
		localConfig.Voice = voice
		localConfig.SampleRate = int(sampleRate)
		localConfig.Channels = int(channels)
		localConfig.BitDepth = int(bitDepth)
		localConfig.Codec = codec
		localConfig.OutputDir = outputDir
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(localConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化本地TTS配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化本地TTS配置失败: %w", err)
		}

	case "fishspeech":
		apiKey := config.getString("apiKey")
		if apiKey == "" {
			apiKey = config.getString("api_key") // 兼容下划线格式
		}
		if apiKey == "" {
			return nil, fmt.Errorf("FishSpeech TTS配置不完整：缺少apiKey")
		}
		providerName = TTS_FISHSPEECH
		referenceID := config.getString("referenceId")
		if referenceID == "" {
			referenceID = config.getString("reference_id") // 兼容下划线格式
		}
		if referenceID == "" {
			referenceID = "default" // 默认值
		}
		latency := config.getString("latency")
		if latency == "" {
			latency = "normal" // 默认值
		}
		version := config.getString("version")
		if version == "" {
			version = "s1" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 24000 // 默认值
		}
		codec := config.getString("codec")
		if codec == "" {
			codec = "wav" // 默认值
		}
		fishspeechConfig := NewFishSpeechConfig(apiKey, referenceID)
		fishspeechConfig.Latency = latency
		fishspeechConfig.Version = version
		fishspeechConfig.SampleRate = int(sampleRate)
		fishspeechConfig.Codec = codec
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(fishspeechConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化FishSpeech配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化FishSpeech配置失败: %w", err)
		}

	case "coqui":
		url := config.getString("url")
		if url == "" {
			return nil, fmt.Errorf("Coqui TTS配置不完整：缺少url")
		}
		providerName = TTS_COQUI
		language := config.getString("language")
		if language == "" {
			language = "en_US" // 默认值
		}
		speaker := config.getString("speaker")
		if speaker == "" {
			speaker = "p226" // 默认值
		}
		sampleRate := config.getInt64("sampleRate")
		if sampleRate == 0 {
			sampleRate = config.getInt64("sample_rate") // 兼容下划线格式
		}
		if sampleRate == 0 {
			sampleRate = 16000 // 默认值
		}
		channels := config.getInt64("channels")
		if channels == 0 {
			channels = 1 // 默认值
		}
		bitDepth := config.getInt64("bitDepth")
		if bitDepth == 0 {
			bitDepth = config.getInt64("bit_depth") // 兼容下划线格式
		}
		if bitDepth == 0 {
			bitDepth = 16 // 默认值
		}
		coquiConfig := NewCoquiTTSOption(url)
		coquiConfig.Language = language
		coquiConfig.Speaker = speaker
		coquiConfig.SampleRate = int(sampleRate)
		coquiConfig.Channels = int(channels)
		coquiConfig.BitDepth = int(bitDepth)
		// 将配置对象转换为 map[string]any
		configBytes, err := json.Marshal(coquiConfig)
		if err != nil {
			return nil, fmt.Errorf("序列化Coqui配置失败: %w", err)
		}
		options = make(map[string]any)
		if err := json.Unmarshal(configBytes, &options); err != nil {
			return nil, fmt.Errorf("反序列化Coqui配置失败: %w", err)
		}

	default:
		return nil, fmt.Errorf("不支持的TTS provider: %s", provider)
	}

	// 使用工厂方法创建服务
	return NewSynthesisService(providerName, options)
}

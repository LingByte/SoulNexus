// Package conversation wires SIP CallSession media to ASR → LLM → TTS using env-driven credentials.
package conversation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sipasr "github.com/LingByte/SoulNexus/pkg/sip/asr"
	sipdtmf "github.com/LingByte/SoulNexus/pkg/sip/dtmf"
	siptts "github.com/LingByte/SoulNexus/pkg/sip/tts"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voice"
	"go.uber.org/zap"
)

var (
	sipSystemPromptMu sync.Mutex
	sipSystemPromptByCallID = map[string]string{}
)

// SetSIPCallSystemPrompt overrides the default LLM system prompt for one call.
// It is consumed when the voice pipeline is attached for this call.
func SetSIPCallSystemPrompt(callID, prompt string) {
	callID = strings.TrimSpace(callID)
	prompt = strings.TrimSpace(prompt)
	if callID == "" || prompt == "" {
		return
	}
	sipSystemPromptMu.Lock()
	sipSystemPromptByCallID[callID] = prompt
	sipSystemPromptMu.Unlock()
}

func popSIPCallSystemPrompt(callID string) string {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return ""
	}
	sipSystemPromptMu.Lock()
	defer sipSystemPromptMu.Unlock()
	v := strings.TrimSpace(sipSystemPromptByCallID[callID])
	delete(sipSystemPromptByCallID, callID)
	return v
}

// VoiceEnv holds SIP voice pipeline settings read with utils.GetEnv (see SoulNexus .env).
type VoiceEnv struct {
	LLMProvider string
	LLMBaseURL  string
	LLMAppID    string
	LLMAPIKey   string
	LLMModel    string

	ASRAppID       string
	ASRSecretID    string
	ASRSecretKey   string
	ASRModelType   string

	TTSAppID       string
	TTSSecretID    string
	TTSSecretKey   string
	TTSVoiceType   int64
	TTSSpeed       int64
	TTSSampleRate  int
}

func voiceEnvFromProcess() VoiceEnv {
	voiceType, _ := strconv.ParseInt(strings.TrimSpace(utils.GetEnv("TTS_VOICE_TYPE")), 10, 64)
	ttsSpeed, _ := strconv.ParseInt(strings.TrimSpace(utils.GetEnv("TTS_SPEED")), 10, 64)
	sr, _ := strconv.Atoi(strings.TrimSpace(utils.GetEnv("TTS_SAMPLE_RATE")))
	if sr <= 0 {
		sr = 16000
	}
	provider := strings.TrimSpace(utils.GetEnv("LLM_PROVIDER"))
	appID := strings.TrimSpace(utils.GetEnv("LLM_APP_ID"))
	if appID == "" {
		appID = strings.TrimSpace(utils.GetEnv("ALIBABA_AI_APP_ID"))
	}
	apiKey := strings.TrimSpace(utils.GetEnv("LLM_APIKEY"))
	if apiKey == "" && strings.EqualFold(provider, "alibaba") {
		apiKey = strings.TrimSpace(utils.GetEnv("ALIBABA_AI_API_KEY"))
	}
	return VoiceEnv{
		LLMProvider: provider,
		LLMBaseURL:  strings.TrimSpace(utils.GetEnv("LLM_BASEURL")),
		LLMAppID:    appID,
		LLMAPIKey:   apiKey,
		LLMModel:    strings.TrimSpace(utils.GetEnv("LLM_MODEL")),

		ASRAppID:     strings.TrimSpace(utils.GetEnv("ASR_APPID")),
		ASRSecretID:  strings.TrimSpace(utils.GetEnv("ASR_SECRET_ID")),
		ASRSecretKey: strings.TrimSpace(utils.GetEnv("ASR_SECRET_KEY")),
		ASRModelType: strings.TrimSpace(utils.GetEnv("ASR_MODEL_TYPE")),

		TTSAppID:      strings.TrimSpace(utils.GetEnv("TTS_APPID")),
		TTSSecretID:   strings.TrimSpace(utils.GetEnv("TTS_SECRET_ID")),
		TTSSecretKey:  strings.TrimSpace(utils.GetEnv("TTS_SECRET_KEY")),
		TTSVoiceType:  voiceType,
		TTSSpeed:      ttsSpeed,
		TTSSampleRate: sr,
	}
}

func (e VoiceEnv) readyForVoice() bool {
	llmReady := e.LLMAPIKey != "" && e.LLMBaseURL != ""
	if strings.EqualFold(e.LLMProvider, "alibaba") {
		// Alibaba provider in pkg/llm consumes AppID in apiUrl slot.
		llmReady = e.LLMAPIKey != "" && strings.TrimSpace(e.LLMAppID) != ""
	}
	return e.ASRAppID != "" && e.ASRSecretID != "" && e.ASRSecretKey != "" &&
		llmReady &&
		e.TTSAppID != "" && e.TTSSecretID != "" && e.TTSSecretKey != ""
}

// AttachVoicePipeline registers a MediaSession processor that feeds decoded PCM into ASR,
// then on final transcripts runs the LLM and streams TTS back to the RTP output.
// Call once per call from the SIP ACK path before MediaSession.Serve() starts (before StartOnACK).
func AttachVoicePipeline(ctx context.Context, cs *sipSession.CallSession, lg *zap.Logger) error {
	if cs == nil {
		return nil
	}
	env := voiceEnvFromProcess()
	if !env.readyForVoice() {
		if lg != nil {
			lg.Info("sip voice pipeline skipped (missing ASR/LLM/TTS env)")
		}
		return nil
	}
	if lg == nil {
		if logger.Lg != nil {
			lg = logger.Lg
		} else {
			lg, _ = zap.NewDevelopment()
		}
	}

	return cs.AttachVoiceConversation(func() error {
		return attachVoiceInner(ctx, cs, env, lg)
	})
}

func sipHangupPhrasesFromEnv() []string {
	s := strings.TrimSpace(utils.GetEnv("SIP_AI_HANGUP_PHRASES"))
	if s == "" {
		return []string{"再见", "拜拜", "挂断", "先挂了", "挂了啊"}
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"再见", "拜拜"}
	}
	return out
}

// sipVADBargeInEnabled is true unless SIP_VAD_BARGE_IN is 0/false/off/no.
func sipVADBargeInEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(utils.GetEnv("SIP_VAD_BARGE_IN")))
	switch v {
	case "", "1", "true", "yes", "on":
		return true
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// sipVADDefaultThreshold RMS 上限（16-bit PCM 与 pkg/voice 一致）：默认提高到更保守值，降低TTS/线路回声误触发。
const sipVADDefaultThreshold = 3200.0

func sipVADThresholdFromEnv() float64 {
	s := strings.TrimSpace(utils.GetEnv("SIP_VAD_THRESHOLD"))
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return f
}

func sipVADConsecutiveFramesFromEnv() int {
	s := strings.TrimSpace(utils.GetEnv("SIP_VAD_CONSEC_FRAMES"))
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// sipASRTriggerPartialEnabled controls whether non-final ASR hypotheses can trigger LLM.
// Default false to avoid premature responses ("抢话") that sound like stutter/choppy dialog.
func sipASRTriggerPartialEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(utils.GetEnv("SIP_ASR_TRIGGER_PARTIAL")))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// sipASRPartialTimeoutMs is a fallback when providers keep emitting partial
// hypotheses but delay/omit final marks on noisy phone lines.
// <=0 disables timeout trigger.
func sipASRPartialTimeoutMs() int {
	s := strings.TrimSpace(utils.GetEnv("SIP_ASR_PARTIAL_TIMEOUT_MS"))
	if s == "" {
		return 1200
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 1200
	}
	if n <= 0 {
		return 0
	}
	if n < 300 {
		return 300
	}
	return n
}

// welcomeWaitFirstRTPMs is how long we wait for the first inbound RTP datagram before playing the welcome clip.
// Default 2000 ms gives the RTP layer time to learn symmetric NAT (remote send target updates from first packet).
// Set SIP_WELCOME_WAIT_FIRST_RTP_MS=0 to skip waiting (e.g. same-LAN tests).
func welcomeWaitFirstRTPMs() int {
	s := strings.TrimSpace(utils.GetEnv("SIP_WELCOME_WAIT_FIRST_RTP_MS"))
	if s == "" {
		return 2000
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 2000
	}
	if v < 0 {
		return 0
	}
	return v
}

func waitFirstRTPBeforeWelcome(ctx context.Context, cs *sipSession.CallSession, lg *zap.Logger, waitMs int) {
	if waitMs <= 0 || cs == nil {
		return
	}
	rtpSess := cs.RTPSession()
	if rtpSess == nil {
		return
	}
	t := time.NewTimer(time.Duration(waitMs) * time.Millisecond)
	defer t.Stop()
	select {
	case <-rtpSess.FirstPacket():
		if lg != nil {
			lg.Info("sip voice: inbound RTP before welcome (symmetric NAT can update send target)",
				zap.String("call_id", cs.CallID))
		}
	case <-t.C:
		if lg != nil {
			lg.Warn("sip voice: no inbound RTP before welcome wait — check client/NAT/firewall or set SIP_WELCOME_WAIT_FIRST_RTP_MS=0 for lab",
				zap.String("call_id", cs.CallID),
				zap.Int("wait_ms", waitMs),
			)
		}
	case <-ctx.Done():
		return
	}
}

func attachVoiceInner(ctx context.Context, cs *sipSession.CallSession, env VoiceEnv, lg *zap.Logger) error {
	ms := cs.MediaSession()
	if ms == nil {
		return fmt.Errorf("sip conversation: nil media session")
	}
	asrOpt := recognizer.NewQcloudASROption(env.ASRAppID, env.ASRSecretID, env.ASRSecretKey)
	if env.ASRModelType != "" {
		asrOpt.ModelType = env.ASRModelType
	}
	asrSvc := recognizer.NewQcloudASR(asrOpt)

	pipe, err := sipasr.New(sipasr.Options{
		ASR:        asrSvc,
		SampleRate: 16000,
		Channels:   1,
		Logger:     lg,
	})
	if err != nil {
		return fmt.Errorf("sip conversation: asr pipeline: %w", err)
	}

	llmModel := env.LLMModel
	if llmModel == "" {
		llmModel = "qwen-plus"
	}
	systemPrompt := popSIPCallSystemPrompt(cs.CallID)
	if systemPrompt == "" {
		systemPrompt = "你是七牛云技术支持语音助手。只用中文、直接回答、少废话：每次最多1-2句，优先给结论和下一步操作，不要举无关示例（如“比如对象存储/CDN/直播云”）。仅在必要时提1个澄清问题；不确定就明确说明并要求最关键的一条信息（报错原文或请求ID）。涉及高风险操作前先提醒并确认。"
	}
	llmEndpointOrAppID := env.LLMBaseURL
	if strings.EqualFold(env.LLMProvider, "alibaba") {
		llmEndpointOrAppID = strings.TrimSpace(env.LLMAppID)
	}
	llmProvider, err := llm.NewLLMProvider(ctx, env.LLMProvider, env.LLMAPIKey, llmEndpointOrAppID, systemPrompt)
	if err != nil {
		return fmt.Errorf("sip conversation: llm provider init: %w", err)
	}
	if ap, ok := llmProvider.(*llm.AlibabaProvider); ok {
		ap.SetActionHandler(func(action string) {
			if action != "transfer_to_agent" {
				return
			}
			if ms == nil || ms.GetContext().Err() != nil {
				lg.Warn("sip voice: transfer action ignored, call already ended",
					zap.String("call_id", cs.CallID))
				return
			}
			wantDigit := strings.TrimSpace(utils.GetEnv("SIP_TRANSFER_TO_AGENT_DIGIT"))
			if wantDigit == "" {
				wantDigit = "0"
			}
			lg.Info("sip voice: alibaba action transfer_to_agent",
				zap.String("call_id", cs.CallID),
				zap.String("digit", wantDigit),
			)
			tryTransferToAgent(context.Background(), cs.CallID, wantDigit, lg)
		})
	}
	lg.Info("sip voice pipeline config",
		zap.String("llm_model", llmModel),
		zap.String("llm_provider", env.LLMProvider),
		zap.String("llm_endpoint_or_app_id", llmEndpointOrAppID),
		zap.String("asr_model", asrOpt.ModelType),
		zap.Int64("tts_speed", env.TTSSpeed),
		zap.Int("tts_sample_rate", env.TTSSampleRate),
	)
	if strings.Contains(strings.ToLower(asrOpt.ModelType), "8k") {
		lg.Warn("sip voice: ASR is 8k; media PCM is 16k so audio is resampled. For better quality set ASR_MODEL_TYPE=16k_zh",
			zap.String("asr_model", asrOpt.ModelType),
		)
	}

	voiceType := env.TTSVoiceType
	if voiceType == 0 {
		voiceType = 101007 // 知性女声（智娜）
	}
	ttsCfg := synthesizer.NewQcloudTTSConfig(env.TTSAppID, env.TTSSecretID, env.TTSSecretKey, voiceType, "pcm", env.TTSSampleRate)
	ttsCfg.Speed = env.TTSSpeed
	qcTTS := synthesizer.NewQCloudService(ttsCfg)
	ttsStream := &qcloudTTSStream{svc: qcTTS}

	var turnMu sync.Mutex
	inFlight := false
	asrState := NewASRStateManager()
	partialTimeoutMs := sipASRPartialTimeoutMs()
	var partialMu sync.Mutex
	var partialTimer *time.Timer
	var pendingPartial string

	ttsPipe, err := siptts.New(siptts.Config{
		Service:       ttsStream,
		SampleRate:    env.TTSSampleRate,
		Channels:      1,
		FrameDuration: 20 * time.Millisecond,
		// Match RTP real-time pacing so the far end does not receive whole replies in a few ms bursts.
		PaceRealtime: true,
		SendPCMFrame: func(frame []byte) error {
			if len(frame) == 0 {
				return nil
			}
			pkt := &media.AudioPacket{
				Payload:       frame,
				IsSynthesized: true,
			}
			ms.SendToOutput("sip-voice-tts", pkt)
			return nil
		},
		Logger: lg,
	})
	if err != nil {
		return fmt.Errorf("sip conversation: tts pipeline: %w", err)
	}

	var ttsPlaying atomic.Bool
	var ttsStartedAtNS atomic.Int64
	var welcomePlaying atomic.Bool
	var vadDet *voice.VADDetector
	if sipVADBargeInEnabled() {
		vadDet = voice.NewVADDetector()
		vadDet.SetLogger(lg)
		thrEnv := sipVADThresholdFromEnv()
		thr := sipVADDefaultThreshold
		if thrEnv > 0 {
			thr = thrEnv
		}
		vadDet.SetThreshold(thr)
		cf := sipVADConsecutiveFramesFromEnv()
		if cf < 1 {
			// ~20ms/frame; phone echo needs sustained energy above threshold (not single spikes).
			cf = 8
		}
		vadDet.SetConsecutiveFrames(cf)
		lg.Info("sip voice: RMS VAD barge-in enabled (TTS playback only)",
			zap.Float64("threshold_effective", thr),
			zap.Float64("threshold_env_override", thrEnv),
			zap.Int("consecutive_frames", cf),
		)
	} else {
		lg.Info("sip voice: RMS VAD barge-in disabled (SIP_VAD_BARGE_IN)")
	}

	asrInRate := 16000
	asrOutRate := 16000
	if strings.Contains(strings.ToLower(asrOpt.ModelType), "8k") {
		asrOutRate = 8000
	}

	// Same incremental strategy as pkg/hardware: ASRStateManager extracts new sentences from
	// cumulative QCloud text without restarting the recognizer.
	triggerTurn := func(userText string, asrIsFinal bool, trigger string) {
		userText = strings.TrimSpace(userText)
		if userText == "" {
			return
		}
		go func(userText string, asrIsFinal bool, trigger string) {
			if ms == nil || ms.GetContext().Err() != nil {
				return
			}
			turnMu.Lock()
			if inFlight {
				turnMu.Unlock()
				return
			}
			inFlight = true
			turnMu.Unlock()

			defer func() {
				turnMu.Lock()
				inFlight = false
				turnMu.Unlock()
			}()

			lg.Info("sip voice asr trigger",
				zap.String("call_id", cs.CallID),
				zap.String("user_text", userText),
				zap.Bool("asr_isFinal", asrIsFinal),
				zap.String("trigger", trigger),
			)

			ttsPipe.Start(ms.GetContext())
			defer func() {
				ttsPlaying.Store(false)
				ttsStartedAtNS.Store(0)
				ttsPipe.Stop()
			}()
			ttsPlaying.Store(true)
			ttsStartedAtNS.Store(time.Now().UnixNano())
			reply, err := streamLLMToTTS(ms.GetContext(), llmProvider, llmModel, userText, ttsPipe, lg)
			if err != nil {
				lg.Warn("sip voice llm/tts", zap.Error(err))
				return
			}
			lg.Info("sip voice llm reply", zap.Int("reply_chars", len(reply)))
			asrProv := "qcloud_asr"
			if env.ASRModelType != "" {
				asrProv = env.ASRModelType
			}
			go persistSIPTurn(context.Background(), cs.CallID, userText, reply, asrProv, llmModel, "qcloud_tts")
		}(userText, asrIsFinal, trigger)
	}

	pipe.SetTextCallback(func(text string, isFinal bool) {
		if ms == nil || ms.GetContext().Err() != nil {
			return
		}
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			return
		}
		incremental := asrState.UpdateASRText(trimmed, isFinal)
		if incremental == "" {
			return
		}
		if isFillerOnlyUtterance(incremental) {
			lg.Debug("sip voice asr skip filler-only", zap.String("text", incremental))
			return
		}
		if isFinal {
			partialMu.Lock()
			pendingPartial = ""
			if partialTimer != nil {
				partialTimer.Stop()
			}
			partialMu.Unlock()
			triggerTurn(incremental, true, "final")
			return
		}
		if sipASRTriggerPartialEnabled() {
			triggerTurn(incremental, false, "partial")
			return
		}
		if partialTimeoutMs <= 0 {
			return
		}
		partialMu.Lock()
		pendingPartial = incremental
		if partialTimer == nil {
			partialTimer = time.AfterFunc(time.Duration(partialTimeoutMs)*time.Millisecond, func() {
				partialMu.Lock()
				text := strings.TrimSpace(pendingPartial)
				pendingPartial = ""
				partialMu.Unlock()
				if text == "" {
					return
				}
				triggerTurn(text, false, "partial-timeout")
			})
		} else {
			partialTimer.Reset(time.Duration(partialTimeoutMs) * time.Millisecond)
		}
		partialMu.Unlock()
	})

	pipe.SetErrorCallback(func(err error, fatal bool) {
		lg.Warn("sip voice asr", zap.Error(err), zap.Bool("fatal", fatal))
	})

	go func() {
		<-ms.GetContext().Done()
		partialMu.Lock()
		pendingPartial = ""
		if partialTimer != nil {
			partialTimer.Stop()
		}
		partialMu.Unlock()
	}()

	proc := media.NewPacketProcessor("sip-voice-asr-feed", media.PriorityHigh,
		func(c context.Context, _ *media.MediaSession, packet media.MediaPacket) error {
			ap, ok := packet.(*media.AudioPacket)
			if !ok || ap == nil || len(ap.Payload) == 0 {
				return nil
			}
			if ap.IsSynthesized {
				return nil
			}
			// Inbound greeting is a mandatory opening. Ignore user audio until greeting finishes.
			if welcomePlaying.Load() {
				return nil
			}
			pcm16 := ap.Payload
			pcmASR := pcm16
			if asrOutRate != asrInRate {
				out, err := media.ResamplePCM(pcm16, asrInRate, asrOutRate)
				if err != nil {
					lg.Debug("sip voice resample", zap.Error(err))
					return nil
				}
				pcmASR = out
			}
			// RMS VAD on 16 kHz PCM (same as media decode path); only while TTS is playing.
			if vadDet != nil && ttsPlaying.Load() {
				// Ignore very early frames right after TTS starts; they are often acoustic echo artifacts.
				if started := ttsStartedAtNS.Load(); started > 0 && time.Since(time.Unix(0, started)) < 700*time.Millisecond {
					return nil
				}
			}
			if vadDet != nil && ttsPlaying.Load() && vadDet.CheckBargeIn(pcm16, true) {
				lg.Info("sip voice: RMS barge-in, stopping TTS", zap.String("call_id", cs.CallID))
				ttsPipe.Stop()
				ttsPlaying.Store(false)
				ttsStartedAtNS.Store(0)
			}
			err := pipe.ProcessPCM(c, pcmASR)
			if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				return nil
			}
			return err
		})

	ms.RegisterProcessor(proc)

	sipdtmf.AttachProcessor(ms, "sip-dtmf", func(_ context.Context, digit string) {
		lg.Info("sip dtmf", zap.String("digit", digit), zap.String("call_id", cs.CallID))
		tryTransferToAgent(context.Background(), cs.CallID, digit, lg)
	})

	// Start RTP read/write before welcome so the first inbound packet can update symmetric RTP
	// (see pkg/sip/rtp/session.go) before we send audio to the SDP-private address.
	cs.StartOnACK()
	if w := welcomeWaitFirstRTPMs(); w > 0 {
		waitFirstRTPBeforeWelcome(ms.GetContext(), cs, lg, w)
	}

	welcomePlaying.Store(true)
	go func() {
		defer welcomePlaying.Store(false)
		if err := playWelcomeWav(ms.GetContext(), ms, lg, env.TTSSampleRate); err != nil {
			lg.Warn("sip voice welcome playback failed", zap.String("call_id", cs.CallID), zap.Error(err))
			return
		}
		lg.Info("sip voice welcome playback finished", zap.String("call_id", cs.CallID))
	}()

	lg.Info("sip voice pipeline attached", zap.String("call_id", cs.CallID))
	return nil
}

func playWelcomeWav(ctx context.Context, ms *media.MediaSession, lg *zap.Logger, sampleRate int) error {
	if ms == nil {
		return fmt.Errorf("media session is nil")
	}
	path := strings.TrimSpace(utils.GetEnv("SIP_WELCOME_WAV_PATH"))
	if path == "" {
		path = "scripts/welcome.wav"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Clean(path)
	}
	pcm, err := loadWAVAsPCM16Mono(path, sampleRate)
	if err != nil {
		return fmt.Errorf("load welcome wav: %w", err)
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	bytesPerFrame := sampleRate * 2 * 20 / 1000 // 16-bit mono, 20ms
	if bytesPerFrame <= 0 {
		bytesPerFrame = 640
	}
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	if lg != nil {
		lg.Info("sip voice welcome playback started", zap.Int("bytes", len(pcm)))
	}

	for off := 0; off < len(pcm); off += bytesPerFrame {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		end := off + bytesPerFrame
		if end > len(pcm) {
			end = len(pcm)
		}
		frame := pcm[off:end]
		if len(frame) == 0 {
			continue
		}
		ms.SendToOutput("sip-voice-welcome", &media.AudioPacket{
			Payload:       frame,
			IsSynthesized: true,
		})
	}
	return nil
}

func streamLLMToTTS(ctx context.Context, llmProvider llm.LLMProvider, model, userText string, ttsPipe *siptts.Pipeline, lg *zap.Logger) (string, error) {
	if llmProvider == nil {
		return "", fmt.Errorf("nil llm provider")
	}
	if ttsPipe == nil {
		return "", fmt.Errorf("nil tts pipe")
	}
	var full strings.Builder
	var seg strings.Builder
	flush := func(force bool) error {
		s := strings.TrimSpace(seg.String())
		if s == "" {
			return nil
		}
		if !force {
			runes := []rune(s)
			last := runes[len(runes)-1]
			if !strings.ContainsRune("。！？.!?,，；;:", last) && len(runes) < 18 {
				return nil
			}
		}
		seg.Reset()
		s = normalizeTTSText(s)
		if s == "" {
			return nil
		}
		return ttsPipe.Speak(s)
	}
	// Alibaba App API usually returns one JSON message per turn; non-streaming avoids long
	// silent waits when SSE chunks are sparse on some networks.
	if _, isAlibaba := llmProvider.(*llm.AlibabaProvider); isAlibaba {
		reply, err := llmProvider.Query(userText, model)
		if err != nil {
			return "", err
		}
		reply = normalizeTTSText(reply)
		if reply == "" {
			return "", nil
		}
		if err := ttsPipe.Speak(reply); err != nil {
			if errors.Is(err, context.Canceled) {
				return reply, nil
			}
			return "", err
		}
		return strings.TrimSpace(reply), nil
	}
	options := llm.QueryOptions{Model: model, Stream: true}
	reply, err := llmProvider.QueryStream(userText, options, func(piece string, _ bool) error {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			return nil
		}
		full.WriteString(piece)
		seg.WriteString(piece)
		return flush(false)
	})
	if err != nil {
		// fallback to non-streaming so behavior stays stable even if provider stream fails.
		reply, err = llmProvider.Query(userText, model)
		if err != nil {
			return "", err
		}
		reply = normalizeTTSText(reply)
		if reply == "" {
			if lg != nil {
				lg.Warn("sip voice tts skip empty/invalid reply after sanitize")
			}
			return "", nil
		}
		if err := ttsPipe.Speak(reply); err != nil {
			if errors.Is(err, context.Canceled) {
				if lg != nil {
					lg.Info("sip voice tts stopped (barge-in or cancel)")
				}
				return reply, nil
			}
			return "", err
		}
		return strings.TrimSpace(reply), nil
	}
	if strings.TrimSpace(reply) == "" {
		reply = full.String()
	}
	if err := flush(true); err != nil {
		if errors.Is(err, context.Canceled) {
			return strings.TrimSpace(reply), nil
		}
		return "", err
	}
	return strings.TrimSpace(reply), nil
}

// SpeakTextOnce sends one synthesized sentence to the current SIP media output.
// It is used by outbound script runtime for deterministic "say" steps.
func SpeakTextOnce(ctx context.Context, cs *sipSession.CallSession, text string, lg *zap.Logger) error {
	if cs == nil || strings.TrimSpace(cs.CallID) == "" {
		return fmt.Errorf("sip conversation: nil call session")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	ms := cs.MediaSession()
	if ms == nil {
		return fmt.Errorf("sip conversation: media session not ready")
	}
	if lg == nil {
		if logger.Lg != nil {
			lg = logger.Lg
		} else {
			lg = zap.NewNop()
		}
	}
	env := voiceEnvFromProcess()
	if env.TTSAppID == "" || env.TTSSecretID == "" || env.TTSSecretKey == "" {
		return fmt.Errorf("sip conversation: missing TTS credentials")
	}
	voiceType := env.TTSVoiceType
	if voiceType == 0 {
		voiceType = 101007 // 知性女声（智娜）
	}
	ttsCfg := synthesizer.NewQcloudTTSConfig(env.TTSAppID, env.TTSSecretID, env.TTSSecretKey, voiceType, "pcm", env.TTSSampleRate)
	ttsCfg.Speed = env.TTSSpeed
	qcTTS := synthesizer.NewQCloudService(ttsCfg)
	ttsStream := &qcloudTTSStream{svc: qcTTS}

	ttsPipe, err := siptts.New(siptts.Config{
		Service:       ttsStream,
		SampleRate:    env.TTSSampleRate,
		Channels:      1,
		FrameDuration: 20 * time.Millisecond,
		PaceRealtime:  true,
		SendPCMFrame: func(frame []byte) error {
			if len(frame) == 0 {
				return nil
			}
			ms.SendToOutput("sip-script-say", &media.AudioPacket{
				Payload:       frame,
				IsSynthesized: true,
			})
			return nil
		},
		Logger: lg,
	})
	if err != nil {
		return fmt.Errorf("sip conversation: tts pipeline: %w", err)
	}
	runCtx := ctx
	if runCtx == nil {
		runCtx = ms.GetContext()
	}
	ttsPipe.Start(runCtx)
	defer ttsPipe.Stop()
	return ttsPipe.Speak(text)
}

// qcloudTTSStream adapts synthesizer.QCloudService to siptts.Service (streaming PCM chunks).
type qcloudTTSStream struct {
	svc *synthesizer.QCloudService
}

func (q *qcloudTTSStream) SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error {
	if q == nil || q.svc == nil {
		return fmt.Errorf("sip conversation: nil tts")
	}
	// QCloud Synthesize blocks until Wait(); tie it to ctx so barge-in Stop() returns quickly.
	// Background avoids canceling the SDK mid-handshake; OnMessage uses ctx to drop audio after cancel.
	done := make(chan error, 1)
	go func() {
		h := &ttsStreamHandler{callback: callback, ctx: ctx}
		done <- q.svc.Synthesize(context.Background(), h, text)
	}()
	select {
	case <-ctx.Done():
		return context.Canceled
	case err := <-done:
		return err
	}
}

type ttsStreamHandler struct {
	ctx        context.Context
	callback   func([]byte) error
	firstChunk bool
}

func (h *ttsStreamHandler) OnMessage(data []byte) {
	if h == nil || len(data) == 0 {
		return
	}
	if h.ctx != nil && h.ctx.Err() != nil {
		return
	}
	if !h.firstChunk {
		h.firstChunk = true
		data = encoder.StripWavHeader(data)
	}
	_ = h.callback(data)
}

func (h *ttsStreamHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}

// isFillerOnlyUtterance filters hesitation sounds that real-time ASR often emits on noisy or low-volume audio.
func isFillerOnlyUtterance(text string) bool {
	t := strings.TrimSpace(text)
	t = strings.Trim(t, "。！？.!?…~～、，,；;：:《》<>（）()[]【】“”\"'`")
	t = strings.TrimSpace(t)
	if t == "" {
		return true
	}
	for _, r := range t {
		switch r {
		case '嗯', '唔', '呃', '啊', '哦', '噢', '诶', '欸', '哈', '哼', '额',
			' ', '\t', '\n', '\r', '，', ',', '。', '.', '！', '!', '？', '?', '、', '…', '~', '～', '-', '—':
			continue
		default:
			return false
		}
	}
	return true
}

// normalizeTTSText removes segments that TTS providers commonly reject:
// empty/punctuation-only strings and strings without CJK/letters/digits after cleanup.
func normalizeTTSText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		// strip control chars
		if r < 0x20 {
			continue
		}
		b.WriteRune(r)
	}
	s = strings.TrimSpace(b.String())
	if s == "" {
		return ""
	}
	onlyPunct := true
	for _, r := range s {
		if (r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 0x4e00 && r <= 0x9fff) {
			onlyPunct = false
			break
		}
		switch r {
		case ' ', '\t', '\n', '\r', '，', ',', '。', '.', '！', '!', '？', '?',
			'；', ';', '：', ':', '、', '…', '-', '—', '~', '～', '“', '”', '"', '\'', '`', '(', ')', '（', '）', '[', ']', '【', '】', '<', '>', '《', '》':
			// punctuation/noise
		default:
			onlyPunct = false
			break
		}
	}
	if onlyPunct {
		return ""
	}
	return s
}

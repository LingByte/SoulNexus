// Package conversation wires SIP CallSession media to ASR → LLM → TTS using env-driven credentials.
package conversation

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
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
	"go.uber.org/zap"
)

// VoiceEnv holds SIP voice pipeline settings read with utils.GetEnv (see SoulNexus .env).
type VoiceEnv struct {
	LLMProvider string
	LLMBaseURL  string
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
	TTSSampleRate  int
}

func voiceEnvFromProcess() VoiceEnv {
	voiceType, _ := strconv.ParseInt(strings.TrimSpace(utils.GetEnv("TTS_VOICE_TYPE")), 10, 64)
	sr, _ := strconv.Atoi(strings.TrimSpace(utils.GetEnv("TTS_SAMPLE_RATE")))
	if sr <= 0 {
		sr = 16000
	}
	return VoiceEnv{
		LLMProvider: strings.TrimSpace(utils.GetEnv("LLM_PROVIDER")),
		LLMBaseURL:  strings.TrimSpace(utils.GetEnv("LLM_BASEURL")),
		LLMAPIKey:   strings.TrimSpace(utils.GetEnv("LLM_APIKEY")),
		LLMModel:    strings.TrimSpace(utils.GetEnv("LLM_MODEL")),

		ASRAppID:     strings.TrimSpace(utils.GetEnv("ASR_APPID")),
		ASRSecretID:  strings.TrimSpace(utils.GetEnv("ASR_SECRET_ID")),
		ASRSecretKey: strings.TrimSpace(utils.GetEnv("ASR_SECRET_KEY")),
		ASRModelType: strings.TrimSpace(utils.GetEnv("ASR_MODEL_TYPE")),

		TTSAppID:      strings.TrimSpace(utils.GetEnv("TTS_APPID")),
		TTSSecretID:   strings.TrimSpace(utils.GetEnv("TTS_SECRET_ID")),
		TTSSecretKey:  strings.TrimSpace(utils.GetEnv("TTS_SECRET_KEY")),
		TTSVoiceType:  voiceType,
		TTSSampleRate: sr,
	}
}

func (e VoiceEnv) readyForVoice() bool {
	return e.ASRAppID != "" && e.ASRSecretID != "" && e.ASRSecretKey != "" &&
		e.LLMAPIKey != "" && e.LLMBaseURL != "" &&
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
	llmHandler := llm.NewLLMHandler(ctx, env.LLMAPIKey, env.LLMBaseURL,
		"You are a helpful Mandarin voice assistant on a phone call. Reply in clear spoken-style sentences, stay brief (one or two sentences when possible).")
	llmHandler.SetModel(llmModel)
	lg.Info("sip voice pipeline config",
		zap.String("llm_model", llmModel),
		zap.String("llm_provider", env.LLMProvider),
		zap.String("asr_model", asrOpt.ModelType),
		zap.Int("tts_sample_rate", env.TTSSampleRate),
	)
	if strings.Contains(strings.ToLower(asrOpt.ModelType), "8k") {
		lg.Warn("sip voice: ASR is 8k; media PCM is 16k so audio is resampled. For better quality set ASR_MODEL_TYPE=16k_zh",
			zap.String("asr_model", asrOpt.ModelType),
		)
	}

	voiceType := env.TTSVoiceType
	if voiceType == 0 {
		voiceType = 1005
	}
	ttsCfg := synthesizer.NewQcloudTTSConfig(env.TTSAppID, env.TTSSecretID, env.TTSSecretKey, voiceType, "pcm", env.TTSSampleRate)
	qcTTS := synthesizer.NewQCloudService(ttsCfg)
	ttsStream := &qcloudTTSStream{svc: qcTTS}

	var turnMu sync.Mutex
	inFlight := false
	asrState := NewASRStateManager()

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

	asrInRate := 16000
	asrOutRate := 16000
	if strings.Contains(strings.ToLower(asrOpt.ModelType), "8k") {
		asrOutRate = 8000
	}

	// Same incremental strategy as pkg/hardware: ASRStateManager extracts new sentences from
	// cumulative QCloud text without restarting the recognizer.
	pipe.SetTextCallback(func(text string, isFinal bool) {
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

		go func(userText string, asrIsFinal bool) {
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
			)

			reply, err := llmHandler.Query(userText, llmModel)
			if err != nil {
				lg.Warn("sip voice llm", zap.Error(err))
				return
			}
			lg.Info("sip voice llm reply", zap.Int("reply_chars", len(reply)))
			ttsPipe.Start(ms.GetContext())
			defer ttsPipe.Stop()
			if err := ttsPipe.Speak(reply); err != nil {
				lg.Warn("sip voice tts", zap.Error(err))
			}
		}(incremental, isFinal)
	})

	pipe.SetErrorCallback(func(err error, fatal bool) {
		lg.Warn("sip voice asr", zap.Error(err), zap.Bool("fatal", fatal))
	})

	proc := media.NewPacketProcessor("sip-voice-asr-feed", media.PriorityHigh,
		func(c context.Context, _ *media.MediaSession, packet media.MediaPacket) error {
			ap, ok := packet.(*media.AudioPacket)
			if !ok || ap == nil || len(ap.Payload) == 0 {
				return nil
			}
			if ap.IsSynthesized {
				return nil
			}
			pcm := ap.Payload
			if asrOutRate != asrInRate {
				out, err := media.ResamplePCM(pcm, asrInRate, asrOutRate)
				if err != nil {
					lg.Debug("sip voice resample", zap.Error(err))
					return nil
				}
				pcm = out
			}
			return pipe.ProcessPCM(c, pcm)
		})

	ms.RegisterProcessor(proc)

	sipdtmf.AttachProcessor(ms, "sip-dtmf", func(_ context.Context, digit string) {
		lg.Info("sip dtmf", zap.String("digit", digit), zap.String("call_id", cs.CallID))
		tryTransferToAgent(context.Background(), cs.CallID, digit, lg)
	})

	lg.Info("sip voice pipeline attached", zap.String("call_id", cs.CallID))
	return nil
}

// qcloudTTSStream adapts synthesizer.QCloudService to siptts.Service (streaming PCM chunks).
type qcloudTTSStream struct {
	svc *synthesizer.QCloudService
}

func (q *qcloudTTSStream) SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error {
	if q == nil || q.svc == nil {
		return fmt.Errorf("sip conversation: nil tts")
	}
	h := &ttsStreamHandler{callback: callback}
	return q.svc.Synthesize(ctx, h, text)
}

type ttsStreamHandler struct {
	callback   func([]byte) error
	firstChunk bool
}

func (h *ttsStreamHandler) OnMessage(data []byte) {
	if h == nil || len(data) == 0 {
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
	t = strings.Trim(t, "。！？.!?…")
	t = strings.TrimSpace(t)
	if t == "" {
		return true
	}
	for _, r := range t {
		switch r {
		case '嗯', '唔', '呃', '啊', '哦', '噢', '诶', '欸', '哈', '哼', '额',
			' ', '\t', '\n', '\r', '，', ',':
			continue
		default:
			return false
		}
	}
	return true
}

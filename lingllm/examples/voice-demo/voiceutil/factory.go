// Package voiceutil wires ASR/TTS/realtime factories for voice-demo examples.
package voiceutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/transport"
	voicetts "github.com/LingByte/lingllm/protocol/voice/tts"
	"github.com/LingByte/lingllm/protocol/voice/xiaozhi"
	"github.com/LingByte/lingllm/realtime"
	"github.com/LingByte/lingllm/recognizer"
	"github.com/LingByte/lingllm/synthesizer"
)

// Factory implements transport.SessionFactory and builds realtime agents.
type Factory struct {
	asrProvider   string
	asrConfig     map[string]interface{}
	ttsProvider   string
	ttsConfig     map[string]interface{}
	realtimeCfg   map[string]any
	systemPrompt  string
	realtimeVoice string
}

// NewFactory loads credentials from environment variables.
func NewFactory() (*Factory, error) {
	f := &Factory{
		asrProvider:   strings.TrimSpace(os.Getenv("ASR_PROVIDER")),
		ttsProvider:   strings.TrimSpace(os.Getenv("TTS_PROVIDER")),
		systemPrompt:  strings.TrimSpace(os.Getenv("REALTIME_SYSTEM_PROMPT")),
		realtimeVoice: strings.TrimSpace(os.Getenv("REALTIME_VOICE")),
	}
	if f.asrProvider == "" {
		f.asrProvider = "volcengine"
	}
	if f.ttsProvider == "" {
		f.ttsProvider = "openai"
	}
	if f.systemPrompt == "" {
		f.systemPrompt = "You are a helpful voice assistant. Reply concisely in the same language the user speaks."
	}
	if f.realtimeVoice == "" {
		f.realtimeVoice = "Cherry"
	}

	var err error
	f.asrConfig, err = loadJSONEnv("ASR_CONFIG_JSON")
	if err != nil {
		return nil, fmt.Errorf("ASR_CONFIG_JSON: %w", err)
	}
	f.ttsConfig, err = loadJSONEnv("TTS_CONFIG_JSON")
	if err != nil {
		return nil, fmt.Errorf("TTS_CONFIG_JSON: %w", err)
	}
	f.realtimeCfg, err = loadJSONEnv("REALTIME_CONFIG_JSON")
	if err != nil {
		return nil, fmt.Errorf("REALTIME_CONFIG_JSON: %w", err)
	}
	mergeEnvCredentials(f)
	return f, nil
}

// RealtimeAgentFactory returns a xiaozhi.RealtimeAgentFactory backed by f.
func (f *Factory) RealtimeAgentFactory() xiaozhi.RealtimeAgentFactory {
	return realtimeFactory{f: f}
}

type realtimeFactory struct{ f *Factory }

func (x realtimeFactory) NewAgent(ctx context.Context, callID string, onEvent func(realtime.Event)) (realtime.Agent, int, int, error) {
	return x.f.newRealtimeAgent(ctx, callID, onEvent)
}

func loadJSONEnv(key string) (map[string]interface{}, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return map[string]interface{}{}, nil
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func mergeEnvCredentials(f *Factory) {
	if f.asrConfig == nil {
		f.asrConfig = map[string]interface{}{}
	}
	if f.ttsConfig == nil {
		f.ttsConfig = map[string]interface{}{}
	}
	if f.realtimeCfg == nil {
		f.realtimeCfg = map[string]any{}
	}
	setIfEmpty(f.asrConfig, "provider", f.asrProvider)
	setIfEmpty(f.ttsConfig, "provider", f.ttsProvider)

	if v := os.Getenv("ASR_APP_ID"); v != "" {
		f.asrConfig["app_id"] = v
	}
	if v := os.Getenv("ASR_TOKEN"); v != "" {
		f.asrConfig["token"] = v
	}
	if v := os.Getenv("ASR_ACCESS_KEY"); v != "" {
		f.asrConfig["access_key"] = v
	}
	if v := os.Getenv("TTS_API_KEY"); v != "" {
		f.ttsConfig["apiKey"] = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		setIfEmpty(f.ttsConfig, "apiKey", v)
		setIfEmpty(f.realtimeCfg, "api_key", v)
	}
}

func setIfEmpty(m map[string]interface{}, key, val string) {
	if val == "" {
		return
	}
	if existing, ok := m[key].(string); !ok || strings.TrimSpace(existing) == "" {
		m[key] = val
	}
}

func (f *Factory) NewASR(_ context.Context, callID string) (asr.Engine, int, error) {
	cfg, err := recognizer.NewTranscriberConfigFromMap(f.asrProvider, f.asrConfig, "zh-CN")
	if err != nil {
		return nil, 0, err
	}
	factory := recognizer.NewTranscriberFactory()
	eng, err := factory.CreateTranscriber(cfg)
	if err != nil {
		return nil, 0, err
	}
	sr := envInt("ASR_SAMPLE_RATE", 16000)
	return &speechEngine{eng: eng, callID: callID}, sr, nil
}

func (f *Factory) NewTTS(_ context.Context, _ string) (voicetts.TTSService, int, error) {
	engine, err := synthesizer.NewAudioSynthesisEngineFromCredential(f.ttsConfig)
	if err != nil {
		return nil, 0, err
	}
	sr := envInt("TTS_SAMPLE_RATE", 16000)
	return voicetts.FromSynthesisEngine(engine), sr, nil
}

func (f *Factory) newRealtimeAgent(ctx context.Context, callID string, onEvent func(realtime.Event)) (realtime.Agent, int, int, error) {
	if len(f.realtimeCfg) == 0 {
		return nil, 0, 0, fmt.Errorf("REALTIME_CONFIG_JSON is empty")
	}
	cfg := make(map[string]any, len(f.realtimeCfg))
	for k, v := range f.realtimeCfg {
		cfg[k] = v
	}
	agent, err := realtime.NewAgentFromCredential(cfg, realtime.Options{
		SystemPrompt:     f.systemPrompt,
		Voice:            f.realtimeVoice,
		InputSampleRate:  envInt("REALTIME_INPUT_SR", 16000),
		OutputSampleRate: envInt("REALTIME_OUTPUT_SR", 24000),
		OnEvent:          onEvent,
	})
	if err != nil {
		return nil, 0, 0, err
	}
	if err := agent.Start(ctx); err != nil {
		return nil, 0, 0, err
	}
	inSR := envInt("REALTIME_INPUT_SR", 16000)
	outSR := envInt("REALTIME_OUTPUT_SR", 24000)
	return agent, inSR, outSR, nil
}

var _ transport.SessionFactory = (*Factory)(nil)

type speechEngine struct {
	eng    recognizer.SpeechRecognitionEngine
	callID string
	mu     sync.Mutex

	onResult func(text string, isFinal bool)
	onError  func(err error, fatal bool)
}

func (e *speechEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eng.Init(func(text string, isLast bool, _ time.Duration, _ string) {
		e.mu.Lock()
		fn := e.onResult
		e.mu.Unlock()
		if fn != nil && strings.TrimSpace(text) != "" {
			fn(text, isLast)
		}
	}, func(err error, fatal bool) {
		e.mu.Lock()
		fn := e.onError
		e.mu.Unlock()
		if fn != nil && err != nil {
			fn(err, fatal)
		}
	})
	return e.eng.ConnAndReceive(e.callID)
}

func (e *speechEngine) Stop() {
	_ = e.eng.StopConn()
}

func (e *speechEngine) SendPCM(pcm []byte, end bool) error {
	if end {
		if err := e.eng.SendEnd(); err != nil {
			return err
		}
	}
	if len(pcm) == 0 {
		return nil
	}
	return e.eng.SendAudioBytes(pcm)
}

func (e *speechEngine) OnResult(callback func(text string, isFinal bool)) {
	e.mu.Lock()
	e.onResult = callback
	e.mu.Unlock()
}

func (e *speechEngine) OnError(callback func(err error, fatal bool)) {
	e.mu.Lock()
	e.onError = callback
	e.mu.Unlock()
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

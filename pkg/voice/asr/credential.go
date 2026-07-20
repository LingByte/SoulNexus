package asr

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/providers"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/recognizer"
	"go.uber.org/zap"
)

// BuildOptions configures tenant ASR pipeline construction.
type BuildOptions struct {
	Env        tenantcfg.VoiceEnv
	SampleRate int
	Logger     *zap.Logger

	InvocationTenantID uint
	InvocationCallID   string
	InvocationProvider string
	InvocationSource   string
}

// BuildEngine resolves tenant/assistant ASR credentials through lingllm.
func BuildEngine(opt BuildOptions) (recognizer.SpeechRecognitionEngine, int, error) {
	cfgMap, provider, err := asrCredentialMap(opt.Env)
	if err != nil {
		return nil, 0, err
	}
	if provider == "" {
		provider = "qcloud"
	}
	tcfg, err := recognizer.NewTranscriberConfigFromMap(provider, cfgMap, "zh-CN")
	if err != nil {
		return nil, 0, fmt.Errorf("voice/asr: config: %w", err)
	}
	factory := recognizer.NewTranscriberFactory()
	eng, err := factory.CreateTranscriber(tcfg)
	if err != nil {
		return nil, 0, fmt.Errorf("voice/asr: create %s: %w", provider, err)
	}
	outRate := opt.SampleRate
	if outRate <= 0 {
		outRate = inferASROutRate(cfgMap, provider)
	}
	if hw := providers.ParseRecognizerHotWords(opt.Env.HotWordsRaw); len(hw) > 0 {
		applyHotWords(eng, hw)
	}
	return eng, outRate, nil
}

// BuildPipeline builds a voice/asr.Pipeline from tenant credentials.
func BuildPipeline(opt BuildOptions) (*Pipeline, int, error) {
	eng, outRate, err := BuildEngine(opt)
	if err != nil {
		return nil, 0, err
	}
	_, provider, _ := asrCredentialMap(opt.Env)
	pipe, err := New(Options{
		ASR:                eng,
		SampleRate:         outRate,
		Channels:           1,
		Logger:             opt.Logger,
		InvocationTenantID: opt.InvocationTenantID,
		InvocationCallID:   opt.InvocationCallID,
		InvocationProvider: firstNonEmpty(opt.InvocationProvider, provider),
		InvocationSource:   opt.InvocationSource,
	})
	if err != nil {
		return nil, 0, err
	}
	return pipe, outRate, nil
}

func asrCredentialMap(env tenantcfg.VoiceEnv) (map[string]any, string, error) {
	raw := env.ASRConfigRaw
	if len(raw) == 0 {
		raw = legacyASRMap(env)
	}
	if len(raw) == 0 {
		return nil, "", fmt.Errorf("voice/asr: missing ASR credentials")
	}
	provider := strings.ToLower(strings.TrimSpace(fmt.Sprint(raw["provider"])))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(env.ASRProvider))
	}
	if provider == "" {
		provider = "qcloud"
	}
	if _, ok := raw["provider"]; !ok {
		raw["provider"] = provider
	}
	mergeLegacyQCloudFields(raw, env)
	return raw, provider, nil
}

func legacyASRMap(env tenantcfg.VoiceEnv) map[string]any {
	if strings.TrimSpace(env.ASRAppID) == "" {
		return nil
	}
	return map[string]any{
		"provider":   firstNonEmpty(env.ASRProvider, "qcloud"),
		"appId":      env.ASRAppID,
		"secretId":   env.ASRSecretID,
		"secretKey":  env.ASRSecretKey,
		"modelType":  env.ASRModelType,
		"model_type": env.ASRModelType,
	}
}

func mergeLegacyQCloudFields(m map[string]any, env tenantcfg.VoiceEnv) {
	setIfEmpty(m, "appId", env.ASRAppID)
	setIfEmpty(m, "app_id", env.ASRAppID)
	setIfEmpty(m, "secretId", env.ASRSecretID)
	setIfEmpty(m, "secret_id", env.ASRSecretID)
	setIfEmpty(m, "secretKey", env.ASRSecretKey)
	setIfEmpty(m, "secret_key", env.ASRSecretKey)
	setIfEmpty(m, "modelType", env.ASRModelType)
	setIfEmpty(m, "model_type", env.ASRModelType)
}

func setIfEmpty(m map[string]any, key, val string) {
	if val == "" {
		return
	}
	if existing, ok := m[key].(string); !ok || strings.TrimSpace(existing) == "" {
		m[key] = val
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func inferASROutRate(cfg map[string]any, provider string) int {
	mt := strings.ToLower(fmt.Sprint(cfg["modelType"]))
	if mt == "" {
		mt = strings.ToLower(fmt.Sprint(cfg["model_type"]))
	}
	if strings.Contains(mt, "8k") {
		return 8000
	}
	if provider == "qcloud" || provider == "tencent" {
		return 16000
	}
	if sr := utils.IntFromAnyOrZero(cfg["sampleRate"]); sr > 0 {
		return sr
	}
	if sr := utils.IntFromAnyOrZero(cfg["sample_rate"]); sr > 0 {
		return sr
	}
	return 16000
}

func applyHotWords(eng recognizer.SpeechRecognitionEngine, hw []recognizer.HotWord) {
	type hotSetter interface {
		SetHotWords([]recognizer.HotWord)
	}
	if s, ok := eng.(hotSetter); ok {
		s.SetHotWords(hw)
	}
}

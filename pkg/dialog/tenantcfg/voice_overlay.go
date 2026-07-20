package tenantcfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// VoiceFieldForTTSProvider returns the JSON key used for timbre on the given TTS provider.
func VoiceFieldForTTSProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "qcloud", "tencent":
		return "voiceType"
	case "aws":
		return "voiceId"
	case "elevenlabs":
		return "voiceId"
	case "volcengine_clone":
		return "assetId"
	case "xunfei_clone":
		return "assetId"
	case "volcengine", "volcengine_stream", "volcengine_llm":
		return "voiceType"
	case "fishaudio", "fishspeech":
		return "reference_id"
	case "coqui", "local_gospeech":
		return "speaker"
	default:
		return "voice"
	}
}

// VoiceFieldForRealtimeProvider returns the JSON key for realtime timbre.
func VoiceFieldForRealtimeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "volcengine_dialogue", "volc_realtime", "doubao_realtime", "volcengine_realtime":
		return "speaker"
	default:
		return "voice"
	}
}

// ApplyTTSVoice overlays assistant-selected timbre onto tenant TTS credentials JSON.
func ApplyTTSVoice(ttsRaw []byte, voice string) []byte {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return ttsRaw
	}
	m := parseJSONMapBytes(ttsRaw)
	if len(m) == 0 {
		return ttsRaw
	}
	provider := strings.ToLower(strings.TrimSpace(strFromMap(m, "provider")))
	voice = normalizeCatalogVoice(provider, "tts", voice)
	if voice == "" {
		return ttsRaw
	}
	key := VoiceFieldForTTSProvider(provider)
	setVoiceValue(m, key, voice)
	out, err := json.Marshal(m)
	if err != nil {
		return ttsRaw
	}
	return out
}

// ApplyRealtimeVoice overlays assistant-selected timbre onto tenant realtime credentials JSON.
func ApplyRealtimeVoice(rtRaw []byte, voice string) []byte {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return rtRaw
	}
	m := parseJSONMapBytes(rtRaw)
	if len(m) == 0 {
		return rtRaw
	}
	provider := strings.ToLower(strings.TrimSpace(strFromMap(m, "provider")))
	voice = normalizeCatalogVoice(provider, "realtime", voice)
	if voice == "" {
		return rtRaw
	}
	key := VoiceFieldForRealtimeProvider(provider)
	setVoiceValue(m, key, voice)
	out, err := json.Marshal(m)
	if err != nil {
		return rtRaw
	}
	return out
}

// normalizeCatalogVoice maps assistant voice to a provider-valid id when possible.
// Unknown ids fall back to the first catalog entry for that provider/mode.
func normalizeCatalogVoice(provider, mode, voice string) string {
	voice = strings.TrimSpace(voice)
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "volcengine_clone" || p == "xunfei_clone" {
		return voice
	}
	res, err := listOverlayVoiceCatalog(provider, mode)
	if err != nil || len(res.Voices) == 0 {
		return voice
	}
	if voice != "" {
		for _, v := range res.Voices {
			if v.ID == voice {
				return voice
			}
		}
	}
	if len(res.Voices) > 0 {
		return res.Voices[0].ID
	}
	return voice
}

func setVoiceValue(m map[string]any, key, voice string) {
	switch key {
	case "voiceType":
		if n, err := strconv.ParseInt(voice, 10, 64); err == nil {
			m[key] = n
			return
		}
		m[key] = voice
	default:
		m[key] = voice
	}
}

var overlayVoiceCatalogAliases = map[string]string{
	"tencent":             "qcloud",
	"volcengine_stream":   "volcengine",
	"volcengine_llm":      "volcengine",
	"dashscope":           "aliyun",
	"qwen":                "aliyun",
	"qwen_omni":           "aliyun_omni",
	"dashscope_omni":      "aliyun_omni",
	"volc_realtime":       "volcengine_dialogue",
	"doubao_realtime":     "volcengine_dialogue",
	"volcengine_realtime": "volcengine_dialogue",
}

type overlayVoiceCatalogFile struct {
	Modes  []string `json:"modes"`
	Voices []struct {
		ID string `json:"id"`
	} `json:"voices"`
}

var (
	overlayVoiceCatalogDirOnce sync.Once
	overlayVoiceCatalogDirPath string
	overlayVoiceCatalogDirErr  error
	overlayVoiceCatalogCache   sync.Map
)

func resolveOverlayVoiceCatalogProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		return ""
	}
	if alias, ok := overlayVoiceCatalogAliases[p]; ok {
		return alias
	}
	return p
}

func overlayVoiceCatalogDir() (string, error) {
	overlayVoiceCatalogDirOnce.Do(func() {
		if env := strings.TrimSpace(os.Getenv("VOICES_CATALOG_DIR")); env != "" {
			if st, err := os.Stat(env); err == nil && st.IsDir() {
				overlayVoiceCatalogDirPath = filepath.Clean(env)
				return
			}
		}
		cwd, err := os.Getwd()
		if err != nil {
			overlayVoiceCatalogDirErr = err
			return
		}
		dir := filepath.Clean(cwd)
		for {
			cand := filepath.Join(dir, "scripts", "voices")
			if st, err := os.Stat(cand); err == nil && st.IsDir() {
				overlayVoiceCatalogDirPath = cand
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				overlayVoiceCatalogDirErr = fmt.Errorf("voice catalog: scripts/voices not found")
				return
			}
			dir = parent
		}
	})
	return overlayVoiceCatalogDirPath, overlayVoiceCatalogDirErr
}

func listOverlayVoiceCatalog(provider, mode string) (overlayVoiceCatalogFile, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "tts"
	}
	resolved := resolveOverlayVoiceCatalogProvider(provider)
	if resolved == "" {
		return overlayVoiceCatalogFile{}, fmt.Errorf("voice catalog: empty provider")
	}
	if v, ok := overlayVoiceCatalogCache.Load(resolved); ok {
		f := v.(overlayVoiceCatalogFile)
		if overlayVoiceCatalogModeAllowed(&f, mode) {
			return f, nil
		}
		return overlayVoiceCatalogFile{}, nil
	}
	root, err := overlayVoiceCatalogDir()
	if err != nil {
		return overlayVoiceCatalogFile{}, err
	}
	raw, err := os.ReadFile(filepath.Join(root, resolved+".json"))
	if err != nil {
		return overlayVoiceCatalogFile{}, err
	}
	var f overlayVoiceCatalogFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return overlayVoiceCatalogFile{}, err
	}
	overlayVoiceCatalogCache.Store(resolved, f)
	if !overlayVoiceCatalogModeAllowed(&f, mode) {
		return overlayVoiceCatalogFile{}, nil
	}
	return f, nil
}

func overlayVoiceCatalogModeAllowed(f *overlayVoiceCatalogFile, mode string) bool {
	if f == nil || len(f.Modes) == 0 {
		return true
	}
	for _, m := range f.Modes {
		if strings.EqualFold(strings.TrimSpace(m), mode) {
			return true
		}
	}
	return false
}

func parseJSONMapBytes(raw []byte) map[string]any {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil || m == nil {
		return map[string]any{}
	}
	return m
}

// StripTTSVoice removes timbre keys from tenant TTS JSON (credentials only).
func StripTTSVoice(ttsRaw []byte) []byte {
	m := parseJSONMapBytes(ttsRaw)
	if len(m) == 0 {
		return ttsRaw
	}
	provider := strings.ToLower(strings.TrimSpace(strFromMap(m, "provider")))
	key := VoiceFieldForTTSProvider(provider)
	delete(m, key)
	for _, k := range []string{"voice", "voiceType", "voiceId", "voice_id", "assetId", "asset_id", "speaker", "reference_id", "referenceId"} {
		delete(m, k)
	}
	out, err := json.Marshal(m)
	if err != nil {
		return ttsRaw
	}
	return out
}

// StripRealtimeVoice removes timbre keys from tenant realtime JSON.
func StripRealtimeVoice(rtRaw []byte) []byte {
	m := parseJSONMapBytes(rtRaw)
	if len(m) == 0 {
		return rtRaw
	}
	for _, k := range []string{"voice", "voiceId", "voice_id", "speaker"} {
		delete(m, k)
	}
	out, err := json.Marshal(m)
	if err != nil {
		return rtRaw
	}
	return out
}

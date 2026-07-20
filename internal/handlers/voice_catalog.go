package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type voiceCatalogEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Label      string `json:"label"`
	Gender     string `json:"gender,omitempty"`
	Locale     string `json:"locale,omitempty"`
	Category   string `json:"category,omitempty"`
	PreviewURL string `json:"previewUrl,omitempty"`
}

type voiceCatalogFile struct {
	Provider   string              `json:"provider"`
	Modes      []string            `json:"modes"`
	VoiceField string              `json:"voiceField"`
	DocURL     string              `json:"docUrl"`
	UpdatedAt  string              `json:"updatedAt"`
	Voices     []voiceCatalogEntry `json:"voices"`
}

type voiceCatalogListResult struct {
	Provider   string              `json:"provider"`
	Mode       string              `json:"mode"`
	VoiceField string              `json:"voiceField,omitempty"`
	DocURL     string              `json:"docUrl,omitempty"`
	Voices     []voiceCatalogEntry `json:"voices"`
	Cached     bool                `json:"cached,omitempty"`
}

type voiceCatalogCacheEntry struct {
	file    *voiceCatalogFile
	modTime time.Time
	loaded  time.Time
}

var voiceCatalogProviderAliases = map[string]string{
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

var (
	voiceCatalogDirOnce sync.Once
	voiceCatalogDirPath string
	voiceCatalogDirErr  error
	voiceCatalogCache   sync.Map // key: provider -> *voiceCatalogCacheEntry
	voiceCatalogLoadMu  sync.Mutex
)

func resolveVoiceCatalogProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		return ""
	}
	if alias, ok := voiceCatalogProviderAliases[p]; ok {
		return alias
	}
	return p
}

func voiceCatalogDir() (string, error) {
	voiceCatalogDirOnce.Do(func() {
		voiceCatalogDirPath, voiceCatalogDirErr = findVoiceCatalogDir()
	})
	return voiceCatalogDirPath, voiceCatalogDirErr
}

func findVoiceCatalogDir() (string, error) {
	if env := strings.TrimSpace(os.Getenv("VOICES_CATALOG_DIR")); env != "" {
		if st, err := os.Stat(env); err == nil && st.IsDir() {
			return filepath.Clean(env), nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := filepath.Clean(cwd)
	for {
		cand := filepath.Join(dir, "scripts", "voices")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("voice catalog: scripts/voices not found (cwd=%s)", cwd)
}

func loadVoiceCatalogFile(provider string) (*voiceCatalogFile, bool, error) {
	provider = resolveVoiceCatalogProvider(provider)
	if provider == "" {
		return nil, false, fmt.Errorf("voice catalog: empty provider")
	}
	root, err := voiceCatalogDir()
	if err != nil {
		return nil, false, err
	}
	path := filepath.Join(root, provider+".json")
	st, err := os.Stat(path)
	if err != nil {
		return nil, false, fmt.Errorf("voice catalog: %s: %w", provider, err)
	}
	if v, ok := voiceCatalogCache.Load(provider); ok {
		if entry, ok := v.(*voiceCatalogCacheEntry); ok && entry != nil && entry.file != nil {
			if entry.modTime.Equal(st.ModTime()) {
				return entry.file, true, nil
			}
		}
	}

	// Serialize disk loads per process; memory hits above skip this lock.
	voiceCatalogLoadMu.Lock()
	defer voiceCatalogLoadMu.Unlock()
	if st2, err2 := os.Stat(path); err2 == nil {
		if cached, ok := voiceCatalogCache.Load(provider); ok {
			if entry, ok := cached.(*voiceCatalogCacheEntry); ok && entry != nil && entry.file != nil && entry.modTime.Equal(st2.ModTime()) {
				return entry.file, true, nil
			}
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("voice catalog: %s: %w", provider, err)
	}
	var f voiceCatalogFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, false, fmt.Errorf("voice catalog: parse %s: %w", provider, err)
	}
	if f.Provider == "" {
		f.Provider = provider
	}
	entry := &voiceCatalogCacheEntry{
		file:    &f,
		modTime: st.ModTime(),
		loaded:  time.Now(),
	}
	if st2, err2 := os.Stat(path); err2 == nil {
		entry.modTime = st2.ModTime()
	}
	voiceCatalogCache.Store(provider, entry)
	return entry.file, false, nil
}

func listVoiceCatalog(provider, mode string) (voiceCatalogListResult, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "tts"
	}
	resolved := resolveVoiceCatalogProvider(provider)
	f, fromCache, err := loadVoiceCatalogFile(resolved)
	if err != nil {
		return voiceCatalogListResult{}, err
	}
	if !voiceCatalogModeAllowed(f, mode) {
		return voiceCatalogListResult{Provider: resolved, Mode: mode, Voices: []voiceCatalogEntry{}, Cached: fromCache}, nil
	}
	voices := make([]voiceCatalogEntry, len(f.Voices))
	copy(voices, f.Voices)
	return voiceCatalogListResult{
		Provider:   resolved,
		Mode:       mode,
		VoiceField: f.VoiceField,
		DocURL:     f.DocURL,
		Voices:     voices,
		Cached:     fromCache,
	}, nil
}

func voiceCatalogModeAllowed(f *voiceCatalogFile, mode string) bool {
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

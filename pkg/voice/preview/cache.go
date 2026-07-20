package preview

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/stores"
)

// Preview cache index lives under data/ (gitignored) — not in scripts/voices/*.json.
// Values are object storage keys, e.g. voice-previews/aliyun_omni/realtime/Tina.wav

var (
	cacheOnce sync.Once
	cachePath string
	cacheErr  error
	cacheMu   sync.Mutex
)

type manifest map[string]string // cacheKey -> objectKey

func cacheKey(provider, mode, voiceID string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + "/" +
		strings.ToLower(strings.TrimSpace(mode)) + "/" +
		strings.TrimSpace(voiceID)
}

func manifestPath() (string, error) {
	cacheOnce.Do(func() {
		if env := strings.TrimSpace(os.Getenv("VOICE_PREVIEW_CACHE_DIR")); env != "" {
			cachePath = filepath.Join(filepath.Clean(env), "manifest.json")
			return
		}
		cachePath = filepath.Join("data", "voice-preview-cache", "manifest.json")
	})
	if cacheErr != nil {
		return "", cacheErr
	}
	return cachePath, nil
}

func loadManifest() (manifest, error) {
	path, err := manifestPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest{}, nil
		}
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("voice preview cache: parse manifest: %w", err)
	}
	if m == nil {
		m = manifest{}
	}
	return m, nil
}

func saveManifest(m manifest) error {
	path, err := manifestPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

// CachedObjectKey returns the stored object key for a voice preview, if indexed.
func CachedObjectKey(provider, mode, voiceID string) (string, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	m, err := loadManifest()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(m[cacheKey(provider, mode, voiceID)]), nil
}

// SetCachedObjectKey records an object key for a voice preview sample.
func SetCachedObjectKey(provider, mode, voiceID, objectKey string) error {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return fmt.Errorf("voice preview cache: empty object key")
	}
	k := cacheKey(provider, mode, voiceID)
	cacheMu.Lock()
	defer cacheMu.Unlock()
	m, err := loadManifest()
	if err != nil {
		return err
	}
	if m == nil {
		m = manifest{}
	}
	if strings.TrimSpace(m[k]) == objectKey {
		return nil
	}
	m[k] = objectKey
	return saveManifest(m)
}

// ResolveObjectKey returns manifest key or checks object storage for the deterministic path.
func ResolveObjectKey(provider, mode, voiceID string) (string, bool, error) {
	if key, err := CachedObjectKey(provider, mode, voiceID); err != nil {
		return "", false, err
	} else if key != "" {
		return key, true, nil
	}
	key := ObjectKey(provider, mode, voiceID)
	store := stores.Default()
	ok, err := store.Exists(key)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	_ = SetCachedObjectKey(provider, mode, voiceID, key)
	return key, true, nil
}

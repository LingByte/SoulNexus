package welcomeaudio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	envAudioCacheDir = "VOICE_AUDIO_CACHE_DIR"
	defaultCacheDir  = "data/audio-cache"
	diskFetchTimeout = 15 * time.Second
)

var (
	diskDirOnce sync.Once
	diskDir     string
)

// DiskCacheDir returns the local directory used to persist downloaded WAVs.
func DiskCacheDir() string {
	diskDirOnce.Do(func() {
		d := strings.TrimSpace(os.Getenv(envAudioCacheDir))
		if d == "" {
			d = defaultCacheDir
		}
		diskDir = d
		_ = os.MkdirAll(diskDir, 0o755)
	})
	return diskDir
}

func diskKey(rawURL string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawURL)))
	return hex.EncodeToString(sum[:16]) + ".wav"
}

func diskPath(rawURL string) string {
	return filepath.Join(DiskCacheDir(), diskKey(rawURL))
}

// HasDiskCache reports whether a successful PrefetchURL result is already on disk.
func HasDiskCache(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	st, err := os.Stat(diskPath(rawURL))
	return err == nil && st.Size() > 0
}

// PrefetchURL downloads a remote WAV to the disk cache (idempotent).
// Uses an independent timeout so call-time context cancel cannot abort warmup.
func PrefetchURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || !strings.HasPrefix(strings.ToLower(rawURL), "http") {
		return nil
	}
	if HasDiskCache(rawURL) {
		return nil
	}
	path := diskPath(rawURL)
	ctx, cancel := context.WithTimeout(context.Background(), diskFetchTimeout)
	defer cancel()
	u, err := parseHTTPURL(rawURL)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: diskFetchTimeout}).Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("%w: GET %d", ErrUnreachable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxBytes+1))
	if err != nil {
		return fmt.Errorf("%w: read body: %v", ErrUnreachable, err)
	}
	if len(body) > MaxBytes {
		return fmt.Errorf("%w: body exceeds %d bytes", ErrUnreachable, MaxBytes)
	}
	if err := ValidateBytes(body); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadDiskWAV(rawURL string) ([]byte, bool) {
	path := diskPath(rawURL)
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return nil, false
	}
	if ValidateBytes(b) != nil {
		_ = os.Remove(path)
		return nil, false
	}
	return b, true
}

func saveDiskWAV(rawURL string, body []byte) {
	if len(body) == 0 {
		return
	}
	path := diskPath(rawURL)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

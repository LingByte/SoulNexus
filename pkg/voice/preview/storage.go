package preview

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/stores"
)

var voiceIDSlugRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// UploadPCM uploads mono PCM16LE as WAV and returns the storage object key.
func UploadPCM(provider, mode, voiceID string, pcm []byte, sampleRate int) (string, error) {
	wav := PCM16LEToWav(pcm, sampleRate)
	if len(wav) == 0 {
		return "", fmt.Errorf("voice preview: empty wav")
	}
	key := ObjectKey(provider, mode, voiceID)
	store := stores.Default()
	if err := store.Write(key, bytes.NewReader(wav)); err != nil {
		return "", err
	}
	return key, nil
}

// ObjectKey builds the deterministic storage key for a voice preview WAV.
func ObjectKey(provider, mode, voiceID string) string {
	prov := strings.ToLower(strings.TrimSpace(provider))
	if prov == "" {
		prov = "unknown"
	}
	m := strings.ToLower(strings.TrimSpace(mode))
	if m == "" {
		m = "tts"
	}
	slug := voiceIDSlugRe.ReplaceAllString(strings.TrimSpace(voiceID), "_")
	if slug == "" {
		slug = "default"
	}
	return path.Join("voice-previews", prov, m, slug+".wav")
}

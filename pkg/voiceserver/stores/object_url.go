package stores

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
)

// PublicObjectURL returns a fetchable URL for an uploaded object. Cloud backends usually return an absolute
// http(s) URL from PublicURL; for local disk, set STORAGE_PUBLIC_BASE_URL (e.g. https://api.example.com/media).
func PublicObjectURL(s Store, bucketName, key string) string {
	if s == nil {
		return ""
	}
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	u := strings.TrimSpace(s.PublicURL(bucketName, key))
	lower := strings.ToLower(u)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return u
	}
	base := strings.TrimSuffix(strings.TrimSpace(utils.GetEnv("STORAGE_PUBLIC_BASE_URL")), "/")
	if base != "" {
		return base + "/" + key
	}
	return u
}

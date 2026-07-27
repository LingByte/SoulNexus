package mediagen

import (
	"strconv"
	"strings"

	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

const (
	defaultSeedreamURL       = "https://ark.cn-beijing.volces.com/api/v3/images/generations"
	defaultSeedreamModel     = "doubao-seedream-5-0-260128"
	defaultSeedanceBaseURL   = "https://ark.cn-beijing.volces.com/api/v3"
	defaultSeedanceModel     = "doubao-seedance-1-5-pro-251215"
	defaultPollIntervalMs    = 10000
	defaultPollMaxAttempts   = 120
	envSeedreamAPIKey        = "SEEDREAM_API_KEY"
	envSeedreamURL           = "SEEDREAM_API_URL"
	envSeedreamModel         = "SEEDREAM_MODEL_ID"
	envSeedanceBaseURL       = "SEEDANCE_API_BASE_URL"
	envSeedanceModel         = "SEEDANCE_MODEL_ID"
	envSeedanceGenerateAudio = "SEEDANCE_GENERATE_AUDIO"
	envVideoPollIntervalMs   = "MEDIAGEN_VIDEO_POLL_INTERVAL_MS"
	envVideoPollMaxAttempts  = "MEDIAGEN_VIDEO_POLL_MAX_ATTEMPTS"
)

// Config holds Volcengine Ark (Seedream / Seedance) and local storage settings.
type Config struct {
	APIKey           string
	SeedreamURL      string
	SeedreamModel    string
	SeedanceBaseURL  string
	SeedanceModel    string
	GenerateAudio    bool
	PollIntervalMs   int
	PollMaxAttempts  int
	UploadBaseDir    string
	ElementsSubdir   string
	VideosSubdir     string
	ReferencesSubdir string
}

func LoadConfig() Config {
	uploadDir := strings.TrimSpace(utils.GetEnv(pkgconst.ENV_UPLOAD_DIR))
	if uploadDir == "" {
		uploadDir = pkgconst.DefaultUploadDir
	}
	generateAudio := true
	if v := strings.TrimSpace(utils.GetEnv(envSeedanceGenerateAudio)); v != "" {
		generateAudio = v == "1" || strings.EqualFold(v, "true")
	}
	return Config{
		APIKey:           strings.TrimSpace(utils.GetEnv(envSeedreamAPIKey)),
		SeedreamURL:      envOrDefault(envSeedreamURL, defaultSeedreamURL),
		SeedreamModel:    envOrDefault(envSeedreamModel, defaultSeedreamModel),
		SeedanceBaseURL:  strings.TrimRight(envOrDefault(envSeedanceBaseURL, defaultSeedanceBaseURL), "/"),
		SeedanceModel:    envOrDefault(envSeedanceModel, defaultSeedanceModel),
		GenerateAudio:    generateAudio,
		PollIntervalMs:   envIntOrDefault(envVideoPollIntervalMs, defaultPollIntervalMs),
		PollMaxAttempts:  envIntOrDefault(envVideoPollMaxAttempts, defaultPollMaxAttempts),
		UploadBaseDir:    uploadDir,
		ElementsSubdir:   "mediagen/elements",
		VideosSubdir:     "mediagen/videos",
		ReferencesSubdir: "mediagen/references",
	}
}

func (c Config) Configured() bool {
	key := strings.TrimSpace(c.APIKey)
	return key != "" && key != "your-api-key-placeholder"
}

func envOrDefault(name, fallback string) string {
	if v := strings.TrimSpace(utils.GetEnv(name)); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(name string, fallback int) int {
	v := strings.TrimSpace(utils.GetEnv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

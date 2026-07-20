package embedder

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/lingllm/utils"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

const defaultHTTPTimeoutSec = 60

// ApplyCustomHeaders attaches user-defined HTTP headers, skipping reserved headers.
func ApplyCustomHeaders(req *http.Request, headers map[string]string) {
	utils.ApplyCustomHeaders(req, headers)
}

// HTTPClientFromConfig builds an HTTP client from embedder config.
func HTTPClientFromConfig(cfg *Config) *http.Client {
	timeout := defaultHTTPTimeoutSec
	if cfg != nil && cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   16,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}
}

// DoRequestWithRetry executes an HTTP request with exponential backoff retries.
func DoRequestWithRetry(
	ctx context.Context,
	client *http.Client,
	maxRetries int,
	buildReq func() (*http.Request, error),
) (*http.Response, error) {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeoutSec * time.Second}
	}
	if maxRetries < 0 {
		maxRetries = 0
	}

	var resp *http.Response
	var err error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			backoff := time.Duration(1<<uint(i-1)) * time.Second
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		var req *http.Request
		req, err = buildReq()
		if err != nil {
			continue
		}

		resp, err = client.Do(req)
		if err == nil {
			return resp, nil
		}
	}

	return nil, err
}

// SanitizeEmbedInputs trims inputs and replaces empty strings with a single space.
func SanitizeEmbedInputs(texts []string) []string {
	if len(texts) == 0 {
		return nil
	}
	sanitized := make([]string, 0, len(texts))
	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			text = " "
		}
		sanitized = append(sanitized, text)
	}
	return sanitized
}

// CustomConfigString reads a string value from CustomConfig.
func CustomConfigString(cfg *Config, key string) string {
	if cfg == nil || cfg.CustomConfig == nil {
		return ""
	}
	v, ok := cfg.CustomConfig[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

// CustomConfigInt reads an int value from CustomConfig.
func CustomConfigInt(cfg *Config, key string) int {
	if cfg == nil || cfg.CustomConfig == nil {
		return 0
	}
	v, ok := cfg.CustomConfig[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// CustomConfigBool reads a bool value from CustomConfig.
func CustomConfigBool(cfg *Config, key string) bool {
	if cfg == nil || cfg.CustomConfig == nil {
		return false
	}
	v, ok := cfg.CustomConfig[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func maxRetriesFromConfig(cfg *Config) int {
	if cfg == nil || cfg.MaxRetries <= 0 {
		return 3
	}
	return cfg.MaxRetries
}

func truncatePromptTokensFromConfig(cfg *Config) int {
	if cfg == nil || cfg.TruncatePromptTokens <= 0 {
		return 511
	}
	return cfg.TruncatePromptTokens
}

func providerNameFromConfig(cfg *Config) string {
	if cfg == nil {
		return ProviderOpenAI
	}
	if p := NormalizeProvider(cfg.Provider); p != "" {
		return p
	}
	return ProviderOpenAI
}

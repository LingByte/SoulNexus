package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                 string
	Issuer               string
	Audience             string
	DatabasePath         string
	CookieDomain         string
	CookieSecure         bool
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	AuthorizationCodeTTL time.Duration
}

func Load() Config {
	return Config{
		Port:                 getEnv("SSO_PORT", "8090"),
		Issuer:               getEnv("SSO_ISSUER", "http://localhost:8090"),
		Audience:             getEnv("SSO_AUDIENCE", "soulnexus-api"),
		DatabasePath:         getEnv("SSO_DB_PATH", "sso.db"),
		CookieDomain:         getEnv("SSO_COOKIE_DOMAIN", ""),
		CookieSecure:         getBool("SSO_COOKIE_SECURE", false),
		AccessTokenTTL:       getDurationFromMinutes("SSO_ACCESS_TOKEN_MINUTES", 10),
		RefreshTokenTTL:      getDurationFromHours("SSO_REFRESH_TOKEN_HOURS", 24*14),
		AuthorizationCodeTTL: getDurationFromMinutes("SSO_AUTH_CODE_MINUTES", 5),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getDurationFromMinutes(key string, fallback int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(fallback) * time.Minute
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return time.Duration(fallback) * time.Minute
	}
	return time.Duration(n) * time.Minute
}

func getDurationFromHours(key string, fallback int) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(fallback) * time.Hour
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return time.Duration(fallback) * time.Hour
	}
	return time.Duration(n) * time.Hour
}

func getBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

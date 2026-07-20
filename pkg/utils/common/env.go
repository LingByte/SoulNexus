package common

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
)

var envCache *cache.ExpiredLRUCache[string, string]

func init() {
	size := 1024
	v, _ := strconv.ParseInt(GetEnv(constants.ENV_CONFIG_CACHE_SIZE), 10, 32)
	if v > 0 {
		size = int(v)
	}

	var configCacheExpired = 10 * time.Second
	exp, err := time.ParseDuration(GetEnv(constants.ENV_CONFIG_CACHE_EXPIRED))
	if err == nil {
		configCacheExpired = exp
	}

	envCache = cache.NewExpiredLRUCache[string, string](size, configCacheExpired)
}

func GetEnv(key string) string {
	v, _ := LookupEnv(key)
	return v
}

// PurgeEnvCacheForTest clears cached environment lookups. Tests that mutate
// process env via t.Setenv should call this in t.Cleanup when other tests
// wire components that read digest secrets from env.
func PurgeEnvCacheForTest() {
	if envCache != nil {
		envCache.Purge()
	}
}

func GetBoolEnv(key string) bool {
	v, _ := strconv.ParseBool(GetEnv(key))
	return v
}

func GetFloatEnv(key string) float64 {
	v, _ := strconv.ParseFloat(GetEnv(key), 64)
	return v
}

func GetIntEnv(key string) int64 {
	v, _ := strconv.ParseInt(GetEnv(key), 10, 64)
	return v
}

// EnvInt reads an integer environment variable. Returns (0, false) when unset or invalid.
func EnvInt(key string) (int, bool) {
	v := strings.TrimSpace(GetEnv(key))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// PositiveIntEnv reads a strictly positive integer env var. Returns (0, false) when unset, invalid, or <= 0.
func PositiveIntEnv(key string) (int, bool) {
	n, ok := EnvInt(key)
	if !ok || n <= 0 {
		return 0, false
	}
	return n, true
}

// EnvFloat reads a float environment variable. Returns (0, false) when unset or invalid.
func EnvFloat(key string) (float64, bool) {
	v := strings.TrimSpace(GetEnv(key))
	if v == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// GetFloatEnvWithDefault gets float environment variable with default value
func GetFloatEnvWithDefault(key string, defaultValue float64) float64 {
	v := GetFloatEnv(key)
	if v == 0 {
		return defaultValue
	}
	return v
}

// GetIntEnvWithDefault gets int environment variable with default value
func GetIntEnvWithDefault(key string, defaultValue int) int {
	v := GetIntEnv(key)
	if v == 0 {
		return defaultValue
	}
	return int(v)
}

// GetStringOrDefault gets environment variable value, returns default if empty.
func GetStringOrDefault(key, defaultValue string) string {
	value := GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetBoolOrDefault gets boolean environment variable value, returns default if empty.
func GetBoolOrDefault(key string, defaultValue bool) bool {
	value := GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return GetBoolEnv(key)
}

// GetIntOrDefault gets integer environment variable value, returns default if zero.
func GetIntOrDefault(key string, defaultValue int) int {
	value := GetIntEnv(key)
	if value == 0 {
		return defaultValue
	}
	return int(value)
}

// GetFloatOrDefault gets float environment variable value, returns default if empty.
func GetFloatOrDefault(key string, defaultValue float64) float64 {
	value := GetEnv(key)
	if value == "" {
		return defaultValue
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return defaultValue
}

// ParseDuration parses duration string with default fallback.
func ParseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

func LookupEnv(key string) (value string, found bool) {
	key = strings.ToUpper(key)
	if v, ok := os.LookupEnv(key); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			if envCache != nil {
				envCache.Add(key, v)
			}
			return v, true
		}
		if envCache != nil {
			envCache.Remove(key)
		}
		return "", true
	}
	if envCache != nil {
		if v, ok := envCache.Get(key); ok {
			return v, true
		}
	}
	if p, ok := findEnvFileUpwards(".env"); ok {
		data, err := os.ReadFile(p)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for i := 0; i < len(lines); i++ {
				v := strings.TrimSpace(lines[i])
				if v == "" || v[0] == '#' || !strings.Contains(v, "=") {
					continue
				}
				vs := strings.SplitN(v, "=", 2)
				k, vv := normalizeEnvKey(vs[0]), trimEnvValue(strings.TrimSpace(vs[1]))
				if k == "" {
					continue
				}

				if envCache != nil {
					envCache.Add(k, vv)
				}
				if k == key {
					return vv, true
				}
			}
		}
	}
	return "", false
}

func trimEnvValue(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && v[0] != '"' && v[0] != '\'' {
		for i := 0; i < len(v); i++ {
			if v[i] != '#' {
				continue
			}
			if i > 0 && (v[i-1] == ' ' || v[i-1] == '\t') {
				v = strings.TrimSpace(v[:i])
				break
			}
		}
	}
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return strings.TrimSpace(v[1 : len(v)-1])
		}
	}
	return v
}

func normalizeEnvKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(k), "export ") {
		k = strings.TrimSpace(k[len("export "):])
	}
	return strings.ToUpper(strings.TrimSpace(k))
}

func findEnvFileUpwards(filename string) (path string, ok bool) {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "", false
	}
	dir, err := os.Getwd()
	if err != nil || strings.TrimSpace(dir) == "" {
		return "", false
	}
	dir = filepath.Clean(dir)
	for {
		cand := filepath.Join(dir, filename)
		if st, err := os.Stat(cand); err == nil && st != nil && !st.IsDir() {
			return cand, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// LoadEnvs loads tagged env fields into a struct pointer.
func LoadEnvs(objPtr any) {
	if objPtr == nil {
		return
	}
	elm := reflect.ValueOf(objPtr).Elem()
	elmType := elm.Type()

	for i := 0; i < elm.NumField(); i++ {
		f := elm.Field(i)
		if !f.CanSet() {
			continue
		}
		keyName := elmType.Field(i).Tag.Get("env")
		if keyName == "-" {
			continue
		}
		if keyName == "" {
			keyName = elmType.Field(i).Name
		}
		switch f.Kind() {
		case reflect.String:
			if v, ok := LookupEnv(keyName); ok {
				f.SetString(v)
			}
		case reflect.Int:
			if v, ok := LookupEnv(keyName); ok {
				if iv, err := strconv.ParseInt(v, 10, 32); err == nil {
					f.SetInt(iv)
				}
			}
		case reflect.Bool:
			if v, ok := LookupEnv(keyName); ok {
				v := strings.ToLower(v)
				if yes, err := strconv.ParseBool(v); err == nil {
					f.SetBool(yes)
				}
			}
		}
	}
}

// LoadEnv loads a .env file into the process environment.
func LoadEnv(env string) error {
	envFile := ".env"
	if env != "" {
		envFile = ".env." + env
	}

	data, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if strings.HasPrefix(strings.ToLower(key), "export ") {
			key = strings.TrimSpace(key[len("export "):])
		}
		value := trimEnvValue(strings.TrimSpace(parts[1]))
		os.Setenv(key, value)
	}

	return nil
}

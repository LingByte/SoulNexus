package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Default Value: 1024
const ENV_CONFIG_CACHE_SIZE = "CONFIG_CACHE_SIZE"

// Default Value: 10s
const ENV_CONFIG_CACHE_EXPIRED = "CONFIG_CACHE_EXPIRED"

type Config struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Key       string `json:"key" gorm:"size:128;uniqueIndex"`
	Desc      string `json:"desc" gorm:"size:200"`
	Autoload  bool   `json:"autoload" gorm:"index"`
	Public    bool   `json:"public" gorm:"index" default:"false"`
	Format    string `json:"format" gorm:"size:20" default:"text" comment:"json,yaml,int,float,bool,text"`
	Value     string
	CreatedAt time.Time `json:"-" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"autoUpdateTime"`
}

var configValueCache *ExpiredLRUCache[string, string]
var envCache *ExpiredLRUCache[string, string]

func init() {
	size := 1024 // fixed size
	v, _ := strconv.ParseInt(GetEnv(ENV_CONFIG_CACHE_SIZE), 10, 32)
	if v > 0 {
		size = int(v)
	}

	var configCacheExpired = 10 * time.Second
	exp, err := time.ParseDuration(GetEnv(ENV_CONFIG_CACHE_EXPIRED))
	if err == nil {
		configCacheExpired = exp
	}

	configValueCache = NewExpiredLRUCache[string, string](size, configCacheExpired)
	envCache = NewExpiredLRUCache[string, string](size, configCacheExpired)
}

func GetEnv(key string) string {
	v, _ := LookupEnv(key)
	return v
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
		// Variable is set in the process environment but empty (e.g. systemd placeholder).
		// Drop any stale cache entry so .env can supply the value below.
		if envCache != nil {
			envCache.Remove(key)
		}
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
	// If the value is not quoted, strip trailing inline comments.
	// Common .env style: KEY=value # comment
	if v != "" && v[0] != '"' && v[0] != '\'' {
		// Only treat # as comment when preceded by whitespace, so values like passwords
		// containing '#' (e.g. "pass##word") are not truncated.
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
	// Support "export KEY=VALUE" style.
	if strings.HasPrefix(strings.ToLower(k), "export ") {
		k = strings.TrimSpace(k[len("export "):])
	}
	return strings.ToUpper(strings.TrimSpace(k))
}

// findEnvFileUpwards searches for filename from the current working directory upwards.
// This makes .env discovery robust when the process starts from a subdirectory (e.g. cmd/server).
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

// load envs to struct
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

func SetValue(db *gorm.DB, key, value, format string, autoload, public bool) {
	key = strings.ToUpper(key)
	configValueCache.Remove(key)

	newV := &Config{
		Key:      key,
		Value:    value,
		Format:   format,
		Autoload: autoload,
		Public:   public,
	}
	result := db.Model(&Config{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "format", "autoload", "public"}),
	}).Create(newV)

	if result.Error != nil {
		logrus.WithFields(logrus.Fields{
			"key":    key,
			"value":  value,
			"format": format,
		}).WithError(result.Error).Warn("config: setValue fail")
	}
}

func GetValue(db *gorm.DB, key string) string {
	key = strings.ToUpper(key)
	cobj, ok := configValueCache.Get(key)
	if ok {
		return cobj
	}

	var v Config
	result := db.Where("key", key).Take(&v)
	if result.Error != nil {
		return ""
	}

	configValueCache.Add(key, v.Value)
	return v.Value
}

func GetIntValue(db *gorm.DB, key string, defaultVal int) int {
	v := GetValue(db, key)
	if v == "" {
		return defaultVal
	}
	val, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultVal
	}
	return int(val)
}

func GetBoolValue(db *gorm.DB, key string) bool {
	v := GetValue(db, key)
	if v == "" {
		return false
	}

	r, _ := strconv.ParseBool(strings.ToLower(v))
	return r
}

func CheckValue(db *gorm.DB, key, defaultValue, format string, autoload, public bool) {
	newV := &Config{
		Key:      strings.ToUpper(key),
		Value:    defaultValue,
		Format:   format,
		Autoload: autoload,
		Public:   public,
	}
	db.Model(&Config{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(newV)
}

func LoadAutoloads(db *gorm.DB) {
	var configs []Config
	db.Where("autoload", true).Find(&configs)
	for _, v := range configs {
		configValueCache.Add(v.Key, v.Value)
	}
}

func LoadPublicConfigs(db *gorm.DB) []Config {
	var configs []Config
	db.Where("public", true).Find(&configs)
	for _, v := range configs {
		configValueCache.Add(v.Key, v.Value)
	}
	return configs
}

// LoadEnv Load .env file based on environment
func LoadEnv(env string) error {
	// Load .env file by default
	envFile := ".env"
	if env != "" {
		envFile = ".env." + env
	}

	// Read .env file
	data, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}

	// Parse .env file
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

// GetStringOrDefault gets environment variable value, returns default if empty
func GetStringOrDefault(key, defaultValue string) string {
	value := GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

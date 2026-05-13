package utils

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
	v, _ := strconv.ParseInt(GetEnv(constants.ENV_CONFIG_CACHE_SIZE), 10, 32)
	if v > 0 {
		size = int(v)
	}

	var configCacheExpired = 10 * time.Second
	exp, err := time.ParseDuration(GetEnv(constants.ENV_CONFIG_CACHE_EXPIRED))
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

// GetStringOrDefault returns the env var when non-empty, otherwise defaultValue.
// Useful for the common "load with sensible fallback" pattern; the bool-returning
// LookupEnv is reserved for callers that need to distinguish "unset" from "empty".
func GetStringOrDefault(key, defaultValue string) string {
	v := GetEnv(key)
	if v == "" {
		return defaultValue
	}
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
		if envCache != nil {
			envCache.Add(key, v)
		}
		return v, true
	}
	if envCache != nil {
		if v, ok := envCache.Get(key); ok {
			return v, true
		}
	}
	data, err := os.ReadFile(".env")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for i := 0; i < len(lines); i++ {
			v := strings.TrimSpace(lines[i])
			if v == "" || v[0] == '#' || !strings.Contains(v, "=") {
				continue
			}
			vs := strings.SplitN(v, "=", 2)
			k, vv := strings.ToUpper(strings.TrimSpace(vs[0])), strings.TrimSpace(vs[1])

			if envCache != nil {
				envCache.Add(k, vv)
			}
			if k == key {
				return vv, true
			}
		}
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

// applyEnvLines parses KEY=VALUE lines (same rules as legacy LoadEnv) and calls os.Setenv.
func applyEnvLines(data []byte) {
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
		value := strings.TrimSpace(parts[1])
		os.Setenv(key, value)
	}
}

// LoadEnv loads a single file: `.env`, or `.env.{env}` when env is non-empty (legacy helper).
func LoadEnv(env string) error {
	envFile := ".env"
	if env != "" {
		envFile = ".env." + env
	}
	data, err := os.ReadFile(envFile)
	if err != nil {
		return err
	}
	applyEnvLines(data)
	return nil
}

// LoadModuleEnvs loads layered env files for one deployable binary (auth, server, sip, …):
//  1. .env-{module}
//  2. .env-{module}.{appEnv} when appEnv is non-empty (overrides earlier keys).
//
// Missing files are skipped with a log line only (same spirit as partial .env loading).
func LoadModuleEnvs(module string, appEnv string) error {
	module = strings.TrimSpace(module)
	if module == "" {
		return errors.New("LoadModuleEnvs: empty module name")
	}
	readOptional := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Note: optional env file %s: %v", path, err)
			return
		}
		applyEnvLines(data)
		log.Printf("Loaded env file %s", path)
	}
	readOptional(fmt.Sprintf(".env-%s", module))
	appEnv = strings.TrimSpace(appEnv)
	if appEnv != "" {
		readOptional(fmt.Sprintf(".env-%s.%s", module, appEnv))
	}
	return nil
}

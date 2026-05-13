package dbconfig

import (
	"log"
	"os"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// Config main configuration structure
type Config struct {
	MachineID int64          `env:"MACHINE_ID"`
	Database  DatabaseConfig `mapstructure:"database"`
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver string `env:"DB_DRIVER"`
	DSN    string `env:"DSN"`
}

var GlobalConfig *Config

func Load() error {
	// 1. Load .env file based on environment (don't error if it doesn't exist, use default values)
	env := os.Getenv("MODE")
	err := utils.LoadEnv(env)
	if err != nil {
		// Only log when .env file doesn't exist, don't affect startup
		log.Printf("Note: .env file not found or failed to load: %v (using default values)", err)
	}

	// 2. Load global configuration
	GlobalConfig = &Config{
		MachineID: utils.GetIntEnv("MACHINE_ID"),
		Database: DatabaseConfig{
			Driver: utils.GetStringOrDefault("DB_DRIVER", "sqlite"),
			DSN:    utils.GetStringOrDefault("DSN", "./ling.db"),
		},
	}
	return nil
}

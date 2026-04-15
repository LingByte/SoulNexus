package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// LogConfigInfo Print global configuration information
func LogConfigInfo() {
	logger.Info("system config load finished")
	logger.Info("global config",
		zap.String("server_name", config.GlobalConfig.Server.Name),
		zap.String("server_desc", config.GlobalConfig.Server.Desc),
		zap.String("server_logo", config.GlobalConfig.Server.Logo),
		zap.String("server_url", config.GlobalConfig.Server.URL),
		zap.String("server_terms_url", config.GlobalConfig.Server.TermsURL),
		zap.String("mode", config.GlobalConfig.Server.Mode),
	)

	logger.Info("base config",
		zap.Int64("machine_id", config.GlobalConfig.MachineID),
		zap.String("addr", config.GlobalConfig.Server.Addr),
		zap.String("db_driver", config.GlobalConfig.Database.Driver),
		zap.String("dsn", config.GlobalConfig.Database.DSN),
		zap.String("monitor_prefix", config.GlobalConfig.Server.MonitorPrefix),
		zap.Bool("language_enabled", config.GlobalConfig.Features.LanguageEnabled),
		zap.String("api_secret_key", config.GlobalConfig.Auth.APISecretKey),
	)

	logger.Info("api config",
		zap.String("api_prefix", config.GlobalConfig.Server.APIPrefix),
		zap.String("docs_prefix", config.GlobalConfig.Server.DocsPrefix),
		zap.String("auth_prefix", config.GlobalConfig.Server.AuthPrefix),
		zap.String("secret_expire_days", config.GlobalConfig.Auth.SecretExpireDays),
		zap.String("session_secret", config.GlobalConfig.Auth.SessionSecret),
	)

	logger.Info("log config",
		zap.String("log_level", config.GlobalConfig.Log.Level),
		zap.String("log_filename", config.GlobalConfig.Log.Filename),
		zap.Int("log_max_size", config.GlobalConfig.Log.MaxSize),
		zap.Int("log_max_age", config.GlobalConfig.Log.MaxAge),
		zap.Int("log_max_backups", config.GlobalConfig.Log.MaxBackups),
	)

	logger.Info("search config",
		zap.Bool("search_enabled", config.GlobalConfig.Features.SearchEnabled),
		zap.String("search_path", config.GlobalConfig.Features.SearchPath),
		zap.Int("search_batch_size", config.GlobalConfig.Features.SearchBatchSize),
	)
	logger.Info("backup config",
		zap.Bool("backup_enabled", config.GlobalConfig.Features.BackupEnabled),
		zap.String("backup_path", config.GlobalConfig.Features.BackupPath),
		zap.String("backup_schedule", config.GlobalConfig.Features.BackupSchedule),
	)

	logger.Info("xun fei config",
		zap.String("xun_fei_ws_api_id", config.GlobalConfig.Services.Voice.Xunfei.WSAppId),
		zap.String("xun_fei_ws_api_secret", config.GlobalConfig.Services.Voice.Xunfei.WSAPISecret),
		zap.String("xun_fei_ws_api_key", config.GlobalConfig.Services.Voice.Xunfei.WSAPIKey),
	)
}

// PrintBannerFromFile Read file and print
func PrintBannerFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	colors := []string{
		"\x1b[38;5;165m",
		"\x1b[38;5;189m",
		"\x1b[38;5;207m",
		"\x1b[38;5;219m",
		"\x1b[38;5;225m",
		"\x1b[38;5;231m",
	}

	for i, line := range lines {
		color := colors[i%len(colors)]
		fmt.Println(color + line + "\x1b[0m")
	}
	return nil
}

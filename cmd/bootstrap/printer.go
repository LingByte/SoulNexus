package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	lingllmversion "github.com/LingByte/lingllm/version"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogConfigInfo Print global configuration information
func LogConfigInfo() {
	logger.Info("system config load finished")
	LogLingLLMVersion()
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
		zap.String("system_timezone", timeutil.Name()),
		zap.String("db_driver", config.GlobalConfig.Database.Driver),
		zap.String("dsn", config.GlobalConfig.Database.DSN),
		zap.String("api_secret_key", config.GlobalConfig.Auth.APISecretKey),
	)

	logger.Info("api config",
		zap.String("api_prefix", config.GlobalConfig.Server.APIPrefix),
		zap.String("docs_prefix", config.GlobalConfig.Server.DocsPrefix),
		zap.String("secret_expire_days", config.GlobalConfig.Auth.SecretExpireDays),
		zap.String("session_secret", config.GlobalConfig.Auth.SessionSecret),
	)

	logger.Info("log config",
		zap.String("log_level", config.GlobalConfig.Log.Level),
		zap.String("log_filename", config.GlobalConfig.Log.Filename),
		zap.Int("log_max_size", config.GlobalConfig.Log.MaxSize),
		zap.Int("log_max_age", config.GlobalConfig.Log.MaxAge),
		zap.Int("log_retention_days", config.GlobalConfig.Log.RetentionDays),
		zap.Int("log_max_backups", config.GlobalConfig.Log.MaxBackups),
	)

	logger.Info("backup config",
		zap.Bool("backup_enabled", utils.GetBoolEnv(pkgconst.ENV_BACKUP_ENABLED)),
		zap.String("backup_path", utils.GetEnv(pkgconst.ENV_BACKUP_PATH)),
		zap.String("backup_schedule", utils.GetEnv(pkgconst.ENV_BACKUP_SCHEDULE)),
		zap.Int("backup_retention_days", int(utils.GetIntEnv(pkgconst.ENV_BACKUP_RETENTION_DAYS))),
	)
}

// PrintBannerFromFile Read file and print, auto-generate if file doesn't exist
func PrintBannerFromFile(filename string, defaultText string) error {
	// Ensure banner file exists, generate if it doesn't
	if err := EnsureBannerFile(filename, defaultText); err != nil {
		return fmt.Errorf("failed to ensure banner file: %w", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	colors := []string{
		"\x1b[38;5;117m", // light blue
		"\x1b[38;5;111m",
		"\x1b[38;5;75m",
		"\x1b[38;5;39m",
		"\x1b[38;5;33m",
		"\x1b[38;5;27m", // dark blue
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		color := colors[i%len(colors)]
		fmt.Println(color + line + "\x1b[0m")
	}
	return nil
}

// ValidateProductionSecurityEnv refuses to start in release/production when
// known fail-open toggles are enabled.
func ValidateProductionSecurityEnv() {
	if !productionLikeRuntime() {
		return
	}
	checks := []struct {
		env  string
		desc string
	}{
		{constants.ENVUploadsRecordingsPublic, "public voice recordings under /uploads"},
		{constants.ENVVoiceDialogAllowEmptyToken, "VoiceDialog WS without token"},
	}
	for _, c := range checks {
		if utils.GetBoolEnv(c.env) {
			logger.Fatal("unsafe env for production",
				zap.String("env", c.env),
				zap.String("reason", c.desc),
			)
		}
	}
}

func productionLikeRuntime() bool {
	if gin.Mode() == gin.ReleaseMode {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(utils.GetEnv(pkgconst.EnvAppEnv))) {
	case pkgconst.ENV_PROD:
		return true
	default:
		return false
	}
}

// LogLingLLMVersion prints embedded lingllm library version metadata at startup.
func LogLingLLMVersion() {
	logger.Info("lingllm",
		zap.String("version", lingllmversion.GetVersion()),
		zap.String("commit", lingllmversion.GetGitCommit()),
		zap.String("built_at", lingllmversion.GetBuildTime()),
		zap.String("go", lingllmversion.GetGoVersion()),
		zap.String("version_info", lingllmversion.GetVersionInfo()),
	)
}

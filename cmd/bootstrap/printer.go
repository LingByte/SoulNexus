package bootstrap

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
		zap.String("server_name", config.GlobalConfig.ServerName),
		zap.String("server_desc", config.GlobalConfig.ServerDesc),
		zap.String("server_logo", config.GlobalConfig.ServerLogo),
		zap.String("server_url", config.GlobalConfig.ServerUrl),
		zap.String("server_terms_url", config.GlobalConfig.ServerTermsUrl),
		zap.String("mode", config.GlobalConfig.Mode),
	)

	logger.Info("base config",
		zap.Int64("machine_id", config.GlobalConfig.MachineID),
		zap.String("addr", config.GlobalConfig.Addr),
		zap.String("db_driver", config.GlobalConfig.DBDriver),
		zap.String("dsn", config.GlobalConfig.DSN),
		zap.String("monitor_prefix", config.GlobalConfig.MonitorPrefix),
		zap.Bool("language_enabled", config.GlobalConfig.LanguageEnabled),
		zap.String("api_secret_key", config.GlobalConfig.APISecretKey),
	)

	logger.Info("api config",
		zap.String("api_prefix", config.GlobalConfig.APIPrefix),
		zap.String("docs_prefix", config.GlobalConfig.DocsPrefix),
		zap.String("admin_prefix", config.GlobalConfig.AdminPrefix),
		zap.String("auth_prefix", config.GlobalConfig.AuthPrefix),
		zap.String("secret_expire_days", config.GlobalConfig.SecretExpireDays),
		zap.String("session_secret", config.GlobalConfig.SessionSecret),
	)

	logger.Info("log config",
		zap.String("log_level", config.GlobalConfig.Log.Level),
		zap.String("log_filename", config.GlobalConfig.Log.Filename),
		zap.Int("log_max_size", config.GlobalConfig.Log.MaxSize),
		zap.Int("log_max_age", config.GlobalConfig.Log.MaxAge),
		zap.Int("log_max_backups", config.GlobalConfig.Log.MaxBackups),
	)

	logger.Info("mail config",
		zap.String("mail_host", config.GlobalConfig.Mail.Host),
		zap.String("mail_username", config.GlobalConfig.Mail.Username),
		zap.String("mail_password", config.GlobalConfig.Mail.Password),
		zap.String("mail_from", config.GlobalConfig.Mail.From),
		zap.Int64("mail_port", config.GlobalConfig.Mail.Port),
	)

	logger.Info("search config",
		zap.Bool("search_enabled", config.GlobalConfig.SearchEnabled),
		zap.String("search_path", config.GlobalConfig.SearchPath),
		zap.Int("search_batch_size", config.GlobalConfig.SearchBatchSize),
	)
	logger.Info("backup config",
		zap.Bool("backup_enabled", config.GlobalConfig.BackupEnabled),
		zap.String("backup_path", config.GlobalConfig.BackupPath),
		zap.String("backup_schedule", config.GlobalConfig.BackupSchedule),
	)

	logger.Info("xun fei config",
		zap.String("xun_fei_ws_api_id", config.GlobalConfig.XunfeiWsAppId),
		zap.String("xun_fei_ws_api_secret", config.GlobalConfig.XunfeiWsApiSecret),
		zap.String("xun_fei_ws_api_secret", config.GlobalConfig.XunfeiWsApiKey),
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

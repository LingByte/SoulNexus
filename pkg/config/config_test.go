package config

import (
	"os"
	"testing"
)

// 为了避免不同用例间互相污染，统一用 t.Setenv 设置环境变量
func setAllEnvs(t *testing.T) {
	t.Setenv("MACHINE_ID", "7")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DSN", "host=127.0.0.1 user=u dbname=d sslmode=disable")
	t.Setenv("ADDR", ":8080")
	t.Setenv("MODE", "release")

	t.Setenv("DOCS_PREFIX", "/docs")
	t.Setenv("API_PREFIX", "/api")
	t.Setenv("ADMIN_PREFIX", "/admin")
	t.Setenv("AUTH_PREFIX", "/auth")

	t.Setenv("SESSION_EXPIRE_DAYS", "14")
	t.Setenv("SESSION_SECRET", "secret-xyz")

	// 日志
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOG_FILENAME", "app.log")
	t.Setenv("LOG_MAX_SIZE", "128")
	t.Setenv("LOG_MAX_AGE", "14")
	t.Setenv("LOG_MAX_BACKUPS", "7")

	// 邮件
	t.Setenv("MAIL_HOST", "smtp.example.com")
	t.Setenv("MAIL_USERNAME", "user@example.com")
	t.Setenv("MAIL_PASSWORD", "pass")
	t.Setenv("MAIL_PORT", "587")
	t.Setenv("MAIL_FROM", "noreply@example.com")

	// LLM
	t.Setenv("LLM_API_KEY", "ak")
	t.Setenv("LLM_BASE_URL", "https://llm.example.com")
	t.Setenv("LLM_MODEL", "gpt-x")

	// Search
	t.Setenv("SEARCH_ENABLED", "1")
	t.Setenv("SEARCH_PATH", "/var/search")
	t.Setenv("SEARCH_BATCH_SIZE", "500")

	t.Setenv("MONITOR_PREFIX", "/monitor")
	t.Setenv("LANGUAGE_ENABLED", "true")
	t.Setenv("API_SECRET_KEY", "api-secret")

	// 备份
	t.Setenv("BACKUP_ENABLED", "true")
	t.Setenv("BACKUP_PATH", "/var/backup")
	t.Setenv("BACKUP_SCHEDULE", "0 2 * * *")

	// 七牛 ASR/TTS
	t.Setenv("QINIU_ASR_API_KEY", "q-asr-ak")
	t.Setenv("QINIU_ASR_BASE_URL", "https://asr.qiniu.example.com")
	t.Setenv("QINIU_TTS_API_KEY", "q-tts-ak")
	t.Setenv("QINIU_TTS_BASE_URL", "https://tts.qiniu.example.com")
}

func TestLoad_WithExplicitAppEnv(t *testing.T) {
	// 显式设置 APP_ENV，触发 util.LoadEnv(env) 的非默认分支
	t.Setenv("APP_ENV", "production")
	setAllEnvs(t)

	// 清空全局，避免前序测试污染
	GlobalConfig = nil

	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if GlobalConfig == nil {
		t.Fatalf("GlobalConfig is nil after Load")
	}

	// 基本字段
	if GlobalConfig.MachineID != 7 {
		t.Fatalf("MachineID=%d, want 7", GlobalConfig.MachineID)
	}
	if GlobalConfig.DBDriver != "postgres" {
		t.Fatalf("DBDriver=%q", GlobalConfig.DBDriver)
	}
	if GlobalConfig.DSN != "host=127.0.0.1 user=u dbname=d sslmode=disable" {
		t.Fatalf("DSN=%q", GlobalConfig.DSN)
	}
	if GlobalConfig.Addr != ":8080" || GlobalConfig.Mode != "release" {
		t.Fatalf("Addr=%q Mode=%q", GlobalConfig.Addr, GlobalConfig.Mode)
	}

	// 路由前缀
	if GlobalConfig.DocsPrefix != "/docs" ||
		GlobalConfig.APIPrefix != "/api" ||
		GlobalConfig.AdminPrefix != "/admin" ||
		GlobalConfig.AuthPrefix != "/auth" {
		t.Fatalf("prefix mismatch: %+v", *GlobalConfig)
	}

	// Session
	if GlobalConfig.SecretExpireDays != "14" || GlobalConfig.SessionSecret != "secret-xyz" {
		t.Fatalf("session mismatch: %+v", *GlobalConfig)
	}

	// 日志
	if GlobalConfig.Log.Level != "info" ||
		GlobalConfig.Log.Filename != "app.log" ||
		GlobalConfig.Log.MaxSize != 128 ||
		GlobalConfig.Log.MaxAge != 14 ||
		GlobalConfig.Log.MaxBackups != 7 {
		t.Fatalf("log config mismatch: %+v", GlobalConfig.Log)
	}

	// 邮件
	if GlobalConfig.Mail.Host != "smtp.example.com" ||
		GlobalConfig.Mail.Username != "user@example.com" ||
		GlobalConfig.Mail.Password != "pass" ||
		GlobalConfig.Mail.Port != 587 ||
		GlobalConfig.Mail.From != "noreply@example.com" {
		t.Fatalf("mail config mismatch: %+v", GlobalConfig.Mail)
	}

	// LLM
	if GlobalConfig.LLMApiKey != "ak" ||
		GlobalConfig.LLMBaseURL != "https://llm.example.com" ||
		GlobalConfig.LLMModel != "gpt-x" {
		t.Fatalf("llm mismatch: %+v", *GlobalConfig)
	}

	// Search
	if !GlobalConfig.SearchEnabled ||
		GlobalConfig.SearchPath != "/var/search" ||
		GlobalConfig.SearchBatchSize != 500 {
		t.Fatalf("search mismatch: enabled=%v path=%q batch=%d",
			GlobalConfig.SearchEnabled, GlobalConfig.SearchPath, GlobalConfig.SearchBatchSize)
	}

	// 其他
	if GlobalConfig.MonitorPrefix != "/monitor" ||
		!GlobalConfig.LanguageEnabled ||
		GlobalConfig.APISecretKey != "api-secret" {
		t.Fatalf("monitor/lang/api mismatch: %+v", *GlobalConfig)
	}

	// 备份
	if !GlobalConfig.BackupEnabled ||
		GlobalConfig.BackupPath != "/var/backup" ||
		GlobalConfig.BackupSchedule != "0 2 * * *" {
		t.Fatalf("backup mismatch: %+v", *GlobalConfig)
	}

	// 七牛 ASR/TTS
	if GlobalConfig.QiniuASRApiKey != "q-asr-ak" ||
		GlobalConfig.QiniuASRBaseURL != "https://asr.qiniu.example.com" ||
		GlobalConfig.QiniuTTSApiKey != "q-tts-ak" ||
		GlobalConfig.QiniuTTSBaseURL != "https://tts.qiniu.example.com" {
		t.Fatalf("qiniu mismatch: %+v", *GlobalConfig)
	}
}

func TestLoad_DefaultsWhenAppEnvEmpty(t *testing.T) {
	// APP_ENV 为空，走默认 development 分支
	_ = os.Unsetenv("APP_ENV")
	setAllEnvs(t)

	GlobalConfig = nil
	if err := Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if GlobalConfig == nil {
		t.Fatalf("GlobalConfig is nil after Load")
	}

	// 抽查几个关键字段，确认仍能正确从环境取值
	if GlobalConfig.MachineID != 7 {
		t.Fatalf("MachineID=%d, want 7", GlobalConfig.MachineID)
	}
	if GlobalConfig.SearchBatchSize != 500 || !GlobalConfig.SearchEnabled {
		t.Fatalf("search config unexpected: %+v", *GlobalConfig)
	}
	if !GlobalConfig.LanguageEnabled || !GlobalConfig.BackupEnabled {
		t.Fatalf("bool parsing unexpected: language=%v backup=%v",
			GlobalConfig.LanguageEnabled, GlobalConfig.BackupEnabled)
	}
	if GlobalConfig.Mail.Port != 587 {
		t.Fatalf("mail port=%d, want 587", GlobalConfig.Mail.Port)
	}
}

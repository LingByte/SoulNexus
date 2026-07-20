package constants

import "time"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

const (
	ENV_LOCAL = "local"
	ENV_DEV   = "dev"
	ENV_PROD  = "prod"

	// Backup env keys
	ENV_BACKUP_ENABLED        = "BACKUP_ENABLED"
	ENV_BACKUP_PATH           = "BACKUP_PATH"
	ENV_BACKUP_SCHEDULE       = "BACKUP_SCHEDULE"
	ENV_BACKUP_RETENTION_DAYS = "BACKUP_RETENTION_DAYS"

	// Database driver names
	DBDriverSQLite = "sqlite"
	DBDriverMySQL  = "mysql"
	DBDriverPG     = "pg"
)

// Server defaults
const (
	DefaultServerAddr = ":8082"
	DefaultUploadDir  = "./data/uploads"
)

// Route prefixes
const (
	UploadRoute = "/uploads"
)

// Timeout / size defaults
const (
	DefaultReadHeaderTimeout  = 30 * time.Second
	DefaultReadTimeout        = 5 * time.Minute
	DefaultIdleTimeout        = 120 * time.Second
	DefaultShutdownTimeout    = 25 * time.Second
	DefaultMaxHeaderBytes     = 1 << 20  // 1 MB
	DefaultMaxMultipartMemory = 32 << 20 // 32 MB
)

// Session defaults
const (
	DefaultSessionExpireDays   = 7
	DefaultSessionRandomKeyLen = 32
)

// Retention / scheduler intervals
const (
	DefaultLogRetentionInterval = 24 * time.Hour
)

// Environment variable keys
const (
	ENV_MODE               = "MODE"
	ENV_PROFILE_AUTO_PPROF = "PROFILE_AUTO_PPROF"
	ENV_UPLOAD_DIR         = "UPLOAD_DIR"
)

// Misc
const (
	DefaultBannerFile    = "banner.txt"
	SignalChannelBufSize = 1
)

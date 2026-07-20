package constants

import "time"

// Banner generation constants
const (
	DefaultBannerText    = "SoulNexus"
	BannerAPIURLTemplate = "https://patorjk.com/software/taag/ajax/convert.php?text=%s&font=doom"
	BannerRefererURL     = "https://patorjk.com/software/taag/"
	BannerAPITimeout     = 10 * time.Second
	BannerUserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	BannerAcceptHeader   = "text/plain, */*"
	DoomFontHeight       = 8
	DoomFontSpaceWidth   = 2
	DoomFontUnknownChar  = "? "
	BannerFilePerm       = 0644
)

// HTML detection substrings for banner API response validation
const (
	HTMLDoctypePrefix = "<!DOCTYPE"
	HTMLTagPrefix     = "<html"
	HTML404Error      = "404 Error"
	HTMLBrTag         = "<br>"
	HTMLBrSelfClose   = "<br/>"
	HTMLBrCloseSpace  = "<br />"
	ASCIIArtChars     = "|_/\\-="
)

// SQL / database initialization constants
const (
	SQLFileExtension       = ".sql"
	SQLStatementTerminator = ";"
	SQLCommentPrefix       = "--"
	SQLHashCommentPrefix   = "#"
	SQLScannerBufferSize   = 1024 * 1024
	DefaultMigrationsDir   = "cmd/bootstrap/migrations"
	SQLAlterTablePrefix    = "ALTER TABLE "
	SQLConvertUTF8MB4      = " CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
)

// System seed actor name
const (
	SystemActorSeed = "seed"
)

// Duplicate key error detection substrings (case-insensitive match)
const (
	ErrSubstrDuplicateEntry   = "duplicate entry"
	ErrSubstrUniqueConstraint = "unique constraint"
)

// JWKS key management defaults
const (
	DefaultMinJWKSKeys              = 3
	DefaultJWKSMinimumKeepOldKeys   = 2
	KeyDirPerm                      = 0700
	DefaultKeyRotationCheckInterval = 24 * time.Hour
	HoursPerDay                     = 24
)

// App-level environment variable keys
const (
	EnvAppEnv = "APP_ENV"
)

// ANSI terminal control codes
const (
	ANSIReset = "\x1b[0m"

	// Banner gradient (light blue → dark blue)
	ANSIBannerGradient1 = "\x1b[38;5;117m"
	ANSIBannerGradient2 = "\x1b[38;5;111m"
	ANSIBannerGradient3 = "\x1b[38;5;75m"
	ANSIBannerGradient4 = "\x1b[38;5;39m"
	ANSIBannerGradient5 = "\x1b[38;5;33m"
	ANSIBannerGradient6 = "\x1b[38;5;27m"
)

// Default site values for seed data
const (
	DefaultSiteURL         = "https://lingecho.com"
	DefaultSiteName        = "SoulNexus"
	DefaultSiteDescription = "SoulNexus - Intelligent Voice Customer Service Platform"
)

// Table names used in bootstrap (raw SQL in seeds)
const (
	TableMailTemplates = "mail_templates"
	TableMailLogs      = "mail_logs"
	TableSMSLogs       = "sms_logs"
)

package bootstrap

import (
	"bufio"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	knmodels "github.com/LingByte/SoulNexus/pkg/knowledge/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification/inbox"
	"github.com/LingByte/SoulNexus/pkg/notification/mail"
	"github.com/LingByte/SoulNexus/pkg/notification/sms"
	"github.com/LingByte/SoulNexus/pkg/task"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// migrationsDirEnv lets operators point goose at a specific on-disk migrations dir.
const migrationsDirEnv = "LINGECHO_MIGRATIONS_DIR"

// Options controls database initialization behavior
type Options struct {
	// InitSQLPath points to a .sql script file (optional); skip if empty
	InitSQLPath string
	// AutoMigrate forces GORM AutoMigrate (dev escape hatch, e.g. -automigrate).
	// AutoMigrate is the schema source of truth and also runs when MigrateSQL is set.
	AutoMigrate bool
	// MigrateSQL enables schema setup (-init): runs GORM AutoMigrate, then applies any
	// hand-written *.sql found in the on-disk migrations dir via goose (skipped if none).
	MigrateSQL bool
	// SeedNonProd whether to write default configuration in non-production environments (default true)
	SeedNonProd bool
}

// SetupDatabase unified entry: connect database -> migrate schema -> optional seed
func SetupDatabase(logWriter io.Writer, opts *Options) (*gorm.DB, error) {
	if opts == nil {
		opts = &Options{AutoMigrate: false, MigrateSQL: false, SeedNonProd: false}
	}

	// 1) Connect to database
	db, err := initDBConn(logWriter)
	if err != nil {
		logger.Error("init database failed", zap.Error(err))
		return nil, err
	}

	initPath := ResolveInitSQLPath(opts.InitSQLPath)

	// Repair snowflake IDs that overflowed into signed INTEGER (SQLite/MySQL) and
	// cannot be scanned into Go uint — must run before workers touch these rows.
	if err := knmodels.RepairKnowledgeSnowflakeNegativeIDs(db); err != nil {
		logger.Warn("repair knowledge snowflake ids failed", zap.Error(err))
	}
	if err := models.RepairAIInvocationLogNegativeIDs(db); err != nil {
		logger.Warn("repair ai_invocation_logs snowflake ids failed", zap.Error(err))
	}

	// 2) Schema.
	//    GORM AutoMigrate (models) is the single source of truth for the schema and runs
	//    whenever schema setup is requested (-init or -automigrate). Goose then applies any
	//    optional hand-written *.sql found in the on-disk migrations dir; if the dir is
	//    missing or empty it is skipped. This avoids AutoMigrate and goose both owning the
	//    full schema — AutoMigrate builds it, goose only layers incremental SQL patches.
	if opts.AutoMigrate || opts.MigrateSQL {
		if err := RunMigrations(db); err != nil {
			logger.Error("gorm automigrate failed", zap.Error(err))
			return nil, err
		}
		logger.Info("GORM AutoMigrate applied (models are the schema source of truth)")
	}
	if opts.MigrateSQL {
		if err := RunVersionedSQLMigrations(db); err != nil {
			logger.Error("versioned sql migration failed", zap.Error(err))
			return nil, err
		}
		logger.Info("schema migration success",
			zap.String("database", config.GlobalConfig.Database.Driver),
			zap.String("dsn", config.GlobalConfig.Database.DSN),
			zap.Bool("automigrate", opts.AutoMigrate || opts.MigrateSQL),
			zap.Bool("goose", opts.MigrateSQL),
		)
	}

	// 3) Permission catalog (required before init-sql role bindings; also runs with -seed)
	if opts.SeedNonProd || initPath != "" {
		if err := models.SyncPermissionCatalog(db); err != nil {
			logger.Error("sync permission catalog failed", zap.Error(err))
			return nil, err
		}
	}

	// 4) -init-sql ONLY: optional tenant/trunk dump (scripts/sql/init.sql). Separate from migrations/.
	if initPath != "" {
		logger.Info("running init sql", zap.String("path", initPath))
		if err := RunInitSQLFromPath(db, initPath); err != nil {
			logger.Error("run init sql failed", zap.String("path", initPath), zap.Error(err))
			return nil, err
		}
		if err := models.BackfillSystemTenantAdminPermissions(db, "init-sql"); err != nil {
			logger.Error("bind tenant admin permissions failed", zap.Error(err))
			return nil, err
		}
		logger.Info("tenant admin permissions backfilled from permission catalog")
	}

	// 5) Non-production: site config + default platform admin (+ re-backfill if seed runs after init-sql)
	if opts.SeedNonProd {
		service := SeedService{db: db}
		if err := service.SeedAll(); err != nil {
			logger.Error("seed failed", zap.Error(err))
			return nil, err
		}
	}

	logger.Info("system bootstrap - database is initialization complete")
	return db, nil
}

// ConnectDatabase opens *gorm.DB from global configuration (no migration/seed).
func ConnectDatabase(logWriter io.Writer) (*gorm.DB, error) {
	return initDBConn(logWriter)
}

// initDBConn creates *gorm.DB based on global configuration
func initDBConn(logWriter io.Writer) (*gorm.DB, error) {
	dbDriver := config.GlobalConfig.Database.Driver
	dsn := config.GlobalConfig.Database.DSN
	return utils.InitDatabase(logWriter, dbDriver, dsn)
}

// RunInitSQL executes SQL statements from a local .sql file segment by segment (split by semicolon ;), idempotent scripts should use IF NOT EXISTS in SQL for protection
func RunInitSQL(db *gorm.DB, sqlFilePath string) error {
	f, err := os.Open(sqlFilePath)
	if err != nil {
		return err
	}
	defer f.Close()
	var (
		sb      strings.Builder
		scanner = bufio.NewScanner(f)
	)
	// Relax token limit (long lines)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		// Ignore comment lines (starting with --) and empty lines
		if trim == "" || strings.HasPrefix(trim, "--") || strings.HasPrefix(trim, "#") {
			continue
		}
		sb.WriteString(line)
		sb.WriteString("\n")
		// Use ; as statement terminator (simple splitting, suitable for most scenarios)
		if strings.HasSuffix(trim, ";") {
			stmt := strings.TrimSpace(sb.String())
			sb.Reset()
			if stmt != "" {
				if err := db.Exec(stmt).Error; err != nil {
					return err
				}
			}
		}
	}
	// Handle remaining content at end of file without semicolon
	rest := strings.TrimSpace(sb.String())
	if rest != "" {
		if err := db.Exec(rest).Error; err != nil {
			return err
		}
	}
	return scanner.Err()
}

// runPostMigrateSQL runs plain *.sql files from a directory (legacy/tests).
// Prefer RunVersionedSQLMigrations (goose) for production.
func runPostMigrateSQL(db *gorm.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		filePath := filepath.Join(migrationsDir, name)
		logger.Info("executing migration script", zap.String("file", filePath))
		if err := RunInitSQL(db, filePath); err != nil {
			logger.Error("migration script failed", zap.String("file", filePath), zap.Error(err))
			return err
		}
	}

	return nil
}

// RunVersionedSQLMigrations applies hand-written goose migrations read from an on-disk
// migrations directory (no embedding). If no directory with *.sql files is found, it is a
// no-op — GORM AutoMigrate already owns the schema. Safe to re-run (goose_db_version).
func RunVersionedSQLMigrations(db *gorm.DB) error {
	if db == nil {
		return apperror.ErrDBNil
	}
	dir := resolveMigrationsDir()
	if dir == "" {
		logger.Info("no on-disk SQL migrations found; skipping goose (GORM AutoMigrate owns schema)")
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := applyGooseDir(sqlDB, config.GlobalConfig.Database.Driver, dir); err != nil {
		return err
	}
	logger.Info("goose migrations applied",
		zap.String("dir", dir),
		zap.String("dialect", gooseDialect(config.GlobalConfig.Database.Driver)),
	)
	return nil
}

// applyGooseDir runs `goose up` against a real filesystem directory.
func applyGooseDir(sqlDB *sql.DB, driver, dir string) error {
	if err := goose.SetDialect(gooseDialect(driver)); err != nil {
		return err
	}
	goose.SetBaseFS(os.DirFS(dir))
	defer goose.SetBaseFS(nil)
	return goose.Up(sqlDB, ".")
}

// resolveMigrationsDir returns the first directory that actually contains *.sql migrations,
// or "" when none exists. Honors the LINGECHO_MIGRATIONS_DIR override first.
func resolveMigrationsDir() string {
	if v := strings.TrimSpace(os.Getenv(migrationsDirEnv)); v != "" {
		if dirHasSQL(v) {
			return v
		}
		return ""
	}
	candidates := []string{
		"cmd/bootstrap/migrations", // running from repo root (dev)
		"migrations",               // running next to the binary / from cmd/bootstrap
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "migrations"))
	}
	for _, c := range candidates {
		if dirHasSQL(c) {
			return c
		}
	}
	return ""
}

// dirHasSQL reports whether dir exists and contains at least one *.sql file.
func dirHasSQL(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), pkgconst.SQLFileExtension) {
			return true
		}
	}
	return false
}

func gooseDialect(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres", "postgresql", "pg":
		return "postgres"
	case "mysql":
		return "mysql"
	default:
		return "sqlite3"
	}
}

// RunMigrations executes entity migration
func RunMigrations(db *gorm.DB) error {
	if db == nil {
		return apperror.ErrDBNil
	}
	if err := utils.MakeMigrates(db, []any{
		&utils.Config{},
		&models.Tenant{},
		&models.TenantGroup{},
		&models.TenantUser{},
		&models.TenantUserGroup{},
		&models.Permission{},
		&models.TenantRole{},
		&models.TenantRolePermission{},
		&models.TenantUserRole{},
		&models.Credential{},
		&models.PlatformAdmin{},
		&models.OperationLog{},
		&knmodels.KnowledgeNamespace{},
		&knmodels.KnowledgeDocument{},
		&knmodels.KnowledgeChunk{},
		&knmodels.KnowledgeUnansweredQuestion{},
		&knmodels.KnowledgeAnsweredQuestion{},
		&knmodels.KnowledgeTypicalQuestion{},
		&knmodels.KnowledgeTypicalQuestionStat{},
		&knmodels.KnowledgeSyncSource{},
		&knmodels.KnowledgeEvalDataset{},
		&models.TenantBill{},
		&models.TenantUsageEvent{},
		&models.BillingPlan{},
		&models.TenantWebhook{},
		&models.TenantWebhookDelivery{},
		&models.LoginHistory{},
		&task.ExecutionTask{},
		&models.Assistant{},
		&models.AssistantMember{},
		&models.AssistantVersion{},
		&models.JSTemplate{},
		&models.JSTemplateUsageLog{},
		&models.WorkflowDefinition{},
		&models.WorkflowInstance{},
		&models.WorkflowVersion{},
		&models.WorkflowPlugin{},
		&models.WorkflowPluginVersion{},
		&models.WorkflowPluginReview{},
		&models.WorkflowPluginInstallation{},
		&models.NodePlugin{},
		&models.NodePluginVersion{},
		&models.NodePluginReview{},
		&models.NodePluginInstallation{},
		&models.VoiceCloneProfile{},
		&models.TenantNluModel{},
		&models.TenantAssistantTool{},
		&models.McpMarketItem{},
		&models.VoiceprintProfile{},
		&models.SpeakerSubject{},
		&models.SpeakerAttribute{},
		&models.SpeakerCredentialRef{},
		&models.VoiceSynthesisHistory{},
		&models.NotificationChannel{},
		&models.TenantIMChannel{},
		&models.DialogConversation{},
		&models.DialogMessage{},
		&models.TenantDialogChannel{},
		&models.TenantDialogSkill{},
		&models.MailTemplate{},
		&models.TenantCallStatsDaily{},
		&models.TenantStats{},
		&models.UserDevice{},
		&inbox.Message{},
		&mail.MailLog{},
		&sms.SMSLog{},
		&models.AIInvocationLog{},
		&models.AIProviderPool{},
		&models.TenantAIPoolGrant{},
	}); err != nil {
		return err
	}
	return nil
}

// ResolveInitSQLPath returns the path only when the operator passed -init-sql explicitly.
// An existing scripts/sql/init.sql on disk does NOT trigger execution by itself.
func ResolveInitSQLPath(flagPath string) string {
	return strings.TrimSpace(flagPath)
}

// RunInitSQLFromPath executes one .sql file or all .sql files in a directory (sorted by name).
func RunInitSQLFromPath(db *gorm.DB, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return RunInitSQL(db, path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), pkgconst.SQLFileExtension) {
			continue
		}
		files = append(files, filepath.Join(path, e.Name()))
	}
	sort.Strings(files)
	for _, f := range files {
		if err := RunInitSQL(db, f); err != nil {
			return err
		}
	}
	return nil
}

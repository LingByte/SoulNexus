package backup

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// StartBackupScheduler starts the backup scheduler
func StartBackupScheduler() {
	c := cron.New()

	// Use Cron expression from configuration
	schedule := config.GlobalConfig.BackupSchedule

	// Add scheduled task
	c.AddFunc(schedule, func() {
		err := ExecuteBackup()
		if err != nil {
			logger.Warn("Backup failed: %v", zap.Error(err))
		} else {
			logger.Info("Backup completed successfully")
		}
	})

	// Start the scheduler
	c.Start()
}

// ExecuteBackup executes database backup according to configuration
func ExecuteBackup() error {
	switch config.GlobalConfig.DBDriver {
	case "sqlite":
		// Execute SQLite backup
		dst := filepath.Join(config.GlobalConfig.BackupPath, fmt.Sprintf("sys_backup_%s.db", time.Now().Format("20060102_150405")))
		return BackupSQLiteDatabase(config.GlobalConfig.DSN, dst)
	case "mysql":
		// Execute MySQL backup
		dst := filepath.Join(config.GlobalConfig.BackupPath, fmt.Sprintf("sys_backup_%s.sql", time.Now().Format("20060102_150405")))
		return BackupMySQLDatabase(config.GlobalConfig.DSN, dst)
	default:
		return fmt.Errorf("unsupported DB_DRIVER: %s", config.GlobalConfig.DBDriver)
	}
}

// BackupSQLiteDatabase performs backup of SQLite database
func BackupSQLiteDatabase(src string, dst string) error {
	// Ensure destination path exists
	backupDir := filepath.Dir(dst)
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		err := os.MkdirAll(backupDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create backup directory: %v", err)
		}
	}

	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %v", err)
	}
	defer destFile.Close()

	// Copy data
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("error copying data: %v", err)
	}

	log.Printf("SQLite database backup completed: %s", dst)
	return nil
}

// BackupMySQLDatabase performs backup of MySQL database
func BackupMySQLDatabase(dsn, dst string) error {
	// Ensure destination path exists
	backupDir := filepath.Dir(dst)
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		err := os.MkdirAll(backupDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create backup directory: %v", err)
		}
	}

	// Use mysqldump to perform backup
	cmd := exec.Command("mysqldump", dsn, ">", dst)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to backup MySQL database: %v", err)
	}

	log.Printf("MySQL database backup completed: %s", dst)
	return nil
}

package backup

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/stores"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultBackupPrefix = "backup"
	manifestFileName    = "backup_manifest.json"
)

// BackupRecord 记录一次备份的元数据
type BackupRecord struct {
	Key      string `json:"key"`       // store 中的对象 key
	FileName string `json:"file_name"` // 原始文件名
	Size     int64  `json:"size"`      // 字节数
	Date     string `json:"date"`      // ISO 8601 时间
	URL      string `json:"url"`       // 完整访问 URL
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

// BackupManifest 本地备份清单，用于追踪已上传的备份并支持过期清理
type BackupManifest struct {
	mu      sync.Mutex
	path    string
	Records []BackupRecord `json:"records"`
}

// loadManifest 从本地文件加载清单
func loadManifest(path string) (*BackupManifest, error) {
	m := &BackupManifest{
		path:    path,
		Records: make([]BackupRecord, 0),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, fmt.Errorf("read manifest file %s: %w", path, err)
	}
	if len(data) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(data, &m.Records); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return m, nil
}

// save 写入清单到本地文件
func (m *BackupManifest) save() error {
	if m.path == "" {
		return fmt.Errorf("manifest path is empty")
	}
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}
	data, err := json.MarshalIndent(m.Records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return os.WriteFile(m.path, data, 0644)
}

// addRecord 追加一条备份记录并持久化
func (m *BackupManifest) addRecord(record BackupRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Records = append(m.Records, record)
	return m.save()
}

// removeRecords 从清单中删除指定记录并持久化
func (m *BackupManifest) removeRecords(indices map[int]bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := make([]BackupRecord, 0, len(m.Records))
	for i, r := range m.Records {
		if !indices[i] {
			filtered = append(filtered, r)
		}
	}
	m.Records = filtered
	return m.save()
}

// manifestPath 生成清单文件的本地路径
func manifestPath() string {
	return filepath.Join(utils.GetEnv(constants.ENV_BACKUP_PATH), manifestFileName)
}

// retentionDays 获取备份保留天数
func retentionDays() int {
	return int(utils.GetIntEnv(constants.ENV_BACKUP_RETENTION_DAYS))
}

// backupKey 生成对象存储中的 key，格式: backup/sys_backup_20260623_150405.sql
func backupKey(ext string) string {
	ts := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s/sys_backup_%s.%s", defaultBackupPrefix, ts, ext)
}

// recordBackupManifest 记录备份到本地清单文件
func recordBackupManifest(key string, size int64, store stores.Store) error {
	mpath := manifestPath()
	manifest, err := loadManifest(mpath)
	if err != nil {
		return err
	}

	record := BackupRecord{
		Key:      key,
		FileName: filepath.Base(key),
		Size:     size,
		Date:     time.Now().UTC().Format(time.RFC3339),
		URL:      store.PublicURL(key),
	}
	if err := manifest.addRecord(record); err != nil {
		return fmt.Errorf("save backup manifest: %w", err)
	}
	logger.Info("Backup uploaded to store",
		zap.String("key", key),
		zap.Int64("size", size),
	)
	return nil
}

// StartBackupScheduler 启动备份调度器，启动时立即执行一次备份
func StartBackupScheduler(db *gorm.DB, dbConfig DatabaseConfig) {
	c := cron.New()
	store := stores.Default()

	go purgeExpiredBackupsSafe(store)

	// 定时备份
	c.AddFunc(utils.GetEnv(constants.ENV_BACKUP_SCHEDULE), func() {
		if err := ExecuteBackup(db, store, dbConfig); err != nil {
			logger.Warn("Scheduled backup failed", zap.Error(err))
		} else {
			logger.Info("Scheduled backup completed successfully")
		}
	})

	// 定时清理过期备份（每6小时执行一次）
	c.AddFunc("0 */6 * * *", func() {
		purgeExpiredBackupsSafe(store)
	})

	c.Start()
}

// purgeExpiredBackupsSafe 带 panic recover 的过期备份清理
func purgeExpiredBackupsSafe(store stores.Store) {
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("backup purge panic recovered", zap.Any("panic", r))
		}
	}()
	if err := purgeExpiredBackups(store); err != nil {
		logger.Warn("Backup purge failed", zap.Error(err))
	}
}

// purgeExpiredBackups 清理超过保留天数的备份对象并在清单中移除记录
func purgeExpiredBackups(store stores.Store) error {
	mpath := manifestPath()
	manifest, err := loadManifest(mpath)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	retention := retentionDays()
	if retention <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -retention)

	var toRemove map[int]bool
	for i, rec := range manifest.Records {
		date, parseErr := time.Parse(time.RFC3339, rec.Date)
		if parseErr != nil {
			date, parseErr = time.Parse("2006-01-02T15:04:05Z", rec.Date)
		}
		if parseErr != nil {
			logger.Warn("Cannot parse backup record date, will remove it",
				zap.String("key", rec.Key),
				zap.String("date", rec.Date),
			)
			if toRemove == nil {
				toRemove = make(map[int]bool)
			}
			toRemove[i] = true
			continue
		}
		if date.Before(cutoff) {
			if toRemove == nil {
				toRemove = make(map[int]bool)
			}
			toRemove[i] = true
		}
	}

	if len(toRemove) == 0 {
		return nil
	}

	removed := 0
	for i := range toRemove {
		rec := manifest.Records[i]
		if err := store.Delete(rec.Key); err != nil {
			logger.Warn("Failed to delete expired backup from store",
				zap.String("key", rec.Key),
				zap.Error(err),
			)
		}
		removed++
		logger.Info("Expired backup deleted",
			zap.String("key", rec.Key),
			zap.String("date", rec.Date),
		)
	}

	if err := manifest.removeRecords(toRemove); err != nil {
		return fmt.Errorf("update manifest after purge: %w", err)
	}

	logger.Info("Backup purge completed",
		zap.Int("removed", removed),
		zap.Int("retention_days", retention),
	)
	return nil
}

// ExecuteBackup 执行数据库备份并上传到对象存储
func ExecuteBackup(db *gorm.DB, store stores.Store, dbConfig DatabaseConfig) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	if store == nil {
		return fmt.Errorf("store is nil")
	}
	switch dbConfig.Driver {
	case constants.DBDriverSQLite:
		return backupSQLiteToStore(dbConfig.DSN, store)
	case constants.DBDriverMySQL:
		return backupMySQLToStore(db, store)
	case constants.DBDriverPG:
		return backupPostgresToStore(db, store)
	default:
		return fmt.Errorf("unsupported DB_DRIVER: %s", dbConfig.Driver)
	}
}

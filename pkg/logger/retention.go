package logger

import (
	"os"
	"path/filepath"

	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"go.uber.org/zap"
)

// PurgeExpiredLogFiles removes regular files under logDir older than retentionDays.
func PurgeExpiredLogFiles(logDir string, retentionDays int) (removed int, err error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	info, err := os.Stat(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return 0, nil
	}

	cutoff := timeutil.Now().In(timeutil.Location()).AddDate(0, 0, -retentionDays)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(logDir, entry.Name())
		fi, statErr := entry.Info()
		if statErr != nil {
			if Lg != nil {
				Lg.Warn("log retention stat failed", zap.String("path", path), zap.Error(statErr))
			}
			continue
		}
		if fi.ModTime().After(cutoff) {
			continue
		}
		if rmErr := os.Remove(path); rmErr != nil {
			if Lg != nil {
				Lg.Warn("log retention remove failed", zap.String("path", path), zap.Error(rmErr))
			}
			continue
		}
		removed++
		if Lg != nil {
			Lg.Info("log retention removed expired file",
				zap.String("path", path),
				zap.Time("modified", fi.ModTime()),
			)
		}
	}
	return removed, nil
}

// LogDirFromFilename returns the directory containing log files derived from LOG_FILENAME.
func LogDirFromFilename(filename string) string {
	if filename == "" {
		return "./logs"
	}
	return filepath.Dir(filename)
}

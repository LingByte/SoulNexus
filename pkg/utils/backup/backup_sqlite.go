package backup

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/stores"
)

// backupSQLiteToStore 备份 SQLite 数据库到对象存储（直接复制 db 文件）
func backupSQLiteToStore(dsn string, store stores.Store) error {
	srcPath := strings.TrimPrefix(dsn, "file:")
	srcPath = strings.TrimSuffix(srcPath, "?cache=shared")
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	key := backupKey("db")
	if err := store.Write(key, bytes.NewReader(data)); err != nil {
		return err
	}
	return recordBackupManifest(key, int64(len(data)), store)
}

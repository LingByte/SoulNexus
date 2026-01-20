//go:build !mysql && !pg

package utils

import (
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func createDatabaseInstance(cfg *gorm.Config, driver, dsn string) (*gorm.DB, error) {
	switch driver {
	case "mysql":
		db, err := gorm.Open(mysql.Open(dsn), cfg)
		if err != nil {
			return nil, err
		}

		// 设置 MySQL 字符集和排序规则，解决 utf8mb4_0900_ai_ci 和 utf8mb3_general_ci 不匹配问题
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}

		// 执行 SET NAMES 确保使用 utf8mb4 字符集
		_, err = sqlDB.Exec("SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci")
		if err != nil {
			// 如果执行失败，记录错误但不阻止连接
			// 因为某些 MySQL 版本可能不支持这个语法
			// 尝试使用更兼容的方式
			_, _ = sqlDB.Exec("SET NAMES utf8mb4")
		}

		return db, nil
	case "pg":
		return gorm.Open(postgres.Open(dsn), cfg)
	}
	if dsn == "" {
		dsn = "file::memory:"
	}
	return gorm.Open(sqlite.Open(dsn), cfg)
}

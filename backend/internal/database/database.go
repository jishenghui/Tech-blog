package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"blog-api/internal/config"
)

// Open 打开 SQLite（WAL + 外键 + busy_timeout），执行迁移并返回 GORM 实例。
func Open(cfg *config.Config) (*gorm.DB, error) {
	if dir := filepath.Dir(cfg.DBPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败: %w", err)
		}
	}

	// _pragma 由 modernc.org/sqlite 逐连接应用，避免连接池内行为不一致。
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
		filepath.ToSlash(cfg.DBPath),
	)

	logLevel := logger.Info
	if cfg.IsProduction() {
		logLevel = logger.Warn
	}
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	var journalMode string
	if err := db.Raw("PRAGMA journal_mode").Row().Scan(&journalMode); err != nil {
		return nil, fmt.Errorf("读取 journal_mode 失败: %w", err)
	}
	if journalMode != "wal" {
		return nil, fmt.Errorf("SQLite 未启用 WAL 模式，当前为 %q", journalMode)
	}

	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Printf("SQLite 已打开: %s（journal_mode=%s）", cfg.DBPath, journalMode)
	return db, nil
}

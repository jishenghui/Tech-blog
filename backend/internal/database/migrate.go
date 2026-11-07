package database

import (
	"gorm.io/gorm"

	"blog-api/internal/model"
)

// AutoMigrate 建表并补充 GORM 不便表达的索引（软删除 + 部分唯一索引）。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Directory{},
		&model.Article{},
		&model.Tag{},
		&model.ArticleTag{},
		&model.Asset{},
		&model.Project{},
		&model.Profile{},
		&model.Todo{},
		&model.ImportJob{},
		&model.Setting{},
	); err != nil {
		return err
	}

	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_directories_slug ON directories(slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_directories_parent ON directories(parent_id, sort_order)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_articles_slug ON articles(slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_articles_dir ON articles(directory_id, status, published_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uk_projects_slug ON projects(slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_assets_sha ON assets(sha256)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_article ON assets(article_id)`,
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

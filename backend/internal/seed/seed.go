package seed

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/model"
)

type fixedDirectory struct {
	Name      string
	Slug      string
	Kind      string
	PageKey   string
	SortOrder int
}

// 固定一级目录。home/projects/tech/todos 为 page 类型、无二级目录；
// other 为 container 类型，用于承接解散后的目录。
var fixedDirectories = []fixedDirectory{
	{Name: "首页", Slug: "home", Kind: "page", PageKey: "home", SortOrder: 0},
	{Name: "项目", Slug: "projects", Kind: "page", PageKey: "projects", SortOrder: 1},
	{Name: "技术栈", Slug: "tech", Kind: "page", PageKey: "tech", SortOrder: 2},
	{Name: "待办", Slug: "todos", Kind: "page", PageKey: "todos", SortOrder: 3},
	{Name: "其他", Slug: "other", Kind: "container", PageKey: "other", SortOrder: 4},
}

// Run 创建唯一管理员（首次启动）与固定一级目录，整体在一个事务内完成。
func Run(db *gorm.DB, cfg *config.Config) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedAdmin(tx, cfg); err != nil {
			return err
		}
		return seedDirectories(tx)
	})
}

func seedAdmin(tx *gorm.DB, cfg *config.Config) error {
	var count int64
	if err := tx.Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("统计管理员数量失败: %w", err)
	}
	if count > 0 {
		return nil
	}

	if cfg.AdminPasswordHash == "" {
		return errors.New("首次启动需要设置 ADMIN_PASSWORD_HASH；可用 `go run . -hash-password 你的密码` 生成 bcrypt 哈希")
	}
	if err := auth.ValidateHash(cfg.AdminPasswordHash); err != nil {
		return fmt.Errorf("ADMIN_PASSWORD_HASH 不是合法的 bcrypt 哈希: %w", err)
	}

	user := model.User{
		Username:     cfg.AdminUsername,
		PasswordHash: cfg.AdminPasswordHash,
		Nickname:     "管理员",
	}
	if err := tx.Create(&user).Error; err != nil {
		return fmt.Errorf("创建管理员失败: %w", err)
	}
	return nil
}

func seedDirectories(tx *gorm.DB) error {
	for _, fixed := range fixedDirectories {
		pageKey := fixed.PageKey

		var dir model.Directory
		err := tx.Unscoped().Where("page_key = ?", fixed.PageKey).First(&dir).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			dir = model.Directory{
				Name:      fixed.Name,
				Slug:      fixed.Slug,
				Kind:      fixed.Kind,
				PageKey:   &pageKey,
				IsFixed:   true,
				Visible:   true,
				SortOrder: fixed.SortOrder,
			}
			if err := tx.Create(&dir).Error; err != nil {
				return fmt.Errorf("创建固定目录 %s 失败: %w", fixed.Name, err)
			}
		case err != nil:
			return fmt.Errorf("查询固定目录 %s 失败: %w", fixed.Name, err)
		case dir.DeletedAt.Valid:
			if err := tx.Unscoped().Model(&dir).Update("deleted_at", nil).Error; err != nil {
				return fmt.Errorf("恢复固定目录 %s 失败: %w", fixed.Name, err)
			}
		}
	}
	return nil
}

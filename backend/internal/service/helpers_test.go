package service_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/database"
	"blog-api/internal/model"
	"blog-api/internal/seed"
	"blog-api/internal/service"
)

type fakeUploader struct {
	base     string
	calls    int
	lastKey  string
	lastMime string
}

func (f *fakeUploader) Upload(_ context.Context, key string, _ []byte, mime string) (string, error) {
	f.calls++
	f.lastKey = key
	f.lastMime = mime
	return f.base + "/" + key, nil
}

type testEnv struct {
	db       *gorm.DB
	dirs     *service.DirectoryService
	articles *service.ArticleService
	tags     *service.TagService
	assets   *service.AssetService
	imports  *service.ImportService
	projects *service.ProjectService
	profiles *service.ProfileService
	todos    *service.TodoService
	uploader *fakeUploader
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	hash, err := auth.HashPassword("test-password")
	if err != nil {
		t.Fatalf("生成测试密码哈希失败: %v", err)
	}

	cfg := &config.Config{
		AppEnv:             config.EnvProduction,
		HTTPAddr:           "127.0.0.1:0",
		DBPath:             filepath.Join(t.TempDir(), "test.db"),
		JWTSecret:          "test-secret-at-least-32-characters-long",
		JWTAccessTTL:       time.Hour,
		JWTRefreshTTL:      24 * time.Hour,
		AdminUsername:      "admin",
		AdminPasswordHash:  hash,
		SiteURL:            "https://example.com",
		APIBaseURL:         "https://api.example.com",
		CORSAllowedOrigins: []string{"https://example.com"},
	}

	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取测试数据库连接池失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := seed.Run(db, cfg); err != nil {
		t.Fatalf("初始化种子数据失败: %v", err)
	}

	renderer := service.NewMarkdownRenderer()
	articleService := service.NewArticleService(db, renderer)
	uploader := &fakeUploader{base: "https://cdn.test"}
	assetService := service.NewAssetService(db, uploader, "test-bucket")
	importService := service.NewImportService(db, assetService, articleService)

	return &testEnv{
		db:       db,
		dirs:     service.NewDirectoryService(db),
		articles: articleService,
		tags:     service.NewTagService(db),
		assets:   assetService,
		imports:  importService,
		projects: service.NewProjectService(db, renderer),
		profiles: service.NewProfileService(db, renderer),
		todos:    service.NewTodoService(db),
		uploader: uploader,
	}
}

func (e *testEnv) fixedDir(t *testing.T, pageKey string) *model.Directory {
	t.Helper()
	var dir model.Directory
	if err := e.db.Where("page_key = ?", pageKey).First(&dir).Error; err != nil {
		t.Fatalf("查询固定目录 %s 失败: %v", pageKey, err)
	}
	return &dir
}

func (e *testEnv) createTopLevel(t *testing.T, name, slug string) *model.Directory {
	t.Helper()
	dir, err := e.dirs.Create(service.CreateDirectoryInput{Name: name, Slug: slug})
	if err != nil {
		t.Fatalf("创建一级目录 %s 失败: %v", name, err)
	}
	return dir
}

func (e *testEnv) createSecondLevel(t *testing.T, parentID uint, name, slug string) *model.Directory {
	t.Helper()
	dir, err := e.dirs.Create(service.CreateDirectoryInput{ParentID: &parentID, Name: name, Slug: slug})
	if err != nil {
		t.Fatalf("创建二级目录 %s 失败: %v", name, err)
	}
	return dir
}

func (e *testEnv) createTag(t *testing.T, name, slug string) *model.Tag {
	t.Helper()
	tag, err := e.tags.Create(service.CreateTagInput{Name: name, Slug: slug})
	if err != nil {
		t.Fatalf("创建标签 %s 失败: %v", name, err)
	}
	return tag
}

func (e *testEnv) createArticle(t *testing.T, directoryID uint, title, content string, tagIDs []uint) *model.Article {
	t.Helper()
	article, err := e.articles.Create(service.CreateArticleInput{
		DirectoryID: directoryID,
		Title:       title,
		ContentMD:   content,
		TagIDs:      tagIDs,
	})
	if err != nil {
		t.Fatalf("创建文章 %s 失败: %v", title, err)
	}
	return article
}

func (e *testEnv) reloadDirectory(t *testing.T, id uint) *model.Directory {
	t.Helper()
	var dir model.Directory
	if err := e.db.First(&dir, id).Error; err != nil {
		t.Fatalf("重新查询目录 %d 失败: %v", id, err)
	}
	return &dir
}

func (e *testEnv) reloadArticle(t *testing.T, id uint) *model.Article {
	t.Helper()
	var article model.Article
	if err := e.db.First(&article, id).Error; err != nil {
		t.Fatalf("重新查询文章 %d 失败: %v", id, err)
	}
	return &article
}

func (e *testEnv) reloadTag(t *testing.T, id uint) *model.Tag {
	t.Helper()
	var tag model.Tag
	if err := e.db.First(&tag, id).Error; err != nil {
		t.Fatalf("重新查询标签 %d 失败: %v", id, err)
	}
	return &tag
}

func directoryCount(t *testing.T, db *gorm.DB, id uint) int64 {
	t.Helper()
	var dir model.Directory
	// 解散后的目录是软删除的，但仍需要检查其残留计数。
	if err := db.Unscoped().First(&dir, id).Error; err != nil {
		t.Fatalf("查询目录 %d 失败: %v", id, err)
	}
	return dir.ArticleCount
}

func tagCount(t *testing.T, db *gorm.DB, id uint) int64 {
	t.Helper()
	var tag model.Tag
	if err := db.First(&tag, id).Error; err != nil {
		t.Fatalf("查询标签 %d 失败: %v", id, err)
	}
	return tag.ArticleCount
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func requireErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("期望错误包含 %q，实际为 nil", substr)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("错误信息不包含 %q: %v", substr, err)
	}
}

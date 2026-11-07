package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/database"
	"blog-api/internal/router"
	"blog-api/internal/seed"
)

func TestM2DirectoryArticleFlow(t *testing.T) {
	server, token := newM2TestServer(t)
	client := &http.Client{}

	// 创建一级目录、二级目录、标签。
	status, env := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/directories", map[string]any{
		"name": "学习",
		"slug": "study",
	}, token)
	requireOK(t, status, env, "创建一级目录")
	var study struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &study); err != nil || study.ID == 0 {
		t.Fatalf("解析一级目录失败: err=%v data=%s", err, env.Data)
	}

	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/directories", map[string]any{
		"parent_id": study.ID,
		"name":      "Go",
		"slug":      "go",
	}, token)
	requireOK(t, status, env, "创建二级目录")
	var goDir struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &goDir); err != nil || goDir.ID == 0 {
		t.Fatalf("解析二级目录失败: err=%v data=%s", err, env.Data)
	}

	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/tags", map[string]any{
		"name": "Go语言",
		"slug": "golang",
	}, token)
	requireOK(t, status, env, "创建标签")
	var tag struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &tag); err != nil || tag.ID == 0 {
		t.Fatalf("解析标签失败: err=%v data=%s", err, env.Data)
	}

	// 创建并发布文章。
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/articles", map[string]any{
		"directory_id": goDir.ID,
		"title":        "并发模型",
		"content_md":   "# 并发模型\n\nGoroutine 调度。\n\n```go\nfmt.Println(\"hi\")\n```\n",
		"tag_ids":      []uint{tag.ID},
	}, token)
	requireOK(t, status, env, "创建文章")
	var article struct {
		ID   uint   `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(env.Data, &article); err != nil || article.ID == 0 {
		t.Fatalf("解析文章失败: err=%v data=%s", err, env.Data)
	}

	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/articles/"+itoa(article.ID)+"/publish", nil, token)
	requireOK(t, status, env, "发布文章")

	// 公开导航包含 study/go。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/nav", nil, "")
	requireOK(t, status, env, "公开导航")
	if !strings.Contains(string(env.Data), `"slug":"study"`) || !strings.Contains(string(env.Data), `"slug":"go"`) {
		t.Fatalf("公开导航缺少目录: %s", env.Data)
	}

	// 公开目录详情。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/directories/go", nil, "")
	requireOK(t, status, env, "公开目录详情")
	var dirPage struct {
		Directory struct {
			Slug string `json:"slug"`
		} `json:"directory"`
		Items []struct {
			ID uint `json:"id"`
		} `json:"items"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(env.Data, &dirPage); err != nil {
		t.Fatalf("解析目录详情失败: %v", err)
	}
	if dirPage.Directory.Slug != "go" || dirPage.Total != 1 || len(dirPage.Items) != 1 {
		t.Fatalf("目录详情不正确: %+v", dirPage)
	}

	// 公开文章详情（中文 slug 需要 URL 编码）。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/articles/"+url.PathEscape(article.Slug), nil, "")
	requireOK(t, status, env, "公开文章详情")
	var publicDetail struct {
		ContentHTML string `json:"content_html"`
	}
	if err := json.Unmarshal(env.Data, &publicDetail); err != nil {
		t.Fatalf("解析公开文章详情失败: %v", err)
	}
	if !strings.Contains(publicDetail.ContentHTML, `class="chroma"`) {
		t.Fatalf("文章详情缺少代码高亮: %s", publicDetail.ContentHTML)
	}

	// 标签列表与标签文章。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/tags", nil, "")
	requireOK(t, status, env, "公开标签列表")
	if !strings.Contains(string(env.Data), `"article_count":1`) {
		t.Fatalf("标签计数不正确: %s", env.Data)
	}
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/tags/golang/articles", nil, "")
	requireOK(t, status, env, "标签文章列表")
	if !strings.Contains(string(env.Data), `"total":1`) {
		t.Fatalf("标签文章数量不正确: %s", env.Data)
	}

	// 搜索。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/search?q=goroutine", nil, "")
	requireOK(t, status, env, "搜索")
	if !strings.Contains(string(env.Data), `"total":1`) {
		t.Fatalf("搜索结果不正确: %s", env.Data)
	}

	// 后台目录树应显示 Go 的已发布计数。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/directories", nil, token)
	requireOK(t, status, env, "后台目录树")
	var tree []struct {
		ID       uint   `json:"id"`
		Slug     string `json:"slug"`
		Children []struct {
			ID           uint   `json:"id"`
			Slug         string `json:"slug"`
			ArticleCount int64  `json:"article_count"`
			DraftCount   int64  `json:"draft_count"`
		} `json:"children"`
	}
	if err := json.Unmarshal(env.Data, &tree); err != nil {
		t.Fatalf("解析后台目录树失败: %v", err)
	}
	var found bool
	for _, node := range tree {
		for _, child := range node.Children {
			if child.Slug == "go" {
				found = true
				if child.ArticleCount != 1 || child.DraftCount != 0 {
					t.Fatalf("Go 目录计数不正确: %+v", child)
				}
			}
		}
	}
	if !found {
		t.Fatal("后台目录树缺少 Go 目录")
	}

	// 解散二级目录，文章应进入 other-other。
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/directories/"+itoa(goDir.ID)+"/dissolve", nil, token)
	requireOK(t, status, env, "解散二级目录")

	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/directories/other-other", nil, "")
	requireOK(t, status, env, "兜底目录详情")
	if !strings.Contains(string(env.Data), `"total":1`) {
		t.Fatalf("文章未进入 other-other: %s", env.Data)
	}

	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/directories/go", nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("已解散目录应返回 404，实际 %d", status)
	}

	// 全量重算。
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/maintenance/recalc", nil, token)
	requireOK(t, status, env, "全量重算")
	if !strings.Contains(string(env.Data), `"articles":1`) {
		t.Fatalf("重算结果不正确: %s", env.Data)
	}

	// 取消发布后公开详情应 404。
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/articles/"+itoa(article.ID)+"/unpublish", nil, token)
	requireOK(t, status, env, "取消发布")
	status, _ = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/articles/"+url.PathEscape(article.Slug), nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("未发布文章应返回 404，实际 %d", status)
	}

	// 删除文章后后台详情应 404。
	status, env = doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/articles/"+itoa(article.ID), nil, token)
	requireOK(t, status, env, "删除文章")
	status, _ = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/articles/"+itoa(article.ID), nil, token)
	if status != http.StatusNotFound {
		t.Fatalf("已删除文章应返回 404，实际 %d", status)
	}
}

func newM2TestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	hash, err := auth.HashPassword("admin123456")
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

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	server := httptest.NewServer(router.New(cfg, db, tokens))
	t.Cleanup(server.Close)

	client := &http.Client{}
	status, env := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "admin123456",
	}, "")
	requireOK(t, status, env, "登录")
	var loginData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(env.Data, &loginData); err != nil || loginData.AccessToken == "" {
		t.Fatalf("解析登录响应失败: err=%v data=%s", err, env.Data)
	}
	return server, loginData.AccessToken
}

func requireOK(t *testing.T, status int, env envelope, action string) {
	t.Helper()
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("%s失败: status=%d code=%d message=%s", action, status, env.Code, env.Message)
	}
}

func itoa(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

package router_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/database"
	"blog-api/internal/model"
	"blog-api/internal/router"
	"blog-api/internal/seed"
)

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestAuthFlowAndSeed(t *testing.T) {
	const (
		oldPassword = "admin123456"
		newPassword = "newpassword123"
	)

	hash, err := auth.HashPassword(oldPassword)
	if err != nil {
		t.Fatalf("生成测试密码哈希失败: %v", err)
	}

	cfg := &config.Config{
		AppEnv:             config.EnvDevelopment,
		HTTPAddr:           "127.0.0.1:0",
		DBPath:             filepath.Join(t.TempDir(), "blog.db"),
		JWTSecret:          "test-secret-at-least-32-characters-long",
		JWTAccessTTL:       time.Hour,
		JWTRefreshTTL:      24 * time.Hour,
		AdminUsername:      "admin",
		AdminPasswordHash:  hash,
		SiteURL:            "http://localhost:4321",
		APIBaseURL:         "http://localhost:8080",
		CORSAllowedOrigins: []string{"http://localhost:4321"},
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

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("创建 cookie jar 失败: %v", err)
	}
	client := &http.Client{Jar: jar}

	// 健康检查
	status, env := doJSON(t, client, http.MethodGet, server.URL+"/healthz", nil, "")
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("healthz 返回异常: status=%d code=%d message=%s", status, env.Code, env.Message)
	}

	// 登录
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": oldPassword,
	}, "")
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("登录失败: status=%d code=%d message=%s", status, env.Code, env.Message)
	}
	var loginData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(env.Data, &loginData); err != nil {
		t.Fatalf("解析登录响应失败: %v", err)
	}
	if loginData.AccessToken == "" {
		t.Fatal("登录响应缺少 access_token")
	}

	// 当前管理员
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/me", nil, loginData.AccessToken)
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("获取当前管理员失败: status=%d code=%d message=%s", status, env.Code, env.Message)
	}
	var me struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(env.Data, &me); err != nil {
		t.Fatalf("解析当前管理员响应失败: %v", err)
	}
	if me.Username != "admin" {
		t.Fatalf("当前管理员用户名不正确: %q", me.Username)
	}

	// 未携带 token 应 401
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/me", nil, "")
	if status != http.StatusUnauthorized || env.Code == 0 {
		t.Fatalf("未鉴权请求应返回 401: status=%d code=%d", status, env.Code)
	}

	// 刷新 access token（refresh token 走 httpOnly Cookie）
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/refresh", nil, "")
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("刷新令牌失败: status=%d code=%d message=%s", status, env.Code, env.Message)
	}
	var refreshData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(env.Data, &refreshData); err != nil {
		t.Fatalf("解析刷新响应失败: %v", err)
	}
	if refreshData.AccessToken == "" {
		t.Fatal("刷新响应缺少 access_token")
	}

	// 修改密码
	status, env = doJSON(t, client, http.MethodPut, server.URL+"/api/v1/auth/password", map[string]string{
		"old_password": oldPassword,
		"new_password": newPassword,
	}, refreshData.AccessToken)
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("修改密码失败: status=%d code=%d message=%s", status, env.Code, env.Message)
	}

	// 旧密码应失败，新密码应成功
	status, _ = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": oldPassword,
	}, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("旧密码登录应返回 401，实际 %d", status)
	}
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": newPassword,
	}, "")
	if status != http.StatusOK || env.Code != 0 {
		t.Fatalf("新密码登录失败: status=%d code=%d message=%s", status, env.Code, env.Message)
	}

	// 种子数据：唯一管理员 + 5 个固定一级目录
	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		t.Fatalf("统计管理员失败: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("管理员数量应为 1，实际 %d", userCount)
	}

	var dirs []model.Directory
	if err := db.Order("sort_order").Find(&dirs).Error; err != nil {
		t.Fatalf("查询目录失败: %v", err)
	}
	if len(dirs) != 5 {
		t.Fatalf("固定目录数量应为 5，实际 %d", len(dirs))
	}
	expected := map[string]string{
		"home": "page", "projects": "page", "tech": "page", "todos": "page", "other": "container",
	}
	for _, dir := range dirs {
		if dir.ParentID != nil {
			t.Fatalf("固定一级目录 %s 不应有 parent_id", dir.Slug)
		}
		if !dir.IsFixed {
			t.Fatalf("目录 %s 应为固定目录", dir.Slug)
		}
		if dir.PageKey == nil || expected[*dir.PageKey] != dir.Kind {
			t.Fatalf("目录 %s 的 page_key/kind 不正确: page_key=%v kind=%s", dir.Slug, dir.PageKey, dir.Kind)
		}
	}
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, token string) (int, envelope) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求 %s %s 失败: %v", method, url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}
	var env envelope
	if len(raw) > 0 && strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("解析响应失败: %v（原文：%s）", err, raw)
		}
	}
	return resp.StatusCode, env
}

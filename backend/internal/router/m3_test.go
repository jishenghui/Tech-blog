package router_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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

type fakeHTTPUploader struct {
	base  string
	calls int
}

func (f *fakeHTTPUploader) Upload(_ context.Context, key string, _ []byte, _ string) (string, error) {
	f.calls++
	return f.base + "/" + key, nil
}

func TestM3AssetAndZipImportFlow(t *testing.T) {
	server, token, uploader := newM3TestServer(t)
	client := &http.Client{}

	// 创建导入目标目录。
	status, env := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/directories", map[string]any{
		"name": "导入",
		"slug": "imports",
	}, token)
	requireOK(t, status, env, "创建一级目录")
	var parent struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &parent); err != nil || parent.ID == 0 {
		t.Fatalf("解析一级目录失败: err=%v data=%s", err, env.Data)
	}

	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/directories", map[string]any{
		"parent_id": parent.ID,
		"name":      "目标",
		"slug":      "target",
	}, token)
	requireOK(t, status, env, "创建二级目录")
	var target struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &target); err != nil || target.ID == 0 {
		t.Fatalf("解析二级目录失败: err=%v data=%s", err, env.Data)
	}

	// 导入 ZIP：文章引用图片，图片应上传并替换为 CDN 地址。
	pngData := makeTestPNG(t)
	zipData := buildTestZip(t, map[string][]byte{
		"post.md":        []byte("# 导入文章\n\n![pic](./images/pic.png)\n"),
		"images/pic.png": pngData,
	})
	zipBody, contentType := multipartBody(t, map[string]string{
		"directory_id": strconv.FormatUint(uint64(target.ID), 10),
		"publish":      "true",
	}, "bundle.zip", zipData)

	status, env = doMultipart(t, client, server.URL+"/api/v1/admin/imports/zip", contentType, zipBody, token)
	requireOK(t, status, env, "上传 ZIP")
	var job struct {
		ID            uint   `json:"id"`
		Status        string `json:"status"`
		MDCount       int    `json:"md_count"`
		ImageCount    int    `json:"image_count"`
		ReplacedCount int    `json:"replaced_count"`
	}
	if err := json.Unmarshal(env.Data, &job); err != nil {
		t.Fatalf("解析导入任务失败: %v", err)
	}
	if job.Status != "success" || job.MDCount != 1 || job.ImageCount != 1 || job.ReplacedCount != 1 {
		t.Fatalf("导入结果不正确: %+v (data=%s)", job, env.Data)
	}
	if uploader.calls != 1 {
		t.Fatalf("图片上传次数应为 1，实际 %d", uploader.calls)
	}

	// 导入历史与详情。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/imports", nil, token)
	requireOK(t, status, env, "导入历史")
	if !bytes.Contains(env.Data, []byte(`"total":1`)) {
		t.Fatalf("导入历史数量不正确: %s", env.Data)
	}
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/imports/"+itoa(job.ID), nil, token)
	requireOK(t, status, env, "导入详情")
	if !bytes.Contains(env.Data, []byte(`"status":"success"`)) {
		t.Fatalf("导入详情状态不正确: %s", env.Data)
	}

	// 公开文章应已发布且正文包含 CDN 图片地址。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/articles", nil, "")
	requireOK(t, status, env, "公开文章列表")
	var page struct {
		Total int64 `json:"total"`
		Items []struct {
			ID   uint   `json:"id"`
			Slug string `json:"slug"`
		} `json:"items"`
	}
	if err := json.Unmarshal(env.Data, &page); err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("公开文章列表不正确: err=%v data=%s", err, env.Data)
	}
	articleID := page.Items[0].ID

	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/articles/"+url.PathEscape(page.Items[0].Slug), nil, "")
	requireOK(t, status, env, "公开文章详情")
	if !bytes.Contains(env.Data, []byte("https://cdn.test/articles/")) {
		t.Fatalf("文章正文缺少 CDN 图片地址: %s", env.Data)
	}

	// 再次上传同一张图片，应命中 sha256 去重，不重复调用上传器。
	imgBody, imgContentType := multipartBody(t, nil, "same.png", pngData)
	status, env = doMultipart(t, client, server.URL+"/api/v1/admin/assets", imgContentType, imgBody, token)
	requireOK(t, status, env, "上传图片")
	if uploader.calls != 1 {
		t.Fatalf("相同图片应去重，上传器调用次数实际 %d", uploader.calls)
	}
	var asset struct {
		ID       uint   `json:"id"`
		URL      string `json:"url"`
		RefCount int    `json:"ref_count"`
	}
	if err := json.Unmarshal(env.Data, &asset); err != nil || asset.ID == 0 {
		t.Fatalf("解析资源失败: err=%v data=%s", err, env.Data)
	}
	if asset.RefCount != 1 {
		t.Fatalf("资源引用计数应为 1，实际 %d", asset.RefCount)
	}

	// 仍被文章引用时不能删除资源。
	status, _ = doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/assets/"+itoa(asset.ID), nil, token)
	if status != http.StatusConflict {
		t.Fatalf("被引用资源删除应返回 409，实际 %d", status)
	}

	// 删除文章后引用归零，资源可删除。
	status, env = doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/articles/"+itoa(articleID), nil, token)
	requireOK(t, status, env, "删除文章")
	status, env = doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/assets/"+itoa(asset.ID), nil, token)
	requireOK(t, status, env, "删除资源")
}

func newM3TestServer(t *testing.T) (*httptest.Server, string, *fakeHTTPUploader) {
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
		QiniuAssetBucket:   "test-bucket",
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

	uploader := &fakeHTTPUploader{base: "https://cdn.test"}
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	server := httptest.NewServer(router.NewWithUploader(cfg, db, tokens, uploader))
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
	return server, loginData.AccessToken, uploader
}

func makeTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("生成测试 PNG 失败: %v", err)
	}
	return buf.Bytes()
}

func buildTestZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, data := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("创建 ZIP 条目 %s 失败: %v", name, err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatalf("写入 ZIP 条目 %s 失败: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 ZIP 失败: %v", err)
	}
	return buf.Bytes()
}

func multipartBody(t *testing.T, fields map[string]string, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("写入表单字段 %s 失败: %v", key, err)
		}
	}
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("创建文件字段失败: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("写入文件字段失败: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 multipart 失败: %v", err)
	}
	return &buf, writer.FormDataContentType()
}

func doMultipart(t *testing.T, client *http.Client, endpoint, contentType string, body *bytes.Buffer, token string) (int, envelope) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", contentType)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return doRequest(t, client, req)
}

func doRequest(t *testing.T, client *http.Client, req *http.Request) (int, envelope) {
	t.Helper()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("请求 %s 失败: %v", req.URL, err)
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

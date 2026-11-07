package router_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestM4FixedPageAPIs(t *testing.T) {
	server, token, _ := newM3TestServer(t)
	client := &http.Client{}

	// 创建项目，包含调试流程与架构图 JSON。
	status, env := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/projects", map[string]any{
		"title":      "Tech Blog",
		"content_md": "# Tech Blog\n\n个人技术博客。",
		"tech_stack": []string{"Go", "Astro", "SQLite"},
		"status":     "published",
		"debug_steps": []map[string]any{
			{
				"id":           "reproduce",
				"title":        "复现 500",
				"description":  "最小请求复现。",
				"image_url":    "/poc/debug-reproduce.svg",
				"video_url":    "",
				"video_poster": "",
				"code":         "fmt.Println(\"hi\")",
				"code_lang":    "go",
				"article_slug": "sqlite-wal-lock",
				"tag_slug":     "sqlite",
			},
		},
		"architecture": map[string]any{
			"nodes": []map[string]any{
				{
					"id": "api", "label": "Go API", "kind": "backend",
					"x": 340, "y": 210, "description": "Gin + GORM",
					"tag_slug": "go", "search_query": "",
				},
			},
			"edges": []map[string]any{},
		},
	}, token)
	requireOK(t, status, env, "创建项目")
	var project struct {
		ID   uint   `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal(env.Data, &project); err != nil || project.ID == 0 {
		t.Fatalf("解析项目失败: err=%v data=%s", err, env.Data)
	}
	if project.Slug != "tech-blog" {
		t.Fatalf("项目 slug 不正确: %q", project.Slug)
	}

	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/projects", nil, "")
	requireOK(t, status, env, "公开项目列表")
	var projects []map[string]any
	if err := json.Unmarshal(env.Data, &projects); err != nil || len(projects) != 1 {
		t.Fatalf("公开项目列表不正确: err=%v data=%s", err, env.Data)
	}
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/projects/tech-blog", nil, "")
	requireOK(t, status, env, "公开项目详情")
	if !bytes.Contains(env.Data, []byte(`"content_html"`)) || !bytes.Contains(env.Data, []byte(`"debug_steps"`)) {
		t.Fatalf("项目详情字段不完整: %s", env.Data)
	}

	// 简历资料读写。
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/admin/profile", nil, token)
	requireOK(t, status, env, "后台首页资料")
	if !bytes.Contains(env.Data, []byte(`"resume_md"`)) {
		t.Fatalf("后台首页资料缺少 resume_md: %s", env.Data)
	}

	status, env = doJSON(t, client, http.MethodPut, server.URL+"/api/v1/admin/profile", map[string]any{
		"headline":   "后端开发",
		"avatar_url": "/poc/avatar.svg",
		"resume_md":  "# 简历\n\nGo 后端。",
		"awards": []map[string]any{
			{"id": "a1", "title": "国家奖学金", "image_url": "/poc/award-1.svg", "date": "2024", "description": "优秀"},
		},
		"experiences":       []map[string]any{},
		"featured_projects": []map[string]any{},
		"contacts":          map[string]any{"email": "me@example.com"},
	}, token)
	requireOK(t, status, env, "更新首页资料")

	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile", nil, "")
	requireOK(t, status, env, "公开首页资料")
	if !bytes.Contains(env.Data, []byte(`"国家奖学金"`)) || !bytes.Contains(env.Data, []byte(`"resume_html"`)) {
		t.Fatalf("公开首页资料不正确: %s", env.Data)
	}

	// 待办 CRUD 与排序。
	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/todos", map[string]any{
		"content": "写 M4 页面",
	}, token)
	requireOK(t, status, env, "创建待办")
	var todo struct {
		ID uint `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &todo); err != nil || todo.ID == 0 {
		t.Fatalf("解析待办失败: err=%v data=%s", err, env.Data)
	}

	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/todos", nil, "")
	requireOK(t, status, env, "公开待办列表")
	var todos []map[string]any
	if err := json.Unmarshal(env.Data, &todos); err != nil || len(todos) != 1 {
		t.Fatalf("公开待办列表不正确: err=%v data=%s", err, env.Data)
	}

	status, env = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/todos/reorder", map[string]any{
		"items": []map[string]any{{"id": todo.ID, "sort_order": 3}},
	}, token)
	requireOK(t, status, env, "待办排序")

	status, env = doJSON(t, client, http.MethodDelete, server.URL+"/api/v1/admin/todos/"+itoa(todo.ID), nil, token)
	requireOK(t, status, env, "删除待办")
	status, env = doJSON(t, client, http.MethodGet, server.URL+"/api/v1/todos", nil, "")
	requireOK(t, status, env, "删除后公开待办列表")
	if err := json.Unmarshal(env.Data, &todos); err != nil || len(todos) != 0 {
		t.Fatalf("删除后待办列表应为空: err=%v data=%s", err, env.Data)
	}

	// 非法 debug_steps 返回 400。
	status, _ = doJSON(t, client, http.MethodPost, server.URL+"/api/v1/admin/projects", map[string]any{
		"title":       "Bad",
		"content_md":  "# Bad",
		"debug_steps": map[string]any{"not": "array"},
	}, token)
	if status != http.StatusBadRequest {
		t.Fatalf("非法 debug_steps 应返回 400，实际 %d", status)
	}
}

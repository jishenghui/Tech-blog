package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"blog-api/internal/service"
)

func TestProjectCRUDWithFixedPageJSON(t *testing.T) {
	env := newTestEnv(t)

	input := service.ProjectInput{
		Title:     "Tech Blog",
		ContentMD: "# Tech Blog\n\n个人技术博客，Astro 前台 + Go API。",
		TechStack: []string{"Go", "Astro", "SQLite"},
		Status:    service.ProjectStatusPublished,
		DebugSteps: json.RawMessage(`[
			{
				"id":"reproduce",
				"title":"复现 500",
				"description":"最小请求复现发布接口的 500。",
				"image_url":"/poc/debug-reproduce.svg",
				"video_url":"https://cdn.test/demo.mp4",
				"video_poster":"/poc/video-poster.svg",
				"code":"fmt.Println(\"hi\")",
				"code_lang":"go",
				"article_slug":"sqlite-wal-lock",
				"tag_slug":"sqlite"
			}
		]`),
		Architecture: json.RawMessage(`{
			"nodes":[
				{"id":"api","label":"Go API","kind":"backend","x":340,"y":210,"description":"Gin + GORM","tag_slug":"go","search_query":""}
			],
			"edges":[]
		}`),
	}

	project, err := env.projects.Create(input)
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	if project.Slug != "tech-blog" {
		t.Fatalf("项目 slug 生成不正确: %q", project.Slug)
	}
	if !strings.Contains(project.ContentHTML, "<h1") {
		t.Fatalf("项目 Markdown 未渲染: %s", project.ContentHTML)
	}
	var techStack []string
	if err := json.Unmarshal([]byte(project.TechStack), &techStack); err != nil || len(techStack) != 3 {
		t.Fatalf("tech_stack 不正确: %s", project.TechStack)
	}

	publicList, err := env.projects.PublicList()
	if err != nil || len(publicList) != 1 {
		t.Fatalf("公开项目列表不正确: err=%v len=%d", err, len(publicList))
	}
	publicDetail, err := env.projects.PublicDetail("tech-blog")
	if err != nil {
		t.Fatalf("公开项目详情失败: %v", err)
	}
	if publicDetail.ContentHTML == "" {
		t.Fatal("公开项目详情缺少 content_html")
	}
	var steps []map[string]any
	if err := json.Unmarshal(publicDetail.DebugSteps, &steps); err != nil || len(steps) != 1 {
		t.Fatalf("debug_steps 不正确: %s", publicDetail.DebugSteps)
	}
	var architecture map[string]any
	if err := json.Unmarshal(publicDetail.Architecture, &architecture); err != nil {
		t.Fatalf("architecture 不正确: %v", err)
	}
	if len(architecture["nodes"].([]any)) != 1 {
		t.Fatalf("architecture 节点数量不正确: %s", publicDetail.Architecture)
	}

	adminDetail, err := env.projects.AdminDetail(project.ID)
	if err != nil {
		t.Fatalf("后台项目详情失败: %v", err)
	}
	if adminDetail.ContentMD != input.ContentMD {
		t.Fatal("后台项目详情缺少 content_md")
	}

	draft, err := env.projects.Create(service.ProjectInput{
		Title:     "Draft Project",
		ContentMD: "# Draft",
		Status:    service.ProjectStatusDraft,
	})
	if err != nil {
		t.Fatalf("创建草稿项目失败: %v", err)
	}
	publicList, _ = env.projects.PublicList()
	if len(publicList) != 1 {
		t.Fatalf("草稿项目不应出现在公开列表: %d", len(publicList))
	}
	adminList, err := env.projects.AdminList()
	if err != nil || len(adminList) != 2 {
		t.Fatalf("后台项目列表不正确: err=%v len=%d", err, len(adminList))
	}

	updatedInput := input
	updatedInput.Title = "Tech Blog Updated"
	updatedInput.Slug = "tech-blog-updated"
	updatedInput.Status = service.ProjectStatusDraft
	updatedInput.Summary = "新的项目摘要"
	updated, err := env.projects.Update(project.ID, updatedInput)
	if err != nil {
		t.Fatalf("更新项目失败: %v", err)
	}
	if updated.Slug != "tech-blog-updated" || updated.Summary != "新的项目摘要" {
		t.Fatalf("项目更新结果不正确: %+v", updated)
	}
	if _, err := env.projects.PublicDetail("tech-blog"); err == nil {
		t.Fatal("旧 slug 不应继续可访问")
	}
	publicList, _ = env.projects.PublicList()
	if len(publicList) != 0 {
		t.Fatalf("全部为草稿时公开列表应为空，实际 %d", len(publicList))
	}

	if err := env.projects.Reorder(service.ReorderProjectsInput{Items: []service.ReorderProjectItem{
		{ID: project.ID, SortOrder: 5},
		{ID: draft.ID, SortOrder: 6},
	}}); err != nil {
		t.Fatalf("项目排序失败: %v", err)
	}
	adminList, _ = env.projects.AdminList()
	if adminList[0].ID != project.ID || adminList[0].SortOrder != 5 {
		t.Fatalf("项目排序结果不正确: %+v", adminList)
	}

	if err := env.projects.Delete(project.ID); err != nil {
		t.Fatalf("删除项目失败: %v", err)
	}
	adminList, _ = env.projects.AdminList()
	if len(adminList) != 1 {
		t.Fatalf("删除后后台项目数量应为 1，实际 %d", len(adminList))
	}

	bad := input
	bad.Slug = "bad-json"
	bad.DebugSteps = json.RawMessage(`{"not":"array"}`)
	_, err = env.projects.Create(bad)
	requireErrorContains(t, err, "debug_steps")

	bad = input
	bad.Slug = "bad-arch"
	bad.Architecture = json.RawMessage(`{"nodes":[],"edges":[{"from":"a","to":"b"}]}`)
	_, err = env.projects.Create(bad)
	requireErrorContains(t, err, "不存在的起点")

	bad = input
	bad.Slug = "bad-status"
	bad.Status = "archived"
	_, err = env.projects.Create(bad)
	requireErrorContains(t, err, "status")

	bad = input
	bad.Slug = draft.Slug
	_, err = env.projects.Create(bad)
	requireErrorContains(t, err, "slug 已存在")
}

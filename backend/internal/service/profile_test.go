package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"blog-api/internal/service"
)

func TestProfileUpdateAndPublicView(t *testing.T) {
	env := newTestEnv(t)

	initial, err := env.profiles.GetPublic()
	if err != nil {
		t.Fatalf("查询初始首页资料失败: %v", err)
	}
	if string(initial.Awards) != "[]" {
		t.Fatalf("初始 awards 应为空数组，实际 %s", initial.Awards)
	}

	input := service.ProfileInput{
		Headline:  "后端开发 / 技术博客作者",
		AvatarURL: "/poc/avatar.svg",
		ResumeMD:  "# 个人简历\n\nGo 后端与前端工程实践。",
		Awards: json.RawMessage(`[
			{"id":"scholarship","title":"国家奖学金","image_url":"/poc/award-1.svg","date":"2024","description":"综合表现优秀"}
		]`),
		Experiences: json.RawMessage(`[
			{"company":"ACME","role":"后端开发","period":"2024.07 - 至今","description":"负责 Go API 与数据服务"}
		]`),
		FeaturedProjects: json.RawMessage(`[
			{"name":"Tech Blog","summary":"个人技术博客","url":"/projects/tech-blog/","cover_url":""}
		]`),
		Contacts: json.RawMessage(`{"email":"me@example.com","github":"https://github.com/jishenghui"}`),
	}

	admin, err := env.profiles.Update(input)
	if err != nil {
		t.Fatalf("更新首页资料失败: %v", err)
	}
	if admin.ResumeMD != input.ResumeMD || !strings.Contains(admin.ResumeHTML, "<h1") {
		t.Fatalf("简历 Markdown/HTML 不正确: md=%q html=%q", admin.ResumeMD, admin.ResumeHTML)
	}
	var awards []map[string]any
	if err := json.Unmarshal(admin.Awards, &awards); err != nil || len(awards) != 1 {
		t.Fatalf("awards 不正确: %s", admin.Awards)
	}
	var contacts map[string]any
	if err := json.Unmarshal(admin.Contacts, &contacts); err != nil || contacts["email"] != "me@example.com" {
		t.Fatalf("contacts 不正确: %s", admin.Contacts)
	}

	public, err := env.profiles.GetPublic()
	if err != nil {
		t.Fatalf("查询公开首页资料失败: %v", err)
	}
	if public.ResumeHTML == "" || public.Headline != input.Headline {
		t.Fatalf("公开首页资料不正确: %+v", public)
	}

	bad := input
	bad.Awards = json.RawMessage(`{"not":"array"}`)
	_, err = env.profiles.Update(bad)
	requireErrorContains(t, err, "awards")

	bad = input
	bad.Contacts = json.RawMessage(`[]`)
	_, err = env.profiles.Update(bad)
	requireErrorContains(t, err, "contacts")
}

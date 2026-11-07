package service_test

import (
	"testing"

	"blog-api/internal/service"
)

func TestTagCRUD(t *testing.T) {
	env := newTestEnv(t)

	tag, err := env.tags.Create(service.CreateTagInput{Name: "前端"})
	if err != nil {
		t.Fatalf("创建标签失败: %v", err)
	}
	if tag.Slug != "前端" {
		t.Fatalf("中文标签名应生成中文 slug，实际 %q", tag.Slug)
	}

	_, err = env.tags.Create(service.CreateTagInput{Name: "前端"})
	requireErrorContains(t, err, "标签名称已存在")

	_, err = env.tags.Create(service.CreateTagInput{Name: "前端2", Slug: "前端"})
	requireErrorContains(t, err, "标签 slug 已存在")

	updated, err := env.tags.Update(tag.ID, service.UpdateTagInput{
		Name: strPtr("前端开发"),
		Slug: strPtr("frontend"),
	})
	if err != nil {
		t.Fatalf("更新标签失败: %v", err)
	}
	if updated.Name != "前端开发" || updated.Slug != "frontend" {
		t.Fatalf("标签更新结果不正确: %+v", updated)
	}

	adminTags, err := env.tags.AdminList()
	if err != nil {
		t.Fatalf("后台标签列表失败: %v", err)
	}
	if len(adminTags) != 1 || adminTags[0].Slug != "frontend" {
		t.Fatalf("后台标签列表不正确: %+v", adminTags)
	}

	publicTags, err := env.tags.PublicList()
	if err != nil {
		t.Fatalf("公开标签列表失败: %v", err)
	}
	if len(publicTags) != 1 {
		t.Fatalf("公开标签数量应为 1，实际 %d", len(publicTags))
	}

	if err := env.tags.Delete(tag.ID); err != nil {
		t.Fatalf("删除标签失败: %v", err)
	}
	if _, err := env.tags.GetBySlug("frontend"); err == nil {
		t.Fatal("已删除标签不应能查询到")
	}
}

package service_test

import (
	"strings"
	"testing"

	"blog-api/internal/model"
	"blog-api/internal/service"
)

func TestArticleLifecycleAndCounts(t *testing.T) {
	env := newTestEnv(t)

	study := env.createTopLevel(t, "学习", "study")
	goDir := env.createSecondLevel(t, study.ID, "Go", "go")
	work := env.createTopLevel(t, "工作", "work")
	rustDir := env.createSecondLevel(t, work.ID, "Rust", "rust")
	golang := env.createTag(t, "Go语言", "golang")
	backend := env.createTag(t, "后端", "backend")

	content := "# 并发模型\n\nGoroutine 调度。\n\n```go\nfmt.Println(\"hello\")\n```\n\n" +
		"```mermaid\ngraph TD;\nA-->B;\n```\n\n行内公式 $x^2$。\n"
	article := env.createArticle(t, goDir.ID, "并发模型", content, []uint{golang.ID, backend.ID})

	if article.Status != service.ArticleStatusDraft {
		t.Fatalf("新文章应为草稿，实际 %q", article.Status)
	}
	if !strings.Contains(article.ContentHTML, `class="chroma"`) {
		t.Fatalf("缺少代码高亮: %s", article.ContentHTML)
	}
	if !strings.Contains(article.ContentHTML, `class="mermaid"`) {
		t.Fatalf("缺少 Mermaid 容器: %s", article.ContentHTML)
	}
	if !strings.Contains(article.ContentHTML, `class="math inline"`) {
		t.Fatalf("缺少数学公式容器: %s", article.ContentHTML)
	}
	if article.WordCount == 0 || article.ReadingTime < 1 {
		t.Fatalf("字数/阅读时长不正确: word_count=%d reading_time=%d", article.WordCount, article.ReadingTime)
	}
	if article.SearchText == "" || !strings.Contains(article.SearchText, "并发模型") {
		t.Fatalf("搜索文本不正确: %q", article.SearchText)
	}
	if article.Summary == "" {
		t.Fatal("摘要不应为空")
	}
	if got := directoryCount(t, env.db, goDir.ID); got != 0 {
		t.Fatalf("草稿不应计入目录计数，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 0 {
		t.Fatalf("草稿不应计入标签计数，实际 %d", got)
	}

	if err := env.articles.Publish(article.ID); err != nil {
		t.Fatalf("发布文章失败: %v", err)
	}
	if got := directoryCount(t, env.db, goDir.ID); got != 1 {
		t.Fatalf("发布后目录计数应为 1，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 1 {
		t.Fatalf("发布后标签计数应为 1，实际 %d", got)
	}
	if got := tagCount(t, env.db, backend.ID); got != 1 {
		t.Fatalf("发布后标签计数应为 1，实际 %d", got)
	}

	page, err := env.articles.PublicList(service.ArticleFilter{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("查询公开文章失败: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("公开文章数量不正确: total=%d items=%d", page.Total, len(page.Items))
	}

	// 一级目录应能聚合其下二级目录的文章。
	subtreePage, err := env.articles.PublicList(service.ArticleFilter{
		DirectoryID: &study.ID,
		Page:        1,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("按一级目录筛选失败: %v", err)
	}
	if subtreePage.Total != 1 {
		t.Fatalf("一级目录应聚合到 1 篇文章，实际 %d", subtreePage.Total)
	}

	detail, err := env.articles.PublicDetail(article.Slug)
	if err != nil {
		t.Fatalf("查询公开文章详情失败: %v", err)
	}
	if detail.ContentHTML == "" || len(detail.Tags) != 2 {
		t.Fatalf("文章详情不完整: html=%d tags=%d", len(detail.ContentHTML), len(detail.Tags))
	}
	if detail.Status != "" || detail.Source != "" {
		t.Fatalf("公开详情不应暴露状态/来源: status=%q source=%q", detail.Status, detail.Source)
	}

	searchPage, err := env.articles.PublicList(service.ArticleFilter{Q: "goroutine", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if searchPage.Total != 1 {
		t.Fatalf("搜索应命中 1 篇，实际 %d", searchPage.Total)
	}

	tagPage, err := env.articles.PublicList(service.ArticleFilter{TagSlug: "golang", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("按标签筛选失败: %v", err)
	}
	if tagPage.Total != 1 {
		t.Fatalf("按标签筛选应命中 1 篇，实际 %d", tagPage.Total)
	}

	adminPage, err := env.articles.AdminList(service.ArticleFilter{
		DirectoryID: &goDir.ID,
		Status:      service.ArticleStatusPublished,
		Page:        1,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("后台文章列表失败: %v", err)
	}
	if adminPage.Total != 1 {
		t.Fatalf("后台筛选应命中 1 篇，实际 %d", adminPage.Total)
	}

	if err := env.articles.SetTags(article.ID, []uint{golang.ID}); err != nil {
		t.Fatalf("覆盖标签失败: %v", err)
	}
	if got := tagCount(t, env.db, backend.ID); got != 0 {
		t.Fatalf("移除标签后计数应归零，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 1 {
		t.Fatalf("保留标签计数应为 1，实际 %d", got)
	}

	if err := env.articles.Move(article.ID, rustDir.ID); err != nil {
		t.Fatalf("移动文章失败: %v", err)
	}
	if got := directoryCount(t, env.db, goDir.ID); got != 0 {
		t.Fatalf("源目录计数应归零，实际 %d", got)
	}
	if got := directoryCount(t, env.db, rustDir.ID); got != 1 {
		t.Fatalf("目标目录计数应为 1，实际 %d", got)
	}

	newTitle := "并发模型（修订）"
	newContent := "# 修订\n\n新的正文。"
	updated, err := env.articles.Update(article.ID, service.UpdateArticleInput{
		Title:     &newTitle,
		ContentMD: &newContent,
	})
	if err != nil {
		t.Fatalf("更新文章失败: %v", err)
	}
	if updated.Slug != article.Slug {
		t.Fatalf("未提供 slug 时不应改变 slug: got=%q want=%q", updated.Slug, article.Slug)
	}
	if !strings.Contains(updated.ContentHTML, "新的正文") {
		t.Fatalf("更新后正文未重新渲染: %s", updated.ContentHTML)
	}

	adminDetail, err := env.articles.AdminDetail(article.ID)
	if err != nil {
		t.Fatalf("查询后台文章详情失败: %v", err)
	}
	if len(adminDetail.TagIDs) != 1 || adminDetail.TagIDs[0] != golang.ID {
		t.Fatalf("后台标签 ID 不正确: %v", adminDetail.TagIDs)
	}
	if adminDetail.ContentMD != newContent {
		t.Fatalf("后台文章未返回 Markdown 原文")
	}

	// 重复 slug 应冲突。
	_, err = env.articles.Create(service.CreateArticleInput{
		DirectoryID: goDir.ID,
		Title:       "重复 slug",
		Slug:        article.Slug,
	})
	requireErrorContains(t, err, "slug 已存在")

	// 文章不能挂在一级目录。
	_, err = env.articles.Create(service.CreateArticleInput{
		DirectoryID: study.ID,
		Title:       "非法目录",
		ContentMD:   "正文",
	})
	requireErrorContains(t, err, "二级目录")

	if err := env.articles.Unpublish(article.ID); err != nil {
		t.Fatalf("取消发布失败: %v", err)
	}
	if got := directoryCount(t, env.db, rustDir.ID); got != 0 {
		t.Fatalf("取消发布后目录计数应归零，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 0 {
		t.Fatalf("取消发布后标签计数应归零，实际 %d", got)
	}

	if err := env.articles.Publish(article.ID); err != nil {
		t.Fatalf("重新发布失败: %v", err)
	}

	// 人为破坏冗余字段后全量重算。
	if err := env.db.Model(&model.Directory{}).Where("id = ?", rustDir.ID).
		UpdateColumn("article_count", 99).Error; err != nil {
		t.Fatalf("破坏目录计数失败: %v", err)
	}
	if err := env.db.Model(&model.Tag{}).Where("id = ?", golang.ID).
		UpdateColumn("article_count", 99).Error; err != nil {
		t.Fatalf("破坏标签计数失败: %v", err)
	}
	if err := env.db.Model(&model.Article{}).Where("id = ?", article.ID).
		UpdateColumn("content_html", "broken").Error; err != nil {
		t.Fatalf("破坏文章 HTML 失败: %v", err)
	}

	recalc, err := env.articles.Recalc()
	if err != nil {
		t.Fatalf("全量重算失败: %v", err)
	}
	if recalc.Articles != 1 {
		t.Fatalf("重算文章数应为 1，实际 %d", recalc.Articles)
	}
	if got := directoryCount(t, env.db, rustDir.ID); got != 1 {
		t.Fatalf("重算后目录计数应为 1，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 1 {
		t.Fatalf("重算后标签计数应为 1，实际 %d", got)
	}
	reloaded := env.reloadArticle(t, article.ID)
	if reloaded.ContentHTML == "broken" {
		t.Fatal("重算后 content_html 未恢复")
	}

	if err := env.articles.Delete(article.ID); err != nil {
		t.Fatalf("删除文章失败: %v", err)
	}
	if got := directoryCount(t, env.db, rustDir.ID); got != 0 {
		t.Fatalf("删除后目录计数应归零，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 0 {
		t.Fatalf("删除后标签计数应归零，实际 %d", got)
	}
	if _, err := env.articles.PublicDetail(article.Slug); err == nil {
		t.Fatal("已删除文章不应能公开访问")
	}
}

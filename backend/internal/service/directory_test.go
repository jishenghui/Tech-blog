package service_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"blog-api/internal/model"
	"blog-api/internal/service"
)

func TestDirectoryDissolveSemantics(t *testing.T) {
	env := newTestEnv(t)

	study := env.createTopLevel(t, "学习", "study")
	goDir := env.createSecondLevel(t, study.ID, "Go", "go")
	rustDir := env.createSecondLevel(t, study.ID, "Rust", "rust")
	golang := env.createTag(t, "Go语言", "golang")

	article := env.createArticle(t, goDir.ID, "Go 并发", "# Go 并发\n\nGoroutine 调度。", []uint{golang.ID})
	if err := env.articles.Publish(article.ID); err != nil {
		t.Fatalf("发布文章失败: %v", err)
	}
	if got := directoryCount(t, env.db, goDir.ID); got != 1 {
		t.Fatalf("Go 目录已发布计数应为 1，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 1 {
		t.Fatalf("标签计数应为 1，实际 %d", got)
	}

	other := env.fixedDir(t, "other")

	// 解散二级目录：文章应进入“其他/其他”，原目录计数归零。
	if err := env.dirs.Dissolve(goDir.ID); err != nil {
		t.Fatalf("解散二级目录失败: %v", err)
	}
	var otherInner model.Directory
	if err := env.db.Where("parent_id = ? AND name = ?", other.ID, "其他").First(&otherInner).Error; err != nil {
		t.Fatalf("查询“其他/其他”失败: %v", err)
	}
	if otherInner.Slug != service.OtherInnerSlug {
		t.Fatalf("“其他/其他”slug 不正确: %q", otherInner.Slug)
	}
	reloadedArticle := env.reloadArticle(t, article.ID)
	if reloadedArticle.DirectoryID == nil || *reloadedArticle.DirectoryID != otherInner.ID {
		t.Fatalf("文章未迁移到“其他/其他”: %+v", reloadedArticle.DirectoryID)
	}
	if got := directoryCount(t, env.db, otherInner.ID); got != 1 {
		t.Fatalf("“其他/其他”计数应为 1，实际 %d", got)
	}
	if got := directoryCount(t, env.db, goDir.ID); got != 0 {
		t.Fatalf("原二级目录计数应归零，实际 %d", got)
	}
	if got := tagCount(t, env.db, golang.ID); got != 1 {
		t.Fatalf("解散不应改变标签计数，实际 %d", got)
	}
	var deletedGo model.Directory
	if err := env.db.First(&deletedGo, goDir.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("原二级目录应被软删除，实际 err=%v", err)
	}

	// 解散一级目录：剩余子目录整体进入“其他”，文章保留。
	if err := env.dirs.Dissolve(study.ID); err != nil {
		t.Fatalf("解散一级目录失败: %v", err)
	}
	rustReloaded := env.reloadDirectory(t, rustDir.ID)
	if rustReloaded.ParentID == nil || *rustReloaded.ParentID != other.ID {
		t.Fatalf("二级目录未迁移到“其他”: %+v", rustReloaded.ParentID)
	}
	var deletedStudy model.Directory
	if err := env.db.First(&deletedStudy, study.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("原一级目录应被软删除，实际 err=%v", err)
	}

	// 固定目录不可解散。
	requireErrorContains(t, env.dirs.Dissolve(other.ID), "固定目录不可解散")

	home := env.fixedDir(t, "home")
	_, err := env.dirs.Create(service.CreateDirectoryInput{
		ParentID: &home.ID,
		Name:     "非法子目录",
		Slug:     "invalid-child",
	})
	requireErrorContains(t, err, "固定页面目录")

	// 二级目录下不能再建目录。
	_, err = env.dirs.Create(service.CreateDirectoryInput{
		ParentID: &rustDir.ID,
		Name:     "第三级",
		Slug:     "third-level",
	})
	requireErrorContains(t, err, "只支持两级目录")

	// 保留 slug。
	_, err = env.dirs.Create(service.CreateDirectoryInput{Name: "文章", Slug: "posts"})
	requireErrorContains(t, err, "系统保留路径")

	// 固定目录不可改名。
	_, err = env.dirs.Update(other.ID, service.UpdateDirectoryInput{Name: strPtr("新名字")})
	requireErrorContains(t, err, "固定目录不可改名")

	// 二级目录可跨一级目录移动。
	work := env.createTopLevel(t, "工作", "work")
	if err := env.dirs.Move(rustDir.ID, &work.ID, nil); err != nil {
		t.Fatalf("移动二级目录失败: %v", err)
	}
	rustReloaded = env.reloadDirectory(t, rustDir.ID)
	if rustReloaded.ParentID == nil || *rustReloaded.ParentID != work.ID {
		t.Fatalf("二级目录未移动到工作目录: %+v", rustReloaded.ParentID)
	}

	// 一级目录不能改为二级目录。
	requireErrorContains(t, env.dirs.Move(work.ID, &other.ID, nil), "一级目录不能移动")
	// 固定目录不可移动。
	requireErrorContains(t, env.dirs.Move(home.ID, nil, nil), "固定目录不可移动")
}

func TestDirectoryTreeMoveAndReorder(t *testing.T) {
	env := newTestEnv(t)

	a := env.createTopLevel(t, "A", "a")
	b := env.createTopLevel(t, "B", "b")
	a1 := env.createSecondLevel(t, a.ID, "A1", "a1")
	a2 := env.createSecondLevel(t, a.ID, "A2", "a2")
	b1 := env.createSecondLevel(t, b.ID, "B1", "b1")

	tree, err := env.dirs.AdminTree()
	if err != nil {
		t.Fatalf("查询后台目录树失败: %v", err)
	}
	if got := findNode(tree, a.ID).Children; len(got) != 2 {
		t.Fatalf("A 的子目录数量应为 2，实际 %d", len(got))
	}
	if got := findNode(tree, b.ID).Children; len(got) != 1 {
		t.Fatalf("B 的子目录数量应为 1，实际 %d", len(got))
	}

	// slug 冲突。
	_, err = env.dirs.Create(service.CreateDirectoryInput{Name: "重复", Slug: "a"})
	requireErrorContains(t, err, "slug 已存在")
	_, err = env.dirs.Update(a.ID, service.UpdateDirectoryInput{Slug: strPtr("b")})
	requireErrorContains(t, err, "slug 已存在")

	// 跨一级目录移动。
	if err := env.dirs.Move(a1.ID, &b.ID, nil); err != nil {
		t.Fatalf("移动 A1 到 B 失败: %v", err)
	}
	if got := env.reloadDirectory(t, a1.ID); got.ParentID == nil || *got.ParentID != b.ID {
		t.Fatalf("A1 未移动到 B: %+v", got.ParentID)
	}

	// 批量排序。
	if err := env.dirs.Reorder(service.ReorderDirectoriesInput{Items: []service.ReorderDirectoryItem{
		{ID: a.ID, ParentID: nil, SortOrder: 20},
		{ID: b.ID, ParentID: nil, SortOrder: 21},
		{ID: a2.ID, ParentID: &a.ID, SortOrder: 5},
		{ID: b1.ID, ParentID: &b.ID, SortOrder: 6},
		{ID: a1.ID, ParentID: &b.ID, SortOrder: 7},
	}}); err != nil {
		t.Fatalf("批量排序失败: %v", err)
	}
	if got := env.reloadDirectory(t, a.ID); got.SortOrder != 20 {
		t.Fatalf("A 排序应为 20，实际 %d", got.SortOrder)
	}
	if got := env.reloadDirectory(t, b.ID); got.SortOrder != 21 {
		t.Fatalf("B 排序应为 21，实际 %d", got.SortOrder)
	}

	// 重复 ID 应失败。
	requireErrorContains(t, env.dirs.Reorder(service.ReorderDirectoriesInput{Items: []service.ReorderDirectoryItem{
		{ID: a.ID, ParentID: nil, SortOrder: 1},
		{ID: a.ID, ParentID: nil, SortOrder: 2},
	}}), "重复目录")

	// 一级目录不能移入二级目录。
	requireErrorContains(t, env.dirs.Move(a.ID, &b.ID, nil), "一级目录不能移动")

	// 隐藏 A 后公开导航不应包含 A。
	if _, err := env.dirs.Update(a.ID, service.UpdateDirectoryInput{Visible: boolPtr(false)}); err != nil {
		t.Fatalf("隐藏目录失败: %v", err)
	}
	nav, err := env.dirs.PublicNav()
	if err != nil {
		t.Fatalf("查询公开导航失败: %v", err)
	}
	for _, node := range nav {
		if node.ID == a.ID {
			t.Fatal("隐藏目录不应出现在公开导航中")
		}
	}

	// B 的子树应包含 B、B1、A1。
	ids, err := env.dirs.SubtreeIDs(b.ID)
	if err != nil {
		t.Fatalf("查询子树失败: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("B 的子树数量应为 3，实际 %d: %v", len(ids), ids)
	}
}

func findNode(nodes []service.DirectoryNode, id uint) service.DirectoryNode {
	for _, node := range nodes {
		if node.ID == id {
			return node
		}
		for _, child := range node.Children {
			if child.ID == id {
				return child
			}
		}
	}
	return service.DirectoryNode{}
}

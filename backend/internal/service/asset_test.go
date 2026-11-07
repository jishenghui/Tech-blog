package service_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"gorm.io/gorm"

	"blog-api/internal/model"
	"blog-api/internal/service"
)

func makePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("生成测试 PNG 失败: %v", err)
	}
	return buf.Bytes()
}

func TestAssetUploadDedupeAndDelete(t *testing.T) {
	env := newTestEnv(t)
	pngData := makePNG(t)

	first, err := env.assets.Upload(context.Background(), service.UploadAssetInput{
		OriginalName: "a.png",
		Data:         pngData,
	})
	if err != nil {
		t.Fatalf("上传图片失败: %v", err)
	}
	if first.ID == 0 || first.Mime != "image/png" || first.Sha256 == "" {
		t.Fatalf("资源信息不完整: %+v", first)
	}
	if !strings.HasPrefix(first.URL, "https://cdn.test/articles/") {
		t.Fatalf("资源 URL 不正确: %s", first.URL)
	}
	if env.uploader.calls != 1 {
		t.Fatalf("上传器调用次数应为 1，实际 %d", env.uploader.calls)
	}

	second, err := env.assets.Upload(context.Background(), service.UploadAssetInput{
		OriginalName: "b.png",
		Data:         pngData,
	})
	if err != nil {
		t.Fatalf("重复上传失败: %v", err)
	}
	if second.ID != first.ID || second.URL != first.URL {
		t.Fatalf("相同内容应复用资源: first=%+v second=%+v", first, second)
	}
	if env.uploader.calls != 1 {
		t.Fatalf("相同内容不应重复上传，实际调用 %d 次", env.uploader.calls)
	}

	var assetCount int64
	if err := env.db.Model(&model.Asset{}).Count(&assetCount).Error; err != nil {
		t.Fatalf("统计资源失败: %v", err)
	}
	if assetCount != 1 {
		t.Fatalf("资源数量应为 1，实际 %d", assetCount)
	}

	if _, err := env.assets.Upload(context.Background(), service.UploadAssetInput{
		OriginalName: "note.txt",
		Data:         []byte("not an image"),
	}); !errors.Is(err, service.ErrNotImage) {
		t.Fatalf("非图片文件应返回 ErrNotImage，实际 %v", err)
	}

	if err := env.db.Model(&model.Asset{}).Where("id = ?", first.ID).
		UpdateColumn("ref_count", 1).Error; err != nil {
		t.Fatalf("设置引用计数失败: %v", err)
	}
	if err := env.assets.Delete(first.ID); err == nil {
		t.Fatal("仍被引用的资源不应允许删除")
	}

	if err := env.db.Model(&model.Asset{}).Where("id = ?", first.ID).
		UpdateColumn("ref_count", 0).Error; err != nil {
		t.Fatalf("重置引用计数失败: %v", err)
	}
	if err := env.assets.Delete(first.ID); err != nil {
		t.Fatalf("删除资源失败: %v", err)
	}
	var deleted model.Asset
	if err := env.db.First(&deleted, first.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("资源应被软删除，实际 err=%v", err)
	}

	third, err := env.assets.Upload(context.Background(), service.UploadAssetInput{
		OriginalName: "c.png",
		Data:         pngData,
	})
	if err != nil {
		t.Fatalf("软删除后重新上传失败: %v", err)
	}
	if third.ID == first.ID {
		t.Fatal("软删除后应创建新的资源记录")
	}
}

func TestAssetRefCountTracksArticles(t *testing.T) {
	env := newTestEnv(t)
	study := env.createTopLevel(t, "学习", "study")
	goDir := env.createSecondLevel(t, study.ID, "Go", "go")

	asset, err := env.assets.Upload(context.Background(), service.UploadAssetInput{
		OriginalName: "pic.png",
		Data:         makePNG(t),
	})
	if err != nil {
		t.Fatalf("上传图片失败: %v", err)
	}

	content := "![pic](" + asset.URL + ")\n\n正文"
	article := env.createArticle(t, goDir.ID, "引用图片", content, nil)

	reload := func() *model.Asset {
		var current model.Asset
		if err := env.db.First(&current, asset.ID).Error; err != nil {
			t.Fatalf("查询资源失败: %v", err)
		}
		return &current
	}
	if got := reload().RefCount; got != 1 {
		t.Fatalf("创建文章后引用计数应为 1，实际 %d", got)
	}

	withoutImage := "没有图片了"
	if _, err := env.articles.Update(article.ID, service.UpdateArticleInput{ContentMD: &withoutImage}); err != nil {
		t.Fatalf("更新文章失败: %v", err)
	}
	if got := reload().RefCount; got != 0 {
		t.Fatalf("移除图片后引用计数应为 0，实际 %d", got)
	}

	if _, err := env.articles.Update(article.ID, service.UpdateArticleInput{ContentMD: &content}); err != nil {
		t.Fatalf("恢复图片失败: %v", err)
	}
	if got := reload().RefCount; got != 1 {
		t.Fatalf("恢复图片后引用计数应为 1，实际 %d", got)
	}

	if err := env.db.Model(&model.Asset{}).Where("id = ?", asset.ID).
		UpdateColumn("ref_count", 99).Error; err != nil {
		t.Fatalf("破坏引用计数失败: %v", err)
	}
	result, err := env.articles.Recalc()
	if err != nil {
		t.Fatalf("全量重算失败: %v", err)
	}
	if result.Assets != 1 {
		t.Fatalf("重算资源数应为 1，实际 %d", result.Assets)
	}
	if got := reload().RefCount; got != 1 {
		t.Fatalf("重算后引用计数应为 1，实际 %d", got)
	}

	if err := env.articles.Delete(article.ID); err != nil {
		t.Fatalf("删除文章失败: %v", err)
	}
	if got := reload().RefCount; got != 0 {
		t.Fatalf("删除文章后引用计数应为 0，实际 %d", got)
	}
}

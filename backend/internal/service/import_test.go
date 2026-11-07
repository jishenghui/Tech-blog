package service_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"blog-api/internal/model"
	"blog-api/internal/service"
)

type zipEntry struct {
	Name string
	Data []byte
}

func makeZip(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.Name, Method: zip.Deflate}
		file, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("创建 ZIP 条目 %s 失败: %v", entry.Name, err)
		}
		if _, err := file.Write(entry.Data); err != nil {
			t.Fatalf("写入 ZIP 条目 %s 失败: %v", entry.Name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭 ZIP 失败: %v", err)
	}
	return buf.Bytes()
}

func TestImportZipValidWithRecursiveRoot(t *testing.T) {
	env := newTestEnv(t)
	study := env.createTopLevel(t, "学习", "study")
	target := env.createSecondLevel(t, study.ID, "Go", "go")

	md := "# 导入文章\n\n![a](./images/pic.png)\n\n![b](images/pic.png)\n"
	zipData := makeZip(t, []zipEntry{
		{Name: "nested/post.md", Data: []byte(md)},
		{Name: "nested/images/pic.png", Data: makePNG(t)},
	})

	job, err := env.imports.ImportZip(context.Background(), service.ZipImportInput{
		DirectoryID: target.ID,
		Filename:    "bundle.zip",
		Size:        int64(len(zipData)),
		Reader:      bytes.NewReader(zipData),
	})
	if err != nil {
		t.Fatalf("导入 ZIP 失败: %v", err)
	}
	if job.Status != "success" {
		t.Fatalf("导入状态应为 success，实际 %q（%s）", job.Status, job.ErrorMessage)
	}
	if job.MDCount != 1 || job.ImageCount != 1 || job.ReplacedCount != 2 {
		t.Fatalf("导入统计不正确: md=%d image=%d replaced=%d", job.MDCount, job.ImageCount, job.ReplacedCount)
	}
	if job.ExtractedRoot != "nested" {
		t.Fatalf("递归下探根路径应为 nested，实际 %q", job.ExtractedRoot)
	}
	if env.uploader.calls != 1 {
		t.Fatalf("图片上传次数应为 1，实际 %d", env.uploader.calls)
	}

	var article model.Article
	if err := env.db.Where("source = ?", "zip").First(&article).Error; err != nil {
		t.Fatalf("查询导入文章失败: %v", err)
	}
	if article.Title != "导入文章" || article.SourceFile != "post.md" {
		t.Fatalf("导入文章标题/来源文件不正确: title=%q file=%q", article.Title, article.SourceFile)
	}
	if article.Status != service.ArticleStatusDraft {
		t.Fatalf("导入文章默认应为草稿，实际 %q", article.Status)
	}
	if strings.Count(article.ContentMD, "https://cdn.test/articles/") != 2 {
		t.Fatalf("图片路径未完全替换: %s", article.ContentMD)
	}
	if strings.Contains(article.ContentMD, "images/pic.png") {
		t.Fatalf("仍残留本地图片路径: %s", article.ContentMD)
	}

	var assetCount int64
	if err := env.db.Model(&model.Asset{}).Count(&assetCount).Error; err != nil {
		t.Fatalf("统计资源失败: %v", err)
	}
	if assetCount != 1 {
		t.Fatalf("资源数量应为 1，实际 %d", assetCount)
	}

	detail, err := env.imports.Get(job.ID)
	if err != nil {
		t.Fatalf("查询导入任务失败: %v", err)
	}
	if detail.ID != job.ID || detail.Meta.Root != "nested" {
		t.Fatalf("导入任务详情不正确: %+v", detail)
	}

	list, err := env.imports.List(1, 10)
	if err != nil {
		t.Fatalf("查询导入历史失败: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != job.ID {
		t.Fatalf("导入历史不正确: %+v", list)
	}
}

func TestImportZipPublishAndDuplicateImage(t *testing.T) {
	env := newTestEnv(t)
	study := env.createTopLevel(t, "学习", "study")
	target := env.createSecondLevel(t, study.ID, "Go", "go")

	pngData := makePNG(t)
	md := "# 发布文章\n\n![a](./images/a.png)\n\n![b](./images/b.png)\n"
	zipData := makeZip(t, []zipEntry{
		{Name: "post.md", Data: []byte(md)},
		{Name: "images/a.png", Data: pngData},
		{Name: "images/b.png", Data: pngData},
	})

	job, err := env.imports.ImportZip(context.Background(), service.ZipImportInput{
		DirectoryID: target.ID,
		Filename:    "publish.zip",
		Size:        int64(len(zipData)),
		Reader:      bytes.NewReader(zipData),
		Publish:     true,
	})
	if err != nil {
		t.Fatalf("导入 ZIP 失败: %v", err)
	}
	if job.Status != "success" || job.MDCount != 1 || job.ImageCount != 2 || job.ReplacedCount != 2 {
		t.Fatalf("导入结果不正确: %+v", job)
	}
	if env.uploader.calls != 1 {
		t.Fatalf("相同内容图片应只上传一次，实际 %d", env.uploader.calls)
	}

	var article model.Article
	if err := env.db.Where("source = ?", "zip").First(&article).Error; err != nil {
		t.Fatalf("查询导入文章失败: %v", err)
	}
	if article.Status != service.ArticleStatusPublished || article.PublishedAt == nil {
		t.Fatalf("导入文章应已发布: status=%q published_at=%v", article.Status, article.PublishedAt)
	}
	if got := directoryCount(t, env.db, target.ID); got != 1 {
		t.Fatalf("发布后目录计数应为 1，实际 %d", got)
	}
}

func TestImportZipInvalidStructures(t *testing.T) {
	env := newTestEnv(t)
	pngData := makePNG(t)

	cases := []struct {
		name    string
		entries []zipEntry
		wantErr string
	}{
		{
			name: "根层含其他文件",
			entries: []zipEntry{
				{Name: "a.md", Data: []byte("# a")},
				{Name: "note.txt", Data: []byte("x")},
			},
			wantErr: "根层级只允许",
		},
		{
			name: "根层含其他目录",
			entries: []zipEntry{
				{Name: "a.md", Data: []byte("# a")},
				{Name: "sub/b.md", Data: []byte("# b")},
			},
			wantErr: "根层级只允许",
		},
		{
			name: "路径穿越",
			entries: []zipEntry{
				{Name: "../evil.md", Data: []byte("# evil")},
			},
			wantErr: "路径穿越",
		},
		{
			name: "缺少 Markdown",
			entries: []zipEntry{
				{Name: "images/pic.png", Data: pngData},
				{Name: "note.txt", Data: []byte("x")},
			},
			wantErr: "根层级只允许",
		},
		{
			name:    "空压缩包",
			entries: []zipEntry{},
			wantErr: "根目录为空",
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parent := env.createTopLevel(t, fmt.Sprintf("导入%d", i), fmt.Sprintf("import-%d", i))
			target := env.createSecondLevel(t, parent.ID, "目标", fmt.Sprintf("target-%d", i))

			zipData := makeZip(t, tc.entries)
			job, err := env.imports.ImportZip(context.Background(), service.ZipImportInput{
				DirectoryID: target.ID,
				Filename:    "invalid.zip",
				Size:        int64(len(zipData)),
				Reader:      bytes.NewReader(zipData),
			})
			if err != nil {
				t.Fatalf("导入请求不应直接失败: %v", err)
			}
			if job.Status != "failed" {
				t.Fatalf("导入状态应为 failed，实际 %q", job.Status)
			}
			if !strings.Contains(job.ErrorMessage, tc.wantErr) {
				t.Fatalf("错误信息应包含 %q，实际 %q", tc.wantErr, job.ErrorMessage)
			}

			var count int64
			if err := env.db.Model(&model.Article{}).Where("directory_id = ?", target.ID).
				Count(&count).Error; err != nil {
				t.Fatalf("统计文章失败: %v", err)
			}
			if count != 0 {
				t.Fatalf("非法压缩包不应创建文章，实际 %d", count)
			}
		})
	}
}

func TestImportZipReplacementFailureDoesNotInterrupt(t *testing.T) {
	env := newTestEnv(t)
	study := env.createTopLevel(t, "学习", "study")
	target := env.createSecondLevel(t, study.ID, "Go", "go")

	md := "# 缺图文章\n\n![missing](./images/missing.png)\n"
	zipData := makeZip(t, []zipEntry{
		{Name: "post.md", Data: []byte(md)},
	})

	job, err := env.imports.ImportZip(context.Background(), service.ZipImportInput{
		DirectoryID: target.ID,
		Filename:    "missing.zip",
		Size:        int64(len(zipData)),
		Reader:      bytes.NewReader(zipData),
	})
	if err != nil {
		t.Fatalf("导入 ZIP 失败: %v", err)
	}
	if job.Status != "success" {
		t.Fatalf("缺图不应中断导入，实际状态 %q（%s）", job.Status, job.ErrorMessage)
	}
	if job.ReplacedCount != 0 || len(job.Meta.Failures) != 1 {
		t.Fatalf("缺图统计不正确: replaced=%d failures=%+v", job.ReplacedCount, job.Meta.Failures)
	}

	var article model.Article
	if err := env.db.Where("source = ?", "zip").First(&article).Error; err != nil {
		t.Fatalf("查询导入文章失败: %v", err)
	}
	if !strings.Contains(article.ContentMD, "./images/missing.png") {
		t.Fatalf("缺图引用应保留原样: %s", article.ContentMD)
	}
}

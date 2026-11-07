package service

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

const (
	// MaxZipUploadSize 是 ZIP 上传大小上限。
	MaxZipUploadSize   = 50 << 20  // 50MB
	maxExtractedSize   = 200 << 20 // 解压后总大小上限
	maxZipEntries      = 2000
	maxMarkdownSize    = 5 << 20
	maxRootDepth       = 20
	maxMetaFileList    = 200
	maxMetaFailureList = 200
)

var driveLetterPattern = regexp.MustCompile(`^[A-Za-z]:/`)

type ImportService struct {
	db       *gorm.DB
	assets   *AssetService
	articles *ArticleService
}

func NewImportService(db *gorm.DB, assets *AssetService, articles *ArticleService) *ImportService {
	return &ImportService{db: db, assets: assets, articles: articles}
}

type ZipImportInput struct {
	DirectoryID uint
	Filename    string
	Size        int64
	Reader      io.Reader
	Publish     bool
}

type ImportFailure struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

type ImportMeta struct {
	Root      string          `json:"root,omitempty"`
	FileCount int             `json:"file_count,omitempty"`
	Files     []string        `json:"files,omitempty"`
	Skipped   []string        `json:"skipped,omitempty"`
	Failures  []ImportFailure `json:"failures,omitempty"`
}

type ImportJobView struct {
	ID               uint       `json:"id"`
	DirectoryID      uint       `json:"directory_id"`
	OriginalFilename string     `json:"original_filename"`
	FileSize         int64      `json:"file_size"`
	Sha256           string     `json:"sha256"`
	Status           string     `json:"status"`
	ExtractedRoot    string     `json:"extracted_root"`
	MDCount          int        `json:"md_count"`
	ImageCount       int        `json:"image_count"`
	ReplacedCount    int        `json:"replaced_count"`
	ErrorMessage     string     `json:"error_message"`
	Meta             ImportMeta `json:"meta"`
	CreatedAt        time.Time  `json:"created_at"`
	FinishedAt       *time.Time `json:"finished_at"`
}

// ImportZip 同步执行一次 ZIP 导入。处理失败会写入 import_jobs 并返回任务详情，
// 只有请求参数/目录校验失败才直接返回错误。
func (s *ImportService) ImportZip(ctx context.Context, in ZipImportInput) (*ImportJobView, error) {
	if in.DirectoryID == 0 {
		return nil, apperr.BadRequest("directory_id 不能为空")
	}
	filename := path.Base(strings.ReplaceAll(strings.TrimSpace(in.Filename), "\\", "/"))
	if filename == "" || filename == "." {
		return nil, apperr.BadRequest("压缩包文件名不能为空")
	}
	if in.Reader == nil {
		return nil, apperr.BadRequest("压缩包内容不能为空")
	}
	if in.Size > MaxZipUploadSize {
		return nil, apperr.BadRequest(fmt.Sprintf("压缩包不能超过 %dMB", MaxZipUploadSize>>20))
	}
	if _, err := validateArticleDirectory(s.db, in.DirectoryID); err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "blog-import-*")
	if err != nil {
		return nil, apperr.Internal("创建临时目录失败", err)
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "upload.zip")
	sha, size, err := saveUpload(in.Reader, zipPath, MaxZipUploadSize)
	if err != nil {
		return nil, err
	}

	job := model.ImportJob{
		DirectoryID:      in.DirectoryID,
		OriginalFilename: filename,
		FileSize:         size,
		Sha256:           sha,
		Status:           "processing",
		Meta:             "{}",
	}
	if err := s.db.Create(&job).Error; err != nil {
		return nil, apperr.Internal("创建导入任务失败", err)
	}

	var (
		imageCount int
		mdCount    int
		replaced   int
		meta       ImportMeta
	)

	finish := func(status, errorMessage string) *ImportJobView {
		now := time.Now().UTC()
		if len(meta.Files) > maxMetaFileList {
			meta.Files = meta.Files[:maxMetaFileList]
		}
		if len(meta.Failures) > maxMetaFailureList {
			meta.Failures = append(meta.Failures[:maxMetaFailureList], ImportFailure{Reason: "其余失败已省略"})
		}
		if len(meta.Skipped) > maxMetaFailureList {
			meta.Skipped = append(meta.Skipped[:maxMetaFailureList], "其余跳过项已省略")
		}

		job.Status = status
		job.ErrorMessage = errorMessage
		job.MDCount = mdCount
		job.ImageCount = imageCount
		job.ReplacedCount = replaced
		job.FinishedAt = &now
		if raw, err := json.Marshal(meta); err == nil {
			job.Meta = string(raw)
		}
		if err := s.db.Save(&job).Error; err != nil {
			log.Printf("更新导入任务 %d 失败: %v", job.ID, err)
		}
		view := toImportJobView(job)
		return &view
	}

	extractDir := filepath.Join(tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return finish("failed", "创建解压目录失败: "+err.Error()), nil
	}

	files, err := extractZip(zipPath, extractDir)
	if err != nil {
		return finish("failed", err.Error()), nil
	}
	meta.FileCount = len(files)
	meta.Files = append(meta.Files, files...)

	root, rootRel, err := effectiveRoot(extractDir)
	if err != nil {
		return finish("failed", err.Error()), nil
	}
	job.ExtractedRoot = rootRel
	meta.Root = rootRel

	mdFiles, imagesDir, err := validateRoot(root)
	if err != nil {
		return finish("failed", err.Error()), nil
	}
	sort.Strings(mdFiles)

	imageMap := make(map[string]string)
	lowerImageMap := make(map[string]string)
	if imagesDir != "" {
		walkErr := filepath.WalkDir(imagesDir, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(imagesDir, filePath)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Size() > MaxImageUploadSize {
				meta.Skipped = append(meta.Skipped, rel+"（超过 10MB）")
				return nil
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return err
			}
			view, err := s.assets.Upload(ctx, UploadAssetInput{OriginalName: rel, Data: data})
			if err != nil {
				if errors.Is(err, ErrNotImage) {
					meta.Skipped = append(meta.Skipped, rel+"（不是图片）")
					return nil
				}
				return err
			}
			imageMap[rel] = view.URL
			lowerImageMap[strings.ToLower(rel)] = view.URL
			imageCount++
			return nil
		})
		if walkErr != nil {
			return finish("failed", "上传图片失败: "+walkErr.Error()), nil
		}
	}

	hardFailures := 0
	for _, mdPath := range mdFiles {
		baseName := filepath.Base(mdPath)
		data, err := readFileLimited(mdPath, maxMarkdownSize)
		if err != nil {
			meta.Failures = append(meta.Failures, ImportFailure{File: baseName, Reason: err.Error()})
			hardFailures++
			continue
		}

		newContent, count, failures := replaceImageReferences(string(data), func(name string) (string, bool) {
			if value, ok := imageMap[name]; ok {
				return value, true
			}
			value, ok := lowerImageMap[strings.ToLower(name)]
			return value, ok
		})
		for i := range failures {
			failures[i].File = baseName
		}
		meta.Failures = append(meta.Failures, failures...)
		replaced += count

		fallbackTitle := strings.TrimSuffix(baseName, filepath.Ext(baseName))
		title := extractMarkdownTitle(newContent, fallbackTitle)
		article, err := s.articles.CreateWithOptions(CreateArticleInput{
			DirectoryID: in.DirectoryID,
			Title:       title,
			ContentMD:   newContent,
		}, CreateArticleOptions{
			Source:     "zip",
			SourceFile: baseName,
		})
		if err != nil {
			meta.Failures = append(meta.Failures, ImportFailure{File: baseName, Reason: err.Error()})
			hardFailures++
			continue
		}
		if in.Publish {
			if err := s.articles.Publish(article.ID); err != nil {
				meta.Failures = append(meta.Failures, ImportFailure{File: baseName, Reason: "发布失败: " + err.Error()})
				hardFailures++
				continue
			}
		}
		mdCount++
	}

	if hardFailures > 0 {
		return finish("failed", fmt.Sprintf("有 %d 个 Markdown 文件处理失败", hardFailures)), nil
	}
	return finish("success", ""), nil
}

func (s *ImportService) List(page, pageSize int) (*Page[ImportJobView], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	if err := s.db.Model(&model.ImportJob{}).Count(&total).Error; err != nil {
		return nil, apperr.Internal("统计导入任务失败", err)
	}

	var jobs []model.ImportJob
	if err := s.db.Order("created_at DESC, id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&jobs).Error; err != nil {
		return nil, apperr.Internal("查询导入任务失败", err)
	}

	items := make([]ImportJobView, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, toImportJobView(job))
	}
	return &Page[ImportJobView]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ImportService) Get(id uint) (*ImportJobView, error) {
	var job model.ImportJob
	if err := s.db.First(&job, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("导入任务不存在")
		}
		return nil, apperr.Internal("查询导入任务失败", err)
	}
	view := toImportJobView(job)
	return &view, nil
}

func toImportJobView(job model.ImportJob) ImportJobView {
	meta := ImportMeta{}
	if job.Meta != "" {
		_ = json.Unmarshal([]byte(job.Meta), &meta)
	}
	return ImportJobView{
		ID:               job.ID,
		DirectoryID:      job.DirectoryID,
		OriginalFilename: job.OriginalFilename,
		FileSize:         job.FileSize,
		Sha256:           job.Sha256,
		Status:           job.Status,
		ExtractedRoot:    job.ExtractedRoot,
		MDCount:          job.MDCount,
		ImageCount:       job.ImageCount,
		ReplacedCount:    job.ReplacedCount,
		ErrorMessage:     job.ErrorMessage,
		Meta:             meta,
		CreatedAt:        job.CreatedAt,
		FinishedAt:       job.FinishedAt,
	}
}

func saveUpload(src io.Reader, dst string, limit int64) (string, int64, error) {
	file, err := os.Create(dst)
	if err != nil {
		return "", 0, apperr.Internal("保存上传文件失败", err)
	}
	defer file.Close()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(src, limit+1))
	if err != nil {
		return "", 0, apperr.Internal("保存上传文件失败", err)
	}
	if written > limit {
		return "", 0, apperr.BadRequest(fmt.Sprintf("压缩包不能超过 %dMB", limit>>20))
	}
	if written == 0 {
		return "", 0, apperr.BadRequest("压缩包内容为空")
	}
	return hex.EncodeToString(hasher.Sum(nil)), written, nil
}

func extractZip(zipPath, destination string) ([]string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("打开压缩包失败: %w", err)
	}
	defer reader.Close()

	if len(reader.File) > maxZipEntries {
		return nil, fmt.Errorf("压缩包条目不能超过 %d 个", maxZipEntries)
	}

	var (
		total int64
		files []string
	)
	for _, f := range reader.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if name == "" {
			continue
		}
		if strings.ContainsRune(name, 0) {
			return nil, errors.New("压缩包包含非法文件名")
		}
		if strings.HasPrefix(name, "/") || driveLetterPattern.MatchString(name) || filepath.IsAbs(name) {
			return nil, fmt.Errorf("压缩包包含非法绝对路径: %s", f.Name)
		}

		clean := path.Clean(name)
		if clean == "." {
			continue
		}
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fmt.Errorf("压缩包包含路径穿越: %s", f.Name)
		}

		target := filepath.Join(destination, filepath.FromSlash(clean))
		rel, err := filepath.Rel(destination, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("压缩包包含路径穿越: %s", f.Name)
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("压缩包不允许符号链接: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, fmt.Errorf("创建解压目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, fmt.Errorf("创建解压目录失败: %w", err)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("读取压缩包条目失败: %w", err)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("写入解压文件失败: %w", err)
		}
		remaining := int64(maxExtractedSize) - total
		written, copyErr := io.Copy(out, io.LimitReader(rc, remaining+1))
		closeErr := out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("写入解压文件失败: %w", copyErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("写入解压文件失败: %w", closeErr)
		}
		if written > remaining {
			return nil, errors.New("压缩包解压后体积过大")
		}
		total += written
		files = append(files, clean)
	}
	return files, nil
}

func effectiveRoot(destination string) (string, string, error) {
	current := destination
	rel := "."
	for depth := 0; depth < maxRootDepth; depth++ {
		entries, err := os.ReadDir(current)
		if err != nil {
			return "", "", fmt.Errorf("读取解压目录失败: %w", err)
		}
		if len(entries) == 1 && entries[0].IsDir() {
			current = filepath.Join(current, entries[0].Name())
			if rel == "." {
				rel = entries[0].Name()
			} else {
				rel = path.Join(rel, entries[0].Name())
			}
			continue
		}
		return current, rel, nil
	}
	return "", "", errors.New("压缩包目录层级过深")
}

func validateRoot(root string) ([]string, string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, "", fmt.Errorf("读取解压根目录失败: %w", err)
	}
	if len(entries) == 0 {
		return nil, "", errors.New("压缩包根目录为空")
	}

	var (
		mdFiles   []string
		imagesDir string
	)
	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() != "images" {
				return nil, "", fmt.Errorf("根层级只允许 images 目录，发现目录 %q", entry.Name())
			}
			if imagesDir != "" {
				return nil, "", errors.New("根层级只能有一个 images 目录")
			}
			imagesDir = filepath.Join(root, entry.Name())
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil, "", fmt.Errorf("根层级只允许 .md 文件与 images 目录，发现文件 %q", entry.Name())
		}
		mdFiles = append(mdFiles, filepath.Join(root, entry.Name()))
	}
	if len(mdFiles) == 0 {
		return nil, "", errors.New("压缩包根层级至少需要一个 .md 文件")
	}
	return mdFiles, imagesDir, nil
}

var imageRefPattern = regexp.MustCompile(`!\[([^\]]*)\]\(\s*\.?/?images/([^)\s]+)\s*\)`)

func replaceImageReferences(content string, lookup func(string) (string, bool)) (string, int, []ImportFailure) {
	replaced := 0
	failures := make([]ImportFailure, 0)

	result := imageRefPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := imageRefPattern.FindStringSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		alt := sub[1]
		rawPath := sub[2]

		name := strings.ReplaceAll(rawPath, "\\", "/")
		name = path.Clean(name)
		name = strings.TrimPrefix(name, "./")
		name = strings.TrimPrefix(name, "/")

		urlValue, ok := lookup(name)
		if !ok {
			if unescaped, err := url.PathUnescape(name); err == nil {
				urlValue, ok = lookup(unescaped)
			}
		}
		if !ok {
			failures = append(failures, ImportFailure{Reason: "图片未找到: " + rawPath})
			return match
		}
		replaced++
		return "![" + alt + "](" + urlValue + ")"
	})
	return result, replaced, failures
}

func extractMarkdownTitle(content, fallback string) string {
	inFence := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			title = strings.TrimSpace(strings.TrimRight(title, "#"))
			if title != "" {
				return title
			}
		}
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "未命名文章"
	}
	return fallback
}

func readFileLimited(filePath string, limit int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("文件超过 %dMB", limit>>20)
	}
	return data, nil
}

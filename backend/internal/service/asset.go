package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

// MaxImageUploadSize 是单张图片的大小上限。
const MaxImageUploadSize = 10 << 20 // 10MB

// ErrNotImage 表示上传内容不是可识别的图片。
var ErrNotImage = errors.New("不是图片文件")

// ObjectUploader 抽象对象存储上传，便于测试时注入假实现。
type ObjectUploader interface {
	Upload(ctx context.Context, key string, data []byte, mime string) (string, error)
}

type AssetService struct {
	db       *gorm.DB
	uploader ObjectUploader
	bucket   string
}

func NewAssetService(db *gorm.DB, uploader ObjectUploader, bucket string) *AssetService {
	return &AssetService{db: db, uploader: uploader, bucket: bucket}
}

type UploadAssetInput struct {
	OriginalName string
	Data         []byte
}

// Upload 计算 sha256 去重，必要时上传七牛云并写入 assets。
func (s *AssetService) Upload(ctx context.Context, in UploadAssetInput) (*AssetView, error) {
	if len(in.Data) == 0 {
		return nil, apperr.BadRequest("图片内容不能为空")
	}
	if len(in.Data) > MaxImageUploadSize {
		return nil, apperr.BadRequest("图片不能超过 10MB")
	}

	mimeType, ext, err := detectImage(in.OriginalName, in.Data)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(in.Data)
	sha := hex.EncodeToString(sum[:])

	var existing model.Asset
	err = s.db.Where("sha256 = ?", sha).First(&existing).Error
	if err == nil {
		view := toAssetView(existing)
		return &view, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Internal("查询资源失败", err)
	}

	if s.uploader == nil {
		return nil, apperr.Unavailable("七牛云图片上传未配置，请在 backend/.env 填写 QINIU_ACCESS_KEY、QINIU_SECRET_KEY、QINIU_ASSET_BUCKET、QINIU_ASSET_DOMAIN")
	}

	key := fmt.Sprintf("articles/%s/%s%s", time.Now().UTC().Format("2006/01"), sha, ext)
	url, err := s.uploader.Upload(ctx, key, in.Data, mimeType)
	if err != nil {
		return nil, apperr.Internal("上传图片失败", err)
	}

	asset := model.Asset{
		Provider:     "qiniu",
		Bucket:       s.bucket,
		ObjectKey:    key,
		URL:          url,
		OriginalName: path.Base(strings.ReplaceAll(in.OriginalName, "\\", "/")),
		Mime:         mimeType,
		Size:         int64(len(in.Data)),
		Sha256:       sha,
	}
	if err := s.db.Create(&asset).Error; err != nil {
		return nil, apperr.Internal("保存资源记录失败", err)
	}

	view := toAssetView(asset)
	return &view, nil
}

// Delete 软删除资源；仍被文章引用时拒绝。
func (s *AssetService) Delete(id uint) error {
	var asset model.Asset
	if err := s.db.First(&asset, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.NotFound("资源不存在")
		}
		return apperr.Internal("查询资源失败", err)
	}
	if asset.RefCount > 0 {
		return apperr.Conflict("资源仍被文章引用，无法删除")
	}
	if err := s.db.Delete(&asset).Error; err != nil {
		return apperr.Internal("删除资源失败", err)
	}
	return nil
}

var imageExtMIME = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".svg":  "image/svg+xml",
	".bmp":  "image/bmp",
	".ico":  "image/x-icon",
	".avif": "image/avif",
}

func detectImage(name string, data []byte) (mimeType, ext string, err error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	ext = strings.ToLower(path.Ext(normalized))

	if ext == ".svg" {
		return "image/svg+xml", ext, nil
	}

	detected := http.DetectContentType(data)
	if strings.HasPrefix(detected, "image/") {
		if ext == "" || imageExtMIME[ext] == "" {
			ext = extFromMIME(detected)
		}
		return detected, ext, nil
	}

	if mimeByExt, ok := imageExtMIME[ext]; ok {
		return mimeByExt, ext, nil
	}
	return "", "", fmt.Errorf("%w: %s", ErrNotImage, name)
}

func extFromMIME(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	case "image/x-icon":
		return ".ico"
	case "image/avif":
		return ".avif"
	default:
		return ".bin"
	}
}

var urlPattern = regexp.MustCompile(`https?://[^\s)\]"'<>]+`)

func extractURLs(content string) []string {
	matches := urlPattern.FindAllString(content, -1)
	urls := make([]string, 0, len(matches))
	for _, raw := range matches {
		url := strings.TrimRight(raw, ".,;:!?")
		if url != "" {
			urls = append(urls, url)
		}
	}
	return urls
}

// syncAssetRefs 按文章正文中的 URL 重新统计资源被多少篇文章引用。
// 传入旧正文与新正文，可覆盖更新/删除场景。
func syncAssetRefs(tx *gorm.DB, contents ...string) error {
	urlSet := make(map[string]struct{})
	for _, content := range contents {
		for _, url := range extractURLs(content) {
			urlSet[url] = struct{}{}
		}
	}
	if len(urlSet) == 0 {
		return nil
	}

	urls := make([]string, 0, len(urlSet))
	for url := range urlSet {
		urls = append(urls, url)
	}

	var existing []string
	if err := tx.Model(&model.Asset{}).Where("url IN ?", urls).Pluck("url", &existing).Error; err != nil {
		return apperr.Internal("查询资源引用失败", err)
	}
	for _, url := range existing {
		var count int64
		if err := tx.Model(&model.Article{}).Where("instr(content_md, ?) > 0", url).Count(&count).Error; err != nil {
			return apperr.Internal("统计资源引用失败", err)
		}
		if err := tx.Model(&model.Asset{}).Where("url = ?", url).
			UpdateColumn("ref_count", count).Error; err != nil {
			return apperr.Internal("更新资源引用计数失败", err)
		}
	}
	return nil
}

// recalcAssetRefs 全量重算资源引用计数，返回涉及的资源数量。
func recalcAssetRefs(tx *gorm.DB) (int, error) {
	if err := tx.Model(&model.Asset{}).Where("deleted_at IS NULL").
		UpdateColumn("ref_count", 0).Error; err != nil {
		return 0, apperr.Internal("重置资源引用计数失败", err)
	}

	var articles []model.Article
	if err := tx.Select("id", "content_md").Find(&articles).Error; err != nil {
		return 0, apperr.Internal("查询文章正文失败", err)
	}

	counts := make(map[string]int64)
	for _, article := range articles {
		seen := make(map[string]struct{})
		for _, url := range extractURLs(article.ContentMD) {
			seen[url] = struct{}{}
		}
		for url := range seen {
			counts[url]++
		}
	}
	if len(counts) == 0 {
		return 0, nil
	}

	urls := make([]string, 0, len(counts))
	for url := range counts {
		urls = append(urls, url)
	}
	var assets []model.Asset
	if err := tx.Where("url IN ?", urls).Find(&assets).Error; err != nil {
		return 0, apperr.Internal("查询资源失败", err)
	}
	for _, asset := range assets {
		if err := tx.Model(&model.Asset{}).Where("id = ?", asset.ID).
			UpdateColumn("ref_count", counts[asset.URL]).Error; err != nil {
			return 0, apperr.Internal("更新资源引用计数失败", err)
		}
	}
	return len(assets), nil
}

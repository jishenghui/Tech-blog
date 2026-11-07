package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

const (
	ArticleStatusDraft     = "draft"
	ArticleStatusPublished = "published"
)

var articleSlugPattern = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

type ArticleService struct {
	db       *gorm.DB
	renderer *MarkdownRenderer
}

func NewArticleService(db *gorm.DB, renderer *MarkdownRenderer) *ArticleService {
	return &ArticleService{db: db, renderer: renderer}
}

type CreateArticleInput struct {
	DirectoryID uint   `json:"directory_id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Summary     string `json:"summary"`
	ContentMD   string `json:"content_md"`
	CoverURL    string `json:"cover_url"`
	IsPinned    bool   `json:"is_pinned"`
	TagIDs      []uint `json:"tag_ids"`
}

type UpdateArticleInput struct {
	Title     *string `json:"title"`
	Slug      *string `json:"slug"`
	Summary   *string `json:"summary"`
	ContentMD *string `json:"content_md"`
	CoverURL  *string `json:"cover_url"`
	IsPinned  *bool   `json:"is_pinned"`
	TagIDs    *[]uint `json:"tag_ids"`
}

type ArticleFilter struct {
	DirectoryID *uint
	Status      string
	TagSlug     string
	Q           string
	Page        int
	PageSize    int
}

type RecalcResult struct {
	Articles    int `json:"articles"`
	Directories int `json:"directories"`
	Tags        int `json:"tags"`
}

func (s *ArticleService) Create(in CreateArticleInput) (*model.Article, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, apperr.BadRequest("文章标题不能为空")
	}
	if utf8.RuneCountInString(title) > 255 {
		return nil, apperr.BadRequest("文章标题不能超过 255 个字符")
	}
	if in.DirectoryID == 0 {
		return nil, apperr.BadRequest("必须指定二级目录")
	}

	rendered, err := s.renderer.Render(title, in.ContentMD)
	if err != nil {
		return nil, apperr.Internal("渲染 Markdown 失败", err)
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = rendered.Excerpt
	}

	providedSlug := strings.TrimSpace(in.Slug)
	var article model.Article
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if _, err := validateArticleDirectory(tx, in.DirectoryID); err != nil {
			return err
		}
		slug, err := resolveNewArticleSlug(tx, title, providedSlug)
		if err != nil {
			return err
		}
		tagIDs, err := validateTagIDs(tx, in.TagIDs)
		if err != nil {
			return err
		}

		directoryID := in.DirectoryID
		article = model.Article{
			DirectoryID: &directoryID,
			Title:       title,
			Slug:        slug,
			Summary:     summary,
			ContentMD:   in.ContentMD,
			ContentHTML: rendered.HTML,
			CoverURL:    strings.TrimSpace(in.CoverURL),
			Status:      ArticleStatusDraft,
			IsPinned:    in.IsPinned,
			Source:      "editor",
			WordCount:   rendered.WordCount,
			ReadingTime: rendered.ReadingTime,
			SearchText:  rendered.SearchText,
		}
		if err := tx.Create(&article).Error; err != nil {
			return apperr.Internal("创建文章失败", err)
		}
		return replaceArticleTags(tx, article.ID, tagIDs, false)
	})
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func (s *ArticleService) Update(id uint, in UpdateArticleInput) (*model.Article, error) {
	var article model.Article
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&article, id).Error; err != nil {
			return articleNotFound(err)
		}

		title := article.Title
		if in.Title != nil {
			title = strings.TrimSpace(*in.Title)
			if title == "" {
				return apperr.BadRequest("文章标题不能为空")
			}
			if utf8.RuneCountInString(title) > 255 {
				return apperr.BadRequest("文章标题不能超过 255 个字符")
			}
		}

		content := article.ContentMD
		if in.ContentMD != nil {
			content = *in.ContentMD
		}
		rendered, err := s.renderer.Render(title, content)
		if err != nil {
			return apperr.Internal("渲染 Markdown 失败", err)
		}

		slug := article.Slug
		if in.Slug != nil {
			slug, err = validateArticleSlug(*in.Slug)
			if err != nil {
				return err
			}
			if slug != article.Slug {
				if err := ensureArticleSlugAvailable(tx, slug, article.ID); err != nil {
					return err
				}
			}
		}

		summary := article.Summary
		if in.Summary != nil {
			summary = strings.TrimSpace(*in.Summary)
		}
		if summary == "" {
			summary = rendered.Excerpt
		}

		coverURL := article.CoverURL
		if in.CoverURL != nil {
			coverURL = strings.TrimSpace(*in.CoverURL)
		}

		updates := map[string]any{
			"title":        title,
			"slug":         slug,
			"summary":      summary,
			"content_md":   content,
			"content_html": rendered.HTML,
			"cover_url":    coverURL,
			"word_count":   rendered.WordCount,
			"reading_time": rendered.ReadingTime,
			"search_text":  rendered.SearchText,
		}
		if in.IsPinned != nil {
			updates["is_pinned"] = *in.IsPinned
		}
		if err := tx.Model(&article).Updates(updates).Error; err != nil {
			return apperr.Internal("保存文章失败", err)
		}

		article.Title = title
		article.Slug = slug
		article.Summary = summary
		article.ContentMD = content
		article.ContentHTML = rendered.HTML
		article.CoverURL = coverURL
		article.WordCount = rendered.WordCount
		article.ReadingTime = rendered.ReadingTime
		article.SearchText = rendered.SearchText
		if in.IsPinned != nil {
			article.IsPinned = *in.IsPinned
		}

		if in.TagIDs != nil {
			tagIDs, err := validateTagIDs(tx, *in.TagIDs)
			if err != nil {
				return err
			}
			if err := replaceArticleTags(tx, article.ID, tagIDs, article.Status == ArticleStatusPublished); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func (s *ArticleService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var article model.Article
		if err := tx.First(&article, id).Error; err != nil {
			return articleNotFound(err)
		}
		if article.Status == ArticleStatusPublished {
			if err := decrementArticleCounts(tx, &article); err != nil {
				return err
			}
		}
		if err := tx.Delete(&article).Error; err != nil {
			return apperr.Internal("删除文章失败", err)
		}
		return nil
	})
}

func (s *ArticleService) Publish(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var article model.Article
		if err := tx.First(&article, id).Error; err != nil {
			return articleNotFound(err)
		}
		if article.Status == ArticleStatusPublished {
			return nil
		}
		if article.DirectoryID == nil {
			return apperr.BadRequest("文章未挂载到二级目录，无法发布")
		}
		if _, err := validateArticleDirectory(tx, *article.DirectoryID); err != nil {
			return err
		}

		rendered, err := s.renderer.Render(article.Title, article.ContentMD)
		if err != nil {
			return apperr.Internal("渲染 Markdown 失败", err)
		}
		summary := article.Summary
		if summary == "" {
			summary = rendered.Excerpt
		}
		now := time.Now().UTC()
		publishedAt := article.PublishedAt
		if publishedAt == nil {
			publishedAt = &now
		}

		updates := map[string]any{
			"status":       ArticleStatusPublished,
			"content_html": rendered.HTML,
			"word_count":   rendered.WordCount,
			"reading_time": rendered.ReadingTime,
			"search_text":  rendered.SearchText,
			"summary":      summary,
			"published_at": publishedAt,
		}
		if err := tx.Model(&article).Updates(updates).Error; err != nil {
			return apperr.Internal("发布文章失败", err)
		}
		if err := incrementDirectoryCount(tx, *article.DirectoryID); err != nil {
			return err
		}
		return incrementTagCounts(tx, article.ID)
	})
}

func (s *ArticleService) Unpublish(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var article model.Article
		if err := tx.First(&article, id).Error; err != nil {
			return articleNotFound(err)
		}
		if article.Status != ArticleStatusPublished {
			return nil
		}
		if err := tx.Model(&article).Update("status", ArticleStatusDraft).Error; err != nil {
			return apperr.Internal("取消发布失败", err)
		}
		if article.DirectoryID != nil {
			if err := decrementDirectoryCount(tx, *article.DirectoryID); err != nil {
				return err
			}
		}
		return decrementTagCounts(tx, article.ID)
	})
}

func (s *ArticleService) Move(id, directoryID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var article model.Article
		if err := tx.First(&article, id).Error; err != nil {
			return articleNotFound(err)
		}
		if _, err := validateArticleDirectory(tx, directoryID); err != nil {
			return err
		}
		if article.DirectoryID != nil && *article.DirectoryID == directoryID {
			return nil
		}
		if article.Status == ArticleStatusPublished {
			if article.DirectoryID != nil {
				if err := decrementDirectoryCount(tx, *article.DirectoryID); err != nil {
					return err
				}
			}
			if err := incrementDirectoryCount(tx, directoryID); err != nil {
				return err
			}
		}
		if err := tx.Model(&article).Update("directory_id", directoryID).Error; err != nil {
			return apperr.Internal("移动文章失败", err)
		}
		return nil
	})
}

func (s *ArticleService) SetTags(id uint, tagIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var article model.Article
		if err := tx.First(&article, id).Error; err != nil {
			return articleNotFound(err)
		}
		ids, err := validateTagIDs(tx, tagIDs)
		if err != nil {
			return err
		}
		return replaceArticleTags(tx, article.ID, ids, article.Status == ArticleStatusPublished)
	})
}

func (s *ArticleService) AdminDetail(id uint) (*AdminArticleDetail, error) {
	var article model.Article
	if err := s.db.First(&article, id).Error; err != nil {
		return nil, articleNotFound(err)
	}

	tagMap, err := s.tagsForArticles([]uint{article.ID})
	if err != nil {
		return nil, err
	}
	dirMap, err := s.directoriesForArticles([]model.Article{article})
	if err != nil {
		return nil, err
	}

	var directory *model.Directory
	if article.DirectoryID != nil {
		directory = dirMap[*article.DirectoryID]
	}
	detail := &AdminArticleDetail{
		ArticleSummary: toArticleSummary(article, tagMap[article.ID], directory),
		ContentMD:      article.ContentMD,
		ContentHTML:    article.ContentHTML,
		TagIDs:         []uint{},
	}
	for _, tag := range tagMap[article.ID] {
		detail.TagIDs = append(detail.TagIDs, tag.ID)
	}
	return detail, nil
}

func (s *ArticleService) AdminList(filter ArticleFilter) (*Page[ArticleSummary], error) {
	filter.normalize()
	if filter.Status != "" && filter.Status != ArticleStatusDraft && filter.Status != ArticleStatusPublished {
		return nil, apperr.BadRequest("status 只能是 draft 或 published")
	}

	countQuery, err := s.buildArticleQuery(filter, true)
	if err != nil {
		return nil, err
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, apperr.Internal("统计文章失败", err)
	}

	findQuery, err := s.buildArticleQuery(filter, true)
	if err != nil {
		return nil, err
	}
	var articles []model.Article
	if err := findQuery.Order("articles.updated_at DESC, articles.id DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&articles).Error; err != nil {
		return nil, apperr.Internal("查询文章失败", err)
	}

	items, err := s.toArticleSummaries(articles, true)
	if err != nil {
		return nil, err
	}
	return &Page[ArticleSummary]{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (s *ArticleService) PublicList(filter ArticleFilter) (*Page[ArticleSummary], error) {
	filter.normalize()

	countQuery, err := s.buildArticleQuery(filter, false)
	if err != nil {
		return nil, err
	}
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, apperr.Internal("统计文章失败", err)
	}

	findQuery, err := s.buildArticleQuery(filter, false)
	if err != nil {
		return nil, err
	}
	var articles []model.Article
	if err := findQuery.Order("articles.is_pinned DESC, articles.published_at DESC, articles.id DESC").
		Limit(filter.PageSize).
		Offset((filter.Page - 1) * filter.PageSize).
		Find(&articles).Error; err != nil {
		return nil, apperr.Internal("查询文章失败", err)
	}

	items, err := s.toArticleSummaries(articles, false)
	if err != nil {
		return nil, err
	}
	return &Page[ArticleSummary]{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func (s *ArticleService) PublicDetail(slug string) (*ArticleDetail, error) {
	var article model.Article
	err := s.db.Where("slug = ? AND status = ?", strings.ToLower(strings.TrimSpace(slug)), ArticleStatusPublished).
		First(&article).Error
	if err != nil {
		return nil, articleNotFound(err)
	}

	tagMap, err := s.tagsForArticles([]uint{article.ID})
	if err != nil {
		return nil, err
	}
	dirMap, err := s.directoriesForArticles([]model.Article{article})
	if err != nil {
		return nil, err
	}

	var directory *model.Directory
	if article.DirectoryID != nil {
		directory = dirMap[*article.DirectoryID]
	}
	summary := toArticleSummary(article, tagMap[article.ID], directory)
	summary.Status = ""
	summary.Source = ""

	return &ArticleDetail{
		ArticleSummary: summary,
		ContentHTML:    article.ContentHTML,
	}, nil
}

// Recalc 全量重算文章派生字段、目录已发布计数与标签计数。
func (s *ArticleService) Recalc() (*RecalcResult, error) {
	result := &RecalcResult{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var articles []model.Article
		if err := tx.Find(&articles).Error; err != nil {
			return apperr.Internal("查询文章失败", err)
		}
		for i := range articles {
			article := &articles[i]
			rendered, err := s.renderer.Render(article.Title, article.ContentMD)
			if err != nil {
				return apperr.Internal("渲染文章失败", err)
			}
			summary := article.Summary
			if summary == "" {
				summary = rendered.Excerpt
			}
			if err := tx.Model(article).Updates(map[string]any{
				"content_html": rendered.HTML,
				"word_count":   rendered.WordCount,
				"reading_time": rendered.ReadingTime,
				"search_text":  rendered.SearchText,
				"summary":      summary,
			}).Error; err != nil {
				return apperr.Internal("重算文章失败", err)
			}
		}
		result.Articles = len(articles)

		// GORM 默认禁止无 WHERE 的全表更新；这里显式限定范围。
		if err := tx.Model(&model.Directory{}).Where("deleted_at IS NULL").
			UpdateColumn("article_count", 0).Error; err != nil {
			return apperr.Internal("重置目录计数失败", err)
		}
		type directoryCount struct {
			DirectoryID uint
			Count       int64
		}
		var directoryCounts []directoryCount
		if err := tx.Model(&model.Article{}).
			Select("directory_id, count(*) AS count").
			Where("status = ? AND directory_id IS NOT NULL", ArticleStatusPublished).
			Group("directory_id").
			Scan(&directoryCounts).Error; err != nil {
			return apperr.Internal("统计目录文章失败", err)
		}
		for _, row := range directoryCounts {
			if err := tx.Model(&model.Directory{}).Where("id = ?", row.DirectoryID).
				UpdateColumn("article_count", row.Count).Error; err != nil {
				return apperr.Internal("更新目录计数失败", err)
			}
		}
		result.Directories = len(directoryCounts)

		if err := tx.Model(&model.Tag{}).Where("1 = 1").
			UpdateColumn("article_count", 0).Error; err != nil {
			return apperr.Internal("重置标签计数失败", err)
		}
		type tagCount struct {
			TagID uint
			Count int64
		}
		var tagCounts []tagCount
		if err := tx.Table("article_tags").
			Select("article_tags.tag_id, count(*) AS count").
			Joins("JOIN articles ON articles.id = article_tags.article_id").
			Where("articles.status = ? AND articles.deleted_at IS NULL", ArticleStatusPublished).
			Group("article_tags.tag_id").
			Scan(&tagCounts).Error; err != nil {
			return apperr.Internal("统计标签文章失败", err)
		}
		for _, row := range tagCounts {
			if err := tx.Model(&model.Tag{}).Where("id = ?", row.TagID).
				UpdateColumn("article_count", row.Count).Error; err != nil {
				return apperr.Internal("更新标签计数失败", err)
			}
		}
		result.Tags = len(tagCounts)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ArticleService) buildArticleQuery(filter ArticleFilter, admin bool) (*gorm.DB, error) {
	query := s.db.Model(&model.Article{})

	if admin {
		if filter.Status != "" {
			query = query.Where("articles.status = ?", filter.Status)
		}
	} else {
		query = query.Where("articles.status = ?", ArticleStatusPublished)
	}

	if filter.DirectoryID != nil {
		ids, err := subtreeIDs(s.db, *filter.DirectoryID)
		if err != nil {
			return nil, err
		}
		query = query.Where("articles.directory_id IN ?", ids)
	}

	if filter.TagSlug != "" {
		var tag model.Tag
		err := s.db.Where("slug = ?", filter.TagSlug).First(&tag).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperr.NotFound("标签不存在")
			}
			return nil, apperr.Internal("查询标签失败", err)
		}
		query = query.Joins("JOIN article_tags ON article_tags.article_id = articles.id").
			Where("article_tags.tag_id = ?", tag.ID)
	}

	if filter.Q != "" {
		query = query.Where("instr(articles.search_text, ?) > 0", strings.ToLower(filter.Q))
	}
	return query, nil
}

func (s *ArticleService) toArticleSummaries(articles []model.Article, admin bool) ([]ArticleSummary, error) {
	ids := make([]uint, 0, len(articles))
	for _, article := range articles {
		ids = append(ids, article.ID)
	}
	tagMap, err := s.tagsForArticles(ids)
	if err != nil {
		return nil, err
	}
	dirMap, err := s.directoriesForArticles(articles)
	if err != nil {
		return nil, err
	}

	items := make([]ArticleSummary, 0, len(articles))
	for _, article := range articles {
		var directory *model.Directory
		if article.DirectoryID != nil {
			directory = dirMap[*article.DirectoryID]
		}
		summary := toArticleSummary(article, tagMap[article.ID], directory)
		if !admin {
			summary.Status = ""
			summary.Source = ""
		}
		items = append(items, summary)
	}
	return items, nil
}

func (s *ArticleService) tagsForArticles(articleIDs []uint) (map[uint][]model.Tag, error) {
	result := make(map[uint][]model.Tag, len(articleIDs))
	if len(articleIDs) == 0 {
		return result, nil
	}

	type row struct {
		ArticleID    uint
		ID           uint
		Name         string
		Slug         string
		Description  string
		ArticleCount int64
	}
	var rows []row
	if err := s.db.Table("article_tags").
		Select("article_tags.article_id, tags.id, tags.name, tags.slug, tags.description, tags.article_count").
		Joins("JOIN tags ON tags.id = article_tags.tag_id").
		Where("article_tags.article_id IN ?", articleIDs).
		Scan(&rows).Error; err != nil {
		return nil, apperr.Internal("查询文章标签失败", err)
	}
	for _, row := range rows {
		result[row.ArticleID] = append(result[row.ArticleID], model.Tag{
			ID:           row.ID,
			Name:         row.Name,
			Slug:         row.Slug,
			Description:  row.Description,
			ArticleCount: row.ArticleCount,
		})
	}
	return result, nil
}

func (s *ArticleService) directoriesForArticles(articles []model.Article) (map[uint]*model.Directory, error) {
	result := make(map[uint]*model.Directory)
	seen := make(map[uint]struct{})
	ids := make([]uint, 0, len(articles))
	for _, article := range articles {
		if article.DirectoryID == nil {
			continue
		}
		id := *article.DirectoryID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return result, nil
	}

	var dirs []model.Directory
	if err := s.db.Where("id IN ?", ids).Find(&dirs).Error; err != nil {
		return nil, apperr.Internal("查询文章目录失败", err)
	}
	for i := range dirs {
		result[dirs[i].ID] = &dirs[i]
	}
	return result, nil
}

func validateArticleDirectory(tx *gorm.DB, id uint) (*model.Directory, error) {
	var dir model.Directory
	if err := tx.First(&dir, id).Error; err != nil {
		return nil, directoryNotFound(err)
	}
	if dir.ParentID == nil {
		return nil, apperr.BadRequest("文章必须挂在二级目录下")
	}

	var parent model.Directory
	if err := tx.First(&parent, *dir.ParentID).Error; err != nil {
		return nil, directoryNotFound(err)
	}
	if parent.Kind != KindContainer {
		return nil, apperr.BadRequest("文章必须挂在容器目录下")
	}
	return &dir, nil
}

func resolveNewArticleSlug(tx *gorm.DB, title, provided string) (string, error) {
	if provided != "" {
		slug, err := validateArticleSlug(provided)
		if err != nil {
			return "", err
		}
		if err := ensureArticleSlugAvailable(tx, slug, 0); err != nil {
			return "", err
		}
		return slug, nil
	}
	return uniqueArticleSlug(tx, generateSlug(title, "post"), 0)
}

func validateArticleSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" {
		return "", apperr.BadRequest("文章 slug 不能为空")
	}
	if utf8.RuneCountInString(slug) > 255 {
		return "", apperr.BadRequest("文章 slug 不能超过 255 个字符")
	}
	if !articleSlugPattern.MatchString(slug) {
		return "", apperr.BadRequest("文章 slug 只能包含字母、数字、下划线和中划线")
	}
	return slug, nil
}

func ensureArticleSlugAvailable(tx *gorm.DB, slug string, excludeID uint) error {
	var count int64
	query := tx.Model(&model.Article{}).Where("slug = ?", slug)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Internal("检查文章 slug 失败", err)
	}
	if count > 0 {
		return apperr.Conflict("文章 slug 已存在")
	}
	return nil
}

func uniqueArticleSlug(tx *gorm.DB, base string, excludeID uint) (string, error) {
	candidate := base
	for i := 2; i <= 1000; i++ {
		var count int64
		query := tx.Model(&model.Article{}).Where("slug = ?", candidate)
		if excludeID != 0 {
			query = query.Where("id <> ?", excludeID)
		}
		if err := query.Count(&count).Error; err != nil {
			return "", apperr.Internal("检查文章 slug 失败", err)
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return "", apperr.Conflict("无法生成唯一的文章 slug")
}

func generateSlug(raw, fallbackPrefix string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = fmt.Sprintf("%s-%d", fallbackPrefix, time.Now().UnixNano())
	}
	runes := []rune(slug)
	if len(runes) > 200 {
		slug = strings.Trim(string(runes[:200]), "-")
	}
	return slug
}

func replaceArticleTags(tx *gorm.DB, articleID uint, newTagIDs []uint, adjustCounts bool) error {
	var current []uint
	if err := tx.Model(&model.ArticleTag{}).Where("article_id = ?", articleID).Pluck("tag_id", &current).Error; err != nil {
		return apperr.Internal("查询文章标签失败", err)
	}

	currentSet := make(map[uint]struct{}, len(current))
	for _, id := range current {
		currentSet[id] = struct{}{}
	}
	newSet := make(map[uint]struct{}, len(newTagIDs))
	for _, id := range newTagIDs {
		newSet[id] = struct{}{}
	}

	var removed []uint
	for _, id := range current {
		if _, keep := newSet[id]; !keep {
			removed = append(removed, id)
		}
	}
	var added []uint
	for _, id := range newTagIDs {
		if _, exists := currentSet[id]; !exists {
			added = append(added, id)
		}
	}

	if len(removed) > 0 {
		if err := tx.Where("article_id = ? AND tag_id IN ?", articleID, removed).Delete(&model.ArticleTag{}).Error; err != nil {
			return apperr.Internal("移除文章标签失败", err)
		}
	}
	for _, tagID := range added {
		if err := tx.Create(&model.ArticleTag{ArticleID: articleID, TagID: tagID}).Error; err != nil {
			return apperr.Internal("添加文章标签失败", err)
		}
	}
	if adjustCounts {
		for _, tagID := range removed {
			if err := decrementTagCount(tx, tagID); err != nil {
				return err
			}
		}
		for _, tagID := range added {
			if err := incrementTagCount(tx, tagID); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTagIDs(tx *gorm.DB, ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return []uint{}, nil
	}

	seen := make(map[uint]struct{}, len(ids))
	unique := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, apperr.BadRequest("标签 ID 无效")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	var count int64
	if err := tx.Model(&model.Tag{}).Where("id IN ?", unique).Count(&count).Error; err != nil {
		return nil, apperr.Internal("校验标签失败", err)
	}
	if int(count) != len(unique) {
		return nil, apperr.BadRequest("存在无效的标签")
	}
	return unique, nil
}

func incrementDirectoryCount(tx *gorm.DB, directoryID uint) error {
	if err := tx.Model(&model.Directory{}).Where("id = ?", directoryID).
		UpdateColumn("article_count", gorm.Expr("article_count + 1")).Error; err != nil {
		return apperr.Internal("更新目录计数失败", err)
	}
	return nil
}

func decrementDirectoryCount(tx *gorm.DB, directoryID uint) error {
	if err := tx.Model(&model.Directory{}).Where("id = ?", directoryID).
		UpdateColumn("article_count", gorm.Expr("CASE WHEN article_count > 0 THEN article_count - 1 ELSE 0 END")).Error; err != nil {
		return apperr.Internal("更新目录计数失败", err)
	}
	return nil
}

func incrementTagCount(tx *gorm.DB, tagID uint) error {
	if err := tx.Model(&model.Tag{}).Where("id = ?", tagID).
		UpdateColumn("article_count", gorm.Expr("article_count + 1")).Error; err != nil {
		return apperr.Internal("更新标签计数失败", err)
	}
	return nil
}

func decrementTagCount(tx *gorm.DB, tagID uint) error {
	if err := tx.Model(&model.Tag{}).Where("id = ?", tagID).
		UpdateColumn("article_count", gorm.Expr("CASE WHEN article_count > 0 THEN article_count - 1 ELSE 0 END")).Error; err != nil {
		return apperr.Internal("更新标签计数失败", err)
	}
	return nil
}

func incrementTagCounts(tx *gorm.DB, articleID uint) error {
	var tagIDs []uint
	if err := tx.Model(&model.ArticleTag{}).Where("article_id = ?", articleID).Pluck("tag_id", &tagIDs).Error; err != nil {
		return apperr.Internal("查询文章标签失败", err)
	}
	for _, tagID := range tagIDs {
		if err := incrementTagCount(tx, tagID); err != nil {
			return err
		}
	}
	return nil
}

func decrementTagCounts(tx *gorm.DB, articleID uint) error {
	var tagIDs []uint
	if err := tx.Model(&model.ArticleTag{}).Where("article_id = ?", articleID).Pluck("tag_id", &tagIDs).Error; err != nil {
		return apperr.Internal("查询文章标签失败", err)
	}
	for _, tagID := range tagIDs {
		if err := decrementTagCount(tx, tagID); err != nil {
			return err
		}
	}
	return nil
}

func decrementArticleCounts(tx *gorm.DB, article *model.Article) error {
	if article.DirectoryID != nil {
		if err := decrementDirectoryCount(tx, *article.DirectoryID); err != nil {
			return err
		}
	}
	return decrementTagCounts(tx, article.ID)
}

func articleNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.NotFound("文章不存在")
	}
	return apperr.Internal("查询文章失败", err)
}

func directoryNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.NotFound("目录不存在")
	}
	return apperr.Internal("查询目录失败", err)
}

func (f *ArticleFilter) normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	f.Status = strings.TrimSpace(f.Status)
	f.TagSlug = strings.ToLower(strings.TrimSpace(f.TagSlug))
	f.Q = strings.TrimSpace(f.Q)
}

package service

import (
	"time"

	"blog-api/internal/model"
)

// Page 是通用分页结果。
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// DirectoryNode 是后台目录树的节点，含草稿计数与隐藏目录。
type DirectoryNode struct {
	ID           uint            `json:"id"`
	ParentID     *uint           `json:"parent_id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Kind         string          `json:"kind"`
	PageKey      *string         `json:"page_key"`
	IsFixed      bool            `json:"is_fixed"`
	Visible      bool            `json:"visible"`
	SortOrder    int             `json:"sort_order"`
	Description  string          `json:"description"`
	ArticleCount int64           `json:"article_count"`
	DraftCount   int64           `json:"draft_count"`
	Children     []DirectoryNode `json:"children"`
}

// NavNode 是公开导航树节点。
type NavNode struct {
	ID       uint      `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Kind     string    `json:"kind"`
	PageKey  *string   `json:"page_key,omitempty"`
	Children []NavNode `json:"children,omitempty"`
}

// DirectoryDetail 是公开目录信息。
type DirectoryDetail struct {
	ID           uint    `json:"id"`
	ParentID     *uint   `json:"parent_id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Kind         string  `json:"kind"`
	PageKey      *string `json:"page_key,omitempty"`
	Description  string  `json:"description"`
	ArticleCount int64   `json:"article_count"`
}

// DirectoryPage 是目录详情页数据：目录信息 + 文章分页。
type DirectoryPage struct {
	Directory DirectoryDetail  `json:"directory"`
	Items     []ArticleSummary `json:"items"`
	Total     int64            `json:"total"`
	Page      int              `json:"page"`
	PageSize  int              `json:"page_size"`
}

// TagView 是标签的公开表示。
type TagView struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	ArticleCount int64  `json:"article_count"`
}

// DirectoryBrief 用于文章上的目录摘要。
type DirectoryBrief struct {
	ID      uint    `json:"id"`
	Name    string  `json:"name"`
	Slug    string  `json:"slug"`
	PageKey *string `json:"page_key,omitempty"`
}

// ArticleSummary 是文章列表项。
type ArticleSummary struct {
	ID          uint            `json:"id"`
	DirectoryID *uint           `json:"directory_id"`
	Title       string          `json:"title"`
	Slug        string          `json:"slug"`
	Summary     string          `json:"summary"`
	CoverURL    string          `json:"cover_url"`
	Status      string          `json:"status,omitempty"`
	IsPinned    bool            `json:"is_pinned"`
	ViewCount   int64           `json:"view_count"`
	WordCount   int             `json:"word_count"`
	ReadingTime int             `json:"reading_time"`
	Source      string          `json:"source,omitempty"`
	PublishedAt *time.Time      `json:"published_at"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Tags        []TagView       `json:"tags"`
	Directory   *DirectoryBrief `json:"directory,omitempty"`
}

// ArticleDetail 是公开文章详情，包含预渲染 HTML。
type ArticleDetail struct {
	ArticleSummary
	ContentHTML string `json:"content_html"`
}

// AdminArticleDetail 是后台文章详情，包含 Markdown 原文与标签 ID。
type AdminArticleDetail struct {
	ArticleSummary
	ContentMD   string `json:"content_md"`
	ContentHTML string `json:"content_html"`
	TagIDs      []uint `json:"tag_ids"`
}

// AssetView 是上传资源的公开表示。
type AssetView struct {
	ID           uint   `json:"id"`
	URL          string `json:"url"`
	ObjectKey    string `json:"object_key"`
	Bucket       string `json:"bucket"`
	OriginalName string `json:"original_name"`
	Mime         string `json:"mime"`
	Size         int64  `json:"size"`
	Sha256       string `json:"sha256"`
	RefCount     int    `json:"ref_count"`
}

// ToDirectoryDetail 供 handler 构造公开目录响应。
func ToDirectoryDetail(dir model.Directory) DirectoryDetail {
	return toDirectoryDetail(dir)
}

// ToTagView 供 handler 构造单个标签响应。
func ToTagView(tag model.Tag) TagView {
	return toTagView(tag)
}

func toNavNode(dir model.Directory) NavNode {
	return NavNode{
		ID:      dir.ID,
		Name:    dir.Name,
		Slug:    dir.Slug,
		Kind:    dir.Kind,
		PageKey: dir.PageKey,
	}
}

func toDirectoryDetail(dir model.Directory) DirectoryDetail {
	return DirectoryDetail{
		ID:           dir.ID,
		ParentID:     dir.ParentID,
		Name:         dir.Name,
		Slug:         dir.Slug,
		Kind:         dir.Kind,
		PageKey:      dir.PageKey,
		Description:  dir.Description,
		ArticleCount: dir.ArticleCount,
	}
}

func toDirectoryNode(dir model.Directory, articleCount, draftCount int64, children []DirectoryNode) DirectoryNode {
	if children == nil {
		children = []DirectoryNode{}
	}
	return DirectoryNode{
		ID:           dir.ID,
		ParentID:     dir.ParentID,
		Name:         dir.Name,
		Slug:         dir.Slug,
		Kind:         dir.Kind,
		PageKey:      dir.PageKey,
		IsFixed:      dir.IsFixed,
		Visible:      dir.Visible,
		SortOrder:    dir.SortOrder,
		Description:  dir.Description,
		ArticleCount: articleCount,
		DraftCount:   draftCount,
		Children:     children,
	}
}

func toTagView(tag model.Tag) TagView {
	return TagView{
		ID:           tag.ID,
		Name:         tag.Name,
		Slug:         tag.Slug,
		Description:  tag.Description,
		ArticleCount: tag.ArticleCount,
	}
}

func toTagViews(tags []model.Tag) []TagView {
	views := make([]TagView, 0, len(tags))
	for _, tag := range tags {
		views = append(views, toTagView(tag))
	}
	return views
}

func toAssetView(asset model.Asset) AssetView {
	return AssetView{
		ID:           asset.ID,
		URL:          asset.URL,
		ObjectKey:    asset.ObjectKey,
		Bucket:       asset.Bucket,
		OriginalName: asset.OriginalName,
		Mime:         asset.Mime,
		Size:         asset.Size,
		Sha256:       asset.Sha256,
		RefCount:     asset.RefCount,
	}
}

func toArticleSummary(article model.Article, tags []model.Tag, directory *model.Directory) ArticleSummary {
	summary := ArticleSummary{
		ID:          article.ID,
		DirectoryID: article.DirectoryID,
		Title:       article.Title,
		Slug:        article.Slug,
		Summary:     article.Summary,
		CoverURL:    article.CoverURL,
		Status:      article.Status,
		IsPinned:    article.IsPinned,
		ViewCount:   article.ViewCount,
		WordCount:   article.WordCount,
		ReadingTime: article.ReadingTime,
		Source:      article.Source,
		PublishedAt: article.PublishedAt,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
		Tags:        toTagViews(tags),
	}
	if directory != nil {
		summary.Directory = &DirectoryBrief{
			ID:      directory.ID,
			Name:    directory.Name,
			Slug:    directory.Slug,
			PageKey: directory.PageKey,
		}
	}
	return summary
}

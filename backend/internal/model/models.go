package model

import (
	"time"

	"gorm.io/gorm"
)

// User 唯一管理员账号。
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Nickname     string     `gorm:"size:64" json:"nickname"`
	AvatarURL    string     `gorm:"size:512" json:"avatar_url"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// Directory 目录树。parent_id 为 NULL 表示一级目录。
type Directory struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ParentID     *uint          `gorm:"index" json:"parent_id"`
	Name         string         `gorm:"size:128;not null" json:"name"`
	Slug         string         `gorm:"size:128;not null" json:"slug"`
	Kind         string         `gorm:"size:16;not null;default:'container'" json:"kind"`
	PageKey      *string        `gorm:"size:32" json:"page_key"`
	IsFixed      bool           `gorm:"not null;default:false" json:"is_fixed"`
	Visible      bool           `gorm:"not null;default:true" json:"visible"`
	SortOrder    int            `gorm:"not null;default:0" json:"sort_order"`
	Description  string         `gorm:"type:text" json:"description"`
	ArticleCount int64          `gorm:"not null;default:0" json:"article_count"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Article 文章。content_html 等冗余字段由写操作在同一事务内维护。
type Article struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	DirectoryID *uint          `gorm:"index" json:"directory_id"`
	Title       string         `gorm:"size:255;not null" json:"title"`
	Slug        string         `gorm:"size:255;not null" json:"slug"`
	Summary     string         `gorm:"type:text" json:"summary"`
	ContentMD   string         `gorm:"type:text;not null;default:''" json:"content_md,omitempty"`
	ContentHTML string         `gorm:"type:text;not null;default:''" json:"content_html,omitempty"`
	CoverURL    string         `gorm:"size:512" json:"cover_url"`
	Status      string         `gorm:"size:16;not null;default:'draft'" json:"status"`
	IsPinned    bool           `gorm:"not null;default:false" json:"is_pinned"`
	ViewCount   int64          `gorm:"not null;default:0" json:"view_count"`
	WordCount   int            `gorm:"not null;default:0" json:"word_count"`
	ReadingTime int            `gorm:"not null;default:0" json:"reading_time"`
	Source      string         `gorm:"size:16;not null;default:'editor'" json:"source"`
	SourceFile  string         `gorm:"size:255" json:"source_file"`
	SearchText  string         `gorm:"type:text" json:"-"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	PublishedAt *time.Time     `json:"published_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Tag 文章标签，技术栈页按 article_count 直接展示。
type Tag struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:64;not null;uniqueIndex" json:"name"`
	Slug         string    `gorm:"size:64;not null;uniqueIndex" json:"slug"`
	Description  string    `gorm:"type:text" json:"description"`
	ArticleCount int64     `gorm:"not null;default:0" json:"article_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ArticleTag 文章与标签的多对多关联。
type ArticleTag struct {
	ArticleID uint      `gorm:"primaryKey" json:"article_id"`
	TagID     uint      `gorm:"primaryKey;index" json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Asset 图片等静态资源，url 为冗余的完整 CDN 地址。
type Asset struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ArticleID    *uint          `gorm:"index" json:"article_id"`
	DirectoryID  *uint          `gorm:"index" json:"directory_id"`
	Provider     string         `gorm:"size:32;not null;default:'qiniu'" json:"provider"`
	Bucket       string         `gorm:"size:64;not null" json:"bucket"`
	ObjectKey    string         `gorm:"size:512;not null" json:"object_key"`
	URL          string         `gorm:"size:512;not null" json:"url"`
	OriginalName string         `gorm:"size:255" json:"original_name"`
	Mime         string         `gorm:"size:128" json:"mime"`
	Size         int64          `gorm:"not null;default:0" json:"size"`
	Sha256       string         `gorm:"size:64;index" json:"sha256"`
	RefCount     int            `gorm:"not null;default:0" json:"ref_count"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Project 固定页面“项目”下的单个项目。
type Project struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Title        string         `gorm:"size:255;not null" json:"title"`
	Slug         string         `gorm:"size:255;not null" json:"slug"`
	Summary      string         `gorm:"type:text" json:"summary"`
	ContentMD    string         `gorm:"type:text;not null;default:''" json:"content_md,omitempty"`
	ContentHTML  string         `gorm:"type:text;not null;default:''" json:"content_html,omitempty"`
	TechStack    string         `gorm:"type:text;not null;default:'[]'" json:"tech_stack"`
	DebugSteps   string         `gorm:"type:text;not null;default:'[]'" json:"debug_steps"`
	Architecture string         `gorm:"type:text;not null;default:'{}'" json:"architecture"`
	RepoURL      string         `gorm:"size:512" json:"repo_url"`
	DemoURL      string         `gorm:"size:512" json:"demo_url"`
	CoverURL     string         `gorm:"size:512" json:"cover_url"`
	SortOrder    int            `gorm:"not null;default:0" json:"sort_order"`
	Status       string         `gorm:"size:16;not null;default:'published'" json:"status"`
	StartedAt    *time.Time     `json:"started_at"`
	EndedAt      *time.Time     `json:"ended_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Profile 首页资料，全表只有 id=1 一行。
type Profile struct {
	ID               uint      `gorm:"primaryKey;check:id = 1" json:"id"`
	Headline         string    `gorm:"size:255" json:"headline"`
	AvatarURL        string    `gorm:"size:512" json:"avatar_url"`
	ResumeMD         string    `gorm:"type:text;not null;default:''" json:"resume_md"`
	ResumeHTML       string    `gorm:"type:text;not null;default:''" json:"resume_html"`
	Awards           string    `gorm:"type:text;not null;default:'[]'" json:"awards"`
	Experiences      string    `gorm:"type:text;not null;default:'[]'" json:"experiences"`
	FeaturedProjects string    `gorm:"type:text;not null;default:'[]'" json:"featured_projects"`
	Contacts         string    `gorm:"type:text;not null;default:'{}'" json:"contacts"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Profile) TableName() string {
	return "profile"
}

// Todo 待办事项。
type Todo struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Content   string     `gorm:"type:text;not null" json:"content"`
	Done      bool       `gorm:"not null;default:false" json:"done"`
	SortOrder int        `gorm:"not null;default:0" json:"sort_order"`
	DueDate   *time.Time `json:"due_date"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ImportJob 压缩包导入任务。
type ImportJob struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	DirectoryID      uint       `gorm:"not null;index" json:"directory_id"`
	OriginalFilename string     `gorm:"size:255;not null" json:"original_filename"`
	FileSize         int64      `gorm:"not null;default:0" json:"file_size"`
	Sha256           string     `gorm:"size:64;index" json:"sha256"`
	Status           string     `gorm:"size:16;not null;default:'pending'" json:"status"`
	ExtractedRoot    string     `gorm:"size:512" json:"extracted_root"`
	MDCount          int        `gorm:"column:md_count;not null;default:0" json:"md_count"`
	ImageCount       int        `gorm:"not null;default:0" json:"image_count"`
	ReplacedCount    int        `gorm:"not null;default:0" json:"replaced_count"`
	ErrorMessage     string     `gorm:"type:text" json:"error_message"`
	Meta             string     `gorm:"type:text;not null;default:'{}'" json:"meta"`
	CreatedAt        time.Time  `json:"created_at"`
	FinishedAt       *time.Time `json:"finished_at"`
}

// Setting 站点级键值配置。
type Setting struct {
	Key       string    `gorm:"primaryKey;size:128" json:"key"`
	Value     string    `gorm:"type:text;not null" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

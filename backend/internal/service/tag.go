package service

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

var tagSlugPattern = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

type TagService struct {
	db *gorm.DB
}

func NewTagService(db *gorm.DB) *TagService {
	return &TagService{db: db}
}

type CreateTagInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type UpdateTagInput struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

func (s *TagService) Create(in CreateTagInput) (*model.Tag, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperr.BadRequest("标签名称不能为空")
	}
	if utf8.RuneCountInString(name) > 64 {
		return nil, apperr.BadRequest("标签名称不能超过 64 个字符")
	}

	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = generateSlug(name, "tag")
		if runes := []rune(slug); len(runes) > 64 {
			slug = strings.Trim(string(runes[:64]), "-")
		}
		if slug == "" {
			slug = generateSlug("", "tag")
		}
	} else {
		var err error
		slug, err = validateTagSlug(slug)
		if err != nil {
			return nil, err
		}
	}

	var tag model.Tag
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := ensureTagNameAvailable(tx, name, 0); err != nil {
			return err
		}
		if err := ensureTagSlugAvailable(tx, slug, 0); err != nil {
			return err
		}
		tag = model.Tag{
			Name:        name,
			Slug:        slug,
			Description: strings.TrimSpace(in.Description),
		}
		if err := tx.Create(&tag).Error; err != nil {
			return apperr.Internal("创建标签失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (s *TagService) Update(id uint, in UpdateTagInput) (*model.Tag, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var tag model.Tag
		if err := tx.First(&tag, id).Error; err != nil {
			return tagNotFound(err)
		}

		updates := map[string]any{}
		if in.Name != nil {
			name := strings.TrimSpace(*in.Name)
			if name == "" {
				return apperr.BadRequest("标签名称不能为空")
			}
			if utf8.RuneCountInString(name) > 64 {
				return apperr.BadRequest("标签名称不能超过 64 个字符")
			}
			if err := ensureTagNameAvailable(tx, name, tag.ID); err != nil {
				return err
			}
			updates["name"] = name
		}
		if in.Slug != nil {
			slug, err := validateTagSlug(*in.Slug)
			if err != nil {
				return err
			}
			if err := ensureTagSlugAvailable(tx, slug, tag.ID); err != nil {
				return err
			}
			updates["slug"] = slug
		}
		if in.Description != nil {
			updates["description"] = strings.TrimSpace(*in.Description)
		}

		if len(updates) > 0 {
			if err := tx.Model(&tag).Updates(updates).Error; err != nil {
				return apperr.Internal("更新标签失败", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var tag model.Tag
	if err := s.db.First(&tag, id).Error; err != nil {
		return nil, tagNotFound(err)
	}
	return &tag, nil
}

func (s *TagService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var tag model.Tag
		if err := tx.First(&tag, id).Error; err != nil {
			return tagNotFound(err)
		}
		if err := tx.Where("tag_id = ?", tag.ID).Delete(&model.ArticleTag{}).Error; err != nil {
			return apperr.Internal("移除文章标签关联失败", err)
		}
		if err := tx.Delete(&tag).Error; err != nil {
			return apperr.Internal("删除标签失败", err)
		}
		return nil
	})
}

func (s *TagService) AdminList() ([]TagView, error) {
	var tags []model.Tag
	if err := s.db.Order("name, id").Find(&tags).Error; err != nil {
		return nil, apperr.Internal("查询标签失败", err)
	}
	return toTagViews(tags), nil
}

func (s *TagService) PublicList() ([]TagView, error) {
	var tags []model.Tag
	if err := s.db.Order("article_count DESC, name, id").Find(&tags).Error; err != nil {
		return nil, apperr.Internal("查询标签失败", err)
	}
	return toTagViews(tags), nil
}

func (s *TagService) GetBySlug(slug string) (*model.Tag, error) {
	var tag model.Tag
	err := s.db.Where("slug = ?", strings.ToLower(strings.TrimSpace(slug))).First(&tag).Error
	if err != nil {
		return nil, tagNotFound(err)
	}
	return &tag, nil
}

func validateTagSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" {
		return "", apperr.BadRequest("标签 slug 不能为空")
	}
	if utf8.RuneCountInString(slug) > 64 {
		return "", apperr.BadRequest("标签 slug 不能超过 64 个字符")
	}
	if !tagSlugPattern.MatchString(slug) {
		return "", apperr.BadRequest("标签 slug 只能包含字母、数字、下划线和中划线")
	}
	return slug, nil
}

func ensureTagNameAvailable(tx *gorm.DB, name string, excludeID uint) error {
	var count int64
	query := tx.Model(&model.Tag{}).Where("name = ?", name)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Internal("检查标签名称失败", err)
	}
	if count > 0 {
		return apperr.Conflict("标签名称已存在")
	}
	return nil
}

func ensureTagSlugAvailable(tx *gorm.DB, slug string, excludeID uint) error {
	var count int64
	query := tx.Model(&model.Tag{}).Where("slug = ?", slug)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Internal("检查标签 slug 失败", err)
	}
	if count > 0 {
		return apperr.Conflict("标签 slug 已存在")
	}
	return nil
}

func tagNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.NotFound("标签不存在")
	}
	return apperr.Internal("查询标签失败", err)
}

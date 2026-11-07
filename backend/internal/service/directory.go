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

const (
	KindPage          = "page"
	KindContainer     = "container"
	PageKeyOther      = "other"
	OtherInnerSlug    = "other-other"
	OtherInnerName    = "其他"
	OtherTopLevelName = "其他"
	OtherTopLevelSlug = "other"
)

var directorySlugPattern = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

// 固定路径与前端保留路径，普通目录不能占用。
var reservedDirectorySlugs = map[string]struct{}{
	"home":        {},
	"projects":    {},
	"tech":        {},
	"todos":       {},
	"other":       {},
	"other-other": {},
	"posts":       {},
	"tags":        {},
	"search":      {},
	"sitemap.xml": {},
	"rss.xml":     {},
	"404.html":    {},
}

type DirectoryService struct {
	db *gorm.DB
}

func NewDirectoryService(db *gorm.DB) *DirectoryService {
	return &DirectoryService{db: db}
}

type CreateDirectoryInput struct {
	ParentID    *uint  `json:"parent_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	SortOrder   *int   `json:"sort_order"`
	Description string `json:"description"`
}

type UpdateDirectoryInput struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Visible     *bool   `json:"visible"`
	SortOrder   *int    `json:"sort_order"`
	Description *string `json:"description"`
}

type ReorderDirectoryItem struct {
	ID        uint  `json:"id"`
	ParentID  *uint `json:"parent_id"`
	SortOrder int   `json:"sort_order"`
}

type ReorderDirectoriesInput struct {
	Items []ReorderDirectoryItem `json:"items"`
}

func (s *DirectoryService) Create(in CreateDirectoryInput) (*model.Directory, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apperr.BadRequest("目录名称不能为空")
	}
	if utf8.RuneCountInString(name) > 128 {
		return nil, apperr.BadRequest("目录名称不能超过 128 个字符")
	}

	slug, err := validateDirectorySlug(in.Slug)
	if err != nil {
		return nil, err
	}

	var dir model.Directory
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if in.ParentID != nil {
			if _, err := validateParentDirectory(tx, *in.ParentID); err != nil {
				return err
			}
		}
		if err := ensureDirectorySlugAvailable(tx, slug, 0); err != nil {
			return err
		}

		sortOrder := 0
		if in.SortOrder != nil {
			sortOrder = *in.SortOrder
		} else {
			next, err := nextSortOrder(tx, in.ParentID)
			if err != nil {
				return err
			}
			sortOrder = next
		}

		dir = model.Directory{
			ParentID:    in.ParentID,
			Name:        name,
			Slug:        slug,
			Kind:        KindContainer,
			Visible:     true,
			SortOrder:   sortOrder,
			Description: strings.TrimSpace(in.Description),
		}
		if err := tx.Create(&dir).Error; err != nil {
			return apperr.Internal("创建目录失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &dir, nil
}

func (s *DirectoryService) Update(id uint, in UpdateDirectoryInput) (*model.Directory, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		dir, err := loadDirectory(tx, id)
		if err != nil {
			return err
		}

		updates := map[string]any{}

		if in.Name != nil {
			name := strings.TrimSpace(*in.Name)
			if name == "" {
				return apperr.BadRequest("目录名称不能为空")
			}
			if utf8.RuneCountInString(name) > 128 {
				return apperr.BadRequest("目录名称不能超过 128 个字符")
			}
			if dir.IsFixed && name != dir.Name {
				return apperr.Forbidden("固定目录不可改名")
			}
			updates["name"] = name
		}

		if in.Slug != nil {
			slug, err := validateDirectorySlug(*in.Slug)
			if err != nil {
				return err
			}
			if dir.IsFixed && slug != dir.Slug {
				return apperr.Forbidden("固定目录不可修改 slug")
			}
			if slug != dir.Slug {
				if err := ensureDirectorySlugAvailable(tx, slug, dir.ID); err != nil {
					return err
				}
				updates["slug"] = slug
			}
		}

		if in.Visible != nil {
			updates["visible"] = *in.Visible
		}
		if in.SortOrder != nil {
			updates["sort_order"] = *in.SortOrder
		}
		if in.Description != nil {
			updates["description"] = strings.TrimSpace(*in.Description)
		}

		if len(updates) > 0 {
			if err := tx.Model(dir).Updates(updates).Error; err != nil {
				return apperr.Internal("更新目录失败", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.getByID(id)
}

func (s *DirectoryService) Move(id uint, parentID *uint, sortOrder *int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		dir, err := loadDirectory(tx, id)
		if err != nil {
			return err
		}
		if dir.IsFixed {
			return apperr.Forbidden("固定目录不可移动")
		}

		if dir.ParentID == nil {
			if parentID != nil {
				return apperr.BadRequest("一级目录不能移动到其他目录下")
			}
		} else {
			if parentID == nil {
				return apperr.BadRequest("二级目录必须挂在一级目录下")
			}
			if *parentID == dir.ID {
				return apperr.BadRequest("不能移动到自身")
			}
			if _, err := validateParentDirectory(tx, *parentID); err != nil {
				return err
			}
		}

		order := 0
		if sortOrder != nil {
			order = *sortOrder
		} else {
			next, err := nextSortOrder(tx, parentID)
			if err != nil {
				return err
			}
			order = next
		}

		if err := tx.Model(dir).Updates(map[string]any{
			"parent_id":  parentID,
			"sort_order": order,
		}).Error; err != nil {
			return apperr.Internal("移动目录失败", err)
		}
		return nil
	})
}

func (s *DirectoryService) Reorder(in ReorderDirectoriesInput) error {
	if len(in.Items) == 0 {
		return apperr.BadRequest("排序列表不能为空")
	}

	seen := make(map[uint]struct{}, len(in.Items))
	for _, item := range in.Items {
		if item.ID == 0 {
			return apperr.BadRequest("目录 ID 不能为空")
		}
		if _, ok := seen[item.ID]; ok {
			return apperr.BadRequest("排序列表中存在重复目录")
		}
		seen[item.ID] = struct{}{}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range in.Items {
			dir, err := loadDirectory(tx, item.ID)
			if err != nil {
				return err
			}

			if dir.IsFixed {
				if !sameParent(dir.ParentID, item.ParentID) {
					return apperr.Forbidden("固定目录不可改变层级")
				}
			} else {
				if dir.ParentID == nil && item.ParentID != nil {
					return apperr.BadRequest("一级目录不能变为二级目录")
				}
				if dir.ParentID != nil && item.ParentID == nil {
					return apperr.BadRequest("二级目录不能变为一级目录")
				}
				if item.ParentID != nil {
					if _, err := validateParentDirectory(tx, *item.ParentID); err != nil {
						return err
					}
				}
			}

			if err := tx.Model(dir).Updates(map[string]any{
				"parent_id":  item.ParentID,
				"sort_order": item.SortOrder,
			}).Error; err != nil {
				return apperr.Internal("更新目录排序失败", err)
			}
		}
		return nil
	})
}

// Dissolve 按需求文档语义解散目录：
// 一级目录：子目录整体移入“其他”，文章保留在原二级目录；
// 二级目录：文章移入“其他/其他”。
func (s *DirectoryService) Dissolve(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		dir, err := loadDirectory(tx, id)
		if err != nil {
			return err
		}
		if dir.IsFixed {
			return apperr.Forbidden("固定目录不可解散")
		}

		other, err := findOtherTopLevel(tx)
		if err != nil {
			return err
		}

		if dir.ParentID == nil {
			return dissolveTopLevel(tx, dir, other)
		}
		return dissolveSecondLevel(tx, dir, other)
	})
}

func dissolveTopLevel(tx *gorm.DB, dir *model.Directory, other *model.Directory) error {
	var articleCount int64
	if err := tx.Model(&model.Article{}).Where("directory_id = ?", dir.ID).Count(&articleCount).Error; err != nil {
		return apperr.Internal("统计目录文章失败", err)
	}
	if articleCount > 0 {
		return apperr.Conflict("该一级目录下存在直属文章，无法解散")
	}

	var children []model.Directory
	if err := tx.Where("parent_id = ?", dir.ID).Order("sort_order, id").Find(&children).Error; err != nil {
		return apperr.Internal("查询子目录失败", err)
	}

	next, err := nextSortOrder(tx, &other.ID)
	if err != nil {
		return err
	}
	for i := range children {
		if err := tx.Model(&children[i]).Updates(map[string]any{
			"parent_id":  other.ID,
			"sort_order": next + i,
		}).Error; err != nil {
			return apperr.Internal("迁移子目录失败", err)
		}
	}

	if err := tx.Delete(dir).Error; err != nil {
		return apperr.Internal("解散一级目录失败", err)
	}
	return nil
}

func dissolveSecondLevel(tx *gorm.DB, dir *model.Directory, other *model.Directory) error {
	var childCount int64
	if err := tx.Model(&model.Directory{}).Where("parent_id = ?", dir.ID).Count(&childCount).Error; err != nil {
		return apperr.Internal("统计子目录失败", err)
	}
	if childCount > 0 {
		return apperr.Conflict("该二级目录下存在子目录，无法解散")
	}

	target, err := findOrCreateOtherInner(tx, other.ID)
	if err != nil {
		return err
	}
	if target.ID == dir.ID {
		return apperr.Conflict("不能解散系统兜底目录“其他/其他”")
	}

	var publishedCount int64
	if err := tx.Model(&model.Article{}).
		Where("directory_id = ? AND status = ?", dir.ID, "published").
		Count(&publishedCount).Error; err != nil {
		return apperr.Internal("统计已发布文章失败", err)
	}

	if err := tx.Model(&model.Article{}).
		Where("directory_id = ?", dir.ID).
		Update("directory_id", target.ID).Error; err != nil {
		return apperr.Internal("迁移文章失败", err)
	}

	if publishedCount > 0 {
		if err := tx.Model(&model.Directory{}).Where("id = ?", target.ID).
			UpdateColumn("article_count", gorm.Expr("article_count + ?", publishedCount)).Error; err != nil {
			return apperr.Internal("更新目标目录计数失败", err)
		}
	}
	if err := tx.Model(&model.Directory{}).Where("id = ?", dir.ID).
		UpdateColumn("article_count", 0).Error; err != nil {
		return apperr.Internal("重置源目录计数失败", err)
	}

	if err := tx.Delete(dir).Error; err != nil {
		return apperr.Internal("解散二级目录失败", err)
	}
	return nil
}

func (s *DirectoryService) AdminTree() ([]DirectoryNode, error) {
	var dirs []model.Directory
	if err := s.db.Order("sort_order, id").Find(&dirs).Error; err != nil {
		return nil, apperr.Internal("查询目录失败", err)
	}

	type statusCount struct {
		DirectoryID uint
		Status      string
		Count       int64
	}
	var rows []statusCount
	if err := s.db.Model(&model.Article{}).
		Select("directory_id, status, count(*) AS count").
		Where("directory_id IS NOT NULL").
		Group("directory_id, status").
		Scan(&rows).Error; err != nil {
		return nil, apperr.Internal("统计目录文章失败", err)
	}

	published := make(map[uint]int64)
	drafts := make(map[uint]int64)
	for _, row := range rows {
		if row.Status == "published" {
			published[row.DirectoryID] = row.Count
		} else if row.Status == "draft" {
			drafts[row.DirectoryID] = row.Count
		}
	}

	childrenByParent := make(map[uint][]DirectoryNode)
	var top []DirectoryNode
	for _, dir := range dirs {
		node := toDirectoryNode(dir, published[dir.ID], drafts[dir.ID], nil)
		if dir.ParentID == nil {
			top = append(top, node)
		} else {
			childrenByParent[*dir.ParentID] = append(childrenByParent[*dir.ParentID], node)
		}
	}
	for i := range top {
		top[i].Children = childrenByParent[top[i].ID]
		if top[i].Children == nil {
			top[i].Children = []DirectoryNode{}
		}
	}
	if top == nil {
		top = []DirectoryNode{}
	}
	return top, nil
}

func (s *DirectoryService) PublicNav() ([]NavNode, error) {
	var dirs []model.Directory
	if err := s.db.Where("visible = ?", true).Order("sort_order, id").Find(&dirs).Error; err != nil {
		return nil, apperr.Internal("查询导航失败", err)
	}

	childrenByParent := make(map[uint][]NavNode)
	var top []NavNode
	for _, dir := range dirs {
		node := toNavNode(dir)
		if dir.ParentID == nil {
			top = append(top, node)
		} else {
			childrenByParent[*dir.ParentID] = append(childrenByParent[*dir.ParentID], node)
		}
	}
	for i := range top {
		top[i].Children = childrenByParent[top[i].ID]
	}
	if top == nil {
		top = []NavNode{}
	}
	return top, nil
}

func (s *DirectoryService) GetBySlug(slug string) (*model.Directory, error) {
	var dir model.Directory
	err := s.db.Where("slug = ?", strings.ToLower(strings.TrimSpace(slug))).First(&dir).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("目录不存在")
	}
	if err != nil {
		return nil, apperr.Internal("查询目录失败", err)
	}
	return &dir, nil
}

// SubtreeIDs 返回目录自身与直接子目录的 ID；文章只挂在二级目录，
// 因此一级目录用于聚合其下所有二级目录的文章。
func (s *DirectoryService) SubtreeIDs(directoryID uint) ([]uint, error) {
	return subtreeIDs(s.db, directoryID)
}

func subtreeIDs(db *gorm.DB, directoryID uint) ([]uint, error) {
	var dir model.Directory
	if err := db.First(&dir, directoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("目录不存在")
		}
		return nil, apperr.Internal("查询目录失败", err)
	}
	if dir.ParentID != nil {
		return []uint{dir.ID}, nil
	}

	var childIDs []uint
	if err := db.Model(&model.Directory{}).Where("parent_id = ?", dir.ID).Pluck("id", &childIDs).Error; err != nil {
		return nil, apperr.Internal("查询子目录失败", err)
	}
	return append([]uint{dir.ID}, childIDs...), nil
}

func (s *DirectoryService) getByID(id uint) (*model.Directory, error) {
	var dir model.Directory
	if err := s.db.First(&dir, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("目录不存在")
		}
		return nil, apperr.Internal("查询目录失败", err)
	}
	return &dir, nil
}

func loadDirectory(tx *gorm.DB, id uint) (*model.Directory, error) {
	var dir model.Directory
	if err := tx.First(&dir, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("目录不存在")
		}
		return nil, apperr.Internal("查询目录失败", err)
	}
	return &dir, nil
}

func validateParentDirectory(tx *gorm.DB, parentID uint) (*model.Directory, error) {
	parent, err := loadDirectory(tx, parentID)
	if err != nil {
		return nil, err
	}
	if parent.ParentID != nil {
		return nil, apperr.BadRequest("只支持两级目录，不能在二级目录下继续创建")
	}
	if parent.Kind != KindContainer {
		return nil, apperr.BadRequest("固定页面目录不能包含子目录")
	}
	return parent, nil
}

func validateDirectorySlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" {
		return "", apperr.BadRequest("目录 slug 不能为空")
	}
	if utf8.RuneCountInString(slug) > 128 {
		return "", apperr.BadRequest("目录 slug 不能超过 128 个字符")
	}
	if !directorySlugPattern.MatchString(slug) {
		return "", apperr.BadRequest("目录 slug 只能包含字母、数字、下划线和中划线")
	}
	if _, reserved := reservedDirectorySlugs[slug]; reserved {
		return "", apperr.Conflict("目录 slug 与系统保留路径冲突")
	}
	return slug, nil
}

func ensureDirectorySlugAvailable(tx *gorm.DB, slug string, excludeID uint) error {
	var count int64
	query := tx.Model(&model.Directory{}).Where("slug = ?", slug)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Internal("检查目录 slug 失败", err)
	}
	if count > 0 {
		return apperr.Conflict("目录 slug 已存在")
	}
	return nil
}

func nextSortOrder(tx *gorm.DB, parentID *uint) (int, error) {
	query := tx.Model(&model.Directory{}).Select("COALESCE(MAX(sort_order), -1)")
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	var maxOrder int
	if err := query.Row().Scan(&maxOrder); err != nil {
		return 0, apperr.Internal("计算排序位置失败", err)
	}
	return maxOrder + 1, nil
}

func findOtherTopLevel(tx *gorm.DB) (*model.Directory, error) {
	var dir model.Directory
	err := tx.Where("page_key = ?", PageKeyOther).First(&dir).Error
	if err == nil {
		return &dir, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Internal("查询“其他”目录失败", err)
	}

	pageKey := PageKeyOther
	order, err := nextSortOrder(tx, nil)
	if err != nil {
		return nil, err
	}
	dir = model.Directory{
		Name:      OtherTopLevelName,
		Slug:      OtherTopLevelSlug,
		Kind:      KindContainer,
		PageKey:   &pageKey,
		IsFixed:   true,
		Visible:   true,
		SortOrder: order,
	}
	if err := tx.Create(&dir).Error; err != nil {
		return nil, apperr.Internal("创建“其他”目录失败", err)
	}
	return &dir, nil
}

func findOrCreateOtherInner(tx *gorm.DB, otherID uint) (*model.Directory, error) {
	var dir model.Directory
	err := tx.Where("parent_id = ? AND name = ?", otherID, OtherInnerName).First(&dir).Error
	if err == nil {
		return &dir, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Internal("查询“其他/其他”目录失败", err)
	}

	parentID := otherID
	order, err := nextSortOrder(tx, &parentID)
	if err != nil {
		return nil, err
	}
	dir = model.Directory{
		ParentID:  &parentID,
		Name:      OtherInnerName,
		Slug:      OtherInnerSlug,
		Kind:      KindContainer,
		Visible:   true,
		SortOrder: order,
	}
	if err := tx.Create(&dir).Error; err != nil {
		return nil, apperr.Internal("创建“其他/其他”目录失败", err)
	}
	return &dir, nil
}

func sameParent(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

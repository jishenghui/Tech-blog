package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

const (
	ProjectStatusDraft     = "draft"
	ProjectStatusPublished = "published"

	maxDebugSteps       = 20
	maxDebugCodeSize    = 20 * 1024
	maxArchitectureNode = 50
	maxArchitectureEdge = 100
	maxTechStackItems   = 20
)

type ProjectService struct {
	db       *gorm.DB
	renderer *MarkdownRenderer
}

func NewProjectService(db *gorm.DB, renderer *MarkdownRenderer) *ProjectService {
	return &ProjectService{db: db, renderer: renderer}
}

type ProjectInput struct {
	Title        string          `json:"title"`
	Slug         string          `json:"slug"`
	Summary      string          `json:"summary"`
	ContentMD    string          `json:"content_md"`
	TechStack    []string        `json:"tech_stack"`
	RepoURL      string          `json:"repo_url"`
	DemoURL      string          `json:"demo_url"`
	CoverURL     string          `json:"cover_url"`
	SortOrder    *int            `json:"sort_order"`
	Status       string          `json:"status"`
	StartedAt    *time.Time      `json:"started_at"`
	EndedAt      *time.Time      `json:"ended_at"`
	DebugSteps   json.RawMessage `json:"debug_steps"`
	Architecture json.RawMessage `json:"architecture"`
}

type ProjectSummary struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Summary   string    `json:"summary"`
	TechStack []string  `json:"tech_stack"`
	CoverURL  string    `json:"cover_url"`
	SortOrder int       `json:"sort_order"`
	Status    string    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProjectDetail struct {
	ProjectSummary
	ContentHTML  string          `json:"content_html"`
	RepoURL      string          `json:"repo_url"`
	DemoURL      string          `json:"demo_url"`
	StartedAt    *time.Time      `json:"started_at"`
	EndedAt      *time.Time      `json:"ended_at"`
	DebugSteps   json.RawMessage `json:"debug_steps"`
	Architecture json.RawMessage `json:"architecture"`
	CreatedAt    time.Time       `json:"created_at"`
}

type AdminProjectDetail struct {
	ProjectDetail
	ContentMD string `json:"content_md"`
}

type ReorderProjectItem struct {
	ID        uint `json:"id"`
	SortOrder int  `json:"sort_order"`
}

type ReorderProjectsInput struct {
	Items []ReorderProjectItem `json:"items"`
}

func (s *ProjectService) Create(in ProjectInput) (*model.Project, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, apperr.BadRequest("项目标题不能为空")
	}
	if utf8.RuneCountInString(title) > 255 {
		return nil, apperr.BadRequest("项目标题不能超过 255 个字符")
	}

	rendered, err := s.renderer.Render(title, in.ContentMD)
	if err != nil {
		return nil, apperr.Internal("渲染项目 Markdown 失败", err)
	}
	techStack, err := validateTechStack(in.TechStack)
	if err != nil {
		return nil, err
	}
	debugSteps, err := validateDebugSteps(in.DebugSteps)
	if err != nil {
		return nil, err
	}
	architecture, err := validateArchitecture(in.Architecture)
	if err != nil {
		return nil, err
	}
	status, err := validateProjectStatus(in.Status, ProjectStatusPublished)
	if err != nil {
		return nil, err
	}
	if err := validateOptionalURL(in.RepoURL, "repo_url"); err != nil {
		return nil, err
	}
	if err := validateOptionalURL(in.DemoURL, "demo_url"); err != nil {
		return nil, err
	}
	if err := validateOptionalURL(in.CoverURL, "cover_url"); err != nil {
		return nil, err
	}
	if err := validateProjectDates(in.StartedAt, in.EndedAt); err != nil {
		return nil, err
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = rendered.Excerpt
	}

	var project model.Project
	err = s.db.Transaction(func(tx *gorm.DB) error {
		slug, err := resolveNewProjectSlug(tx, title, in.Slug)
		if err != nil {
			return err
		}
		order := 0
		if in.SortOrder != nil {
			order = *in.SortOrder
		} else {
			next, err := nextProjectSortOrder(tx)
			if err != nil {
				return err
			}
			order = next
		}

		project = model.Project{
			Title:        title,
			Slug:         slug,
			Summary:      summary,
			ContentMD:    in.ContentMD,
			ContentHTML:  rendered.HTML,
			TechStack:    techStack,
			DebugSteps:   debugSteps,
			Architecture: architecture,
			RepoURL:      strings.TrimSpace(in.RepoURL),
			DemoURL:      strings.TrimSpace(in.DemoURL),
			CoverURL:     strings.TrimSpace(in.CoverURL),
			SortOrder:    order,
			Status:       status,
			StartedAt:    in.StartedAt,
			EndedAt:      in.EndedAt,
		}
		if err := tx.Create(&project).Error; err != nil {
			return apperr.Internal("创建项目失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *ProjectService) Update(id uint, in ProjectInput) (*model.Project, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, apperr.BadRequest("项目标题不能为空")
	}
	if utf8.RuneCountInString(title) > 255 {
		return nil, apperr.BadRequest("项目标题不能超过 255 个字符")
	}

	rendered, err := s.renderer.Render(title, in.ContentMD)
	if err != nil {
		return nil, apperr.Internal("渲染项目 Markdown 失败", err)
	}
	techStack, err := validateTechStack(in.TechStack)
	if err != nil {
		return nil, err
	}
	debugSteps, err := validateDebugSteps(in.DebugSteps)
	if err != nil {
		return nil, err
	}
	architecture, err := validateArchitecture(in.Architecture)
	if err != nil {
		return nil, err
	}
	if err := validateOptionalURL(in.RepoURL, "repo_url"); err != nil {
		return nil, err
	}
	if err := validateOptionalURL(in.DemoURL, "demo_url"); err != nil {
		return nil, err
	}
	if err := validateOptionalURL(in.CoverURL, "cover_url"); err != nil {
		return nil, err
	}
	if err := validateProjectDates(in.StartedAt, in.EndedAt); err != nil {
		return nil, err
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = rendered.Excerpt
	}

	var project model.Project
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&project, id).Error; err != nil {
			return projectNotFound(err)
		}

		status, err := validateProjectStatus(in.Status, project.Status)
		if err != nil {
			return err
		}

		slug := project.Slug
		if strings.TrimSpace(in.Slug) != "" {
			slug, err = validateProjectSlug(in.Slug)
			if err != nil {
				return err
			}
			if slug != project.Slug {
				if err := ensureProjectSlugAvailable(tx, slug, project.ID); err != nil {
					return err
				}
			}
		}

		updates := map[string]any{
			"title":        title,
			"slug":         slug,
			"summary":      summary,
			"content_md":   in.ContentMD,
			"content_html": rendered.HTML,
			"tech_stack":   techStack,
			"debug_steps":  debugSteps,
			"architecture": architecture,
			"repo_url":     strings.TrimSpace(in.RepoURL),
			"demo_url":     strings.TrimSpace(in.DemoURL),
			"cover_url":    strings.TrimSpace(in.CoverURL),
			"status":       status,
			"started_at":   in.StartedAt,
			"ended_at":     in.EndedAt,
		}
		if in.SortOrder != nil {
			updates["sort_order"] = *in.SortOrder
		}
		if err := tx.Model(&project).Updates(updates).Error; err != nil {
			return apperr.Internal("更新项目失败", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var updated model.Project
	if err := s.db.First(&updated, id).Error; err != nil {
		return nil, projectNotFound(err)
	}
	return &updated, nil
}

func (s *ProjectService) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var project model.Project
		if err := tx.First(&project, id).Error; err != nil {
			return projectNotFound(err)
		}
		if err := tx.Delete(&project).Error; err != nil {
			return apperr.Internal("删除项目失败", err)
		}
		return nil
	})
}

func (s *ProjectService) Reorder(in ReorderProjectsInput) error {
	if len(in.Items) == 0 {
		return apperr.BadRequest("排序列表不能为空")
	}
	seen := make(map[uint]struct{}, len(in.Items))
	for _, item := range in.Items {
		if item.ID == 0 {
			return apperr.BadRequest("项目 ID 不能为空")
		}
		if _, ok := seen[item.ID]; ok {
			return apperr.BadRequest("排序列表中存在重复项目")
		}
		seen[item.ID] = struct{}{}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range in.Items {
			var project model.Project
			if err := tx.First(&project, item.ID).Error; err != nil {
				return projectNotFound(err)
			}
			if err := tx.Model(&project).Update("sort_order", item.SortOrder).Error; err != nil {
				return apperr.Internal("更新项目排序失败", err)
			}
		}
		return nil
	})
}

func (s *ProjectService) AdminList() ([]ProjectSummary, error) {
	var projects []model.Project
	if err := s.db.Order("sort_order, id").Find(&projects).Error; err != nil {
		return nil, apperr.Internal("查询项目失败", err)
	}
	items := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		items = append(items, toProjectSummary(project, true))
	}
	return items, nil
}

func (s *ProjectService) PublicList() ([]ProjectSummary, error) {
	var projects []model.Project
	if err := s.db.Where("status = ?", ProjectStatusPublished).Order("sort_order, id").Find(&projects).Error; err != nil {
		return nil, apperr.Internal("查询项目失败", err)
	}
	items := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		items = append(items, toProjectSummary(project, false))
	}
	return items, nil
}

func (s *ProjectService) PublicDetail(slug string) (*ProjectDetail, error) {
	var project model.Project
	err := s.db.Where("slug = ? AND status = ?", strings.ToLower(strings.TrimSpace(slug)), ProjectStatusPublished).
		First(&project).Error
	if err != nil {
		return nil, projectNotFound(err)
	}
	detail := toProjectDetail(project, false)
	return &detail, nil
}

func (s *ProjectService) AdminDetail(id uint) (*AdminProjectDetail, error) {
	var project model.Project
	if err := s.db.First(&project, id).Error; err != nil {
		return nil, projectNotFound(err)
	}
	detail := AdminProjectDetail{
		ProjectDetail: toProjectDetail(project, true),
		ContentMD:     project.ContentMD,
	}
	return &detail, nil
}

func toProjectSummary(project model.Project, admin bool) ProjectSummary {
	status := ""
	if admin {
		status = project.Status
	}
	return ProjectSummary{
		ID:        project.ID,
		Title:     project.Title,
		Slug:      project.Slug,
		Summary:   project.Summary,
		TechStack: parseTechStack(project.TechStack),
		CoverURL:  project.CoverURL,
		SortOrder: project.SortOrder,
		Status:    status,
		UpdatedAt: project.UpdatedAt,
	}
}

func toProjectDetail(project model.Project, admin bool) ProjectDetail {
	summary := toProjectSummary(project, admin)
	return ProjectDetail{
		ProjectSummary: summary,
		ContentHTML:    project.ContentHTML,
		RepoURL:        project.RepoURL,
		DemoURL:        project.DemoURL,
		StartedAt:      project.StartedAt,
		EndedAt:        project.EndedAt,
		DebugSteps:     rawJSONOr(project.DebugSteps, "[]"),
		Architecture:   rawJSONOr(project.Architecture, `{"nodes":[],"edges":[]}`),
		CreatedAt:      project.CreatedAt,
	}
}

func parseTechStack(raw string) []string {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	if items == nil {
		return []string{}
	}
	return items
}

func rawJSONOr(raw, fallback string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return json.RawMessage(fallback)
	}
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}
	return json.RawMessage(fallback)
}

func projectNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperr.NotFound("项目不存在")
	}
	return apperr.Internal("查询项目失败", err)
}

func resolveNewProjectSlug(tx *gorm.DB, title, provided string) (string, error) {
	if provided != "" {
		slug, err := validateProjectSlug(provided)
		if err != nil {
			return "", err
		}
		if err := ensureProjectSlugAvailable(tx, slug, 0); err != nil {
			return "", err
		}
		return slug, nil
	}
	return uniqueProjectSlug(tx, generateSlug(title, "project"), 0)
}

func validateProjectSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" {
		return "", apperr.BadRequest("项目 slug 不能为空")
	}
	if utf8.RuneCountInString(slug) > 255 {
		return "", apperr.BadRequest("项目 slug 不能超过 255 个字符")
	}
	if !articleSlugPattern.MatchString(slug) {
		return "", apperr.BadRequest("项目 slug 只能包含字母、数字、下划线和中划线")
	}
	return slug, nil
}

func ensureProjectSlugAvailable(tx *gorm.DB, slug string, excludeID uint) error {
	var count int64
	query := tx.Model(&model.Project{}).Where("slug = ?", slug)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Internal("检查项目 slug 失败", err)
	}
	if count > 0 {
		return apperr.Conflict("项目 slug 已存在")
	}
	return nil
}

func uniqueProjectSlug(tx *gorm.DB, base string, excludeID uint) (string, error) {
	candidate := base
	for i := 2; i <= 1000; i++ {
		var count int64
		query := tx.Model(&model.Project{}).Where("slug = ?", candidate)
		if excludeID != 0 {
			query = query.Where("id <> ?", excludeID)
		}
		if err := query.Count(&count).Error; err != nil {
			return "", apperr.Internal("检查项目 slug 失败", err)
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return "", apperr.Conflict("无法生成唯一的项目 slug")
}

func nextProjectSortOrder(tx *gorm.DB) (int, error) {
	var maxOrder int
	if err := tx.Model(&model.Project{}).Select("COALESCE(MAX(sort_order), -1)").Row().Scan(&maxOrder); err != nil {
		return 0, apperr.Internal("计算项目排序位置失败", err)
	}
	return maxOrder + 1, nil
}

func validateProjectStatus(raw, fallback string) (string, error) {
	status := strings.TrimSpace(raw)
	if status == "" {
		status = fallback
	}
	if status != ProjectStatusDraft && status != ProjectStatusPublished {
		return "", apperr.BadRequest("项目 status 只能是 draft 或 published")
	}
	return status, nil
}

func validateProjectDates(startedAt, endedAt *time.Time) error {
	if startedAt != nil && endedAt != nil && endedAt.Before(*startedAt) {
		return apperr.BadRequest("项目结束时间不能早于开始时间")
	}
	return nil
}

func validateOptionalURL(raw, field string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if len(value) > 512 {
		return apperr.BadRequest(field + " 不能超过 512 个字符")
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "/") {
		return apperr.BadRequest(field + " 必须是 http(s) 或 / 开头的地址")
	}
	return nil
}

func validateTechStack(items []string) (string, error) {
	if len(items) > maxTechStackItems {
		return "", apperr.BadRequest(fmt.Sprintf("tech_stack 不能超过 %d 项", maxTechStackItems))
	}
	cleaned := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if utf8.RuneCountInString(value) > 64 {
			return "", apperr.BadRequest("tech_stack 单项不能超过 64 个字符")
		}
		cleaned = append(cleaned, value)
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return "", apperr.BadRequest("tech_stack 格式不正确")
	}
	return string(encoded), nil
}

type projectDebugStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	VideoURL    string `json:"video_url"`
	VideoPoster string `json:"video_poster"`
	Code        string `json:"code"`
	CodeLang    string `json:"code_lang"`
	ArticleSlug string `json:"article_slug"`
	TagSlug     string `json:"tag_slug"`
}

func validateDebugSteps(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return "[]", nil
	}
	var steps []projectDebugStep
	if err := json.Unmarshal(raw, &steps); err != nil {
		return "", apperr.BadRequest("debug_steps 必须是 JSON 数组")
	}
	if steps == nil {
		steps = []projectDebugStep{}
	}
	if len(steps) > maxDebugSteps {
		return "", apperr.BadRequest(fmt.Sprintf("debug_steps 不能超过 %d 步", maxDebugSteps))
	}

	seen := make(map[string]struct{}, len(steps))
	for i := range steps {
		step := &steps[i]
		step.Title = strings.TrimSpace(step.Title)
		step.Description = strings.TrimSpace(step.Description)
		step.ImageURL = strings.TrimSpace(step.ImageURL)
		step.VideoURL = strings.TrimSpace(step.VideoURL)
		step.VideoPoster = strings.TrimSpace(step.VideoPoster)
		step.Code = strings.TrimSpace(step.Code)
		step.CodeLang = strings.TrimSpace(step.CodeLang)
		step.ArticleSlug = strings.TrimSpace(step.ArticleSlug)
		step.TagSlug = strings.TrimSpace(step.TagSlug)

		if step.Title == "" {
			return "", apperr.BadRequest(fmt.Sprintf("第 %d 个调试步骤缺少标题", i+1))
		}
		if utf8.RuneCountInString(step.Title) > 128 {
			return "", apperr.BadRequest("调试步骤标题不能超过 128 个字符")
		}
		if utf8.RuneCountInString(step.Description) > 2000 {
			return "", apperr.BadRequest("调试步骤说明不能超过 2000 个字符")
		}
		if len(step.Code) > maxDebugCodeSize {
			return "", apperr.BadRequest("调试步骤代码不能超过 20KB")
		}
		if utf8.RuneCountInString(step.CodeLang) > 32 {
			return "", apperr.BadRequest("调试步骤 code_lang 不能超过 32 个字符")
		}
		for field, value := range map[string]string{
			"image_url":    step.ImageURL,
			"video_url":    step.VideoURL,
			"video_poster": step.VideoPoster,
		} {
			if err := validateOptionalURL(value, field); err != nil {
				return "", err
			}
		}
		if step.ID == "" {
			step.ID = fmt.Sprintf("step-%d", i+1)
		}
		if _, exists := seen[step.ID]; exists {
			return "", apperr.BadRequest("debug_steps 的 id 不能重复")
		}
		seen[step.ID] = struct{}{}
	}

	encoded, err := json.Marshal(steps)
	if err != nil {
		return "", apperr.BadRequest("debug_steps 格式不正确")
	}
	return string(encoded), nil
}

type projectArchitecture struct {
	Nodes []projectArchNode `json:"nodes"`
	Edges []projectArchEdge `json:"edges"`
}

type projectArchNode struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Kind        string  `json:"kind"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Description string  `json:"description"`
	TagSlug     string  `json:"tag_slug"`
	SearchQuery string  `json:"search_query"`
}

type projectArchEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func validateArchitecture(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return `{"nodes":[],"edges":[]}`, nil
	}
	var architecture projectArchitecture
	if err := json.Unmarshal(raw, &architecture); err != nil {
		return "", apperr.BadRequest("architecture 必须是 JSON 对象")
	}
	if architecture.Nodes == nil {
		architecture.Nodes = []projectArchNode{}
	}
	if architecture.Edges == nil {
		architecture.Edges = []projectArchEdge{}
	}
	if len(architecture.Nodes) > maxArchitectureNode {
		return "", apperr.BadRequest(fmt.Sprintf("architecture 节点不能超过 %d 个", maxArchitectureNode))
	}
	if len(architecture.Edges) > maxArchitectureEdge {
		return "", apperr.BadRequest(fmt.Sprintf("architecture 边不能超过 %d 条", maxArchitectureEdge))
	}

	allowedKinds := map[string]struct{}{
		"frontend": {}, "backend": {}, "data": {}, "infra": {}, "admin": {},
	}
	nodeIDs := make(map[string]struct{}, len(architecture.Nodes))
	for i := range architecture.Nodes {
		node := &architecture.Nodes[i]
		node.ID = strings.TrimSpace(node.ID)
		node.Label = strings.TrimSpace(node.Label)
		node.Kind = strings.TrimSpace(node.Kind)
		node.Description = strings.TrimSpace(node.Description)
		node.TagSlug = strings.TrimSpace(node.TagSlug)
		node.SearchQuery = strings.TrimSpace(node.SearchQuery)

		if node.ID == "" {
			return "", apperr.BadRequest("architecture 节点 id 不能为空")
		}
		if _, exists := nodeIDs[node.ID]; exists {
			return "", apperr.BadRequest("architecture 节点 id 不能重复")
		}
		nodeIDs[node.ID] = struct{}{}
		if node.Label == "" {
			return "", apperr.BadRequest("architecture 节点 label 不能为空")
		}
		if node.Kind == "" {
			node.Kind = "infra"
		}
		if _, ok := allowedKinds[node.Kind]; !ok {
			return "", apperr.BadRequest("architecture 节点 kind 只能是 frontend/backend/data/infra/admin")
		}
		if math.IsNaN(node.X) || math.IsInf(node.X, 0) || math.IsNaN(node.Y) || math.IsInf(node.Y, 0) {
			return "", apperr.BadRequest("architecture 节点坐标必须是有效数字")
		}
		if utf8.RuneCountInString(node.Description) > 1000 {
			return "", apperr.BadRequest("architecture 节点说明不能超过 1000 个字符")
		}
	}

	edgeSet := make(map[string]struct{}, len(architecture.Edges))
	for i := range architecture.Edges {
		edge := &architecture.Edges[i]
		edge.From = strings.TrimSpace(edge.From)
		edge.To = strings.TrimSpace(edge.To)
		if _, ok := nodeIDs[edge.From]; !ok {
			return "", apperr.BadRequest("architecture 边引用了不存在的起点")
		}
		if _, ok := nodeIDs[edge.To]; !ok {
			return "", apperr.BadRequest("architecture 边引用了不存在的终点")
		}
		key := edge.From + "->" + edge.To
		if _, exists := edgeSet[key]; exists {
			return "", apperr.BadRequest("architecture 边不能重复")
		}
		edgeSet[key] = struct{}{}
	}

	encoded, err := json.Marshal(architecture)
	if err != nil {
		return "", apperr.BadRequest("architecture 格式不正确")
	}
	return string(encoded), nil
}

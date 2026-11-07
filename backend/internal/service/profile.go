package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"blog-api/internal/apperr"
	"blog-api/internal/model"
)

const (
	maxProfileAwards       = 50
	maxProfileExperiences  = 50
	maxProfileFeatured     = 50
	maxProfileResumeSize   = 100 * 1024
	maxProfileContactsSize = 10 * 1024
)

type ProfileService struct {
	db       *gorm.DB
	renderer *MarkdownRenderer
}

func NewProfileService(db *gorm.DB, renderer *MarkdownRenderer) *ProfileService {
	return &ProfileService{db: db, renderer: renderer}
}

type ProfileInput struct {
	Headline         string          `json:"headline"`
	AvatarURL        string          `json:"avatar_url"`
	ResumeMD         string          `json:"resume_md"`
	Awards           json.RawMessage `json:"awards"`
	Experiences      json.RawMessage `json:"experiences"`
	FeaturedProjects json.RawMessage `json:"featured_projects"`
	Contacts         json.RawMessage `json:"contacts"`
}

type ProfileView struct {
	Headline         string          `json:"headline"`
	AvatarURL        string          `json:"avatar_url"`
	ResumeHTML       string          `json:"resume_html"`
	Awards           json.RawMessage `json:"awards"`
	Experiences      json.RawMessage `json:"experiences"`
	FeaturedProjects json.RawMessage `json:"featured_projects"`
	Contacts         json.RawMessage `json:"contacts"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type AdminProfileView struct {
	ProfileView
	ResumeMD string `json:"resume_md"`
}

func (s *ProfileService) GetPublic() (*ProfileView, error) {
	profile, err := s.ensureProfile()
	if err != nil {
		return nil, err
	}
	view := toProfileView(*profile)
	return &view, nil
}

func (s *ProfileService) GetAdmin() (*AdminProfileView, error) {
	profile, err := s.ensureProfile()
	if err != nil {
		return nil, err
	}
	view := AdminProfileView{
		ProfileView: toProfileView(*profile),
		ResumeMD:    profile.ResumeMD,
	}
	return &view, nil
}

func (s *ProfileService) Update(in ProfileInput) (*AdminProfileView, error) {
	headline := strings.TrimSpace(in.Headline)
	if utf8.RuneCountInString(headline) > 200 {
		return nil, apperr.BadRequest("headline 不能超过 200 个字符")
	}
	avatarURL := strings.TrimSpace(in.AvatarURL)
	if err := validateOptionalURL(avatarURL, "avatar_url"); err != nil {
		return nil, err
	}
	if len(in.ResumeMD) > maxProfileResumeSize {
		return nil, apperr.BadRequest("resume_md 不能超过 100KB")
	}

	awards, err := validateProfileAwards(in.Awards)
	if err != nil {
		return nil, err
	}
	experiences, err := validateProfileExperiences(in.Experiences)
	if err != nil {
		return nil, err
	}
	featuredProjects, err := validateProfileFeaturedProjects(in.FeaturedProjects)
	if err != nil {
		return nil, err
	}
	contacts, err := validateProfileContacts(in.Contacts)
	if err != nil {
		return nil, err
	}

	rendered, err := s.renderer.Render("简历", in.ResumeMD)
	if err != nil {
		return nil, apperr.Internal("渲染简历 Markdown 失败", err)
	}

	profile, err := s.ensureProfile()
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"headline":          headline,
		"avatar_url":        avatarURL,
		"resume_md":         in.ResumeMD,
		"resume_html":       rendered.HTML,
		"awards":            awards,
		"experiences":       experiences,
		"featured_projects": featuredProjects,
		"contacts":          contacts,
	}
	if err := s.db.Model(profile).Updates(updates).Error; err != nil {
		return nil, apperr.Internal("更新首页资料失败", err)
	}
	return s.GetAdmin()
}

func (s *ProfileService) ensureProfile() (*model.Profile, error) {
	var profile model.Profile
	err := s.db.First(&profile, 1).Error
	if err == nil {
		return &profile, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.Internal("查询首页资料失败", err)
	}

	profile = model.Profile{
		ID:               1,
		Awards:           "[]",
		Experiences:      "[]",
		FeaturedProjects: "[]",
		Contacts:         "{}",
	}
	if err := s.db.Create(&profile).Error; err != nil {
		return nil, apperr.Internal("创建首页资料失败", err)
	}
	return &profile, nil
}

func toProfileView(profile model.Profile) ProfileView {
	return ProfileView{
		Headline:         profile.Headline,
		AvatarURL:        profile.AvatarURL,
		ResumeHTML:       profile.ResumeHTML,
		Awards:           rawJSONOr(profile.Awards, "[]"),
		Experiences:      rawJSONOr(profile.Experiences, "[]"),
		FeaturedProjects: rawJSONOr(profile.FeaturedProjects, "[]"),
		Contacts:         rawJSONOr(profile.Contacts, "{}"),
		UpdatedAt:        profile.UpdatedAt,
	}
}

type profileAward struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	ImageURL    string `json:"image_url"`
	Date        string `json:"date"`
	Description string `json:"description"`
}

func validateProfileAwards(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return "[]", nil
	}
	var awards []profileAward
	if err := json.Unmarshal(raw, &awards); err != nil {
		return "", apperr.BadRequest("awards 必须是 JSON 数组")
	}
	if awards == nil {
		awards = []profileAward{}
	}
	if len(awards) > maxProfileAwards {
		return "", apperr.BadRequest(fmt.Sprintf("awards 不能超过 %d 条", maxProfileAwards))
	}

	seen := make(map[string]struct{}, len(awards))
	for i := range awards {
		award := &awards[i]
		award.Title = strings.TrimSpace(award.Title)
		award.ImageURL = strings.TrimSpace(award.ImageURL)
		award.Date = strings.TrimSpace(award.Date)
		award.Description = strings.TrimSpace(award.Description)
		if award.Title == "" {
			return "", apperr.BadRequest(fmt.Sprintf("第 %d 个奖项缺少标题", i+1))
		}
		if utf8.RuneCountInString(award.Title) > 128 {
			return "", apperr.BadRequest("奖项标题不能超过 128 个字符")
		}
		if utf8.RuneCountInString(award.Description) > 1000 {
			return "", apperr.BadRequest("奖项说明不能超过 1000 个字符")
		}
		if err := validateOptionalURL(award.ImageURL, "image_url"); err != nil {
			return "", err
		}
		if award.ID == "" {
			award.ID = fmt.Sprintf("award-%d", i+1)
		}
		if _, exists := seen[award.ID]; exists {
			return "", apperr.BadRequest("awards 的 id 不能重复")
		}
		seen[award.ID] = struct{}{}
	}

	encoded, err := json.Marshal(awards)
	if err != nil {
		return "", apperr.BadRequest("awards 格式不正确")
	}
	return string(encoded), nil
}

type profileExperience struct {
	Company     string `json:"company"`
	Role        string `json:"role"`
	Period      string `json:"period"`
	Description string `json:"description"`
}

func validateProfileExperiences(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return "[]", nil
	}
	var experiences []profileExperience
	if err := json.Unmarshal(raw, &experiences); err != nil {
		return "", apperr.BadRequest("experiences 必须是 JSON 数组")
	}
	if experiences == nil {
		experiences = []profileExperience{}
	}
	if len(experiences) > maxProfileExperiences {
		return "", apperr.BadRequest(fmt.Sprintf("experiences 不能超过 %d 条", maxProfileExperiences))
	}
	for i := range experiences {
		experience := &experiences[i]
		experience.Company = strings.TrimSpace(experience.Company)
		experience.Role = strings.TrimSpace(experience.Role)
		experience.Period = strings.TrimSpace(experience.Period)
		experience.Description = strings.TrimSpace(experience.Description)
		if experience.Company == "" || experience.Role == "" {
			return "", apperr.BadRequest(fmt.Sprintf("第 %d 段经历缺少公司或职位", i+1))
		}
		if utf8.RuneCountInString(experience.Description) > 1000 {
			return "", apperr.BadRequest("经历说明不能超过 1000 个字符")
		}
	}
	encoded, err := json.Marshal(experiences)
	if err != nil {
		return "", apperr.BadRequest("experiences 格式不正确")
	}
	return string(encoded), nil
}

type profileFeaturedProject struct {
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
	CoverURL string `json:"cover_url"`
}

func validateProfileFeaturedProjects(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return "[]", nil
	}
	var projects []profileFeaturedProject
	if err := json.Unmarshal(raw, &projects); err != nil {
		return "", apperr.BadRequest("featured_projects 必须是 JSON 数组")
	}
	if projects == nil {
		projects = []profileFeaturedProject{}
	}
	if len(projects) > maxProfileFeatured {
		return "", apperr.BadRequest(fmt.Sprintf("featured_projects 不能超过 %d 条", maxProfileFeatured))
	}
	for i := range projects {
		project := &projects[i]
		project.Name = strings.TrimSpace(project.Name)
		project.Summary = strings.TrimSpace(project.Summary)
		project.URL = strings.TrimSpace(project.URL)
		project.CoverURL = strings.TrimSpace(project.CoverURL)
		if project.Name == "" {
			return "", apperr.BadRequest(fmt.Sprintf("第 %d 个推荐项目缺少名称", i+1))
		}
		if err := validateOptionalURL(project.URL, "featured_projects.url"); err != nil {
			return "", err
		}
		if err := validateOptionalURL(project.CoverURL, "featured_projects.cover_url"); err != nil {
			return "", err
		}
	}
	encoded, err := json.Marshal(projects)
	if err != nil {
		return "", apperr.BadRequest("featured_projects 格式不正确")
	}
	return string(encoded), nil
}

func validateProfileContacts(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return "{}", nil
	}
	var contacts map[string]any
	if err := json.Unmarshal(raw, &contacts); err != nil {
		return "", apperr.BadRequest("contacts 必须是 JSON 对象")
	}
	if contacts == nil {
		contacts = map[string]any{}
	}
	encoded, err := json.Marshal(contacts)
	if err != nil {
		return "", apperr.BadRequest("contacts 格式不正确")
	}
	if len(encoded) > maxProfileContactsSize {
		return "", apperr.BadRequest("contacts 不能超过 10KB")
	}
	return string(encoded), nil
}

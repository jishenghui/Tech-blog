package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"blog-api/internal/apperr"
	"blog-api/internal/response"
	"blog-api/internal/service"
)

type ArticleHandler struct {
	articles *service.ArticleService
}

func NewArticleHandler(articles *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articles: articles}
}

func (h *ArticleHandler) AdminList(c *gin.Context) {
	directoryID, err := optionalUintQuery(c, "directory_id")
	if err != nil {
		respondError(c, err)
		return
	}
	page, pageSize, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}

	result, err := h.articles.AdminList(service.ArticleFilter{
		DirectoryID: directoryID,
		Status:      c.Query("status"),
		Q:           c.Query("q"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *ArticleHandler) AdminDetail(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	detail, err := h.articles.AdminDetail(id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *ArticleHandler) Create(c *gin.Context) {
	var in service.CreateArticleInput
	if !bindJSON(c, &in) {
		return
	}
	article, err := h.articles.Create(in)
	if err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, article.ID)
}

func (h *ArticleHandler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in service.UpdateArticleInput
	if !bindJSON(c, &in) {
		return
	}
	article, err := h.articles.Update(id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, article.ID)
}

func (h *ArticleHandler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.articles.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *ArticleHandler) Publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.articles.Publish(id); err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, id)
}

func (h *ArticleHandler) Unpublish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.articles.Unpublish(id); err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, id)
}

func (h *ArticleHandler) Move(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in struct {
		DirectoryID uint `json:"directory_id"`
	}
	if !bindJSON(c, &in) {
		return
	}
	if in.DirectoryID == 0 {
		respondError(c, apperr.BadRequest("directory_id 不能为空"))
		return
	}
	if err := h.articles.Move(id, in.DirectoryID); err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, id)
}

func (h *ArticleHandler) SetTags(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in struct {
		TagIDs []uint `json:"tag_ids"`
	}
	if !bindJSON(c, &in) {
		return
	}
	if err := h.articles.SetTags(id, in.TagIDs); err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, id)
}

func (h *ArticleHandler) PublicList(c *gin.Context) {
	directoryID, err := optionalUintQuery(c, "directory_id")
	if err != nil {
		respondError(c, err)
		return
	}
	page, pageSize, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}

	result, err := h.articles.PublicList(service.ArticleFilter{
		DirectoryID: directoryID,
		TagSlug:     c.Query("tag"),
		Q:           c.Query("q"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *ArticleHandler) PublicDetail(c *gin.Context) {
	detail, err := h.articles.PublicDetail(c.Param("slug"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *ArticleHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		respondError(c, apperr.BadRequest("搜索关键词不能为空"))
		return
	}
	page, pageSize, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}

	result, err := h.articles.PublicList(service.ArticleFilter{
		Q:        query,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *ArticleHandler) respondDetail(c *gin.Context, id uint) {
	detail, err := h.articles.AdminDetail(id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, detail)
}

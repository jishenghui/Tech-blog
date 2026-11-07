package handler

import (
	"github.com/gin-gonic/gin"

	"blog-api/internal/response"
	"blog-api/internal/service"
)

type TagHandler struct {
	tags     *service.TagService
	articles *service.ArticleService
}

func NewTagHandler(tags *service.TagService, articles *service.ArticleService) *TagHandler {
	return &TagHandler{tags: tags, articles: articles}
}

func (h *TagHandler) AdminList(c *gin.Context) {
	tags, err := h.tags.AdminList()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, tags)
}

func (h *TagHandler) Create(c *gin.Context) {
	var in service.CreateTagInput
	if !bindJSON(c, &in) {
		return
	}
	tag, err := h.tags.Create(in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, service.ToTagView(*tag))
}

func (h *TagHandler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in service.UpdateTagInput
	if !bindJSON(c, &in) {
		return
	}
	tag, err := h.tags.Update(id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, service.ToTagView(*tag))
}

func (h *TagHandler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.tags.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *TagHandler) PublicList(c *gin.Context) {
	tags, err := h.tags.PublicList()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, tags)
}

func (h *TagHandler) PublicArticles(c *gin.Context) {
	page, pageSize, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}
	result, err := h.articles.PublicList(service.ArticleFilter{
		TagSlug:  c.Param("slug"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, result)
}

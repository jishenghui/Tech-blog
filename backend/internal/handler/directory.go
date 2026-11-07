package handler

import (
	"github.com/gin-gonic/gin"

	"blog-api/internal/response"
	"blog-api/internal/service"
)

type DirectoryHandler struct {
	directories *service.DirectoryService
	articles    *service.ArticleService
}

func NewDirectoryHandler(directories *service.DirectoryService, articles *service.ArticleService) *DirectoryHandler {
	return &DirectoryHandler{directories: directories, articles: articles}
}

func (h *DirectoryHandler) AdminTree(c *gin.Context) {
	tree, err := h.directories.AdminTree()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, tree)
}

func (h *DirectoryHandler) Create(c *gin.Context) {
	var in service.CreateDirectoryInput
	if !bindJSON(c, &in) {
		return
	}
	dir, err := h.directories.Create(in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, dir)
}

func (h *DirectoryHandler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in service.UpdateDirectoryInput
	if !bindJSON(c, &in) {
		return
	}
	dir, err := h.directories.Update(id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, dir)
}

func (h *DirectoryHandler) Dissolve(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.directories.Dissolve(id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DirectoryHandler) Move(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in struct {
		ParentID  *uint `json:"parent_id"`
		SortOrder *int  `json:"sort_order"`
	}
	if !bindJSON(c, &in) {
		return
	}
	if err := h.directories.Move(id, in.ParentID, in.SortOrder); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DirectoryHandler) Reorder(c *gin.Context) {
	var in service.ReorderDirectoriesInput
	if !bindJSON(c, &in) {
		return
	}
	if err := h.directories.Reorder(in); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DirectoryHandler) PublicNav(c *gin.Context) {
	nodes, err := h.directories.PublicNav()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nodes)
}

func (h *DirectoryHandler) PublicDetail(c *gin.Context) {
	dir, err := h.directories.GetBySlug(c.Param("slug"))
	if err != nil {
		respondError(c, err)
		return
	}

	page, pageSize, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}

	articles, err := h.articles.PublicList(service.ArticleFilter{
		DirectoryID: &dir.ID,
		TagSlug:     c.Query("tag"),
		Q:           c.Query("q"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	response.OK(c, service.DirectoryPage{
		Directory: service.ToDirectoryDetail(*dir),
		Items:     articles.Items,
		Total:     articles.Total,
		Page:      articles.Page,
		PageSize:  articles.PageSize,
	})
}

package handler

import (
	"github.com/gin-gonic/gin"

	"blog-api/internal/response"
	"blog-api/internal/service"
)

type ProjectHandler struct {
	projects *service.ProjectService
}

func NewProjectHandler(projects *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

func (h *ProjectHandler) PublicList(c *gin.Context) {
	items, err := h.projects.PublicList()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *ProjectHandler) PublicDetail(c *gin.Context) {
	detail, err := h.projects.PublicDetail(c.Param("slug"))
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *ProjectHandler) AdminList(c *gin.Context) {
	items, err := h.projects.AdminList()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *ProjectHandler) AdminDetail(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	detail, err := h.projects.AdminDetail(id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, detail)
}

func (h *ProjectHandler) Create(c *gin.Context) {
	var in service.ProjectInput
	if !bindJSON(c, &in) {
		return
	}
	project, err := h.projects.Create(in)
	if err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, project.ID)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var in service.ProjectInput
	if !bindJSON(c, &in) {
		return
	}
	project, err := h.projects.Update(id, in)
	if err != nil {
		respondError(c, err)
		return
	}
	h.respondDetail(c, project.ID)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.projects.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *ProjectHandler) Reorder(c *gin.Context) {
	var in service.ReorderProjectsInput
	if !bindJSON(c, &in) {
		return
	}
	if err := h.projects.Reorder(in); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *ProjectHandler) respondDetail(c *gin.Context, id uint) {
	detail, err := h.projects.AdminDetail(id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, detail)
}

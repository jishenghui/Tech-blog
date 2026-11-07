package handler

import (
	"github.com/gin-gonic/gin"

	"blog-api/internal/response"
	"blog-api/internal/service"
)

type ProfileHandler struct {
	profiles *service.ProfileService
}

func NewProfileHandler(profiles *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

func (h *ProfileHandler) PublicGet(c *gin.Context) {
	view, err := h.profiles.GetPublic()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *ProfileHandler) AdminGet(c *gin.Context) {
	view, err := h.profiles.GetAdmin()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *ProfileHandler) AdminUpdate(c *gin.Context) {
	var in service.ProfileInput
	if !bindJSON(c, &in) {
		return
	}
	view, err := h.profiles.Update(in)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

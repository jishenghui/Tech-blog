package handler

import (
	"github.com/gin-gonic/gin"

	"blog-api/internal/response"
	"blog-api/internal/service"
)

type MaintenanceHandler struct {
	articles *service.ArticleService
}

func NewMaintenanceHandler(articles *service.ArticleService) *MaintenanceHandler {
	return &MaintenanceHandler{articles: articles}
}

func (h *MaintenanceHandler) Recalc(c *gin.Context) {
	result, err := h.articles.Recalc()
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, result)
}

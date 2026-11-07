package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"blog-api/internal/apperr"
	"blog-api/internal/response"
	"blog-api/internal/service"
)

type AssetHandler struct {
	assets *service.AssetService
}

func NewAssetHandler(assets *service.AssetService) *AssetHandler {
	return &AssetHandler{assets: assets}
}

func (h *AssetHandler) Upload(c *gin.Context) {
	// 多留 1MB 给 multipart 头部；实际图片大小由 service 再校验。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxImageUploadSize+1<<20)

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传图片文件（字段名 file）")
		return
	}
	if file.Size > service.MaxImageUploadSize {
		response.BadRequest(c, "图片不能超过 10MB")
		return
	}

	src, err := file.Open()
	if err != nil {
		respondError(c, apperr.Internal("读取上传文件失败", err))
		return
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, service.MaxImageUploadSize+1))
	if err != nil {
		respondError(c, apperr.Internal("读取上传文件失败", err))
		return
	}
	if int64(len(data)) > service.MaxImageUploadSize {
		response.BadRequest(c, "图片不能超过 10MB")
		return
	}

	view, err := h.assets.Upload(c.Request.Context(), service.UploadAssetInput{
		OriginalName: file.Filename,
		Data:         data,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotImage) {
			response.BadRequest(c, "只支持图片文件")
			return
		}
		respondError(c, err)
		return
	}
	response.OK(c, view)
}

func (h *AssetHandler) Delete(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.assets.Delete(id); err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, nil)
}

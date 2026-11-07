package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"blog-api/internal/apperr"
	"blog-api/internal/response"
	"blog-api/internal/service"
)

type ImportHandler struct {
	imports *service.ImportService
}

func NewImportHandler(imports *service.ImportService) *ImportHandler {
	return &ImportHandler{imports: imports}
}

func (h *ImportHandler) UploadZip(c *gin.Context) {
	// 多留 1MB 给 multipart 头部；实际 ZIP 大小由 service 再校验。
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxZipUploadSize+1<<20)

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传 ZIP 文件（字段名 file）")
		return
	}

	directoryID, err := strconv.ParseUint(c.PostForm("directory_id"), 10, 64)
	if err != nil || directoryID == 0 {
		response.BadRequest(c, "directory_id 必须是正整数")
		return
	}
	publish := c.PostForm("publish") == "true" || c.PostForm("publish") == "1"

	src, err := file.Open()
	if err != nil {
		respondError(c, apperr.Internal("读取上传文件失败", err))
		return
	}
	defer src.Close()

	job, err := h.imports.ImportZip(c.Request.Context(), service.ZipImportInput{
		DirectoryID: uint(directoryID),
		Filename:    file.Filename,
		Size:        file.Size,
		Reader:      src,
		Publish:     publish,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, job)
}

func (h *ImportHandler) List(c *gin.Context) {
	page, pageSize, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}
	result, err := h.imports.List(page, pageSize)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *ImportHandler) Detail(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	job, err := h.imports.Get(id)
	if err != nil {
		respondError(c, err)
		return
	}
	response.OK(c, job)
}

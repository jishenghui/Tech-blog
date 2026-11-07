package handler

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"blog-api/internal/apperr"
	"blog-api/internal/response"
)

func respondError(c *gin.Context, err error) {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		if appErr.Status >= 500 {
			log.Printf("请求处理失败: %v", appErr)
		}
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	log.Printf("未处理错误: %v", err)
	response.Internal(c, "服务器内部错误")
}

func idParam(c *gin.Context) (uint, bool) {
	raw := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		respondError(c, apperr.BadRequest("无效的 ID"))
		return 0, false
	}
	return uint(id), true
}

func pagination(c *gin.Context) (int, int, error) {
	page := 1
	pageSize := 20

	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return 0, 0, apperr.BadRequest("page 必须是正整数")
		}
		page = value
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return 0, 0, apperr.BadRequest("page_size 必须在 1-100 之间")
		}
		pageSize = value
	}
	return page, pageSize, nil
}

func optionalUintQuery(c *gin.Context, key string) (*uint, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return nil, apperr.BadRequest(key + " 必须是正整数")
	}
	id := uint(value)
	return &id, nil
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		respondError(c, apperr.BadRequest("请求参数格式不正确"))
		return false
	}
	return true
}

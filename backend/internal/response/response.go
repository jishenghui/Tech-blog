package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope 是所有接口共用的响应信封。
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	CodeBadRequest   = 40001
	CodeUnauthorized = 40101
	CodeForbidden    = 40301
	CodeNotFound     = 40401
	CodeInternal     = 50001
	CodeUnavailable  = 50301
)

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data})
}

func Error(c *gin.Context, status, code int, message string) {
	c.JSON(status, Envelope{Code: code, Message: message})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, CodeForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, CodeInternal, message)
}

func Unavailable(c *gin.Context, message string) {
	Error(c, http.StatusServiceUnavailable, CodeUnavailable, message)
}

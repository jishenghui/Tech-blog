package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS 仅对公开读接口放行配置中的站点来源。
// 后台接口依赖 admin 子域同源反代，不开放跨域。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.TrimRight(origin, "/")] = struct{}{}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 后台管理与鉴权接口依赖同源反代，不开放跨域。
		if strings.HasPrefix(path, "/api/v1/admin/") || strings.HasPrefix(path, "/api/v1/auth/") {
			c.Next()
			return
		}

		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Add("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Header("Access-Control-Max-Age", "86400")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

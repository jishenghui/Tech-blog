package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"blog-api/internal/auth"
	"blog-api/internal/response"
)

const (
	ContextUserID   = "auth.user_id"
	ContextUsername = "auth.username"
)

// Auth 校验 Authorization: Bearer <access_token>，并把管理员身份写入上下文。
func Auth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Unauthorized(c, "未登录或登录已过期")
			c.Abort()
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(parts[1]), auth.TokenTypeAccess)
		if err != nil {
			response.Unauthorized(c, "未登录或登录已过期")
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Next()
	}
}

package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/response"
)

// New 组装 HTTP 路由。公开读接口与后台写接口共用 /api/v1 前缀。
func New(cfg *config.Config, db *gorm.DB, tokens *auth.TokenManager) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSAllowedOrigins))

	healthHandler := handler.NewHealthHandler(db)
	authHandler := handler.NewAuthHandler(db, tokens, cfg)

	r.GET("/healthz", healthHandler.Check)

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
			authGroup.POST("/logout", authHandler.Logout)
			authGroup.GET("/me", middleware.Auth(tokens), authHandler.Me)
			authGroup.PUT("/password", middleware.Auth(tokens), authHandler.ChangePassword)
		}

		// M2 起在此挂载目录/文章等后台路由；现在仅保留鉴权分组。
		_ = v1.Group("/admin", middleware.Auth(tokens))
	}

	r.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "接口不存在")
	})

	return r
}

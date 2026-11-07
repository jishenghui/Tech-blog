package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/response"
	"blog-api/internal/service"
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

	renderer := service.NewMarkdownRenderer()
	directoryService := service.NewDirectoryService(db)
	articleService := service.NewArticleService(db, renderer)
	tagService := service.NewTagService(db)

	directoryHandler := handler.NewDirectoryHandler(directoryService, articleService)
	articleHandler := handler.NewArticleHandler(articleService)
	tagHandler := handler.NewTagHandler(tagService, articleService)
	maintenanceHandler := handler.NewMaintenanceHandler(articleService)

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

		// 公开读接口：无鉴权，CORS 仅放行站点来源。
		v1.GET("/nav", directoryHandler.PublicNav)
		v1.GET("/directories/:slug", directoryHandler.PublicDetail)
		v1.GET("/articles", articleHandler.PublicList)
		v1.GET("/articles/:slug", articleHandler.PublicDetail)
		v1.GET("/tags", tagHandler.PublicList)
		v1.GET("/tags/:slug/articles", tagHandler.PublicArticles)
		v1.GET("/search", articleHandler.Search)

		admin := v1.Group("/admin", middleware.Auth(tokens))
		{
			admin.GET("/directories", directoryHandler.AdminTree)
			admin.POST("/directories", directoryHandler.Create)
			admin.POST("/directories/reorder", directoryHandler.Reorder)
			admin.PUT("/directories/:id", directoryHandler.Update)
			admin.POST("/directories/:id/dissolve", directoryHandler.Dissolve)
			admin.POST("/directories/:id/move", directoryHandler.Move)

			admin.GET("/articles", articleHandler.AdminList)
			admin.POST("/articles", articleHandler.Create)
			admin.GET("/articles/:id", articleHandler.AdminDetail)
			admin.PUT("/articles/:id", articleHandler.Update)
			admin.DELETE("/articles/:id", articleHandler.Delete)
			admin.POST("/articles/:id/publish", articleHandler.Publish)
			admin.POST("/articles/:id/unpublish", articleHandler.Unpublish)
			admin.POST("/articles/:id/move", articleHandler.Move)
			admin.PUT("/articles/:id/tags", articleHandler.SetTags)

			admin.GET("/tags", tagHandler.AdminList)
			admin.POST("/tags", tagHandler.Create)
			admin.PUT("/tags/:id", tagHandler.Update)
			admin.DELETE("/tags/:id", tagHandler.Delete)

			admin.POST("/maintenance/recalc", maintenanceHandler.Recalc)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "接口不存在")
	})

	return r
}

package router

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/response"
	"blog-api/internal/service"
	"blog-api/internal/storage"
)

// New 组装 HTTP 路由。公开读接口与后台写接口共用 /api/v1 前缀。
func New(cfg *config.Config, db *gorm.DB, tokens *auth.TokenManager) *gin.Engine {
	var uploader service.ObjectUploader
	if cfg.QiniuConfigured() {
		uploader = storage.NewQiniuUploader(cfg)
	} else {
		log.Println("七牛云未配置：图片上传与 ZIP 导入暂不可用（请在 backend/.env 填写 QINIU_ACCESS_KEY、QINIU_SECRET_KEY、QINIU_ASSET_BUCKET、QINIU_ASSET_DOMAIN）")
	}
	return NewWithUploader(cfg, db, tokens, uploader)
}

// NewWithUploader 允许测试注入假的对象存储上传器。
func NewWithUploader(cfg *config.Config, db *gorm.DB, tokens *auth.TokenManager, uploader service.ObjectUploader) *gin.Engine {
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
	assetService := service.NewAssetService(db, uploader, cfg.QiniuAssetBucket)
	importService := service.NewImportService(db, assetService, articleService)
	projectService := service.NewProjectService(db, renderer)
	profileService := service.NewProfileService(db, renderer)
	todoService := service.NewTodoService(db)

	directoryHandler := handler.NewDirectoryHandler(directoryService, articleService)
	articleHandler := handler.NewArticleHandler(articleService)
	tagHandler := handler.NewTagHandler(tagService, articleService)
	assetHandler := handler.NewAssetHandler(assetService)
	importHandler := handler.NewImportHandler(importService)
	projectHandler := handler.NewProjectHandler(projectService)
	profileHandler := handler.NewProfileHandler(profileService)
	todoHandler := handler.NewTodoHandler(todoService)
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
		v1.GET("/projects", projectHandler.PublicList)
		v1.GET("/projects/:slug", projectHandler.PublicDetail)
		v1.GET("/profile", profileHandler.PublicGet)
		v1.GET("/todos", todoHandler.PublicList)

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

			admin.POST("/assets", assetHandler.Upload)
			admin.DELETE("/assets/:id", assetHandler.Delete)

			admin.POST("/imports/zip", importHandler.UploadZip)
			admin.GET("/imports", importHandler.List)
			admin.GET("/imports/:id", importHandler.Detail)

			admin.GET("/projects", projectHandler.AdminList)
			admin.POST("/projects", projectHandler.Create)
			admin.POST("/projects/reorder", projectHandler.Reorder)
			admin.GET("/projects/:id", projectHandler.AdminDetail)
			admin.PUT("/projects/:id", projectHandler.Update)
			admin.DELETE("/projects/:id", projectHandler.Delete)

			admin.GET("/profile", profileHandler.AdminGet)
			admin.PUT("/profile", profileHandler.AdminUpdate)

			admin.GET("/todos", todoHandler.AdminList)
			admin.POST("/todos", todoHandler.Create)
			admin.POST("/todos/reorder", todoHandler.Reorder)
			admin.PUT("/todos/:id", todoHandler.Update)
			admin.DELETE("/todos/:id", todoHandler.Delete)

			admin.POST("/maintenance/recalc", maintenanceHandler.Recalc)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "接口不存在")
	})

	return r
}

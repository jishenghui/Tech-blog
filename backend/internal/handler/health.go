package handler

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"blog-api/internal/response"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Check 返回服务与数据库的就绪状态。
func (h *HealthHandler) Check(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil {
		log.Printf("健康检查获取连接池失败: %v", err)
		response.Unavailable(c, "数据库不可用")
		return
	}
	if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		log.Printf("健康检查数据库 ping 失败: %v", err)
		response.Unavailable(c, "数据库不可用")
		return
	}
	response.OK(c, gin.H{"status": "ok"})
}

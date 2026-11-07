package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/response"
)

const refreshCookieName = "blog_refresh_token"

const refreshCookiePath = "/api/v1/auth"

type AuthHandler struct {
	db     *gorm.DB
	tokens *auth.TokenManager
	cfg    *config.Config
}

func NewAuthHandler(db *gorm.DB, tokens *auth.TokenManager, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, tokens: tokens, cfg: cfg}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 校验用户名密码，返回 access token，并把 refresh token 写入 httpOnly Cookie。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "用户名和密码不能为空")
		return
	}

	var user model.User
	err := h.db.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Unauthorized(c, "用户名或密码错误")
			return
		}
		log.Printf("查询管理员失败: %v", err)
		response.Internal(c, "登录失败")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		response.Unauthorized(c, "用户名或密码错误")
		return
	}

	access, refresh, err := h.tokens.GeneratePair(user.ID, user.Username)
	if err != nil {
		log.Printf("签发令牌失败: %v", err)
		response.Internal(c, "登录失败")
		return
	}

	now := time.Now().UTC()
	if err := h.db.Model(&user).Update("last_login_at", now).Error; err != nil {
		log.Printf("更新最后登录时间失败: %v", err)
	}
	user.LastLoginAt = &now

	h.setRefreshCookie(c, refresh)
	response.OK(c, gin.H{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   int64(h.tokens.AccessTTL().Seconds()),
		"user":         userVO(user),
	})
}

// Refresh 使用 httpOnly Cookie 中的 refresh token 换取新的 access token，并轮换 refresh token。
func (h *AuthHandler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(refreshCookieName)
	if err != nil || raw == "" {
		response.Unauthorized(c, "登录已过期，请重新登录")
		return
	}

	claims, err := h.tokens.Parse(raw, auth.TokenTypeRefresh)
	if err != nil {
		h.clearRefreshCookie(c)
		response.Unauthorized(c, "登录已过期，请重新登录")
		return
	}

	var user model.User
	if err := h.db.First(&user, claims.UserID).Error; err != nil {
		h.clearRefreshCookie(c)
		response.Unauthorized(c, "登录已过期，请重新登录")
		return
	}

	access, refresh, err := h.tokens.GeneratePair(user.ID, user.Username)
	if err != nil {
		log.Printf("刷新令牌失败: %v", err)
		response.Internal(c, "刷新登录失败")
		return
	}

	h.setRefreshCookie(c, refresh)
	response.OK(c, gin.H{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   int64(h.tokens.AccessTTL().Seconds()),
	})
}

// Logout 清除 refresh Cookie。access token 无状态，由客户端丢弃。
func (h *AuthHandler) Logout(c *gin.Context) {
	h.clearRefreshCookie(c)
	response.OK(c, nil)
}

// Me 返回当前管理员信息。
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint(middleware.ContextUserID)
	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Unauthorized(c, "账号不存在")
			return
		}
		log.Printf("查询当前管理员失败: %v", err)
		response.Internal(c, "获取账号信息失败")
		return
	}
	response.OK(c, userVO(user))
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改管理员密码。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "旧密码和新密码不能为空")
		return
	}
	if len(req.NewPassword) < 8 {
		response.BadRequest(c, "新密码至少 8 个字符")
		return
	}
	if len([]byte(req.NewPassword)) > 72 {
		response.BadRequest(c, "新密码不能超过 72 字节")
		return
	}
	if req.OldPassword == req.NewPassword {
		response.BadRequest(c, "新密码不能与旧密码相同")
		return
	}

	userID := c.GetUint(middleware.ContextUserID)
	var user model.User
	if err := h.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Unauthorized(c, "账号不存在")
			return
		}
		log.Printf("查询当前管理员失败: %v", err)
		response.Internal(c, "修改密码失败")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.OldPassword) {
		response.BadRequest(c, "旧密码不正确")
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		log.Printf("生成新密码哈希失败: %v", err)
		response.Internal(c, "修改密码失败")
		return
	}
	if err := h.db.Model(&user).Update("password_hash", hash).Error; err != nil {
		log.Printf("更新密码失败: %v", err)
		response.Internal(c, "修改密码失败")
		return
	}
	response.OK(c, nil)
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		refreshCookieName,
		token,
		int(h.tokens.RefreshTTL().Seconds()),
		refreshCookiePath,
		"",
		h.cfg.IsProduction(),
		true,
	)
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", h.cfg.IsProduction(), true)
}

type userInfo struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	AvatarURL   string     `json:"avatar_url"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

func userVO(user model.User) userInfo {
	return userInfo{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		AvatarURL:   user.AvatarURL,
		LastLoginAt: user.LastLoginAt,
	}
}

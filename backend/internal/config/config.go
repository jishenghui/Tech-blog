package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Config 保存后端运行所需的全部配置。敏感项只从环境变量读取。
type Config struct {
	AppEnv             string
	HTTPAddr           string
	DBPath             string
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	AdminUsername      string
	AdminPasswordHash  string
	SiteURL            string
	APIBaseURL         string
	CORSAllowedOrigins []string

	QiniuAccessKey    string
	QiniuSecretKey    string
	QiniuStaticBucket string
	QiniuAssetBucket  string
	QiniuAssetDomain  string

	GitHubToken string
	GitHubRepo  string
}

// Load 读取 backend/.env（可选）与环境变量，并校验必填项。
func Load() (*Config, error) {
	// 本地开发使用 backend/.env；文件不存在时忽略，生产环境由 systemd 注入。
	loadDotEnv(".env")

	accessTTL, err := durationEnv("JWT_ACCESS_TTL", 2*time.Hour)
	if err != nil {
		return nil, err
	}
	refreshTTL, err := durationEnv("JWT_REFRESH_TTL", 168*time.Hour)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AppEnv:             env("APP_ENV", EnvDevelopment),
		HTTPAddr:           env("HTTP_ADDR", "127.0.0.1:8080"),
		DBPath:             env("DB_PATH", "./data/blog.db"),
		JWTSecret:          strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTAccessTTL:       accessTTL,
		JWTRefreshTTL:      refreshTTL,
		AdminUsername:      env("ADMIN_USERNAME", "admin"),
		AdminPasswordHash:  strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH")),
		SiteURL:            strings.TrimSpace(os.Getenv("SITE_URL")),
		APIBaseURL:         strings.TrimSpace(os.Getenv("API_BASE_URL")),
		CORSAllowedOrigins: splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		QiniuAccessKey:     strings.TrimSpace(os.Getenv("QINIU_ACCESS_KEY")),
		QiniuSecretKey:     strings.TrimSpace(os.Getenv("QINIU_SECRET_KEY")),
		QiniuStaticBucket:  env("QINIU_STATIC_BUCKET", "blog-static"),
		QiniuAssetBucket:   env("QINIU_ASSET_BUCKET", "blog-assets"),
		QiniuAssetDomain:   strings.TrimSpace(os.Getenv("QINIU_ASSET_DOMAIN")),
		GitHubToken:        strings.TrimSpace(os.Getenv("GITHUB_TOKEN")),
		GitHubRepo:         env("GITHUB_REPO", "jishenghui/Tech-blog"),
	}

	if cfg.AppEnv == EnvDevelopment {
		if cfg.SiteURL == "" {
			cfg.SiteURL = "http://localhost:4321"
		}
		if cfg.APIBaseURL == "" {
			cfg.APIBaseURL = "http://localhost:8080"
		}
		if len(cfg.CORSAllowedOrigins) == 0 {
			cfg.CORSAllowedOrigins = []string{"http://localhost:4321"}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate 在启动时尽早暴露配置错误。
func (c *Config) Validate() error {
	if c.AppEnv != EnvDevelopment && c.AppEnv != EnvProduction {
		return fmt.Errorf("APP_ENV 只能是 %s 或 %s，当前为 %q", EnvDevelopment, EnvProduction, c.AppEnv)
	}
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET 不能为空")
	}
	if c.AppEnv == EnvProduction && len(c.JWTSecret) < 32 {
		return errors.New("生产环境 JWT_SECRET 长度至少 32 个字符")
	}
	if strings.TrimSpace(c.AdminUsername) == "" {
		return errors.New("ADMIN_USERNAME 不能为空")
	}
	if c.JWTAccessTTL <= 0 || c.JWTRefreshTTL <= 0 {
		return errors.New("JWT_ACCESS_TTL 与 JWT_REFRESH_TTL 必须为正数")
	}
	if c.JWTRefreshTTL <= c.JWTAccessTTL {
		return errors.New("JWT_REFRESH_TTL 必须大于 JWT_ACCESS_TTL")
	}
	if c.HTTPAddr == "" {
		return errors.New("HTTP_ADDR 不能为空")
	}
	if c.DBPath == "" {
		return errors.New("DB_PATH 不能为空")
	}
	if c.AppEnv == EnvProduction {
		if c.SiteURL == "" {
			return errors.New("生产环境必须设置 SITE_URL")
		}
		if c.APIBaseURL == "" {
			return errors.New("生产环境必须设置 API_BASE_URL")
		}
		if len(c.CORSAllowedOrigins) == 0 {
			return errors.New("生产环境必须设置 CORS_ALLOWED_ORIGINS")
		}
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法的时长（如 2h、168h）: %w", key, err)
	}
	return d, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

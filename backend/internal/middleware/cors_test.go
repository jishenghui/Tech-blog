package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSOnlyAllowsConfiguredOriginOnPublicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CORS([]string{"http://localhost:4321"}))
	r.GET("/api/v1/articles", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

	tests := []struct {
		name       string
		method     string
		path       string
		origin     string
		wantStatus int
		wantAllow  bool
	}{
		{
			name:       "允许的站点来源访问公开接口",
			method:     http.MethodGet,
			path:       "/api/v1/articles",
			origin:     "http://localhost:4321",
			wantStatus: http.StatusOK,
			wantAllow:  true,
		},
		{
			name:       "未配置的来源不返回 CORS 头",
			method:     http.MethodGet,
			path:       "/api/v1/articles",
			origin:     "http://evil.example",
			wantStatus: http.StatusOK,
			wantAllow:  false,
		},
		{
			name:       "预检请求返回 204 与 CORS 头",
			method:     http.MethodOptions,
			path:       "/api/v1/articles",
			origin:     "http://localhost:4321",
			wantStatus: http.StatusNoContent,
			wantAllow:  true,
		},
		{
			name:       "鉴权接口不开放 CORS",
			method:     http.MethodPost,
			path:       "/api/v1/auth/login",
			origin:     "http://localhost:4321",
			wantStatus: http.StatusOK,
			wantAllow:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Origin", tt.origin)
			if tt.method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", http.MethodGet)
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("状态码不正确: got=%d want=%d", w.Code, tt.wantStatus)
			}
			gotAllow := w.Header().Get("Access-Control-Allow-Origin") != ""
			if gotAllow != tt.wantAllow {
				t.Fatalf("CORS 头不符合预期: got=%v want=%v headers=%v", gotAllow, tt.wantAllow, w.Header())
			}
		})
	}
}

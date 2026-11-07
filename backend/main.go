package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blog-api/internal/auth"
	"blog-api/internal/config"
	"blog-api/internal/database"
	"blog-api/internal/router"
	"blog-api/internal/seed"
)

func main() {
	var hashSource string
	flag.StringVar(&hashSource, "hash-password", "", "生成 ADMIN_PASSWORD_HASH 所需的 bcrypt 哈希后退出")
	flag.Parse()

	if hashSource != "" {
		hash, err := auth.HashPassword(hashSource)
		if err != nil {
			log.Fatalf("生成密码哈希失败: %v", err)
		}
		fmt.Println(hash)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	if err := seed.Run(db, cfg); err != nil {
		log.Fatalf("初始化种子数据失败: %v", err)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	engine := router.New(cfg, db, tokens)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("blog-api 已启动，监听 %s（APP_ENV=%s）", cfg.HTTPAddr, cfg.AppEnv)
		errCh <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务异常退出: %v", err)
		}
	case sig := <-quit:
		log.Printf("收到信号 %s，正在关闭服务...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("关闭 HTTP 服务出错: %v", err)
		}
	}

	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			log.Printf("关闭数据库失败: %v", err)
		}
	}
	log.Println("服务已退出")
}

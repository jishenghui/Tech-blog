package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/qiniu/go-sdk/v7/auth"
	qiniustorage "github.com/qiniu/go-sdk/v7/storage"

	"blog-api/internal/config"
)

// QiniuUploader 把图片上传到七牛云 Kodo，并返回对象 URL。
// 区域信息在首次上传时通过 UC API 自动获取并缓存。
type QiniuUploader struct {
	accessKey string
	secretKey string
	bucket    string
	domain    string

	mu       sync.Mutex
	uploader *qiniustorage.FormUploader
}

func NewQiniuUploader(cfg *config.Config) *QiniuUploader {
	return &QiniuUploader{
		accessKey: cfg.QiniuAccessKey,
		secretKey: cfg.QiniuSecretKey,
		bucket:    cfg.QiniuAssetBucket,
		domain:    strings.TrimRight(cfg.QiniuAssetDomain, "/"),
	}
}

func (u *QiniuUploader) Upload(ctx context.Context, key string, data []byte, mime string) (string, error) {
	uploader, err := u.formUploader()
	if err != nil {
		return "", err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	mac := auth.New(u.accessKey, u.secretKey)
	policy := qiniustorage.PutPolicy{Scope: u.bucket}
	token := policy.UploadToken(mac)

	ret := qiniustorage.PutRet{}
	extra := qiniustorage.PutExtra{MimeType: mime}
	if err := uploader.Put(ctx, &ret, token, key, bytes.NewReader(data), int64(len(data)), &extra); err != nil {
		return "", fmt.Errorf("七牛云上传失败: %w", err)
	}
	return u.domain + "/" + key, nil
}

func (u *QiniuUploader) formUploader() (*qiniustorage.FormUploader, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.uploader != nil {
		return u.uploader, nil
	}

	region, err := qiniustorage.GetRegionWithOptions(u.accessKey, u.bucket, qiniustorage.DefaultUCApiOptions())
	if err != nil {
		return nil, fmt.Errorf("获取七牛云存储区域失败: %w", err)
	}
	u.uploader = qiniustorage.NewFormUploader(&qiniustorage.Config{
		Region:   region,
		UseHTTPS: true,
	})
	return u.uploader, nil
}

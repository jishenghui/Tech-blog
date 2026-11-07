package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvDoesNotExpandDollarSigns(t *testing.T) {
	const (
		hashKey     = "BLOG_TEST_DOTENV_HASH"
		quoteKey    = "BLOG_TEST_DOTENV_QUOTED"
		existingKey = "BLOG_TEST_DOTENV_EXISTING"
	)
	const hash = `$2a$10$vcQK1C0FAgRVSrozSGVz8uPnv36WCzuILQ0pNWU/7esZn60Y/gtzC`

	t.Setenv(existingKey, "from-env")
	t.Cleanup(func() {
		_ = os.Unsetenv(hashKey)
		_ = os.Unsetenv(quoteKey)
	})

	path := filepath.Join(t.TempDir(), ".env")
	content := "# 注释行\n" +
		hashKey + "=" + hash + "\n" +
		quoteKey + "='带引号的值'\n" +
		existingKey + "=from-file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入测试 .env 失败: %v", err)
	}

	loadDotEnv(path)

	if got := os.Getenv(hashKey); got != hash {
		t.Fatalf("bcrypt 哈希被改写: got=%q want=%q", got, hash)
	}
	if got := os.Getenv(quoteKey); got != "带引号的值" {
		t.Fatalf("引号未被正确去除: %q", got)
	}
	if got := os.Getenv(existingKey); got != "from-env" {
		t.Fatalf("已存在的环境变量不应被 .env 覆盖: %q", got)
	}
}

package auth

import (
	"testing"
	"time"
)

func TestGeneratePairAndParse(t *testing.T) {
	manager := NewTokenManager("test-secret-at-least-32-characters-long", time.Hour, 24*time.Hour)

	access, refresh, err := manager.GeneratePair(7, "admin")
	if err != nil {
		t.Fatalf("签发令牌失败: %v", err)
	}
	if access == refresh {
		t.Fatal("access token 与 refresh token 不应相同")
	}

	accessClaims, err := manager.Parse(access, TokenTypeAccess)
	if err != nil {
		t.Fatalf("解析 access token 失败: %v", err)
	}
	if accessClaims.UserID != 7 || accessClaims.Username != "admin" {
		t.Fatalf("access token 声明不正确: %+v", accessClaims)
	}

	refreshClaims, err := manager.Parse(refresh, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("解析 refresh token 失败: %v", err)
	}
	if refreshClaims.UserID != 7 {
		t.Fatalf("refresh token 声明不正确: %+v", refreshClaims)
	}

	if _, err := manager.Parse(refresh, TokenTypeAccess); err == nil {
		t.Fatal("refresh token 不应能当作 access token 使用")
	}
	if _, err := manager.Parse(access+"tampered", TokenTypeAccess); err == nil {
		t.Fatal("被篡改的 token 不应通过校验")
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	manager := NewTokenManager("test-secret-at-least-32-characters-long", -time.Minute, time.Hour)
	access, _, err := manager.GeneratePair(1, "admin")
	if err != nil {
		t.Fatalf("签发令牌失败: %v", err)
	}
	if _, err := manager.Parse(access, TokenTypeAccess); err == nil {
		t.Fatal("过期 token 不应通过校验")
	}
}

package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"

	tokenIssuer = "blog-api"
)

// ErrInvalidToken 统一表示令牌无效、过期或类型不符，避免向客户端泄露细节。
var ErrInvalidToken = errors.New("令牌无效或已过期")

// Claims 是 access/refresh token 共用的声明。
type Claims struct {
	UserID    uint   `json:"uid"`
	Username  string `json:"username"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// TokenManager 负责签发与校验 HS256 JWT。
type TokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenManager(secret string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (m *TokenManager) AccessTTL() time.Duration {
	return m.accessTTL
}

func (m *TokenManager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

// GeneratePair 同时签发 access token 与 refresh token。
func (m *TokenManager) GeneratePair(userID uint, username string) (access, refresh string, err error) {
	now := time.Now().UTC()
	access, err = m.sign(userID, username, TokenTypeAccess, now, m.accessTTL)
	if err != nil {
		return "", "", err
	}
	refresh, err = m.sign(userID, username, TokenTypeRefresh, now, m.refreshTTL)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (m *TokenManager) sign(userID uint, username, tokenType string, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Username:  username,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse 校验签名、有效期、签发者与 token 类型。
func (m *TokenManager) Parse(raw, expectedType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
	)
	if err != nil || !token.Valid || claims.TokenType != expectedType || claims.UserID == 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

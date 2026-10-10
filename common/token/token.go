// Package token JWT(HS256) 签发与解析。
// Claims 字段与 repo_backend hono/jwt payload 及 Redis 中 AuthUser JSON 完全对齐：
// userId / email / username / tenantId / role / isSuperUser / exp
package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TTL 登录态有效期 1 天
const TTL = 24 * time.Hour

// MerchantRole 普通用户角色（与 repo_backend 一致）
const MerchantRole = "merchant"

type Claims struct {
	UserID      int64   `json:"userId"`
	Email       string  `json:"email"`
	Username    *string `json:"username"`
	TenantID    *int64  `json:"tenantId"`
	Role        string  `json:"role"`
	IsSuperUser bool    `json:"isSuperUser"`
	jwt.RegisteredClaims
}

// Sign 使用 HS256 + 共享密钥签发 token
func Sign(secret string, c *Claims) (string, error) {
	if c.RegisteredClaims.ExpiresAt == nil {
		c.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(TTL))
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// Parse 解析并校验签名与过期时间
func Parse(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

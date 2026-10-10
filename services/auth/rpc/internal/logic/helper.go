package logic

import (
	"context"
	"database/sql"
	"encoding/json"

	"sso_login/common/token"
	"sso_login/services/auth/rpc/auth"
)

// Redis key 与 TTL，命名与 repo_backend 完全一致
const (
	ttlSession     = 24 * 60 * 60 // token:{token} 登录态 1 天
	ttlCaptcha     = 5 * 60       // captcha:login:{id} 5 分钟
	ttlNonce       = 5 * 60       // nonce:{nonce} 5 分钟
	ttlFreeze      = 60 * 60      // 失败计数 / 锁定窗口 1 小时
	ttlOAuthCode   = 5 * 60       // oauth:code:{code} 授权码 5 分钟
	passwordMaxTry = 6            // 窗口内失败 6 次锁定
)

func captchaKey(id string) string  { return "captcha:login:" + id }
func nonceKey(nonce string) string { return "nonce:" + nonce }
func lockedKey(email string) string {
	return "login:locked:" + email
}
func failedKey(email string) string {
	return "login:failed:" + email
}
func sessionKey(t string) string { return "token:" + t }
func oauthCodeKey(code string) string {
	return "oauth:code:" + code
}

// oauthCodePayload 授权码内容（Redis JSON），一次性使用
type oauthCodePayload struct {
	UserId      int64  `json:"user_id"`
	ClientId    string `json:"client_id"`
	RedirectUri string `json:"redirect_uri"`
}

// saveSession 将登录态 JSON 写入 Redis（与 repo_backend token:{token} 结构一致）
func saveSession(ctx context.Context, r redisWriter, key string, c *token.Claims) error {
	body, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return r.SetexCtx(ctx, key, string(body), ttlSession)
}

// loadSession 读取并解析登录态；不存在返回 (nil, nil)
func loadSession(ctx context.Context, r redisReader, key string) (*token.Claims, error) {
	raw, err := r.GetCtx(ctx, key)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, nil
	}
	c := &token.Claims{}
	if err := json.Unmarshal([]byte(raw), c); err != nil {
		return nil, err
	}
	return c, nil
}

// claimsToProto 内部 Claims → gRPC UserInfo
func claimsToProto(c *token.Claims) *auth.UserInfo {
	u := &auth.UserInfo{
		UserId:      c.UserID,
		Email:       c.Email,
		Role:        c.Role,
		IsSuperUser: c.IsSuperUser,
	}
	if c.Username != nil {
		u.Username = *c.Username
	}
	if c.TenantID != nil {
		u.TenantId = *c.TenantID
	}
	if c.ExpiresAt != nil {
		u.Exp = c.ExpiresAt.Unix()
	}
	return u
}

// nullString sql.NullString 便捷构造
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// 最小化接口，便于在 logic 中直接使用 svcCtx.Redis 并让辅助函数可测试
type redisReader interface {
	GetCtx(ctx context.Context, key string) (string, error)
}

type redisWriter interface {
	SetexCtx(ctx context.Context, key, value string, seconds int) error
}

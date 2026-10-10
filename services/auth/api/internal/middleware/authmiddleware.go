package middleware

import (
	"context"
	"net/http"
	"strings"

	"sso_login/services/auth/rpc/authclient"
)

type ctxKey string

// UserContextKey 中间件校验通过后，将 *authclient.UserInfo 注入 request context
const UserContextKey ctxKey = "ssoAuthUser"

// TokenContextKey 将 token 原文注入 request context，供登出/改密等 logic 使用
const TokenContextKey ctxKey = "ssoAuthToken"

// AuthMiddleware 登录态校验中间件（语义对齐 repo_backend authMiddleware）：
// 从 authorization 头取 token（兼容 "Bearer xxx"），经 gRPC 调 auth.rpc ValidateSession
// 校验 Redis 会话；无效返回 401 空响应，有效则把用户信息注入 context 后放行。
type AuthMiddleware struct {
	AuthClient authclient.Auth
}

func NewAuthMiddleware(c authclient.Auth) *AuthMiddleware {
	return &AuthMiddleware{AuthClient: c}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get("authorization"))
		token = strings.TrimPrefix(token, "Bearer ")
		token = strings.TrimSpace(token)
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		resp, err := m.AuthClient.ValidateSession(r.Context(),
			&authclient.ValidateSessionReq{Token: token})
		if err != nil || resp == nil || resp.Code != 200 || resp.User == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, resp.User)
		ctx = context.WithValue(ctx, TokenContextKey, token)
		next(w, r.WithContext(ctx))
	}
}

// OAuth2 授权码模式协议端点（独立于 goctl 生成的 JSON 路由，因需要 302 重定向 / 表单解析 / 静态页输出）
package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/rest"
)

// ssoCookieName SSO 域会话 cookie（httpOnly，由登录接口成功时写入）
const ssoCookieName = "sso_token"

// RegisterOAuthRoutes 在 REST 服务上注册 OAuth2 协议端点与 SSO 页面
func RegisterOAuthRoutes(server *rest.Server, svcCtx *svc.ServiceContext, loginPage, demoPage []byte) {
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/oauth/authorize", Handler: oauthAuthorizeHandler(svcCtx)})
	server.AddRoute(rest.Route{Method: http.MethodPost, Path: "/oauth/token", Handler: oauthTokenHandler(svcCtx)})
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/oauth/userinfo", Handler: oauthUserinfoHandler(svcCtx)})
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/oauth/clientinfo", Handler: oauthClientinfoHandler(svcCtx)})
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/oauth/login", Handler: staticPageHandler(loginPage)})
	// SSO 单点登出：业务系统退出后引导用户来此清除 SSO 域 cookie，再 302 回跳
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/oauth/logout", Handler: oauthLogoutHandler()})
	// 演示业务系统（模拟第三方站点，含 /callback 回跳路径）
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/demo-app", Handler: staticPageHandler(demoPage)})
	server.AddRoute(rest.Route{Method: http.MethodGet, Path: "/demo-app/callback", Handler: staticPageHandler(demoPage)})
}

func staticPageHandler(page []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	}
}

// oauthAuthorizeHandler GET /oauth/authorize?client_id&redirect_uri&response_type=code&state
// 已登录（sso_token cookie）→ 颁发授权码并 302 回业务系统；未登录 → 302 到 SSO 独立登录页
func oauthAuthorizeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		clientId, redirectUri, responseType, state :=
			q.Get("client_id"), q.Get("redirect_uri"), q.Get("response_type"), q.Get("state")

		if clientId == "" || redirectUri == "" {
			oauthErrorPage(w, "缺少 client_id 或 redirect_uri 参数", http.StatusBadRequest)
			return
		}
		if responseType != "code" {
			oauthErrorPage(w, "仅支持 response_type=code", http.StatusBadRequest)
			return
		}

		// 1. 通过 SSO 域 cookie 识别登录态
		userId := int64(0)
		if tk := readSSOToken(r); tk != "" {
			vs, err := svcCtx.AuthRpc.ValidateSession(r.Context(), &authclient.ValidateSessionReq{Token: tk})
			if err == nil && vs.Code == 200 {
				userId = vs.User.UserId
			}
		}
		if userId == 0 {
			// 未登录 → 跳转 SSO 独立登录页（携带原授权参数）
			loginUrl := "/oauth/login?" + url.Values{
				"client_id":     {clientId},
				"redirect_uri":  {redirectUri},
				"response_type": {responseType},
				"state":         {state},
			}.Encode()
			http.Redirect(w, r, loginUrl, http.StatusFound)
			return
		}

		// 2. 已登录 → 为业务系统颁发一次性授权码
		ar, err := svcCtx.AuthRpc.OAuthAuthorize(r.Context(), &authclient.OAuthAuthorizeReq{
			UserId:      userId,
			ClientId:    clientId,
			RedirectUri: redirectUri,
		})
		if err != nil {
			oauthErrorPage(w, "授权服务暂不可用，请稍后重试", http.StatusBadGateway)
			return
		}
		if ar.Code != 200 {
			// client / redirect_uri 校验失败时绝不能重定向到未验证的 URI
			oauthErrorPage(w, ar.Message, http.StatusBadRequest)
			return
		}

		// 3. 302 携带 code 与 state 回业务系统
		u, err := url.Parse(redirectUri)
		if err != nil {
			oauthErrorPage(w, "redirect_uri 非法", http.StatusBadRequest)
			return
		}
		qs := u.Query()
		qs.Set("code", ar.AuthCode)
		if state != "" {
			qs.Set("state", state)
		}
		u.RawQuery = qs.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	}
}

// oauthTokenHandler POST /oauth/token（form-urlencoded 或 JSON）
// 业务系统服务端用 code + client 凭证换 access_token
func oauthTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var clientId, clientSecret, code, redirectUri, grantType string
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			clientId, clientSecret = body["client_id"], body["client_secret"]
			code, redirectUri = body["code"], body["redirect_uri"]
			grantType = body["grant_type"]
		} else {
			_ = r.ParseForm()
			clientId = r.PostFormValue("client_id")
			clientSecret = r.PostFormValue("client_secret")
			code = r.PostFormValue("code")
			redirectUri = r.PostFormValue("redirect_uri")
			grantType = r.PostFormValue("grant_type")
		}
		if grantType != "" && grantType != "authorization_code" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "unsupported_grant_type", "error_description": "仅支持 authorization_code",
			})
			return
		}

		resp, err := svcCtx.AuthRpc.OAuthExchangeToken(r.Context(), &authclient.OAuthExchangeReq{
			ClientId:     clientId,
			ClientSecret: clientSecret,
			AuthCode:     code,
			RedirectUri:  redirectUri,
		})
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"error": "server_error", "error_description": "授权服务暂不可用",
			})
			return
		}
		if resp.Code != 200 {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": "invalid_grant", "error_description": resp.Message,
			})
			return
		}
		// 标准 OAuth2 token 响应，附带 userinfo 便于业务系统直接展示
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": resp.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   resp.ExpiresIn,
			"userinfo":     resp.User,
		})
	}
}

// oauthUserinfoHandler GET /oauth/userinfo
// token 来源：authorization 头（Bearer xxx 或原文）→ sso_token cookie
func oauthUserinfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tk := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tk == "" {
			tk = readSSOToken(r)
		}
		if tk == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"code": 401, "message": "缺少 token"})
			return
		}
		vs, err := svcCtx.AuthRpc.ValidateSession(r.Context(), &authclient.ValidateSessionReq{Token: tk})
		if err != nil || vs.Code != 200 {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"code": 401, "message": "会话无效或已过期"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"code": 200, "message": "ok", "data": vs.User})
	}
}

// oauthClientinfoHandler GET /oauth/clientinfo?client_id=
// 登录页展示应用名称（仅公开 name，不含任何凭证信息）
func oauthClientinfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientId := r.URL.Query().Get("client_id")
		resp, err := svcCtx.AuthRpc.OAuthClientInfo(r.Context(), &authclient.OAuthClientReq{ClientId: clientId})
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"code": 10015, "message": "授权服务暂不可用"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"code": resp.Code, "message": resp.Message, "name": resp.Name, "valid": resp.Valid})
	}
}

func readSSOToken(r *http.Request) string {
	if c, err := r.Cookie(ssoCookieName); err == nil {
		return c.Value
	}
	return ""
}

// oauthLogoutHandler GET /oauth/logout?redirect=<回跳地址>
// 业务系统退出后引导用户来此清除 SSO 域会话 cookie，再 302 回跳到调用方提供的地址。
// 设计目标：让"单点登出"在跨域 httpOnly cookie 场景下可落地——业务系统无法跨域清 SSO cookie，
// 只能让浏览器导航到本端点（顶层导航带 cookie），SSO 清完 cookie 再把用户送回业务系统。
// 安全：redirect 必须是 http/https 绝对地址，拒绝 javascript:/data: 等协议，避免 open-redirect XSS。
func oauthLogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirect := r.URL.Query().Get("redirect")
		// 无论是否带 redirect，都先清 SSO cookie
		http.SetCookie(w, &http.Cookie{
			Name:     ssoCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		if redirect == "" {
			// 无回跳目标：展示轻量登出完成页，提供返回入口
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8">
<title>已退出登录</title><style>body{font-family:system-ui;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f5f6fa}
.card{background:#fff;padding:40px 48px;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);text-align:center}
h2{color:#1a1a1a;margin:0 0 12px}p{color:#666;margin:0}</style></head>
<body><div class="card"><h2>已退出 SSO 登录</h2><p>请关闭本页或重新登录。</p></div></body></html>`)
			return
		}
		u, err := url.Parse(redirect)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			// 非法回跳：只清 cookie，不跳转，避免 open-redirect
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8">
<title>退出失败</title></head><body><p>回跳地址非法，SSO 会话已清除。</p></body></html>`)
			return
		}
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func oauthErrorPage(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8">
<title>授权失败</title><style>body{font-family:system-ui;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;background:#f5f6fa}
.card{background:#fff;padding:40px 48px;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);text-align:center}
h2{color:#e5484d;margin:0 0 12px}p{color:#666;margin:0 0 24px}a{color:#1677ff;text-decoration:none}</style></head>
<body><div class="card"><h2>授权失败</h2><p>%s</p><a href="/oauth/login">返回 SSO 登录</a></div></body></html>`,
		html.EscapeString(msg))
}

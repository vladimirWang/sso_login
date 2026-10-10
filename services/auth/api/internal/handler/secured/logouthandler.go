// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package secured

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"sso_login/services/auth/api/internal/logic/secured"
	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
)

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LogoutReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := secured.NewLogoutLogic(r.Context(), svcCtx)
		resp, err := l.Logout(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		// 登出成功：清除 SSO 域会话 cookie
		if resp.Code == 200 {
			http.SetCookie(w, &http.Cookie{
				Name:     "sso_token",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   -1,
			})
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

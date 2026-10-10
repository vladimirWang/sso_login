// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package common

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"sso_login/services/auth/api/internal/logic/common"
	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
)

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := common.NewLoginLogic(r.Context(), svcCtx)
		resp, err := l.Login(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		// 登录成功：token 写入 SSO 域 httpOnly cookie，供 /oauth/authorize 识别登录态
		if resp.Code == 200 && resp.Data != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "sso_token",
				Value:    resp.Data,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   86400,
			})
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

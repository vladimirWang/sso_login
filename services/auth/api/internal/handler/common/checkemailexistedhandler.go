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

func CheckEmailExistedHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EmailPath
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := common.NewCheckEmailExistedLogic(r.Context(), svcCtx)
		resp, err := l.CheckEmailExisted(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}

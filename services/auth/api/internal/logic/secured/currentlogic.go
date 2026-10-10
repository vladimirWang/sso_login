package secured

import (
	"context"

	"sso_login/common/errcode"
	"sso_login/services/auth/api/internal/middleware"
	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type CurrentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCurrentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CurrentLogic {
	return &CurrentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Current 用户信息已由 Auth 中间件经 gRPC ValidateSession 注入 ctx，直接返回
func (l *CurrentLogic) Current() (resp *types.CurrentResp, err error) {
	u, ok := l.ctx.Value(middleware.UserContextKey).(*authclient.UserInfo)
	if !ok || u == nil {
		return &types.CurrentResp{Code: errcode.SessionInvalid, Message: "未登录或登录已过期"}, nil
	}
	return &types.CurrentResp{
		Code:    errcode.Success,
		Message: "获取用户信息成功",
		Data: types.UserInfo{
			UserId:      u.UserId,
			Email:       u.Email,
			Username:    u.Username,
			TenantId:    u.TenantId,
			Role:        u.Role,
			IsSuperUser: u.IsSuperUser,
			Exp:         u.Exp,
		},
	}, nil
}

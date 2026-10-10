package secured

import (
	"context"

	"sso_login/services/auth/api/internal/middleware"
	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout(req *types.LogoutReq) (resp *types.BaseResp, err error) {
	token, _ := l.ctx.Value(middleware.TokenContextKey).(string)
	r, err := l.svcCtx.AuthRpc.Logout(l.ctx, &authclient.LogoutReq{Token: token})
	if err != nil {
		return nil, err
	}
	return &types.BaseResp{Code: r.Code, Message: r.Message}, nil
}

package secured

import (
	"context"

	"sso_login/services/auth/api/internal/middleware"
	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdatePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdatePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePasswordLogic {
	return &UpdatePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdatePasswordLogic) UpdatePassword(req *types.UpdatePasswordReq) (resp *types.BaseResp, err error) {
	token, _ := l.ctx.Value(middleware.TokenContextKey).(string)
	r, err := l.svcCtx.AuthRpc.UpdatePassword(l.ctx, &authclient.UpdatePasswordReq{
		Token:    token,
		Current:  req.Current,
		Password: req.Password,
		Nonce:    req.Nonce,
	})
	if err != nil {
		return nil, err
	}
	return &types.BaseResp{Code: r.Code, Message: r.Message}, nil
}

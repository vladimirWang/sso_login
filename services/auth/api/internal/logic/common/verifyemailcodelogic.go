// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package common

import (
	"context"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVerifyEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyEmailCodeLogic {
	return &VerifyEmailCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VerifyEmailCodeLogic) VerifyEmailCode(req *types.EmailCodeReq) (resp *types.EmailCodeResp, err error) {
	r, err := l.svcCtx.AuthRpc.VerifyEmailCode(l.ctx, &authclient.EmailCodeReq{
		Email: req.Email,
		Code:  req.Code,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmailCodeResp{Code: r.Code, Message: r.Message}, nil
}

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

type SendEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendEmailCodeLogic) SendEmailCode(req *types.EmailCodeReq) (resp *types.EmailCodeResp, err error) {
	r, err := l.svcCtx.AuthRpc.SendEmailCode(l.ctx, &authclient.EmailCodeReq{
		Email: req.Email,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmailCodeResp{Code: r.Code, Message: r.Message, Data: r.Ttl}, nil
}

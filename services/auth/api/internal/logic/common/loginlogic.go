package common

import (
	"context"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	r, err := l.svcCtx.AuthRpc.Login(l.ctx, &authclient.LoginReq{
		Email:       req.Email,
		Password:    req.Password,
		CaptchaId:   req.CaptchaId,
		CaptchaText: req.CaptchaText,
		Nonce:       req.Nonce,
	})
	if err != nil {
		return nil, err
	}
	// data 为 token 原文（与 repo_backend 登录响应一致）
	return &types.LoginResp{Code: r.Code, Message: r.Message, Data: r.Token}, nil
}

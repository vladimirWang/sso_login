package common

import (
	"context"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSaltLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSaltLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSaltLogic {
	return &GetSaltLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSaltLogic) GetSalt(req *types.EmailPath) (resp *types.SaltResp, err error) {
	r, err := l.svcCtx.AuthRpc.GetSalt(l.ctx, &authclient.EmailReq{Email: req.Email})
	if err != nil {
		return nil, err
	}
	return &types.SaltResp{Code: r.Code, Message: r.Message, Data: r.Salt}, nil
}

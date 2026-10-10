package common

import (
	"context"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNonceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetNonceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNonceLogic {
	return &GetNonceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetNonceLogic) GetNonce() (resp *types.NonceResp, err error) {
	r, err := l.svcCtx.AuthRpc.GetNonce(l.ctx, nil)
	if err != nil {
		return nil, err
	}
	return &types.NonceResp{Code: r.Code, Message: r.Message, Data: r.Nonce}, nil
}

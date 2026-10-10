package logic

import (
	"context"

	"sso_login/common/crypto"
	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetNonceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetNonceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetNonceLogic {
	return &GetNonceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetNonce 生成一次性 nonce（对齐 repo_backend util.getNonce）
func (l *GetNonceLogic) GetNonce(in *auth.NonceReq) (*auth.NonceResp, error) {
	return &auth.NonceResp{
		Code:    errcode.Success,
		Message: "nonce生成成功",
		Nonce:   crypto.GenerateNonce(),
	}, nil
}

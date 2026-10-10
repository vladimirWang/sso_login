package common

import (
	"context"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckEmailExistedLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckEmailExistedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckEmailExistedLogic {
	return &CheckEmailExistedLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckEmailExistedLogic) CheckEmailExisted(req *types.EmailPath) (resp *types.EmailExistedResp, err error) {
	r, err := l.svcCtx.AuthRpc.CheckEmailExisted(l.ctx, &authclient.EmailReq{Email: req.Email})
	if err != nil {
		return nil, err
	}
	return &types.EmailExistedResp{
		Code:    r.Code,
		Message: r.Message,
		Data:    r.Existed,
	}, nil
}

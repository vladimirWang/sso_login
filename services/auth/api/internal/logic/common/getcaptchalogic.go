package common

import (
	"context"

	"sso_login/services/auth/api/internal/svc"
	"sso_login/services/auth/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaptchaLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaptchaLogic {
	return &GetCaptchaLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaptchaLogic) GetCaptcha() (resp *types.CaptchaResp, err error) {
	r, err := l.svcCtx.AuthRpc.GetCaptcha(l.ctx, nil)
	if err != nil {
		return nil, err
	}
	return &types.CaptchaResp{
		Code:    r.Code,
		Message: r.Message,
		Data: types.CaptchaData{
			Image:     r.Image,
			CaptchaId: r.CaptchaId,
		},
	}, nil
}

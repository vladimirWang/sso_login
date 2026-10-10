package logic

import (
	"context"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CurrentUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CurrentUserLogic {
	return &CurrentUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CurrentUser 按 token 取当前登录用户（对齐 repo_backend getCurrentUser）
func (l *CurrentUserLogic) CurrentUser(in *auth.CurrentReq) (*auth.CurrentResp, error) {
	claims, err := loadSession(l.ctx, l.svcCtx.Redis, sessionKey(in.Token))
	if err != nil {
		l.Errorf("load session error: %v", err)
		return &auth.CurrentResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if claims == nil {
		return &auth.CurrentResp{Code: errcode.SessionInvalid, Message: "未登录或登录已过期"}, nil
	}
	return &auth.CurrentResp{
		Code:    errcode.Success,
		Message: "获取用户信息成功",
		User:    claimsToProto(claims),
	}, nil
}

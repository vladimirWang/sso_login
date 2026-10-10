package logic

import (
	"context"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewValidateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateSessionLogic {
	return &ValidateSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ValidateSession 供其他业务系统 / API 网关接入 SSO 使用：
// 校验 token 对应的 Redis 会话是否存在，存在则返回用户信息。
// 会话有效性以 Redis token:{token} 为准（登出即失效），与 repo_backend 鉴权语义一致。
func (l *ValidateSessionLogic) ValidateSession(in *auth.ValidateSessionReq) (*auth.ValidateSessionResp, error) {
	if in.Token == "" {
		return &auth.ValidateSessionResp{Code: errcode.SessionInvalid, Message: "缺少token"}, nil
	}
	claims, err := loadSession(l.ctx, l.svcCtx.Redis, sessionKey(in.Token))
	if err != nil {
		l.Errorf("load session error: %v", err)
		return &auth.ValidateSessionResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if claims == nil {
		return &auth.ValidateSessionResp{Code: errcode.SessionInvalid, Message: "未登录或登录已过期"}, nil
	}
	return &auth.ValidateSessionResp{
		Code:    errcode.Success,
		Message: "ok",
		User:    claimsToProto(claims),
	}, nil
}

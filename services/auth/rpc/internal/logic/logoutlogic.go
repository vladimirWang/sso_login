package logic

import (
	"context"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/event"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Logout 登出（对齐 repo_backend logoutUser）：删除 Redis 会话即立即失效。
// 接口幂等：会话不存在也返回成功。
func (l *LogoutLogic) Logout(in *auth.LogoutReq) (*auth.BaseResp, error) {
	key := sessionKey(in.Token)

	// 先取会话用于事件通知，再删除
	if claims, err := loadSession(l.ctx, l.svcCtx.Redis, key); err != nil {
		l.Errorf("load session error: %v", err)
	} else if claims != nil {
		if err = l.svcCtx.Publisher.Publish(l.ctx, event.RoutingUserLogout, event.Event{
			UserID:   claims.UserID,
			Email:    claims.Email,
			TenantID: claims.TenantID,
		}); err != nil {
			l.Errorf("publish logout event error: %v", err)
		}
	}

	if _, err := l.svcCtx.Redis.DelCtx(l.ctx, key); err != nil {
		l.Errorf("del session error: %v", err)
		return &auth.BaseResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	return &auth.BaseResp{Code: errcode.Success, Message: "用户登出成功"}, nil
}

package logic

import (
	"context"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type OAuthClientInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOAuthClientInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OAuthClientInfoLogic {
	return &OAuthClientInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询 client 基本信息（登录页展示应用名称）
func (l *OAuthClientInfoLogic) OAuthClientInfo(in *auth.OAuthClientReq) (*auth.OAuthClientResp, error) {
	client, err := l.svcCtx.OAuthClientModel.FindOneByClientId(l.ctx, in.ClientId)
	if err == sqlx.ErrNotFound {
		return &auth.OAuthClientResp{Code: errcode.Success, Valid: false}, nil
	}
	if err != nil {
		l.Errorf("find oauth client error: %v", err)
		return &auth.OAuthClientResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	return &auth.OAuthClientResp{
		Code:  errcode.Success,
		Name:  client.Name,
		Valid: client.Status == "ACTIVE",
	}, nil
}

package logic

import (
	"context"
	"encoding/json"

	"sso_login/common/crypto"
	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type OAuthAuthorizeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOAuthAuthorizeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OAuthAuthorizeLogic {
	return &OAuthAuthorizeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// OAuth2 授权码模式：已登录用户为业务系统颁发授权码（5 分钟一次性）
func (l *OAuthAuthorizeLogic) OAuthAuthorize(in *auth.OAuthAuthorizeReq) (*auth.OAuthAuthorizeResp, error) {
	// 1. client 必须已注册且启用
	client, err := l.svcCtx.OAuthClientModel.FindOneByClientId(l.ctx, in.ClientId)
	if err == sqlx.ErrNotFound {
		return &auth.OAuthAuthorizeResp{Code: errcode.ValidationError, Message: "client_id 无效"}, nil
	}
	if err != nil {
		l.Errorf("find oauth client error: %v", err)
		return &auth.OAuthAuthorizeResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if client.Status != "ACTIVE" {
		return &auth.OAuthAuthorizeResp{Code: errcode.Forbidden, Message: "应用已被停用"}, nil
	}
	// 2. redirect_uri 必须与注册的完全一致（防开放重定向）
	if !client.ValidateRedirectUri(in.RedirectUri) {
		return &auth.OAuthAuthorizeResp{Code: errcode.ValidationError, Message: "redirect_uri 未在应用中注册"}, nil
	}

	// 3. 生成一次性授权码，Redis 保存 5 分钟
	code := crypto.GenerateSalt()
	payload, _ := json.Marshal(oauthCodePayload{
		UserId:      in.UserId,
		ClientId:    in.ClientId,
		RedirectUri: in.RedirectUri,
	})
	if err = l.svcCtx.Redis.SetexCtx(l.ctx, oauthCodeKey(code), string(payload), ttlOAuthCode); err != nil {
		l.Errorf("save oauth code error: %v", err)
		return &auth.OAuthAuthorizeResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	return &auth.OAuthAuthorizeResp{
		Code:     errcode.Success,
		Message:  "ok",
		AuthCode: code,
	}, nil
}

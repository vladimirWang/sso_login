package logic

import (
	"context"
	"encoding/json"

	"sso_login/common/crypto"
	"sso_login/common/errcode"
	"sso_login/common/token"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/event"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type OAuthExchangeTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOAuthExchangeTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OAuthExchangeTokenLogic {
	return &OAuthExchangeTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// OAuthExchangeToken 业务系统用授权码 + client 凭证换 access_token（JWT + Redis 会话，
// 会话结构与普通登录完全一致，可被 ValidateSession / logout 统一管理）
func (l *OAuthExchangeTokenLogic) OAuthExchangeToken(in *auth.OAuthExchangeReq) (*auth.OAuthExchangeResp, error) {
	// 1. 校验 client 凭证（secret 存储 sha256 后的十六进制）
	client, err := l.svcCtx.OAuthClientModel.FindOneByClientId(l.ctx, in.ClientId)
	if err == sqlx.ErrNotFound {
		return oauthExchangeFail(errcode.ValidationError, "client_id 无效"), nil
	}
	if err != nil {
		l.Errorf("find oauth client error: %v", err)
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}
	if client.Status != "ACTIVE" {
		return oauthExchangeFail(errcode.Forbidden, "应用已被停用"), nil
	}
	if crypto.Sha256(in.ClientSecret) != client.ClientSecret {
		return oauthExchangeFail(errcode.ValidationError, "client_secret 不正确"), nil
	}

	// 2. 消费授权码：读取后立即删除，保证一次性
	cKey := oauthCodeKey(in.AuthCode)
	raw, err := l.svcCtx.Redis.GetCtx(l.ctx, cKey)
	if err != nil {
		l.Errorf("get oauth code error: %v", err)
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}
	if raw == "" {
		return oauthExchangeFail(errcode.ValidationError, "授权码无效或已过期"), nil
	}
	if _, err = l.svcCtx.Redis.DelCtx(l.ctx, cKey); err != nil {
		l.Errorf("del oauth code error: %v", err)
	}
	var p oauthCodePayload
	if err = json.Unmarshal([]byte(raw), &p); err != nil {
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}
	// 3. 授权码必须与请求方及当初授权的 redirect_uri 一致
	if p.ClientId != in.ClientId || p.RedirectUri != in.RedirectUri {
		return oauthExchangeFail(errcode.ValidationError, "授权码与请求参数不匹配"), nil
	}

	// 4. 重新查用户并签发 JWT（会话有效期 1 天，与密码登录一致）
	user, err := l.svcCtx.UserModel.FindOneById(l.ctx, p.UserId)
	if err == sqlx.ErrNotFound {
		return oauthExchangeFail(errcode.UserNotFound, "用户不存在"), nil
	}
	if err != nil {
		l.Errorf("find user error: %v", err)
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}
	isSuper, err := l.svcCtx.TenantModel.SuperUserExists(l.ctx, user.Id)
	if err != nil {
		l.Errorf("check super user error: %v", err)
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}
	var usernamePtr *string
	if user.Username.Valid {
		usernamePtr = &user.Username.String
	}
	tenantId := user.TenantId
	claims := &token.Claims{
		UserID:      user.Id,
		Email:       user.Email,
		Username:    usernamePtr,
		TenantID:    &tenantId,
		Role:        token.MerchantRole,
		IsSuperUser: isSuper,
	}
	accessToken, err := token.Sign(l.svcCtx.Config.JwtSecret, claims)
	if err != nil {
		l.Errorf("sign token error: %v", err)
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}
	if err = saveSession(l.ctx, l.svcCtx.Redis, sessionKey(accessToken), claims); err != nil {
		l.Errorf("save session error: %v", err)
		return oauthExchangeFail(errcode.SystemError, "系统错误"), nil
	}

	// 免密登录事件（MQ 异常不影响发放结果）
	if err = l.svcCtx.Publisher.Publish(l.ctx, event.RoutingUserLogin, event.Event{
		UserID:   user.Id,
		Email:    user.Email,
		TenantID: &tenantId,
	}); err != nil {
		l.Errorf("publish oauth login event error: %v", err)
	}

	return &auth.OAuthExchangeResp{
		Code:        errcode.Success,
		Message:     "ok",
		AccessToken: accessToken,
		ExpiresIn:   int64(token.TTL.Seconds()),
		User:        claimsToProto(claims),
	}, nil
}

func oauthExchangeFail(code int32, msg string) *auth.OAuthExchangeResp {
	return &auth.OAuthExchangeResp{Code: code, Message: msg}
}

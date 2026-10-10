package logic

import (
	"context"
	"strings"

	"sso_login/common/crypto"
	"sso_login/common/errcode"
	"sso_login/common/token"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/event"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Login 普通用户登录，流程与 repo_backend userService.loginUser 完全一致：
// nonce 防重放 → 图形验证码(一次性) → 用户存在性 → 账号锁定 →
// 密码比对(失败计数/锁定) → 签发 JWT → token 存 Redis → 发布登录事件
func (l *LoginLogic) Login(in *auth.LoginReq) (*auth.LoginResp, error) {
	// 1. nonce 校验：5 分钟有效 + Redis 原子防重放（SET NX EX）
	nonceTs, err := crypto.ParseNonce(in.Nonce)
	if err != nil || crypto.IsNonceExpired(nonceTs) {
		return &auth.LoginResp{Code: errcode.NonceInvalid, Message: "nonce无效"}, nil
	}
	firstUse, err := l.svcCtx.Redis.SetnxExCtx(l.ctx, nonceKey(in.Nonce), "1", ttlNonce)
	if err != nil {
		l.Errorf("check nonce redis error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if !firstUse {
		return &auth.LoginResp{Code: errcode.NonceInvalid, Message: "nonce无效"}, nil
	}

	// 2. 图形验证码：比对后立即删除，防止重复使用（大小写不敏感）
	cKey := captchaKey(in.CaptchaId)
	stored, err := l.svcCtx.Redis.GetCtx(l.ctx, cKey)
	if err != nil {
		l.Errorf("get captcha redis error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if stored == "" {
		return &auth.LoginResp{Code: errcode.CaptchaExpired, Message: "验证码已过期"}, nil
	}
	if _, err = l.svcCtx.Redis.DelCtx(l.ctx, cKey); err != nil {
		l.Errorf("del captcha redis error: %v", err)
	}
	if !strings.EqualFold(stored, in.CaptchaText) {
		return &auth.LoginResp{Code: errcode.CaptchaIncorrect, Message: "验证码不正确"}, nil
	}

	// 3. 用户存在性
	user, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, in.Email)
	if err == sqlx.ErrNotFound {
		return &auth.LoginResp{Code: errcode.UserNotFound, Message: "用户不存在"}, nil
	}
	if err != nil {
		l.Errorf("find user error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	// 4. 账号是否冻结
	lockVal, err := l.svcCtx.Redis.GetCtx(l.ctx, lockedKey(in.Email))
	if err != nil {
		l.Errorf("get lock redis error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if lockVal != "" {
		return &auth.LoginResp{Code: errcode.AccountLocked, Message: "账号已锁定"}, nil
	}

	// 5. 密码比对：客户端提交 sha256(库中password + "_" + nonce)
	fKey := failedKey(in.Email)
	if crypto.Sha256(user.Password+"_"+in.Nonce) != in.Password {
		// 1 小时窗口内累计失败 6 次锁定账号
		oldCount, _ := l.svcCtx.Redis.GetCtx(l.ctx, fKey)
		if oldCount != "" {
			if _, err = l.svcCtx.Redis.IncrCtx(l.ctx, fKey); err != nil {
				l.Errorf("incr failed count error: %v", err)
			}
			if err = l.svcCtx.Redis.ExpireCtx(l.ctx, fKey, ttlFreeze); err != nil {
				l.Errorf("expire failed count error: %v", err)
			}
			// 旧计数为 5 时，本次是第 6 次失败 → 锁定
			if oldCount == "5" {
				if err = l.svcCtx.Redis.SetexCtx(l.ctx, lockedKey(in.Email), "1", ttlFreeze); err != nil {
					l.Errorf("lock account error: %v", err)
				}
			}
		} else {
			if err = l.svcCtx.Redis.SetexCtx(l.ctx, fKey, "1", ttlFreeze); err != nil {
				l.Errorf("set failed count error: %v", err)
			}
		}
		return &auth.LoginResp{Code: errcode.PasswordIncorrect, Message: "密码不正确"}, nil
	}
	// 密码正确，清空失败计数
	if _, err = l.svcCtx.Redis.DelCtx(l.ctx, fKey); err != nil {
		l.Errorf("clear failed count error: %v", err)
	}

	// 6. 签发 JWT（HS256，有效期 1 天；查 tenants.super_user_id 判断租户超管）
	isSuper, err := l.svcCtx.TenantModel.SuperUserExists(l.ctx, user.Id)
	if err != nil {
		l.Errorf("check super user error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	var usernamePtr *string
	if user.Username.Valid {
		usernamePtr = &user.Username.String
	}
	tenantId := user.TenantId
	claims := &token.Claims{
		UserID:      user.Id,
		Email:       user.Email,
		Username:    usernamePtr, // 为空时保持与 repo_backend 一致的 null
		TenantID:    &tenantId,
		Role:        token.MerchantRole,
		IsSuperUser: isSuper,
	}
	jwtToken, err := token.Sign(l.svcCtx.Config.JwtSecret, claims)
	if err != nil {
		l.Errorf("sign token error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	// 7. token 存 Redis（登出删除即立即失效）
	if err = saveSession(l.ctx, l.svcCtx.Redis, sessionKey(jwtToken), claims); err != nil {
		l.Errorf("save session error: %v", err)
		return &auth.LoginResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	// 登录成功事件（MQ 异常不影响登录结果）
	if err = l.svcCtx.Publisher.Publish(l.ctx, event.RoutingUserLogin, event.Event{
		UserID:   user.Id,
		Email:    user.Email,
		TenantID: &tenantId,
	}); err != nil {
		l.Errorf("publish login event error: %v", err)
	}

	return &auth.LoginResp{
		Code:    errcode.Success,
		Message: "用户登录成功",
		Token:   jwtToken,
		User:    claimsToProto(claims),
	}, nil
}

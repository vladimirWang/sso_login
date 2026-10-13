package logic

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/mail"
	"strings"

	"sso_login/common/email"
	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type SendEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SendEmailCode 发送注册邮箱验证码：
// 邮箱格式校验 → 已注册拦截 → 60 秒频控 → 生成 6 位数字存 Redis（5 分钟）→ QQ SMTP 发送。
// 频控 key 仅在发送成功后写入，发送失败允许立即重试。
func (l *SendEmailCodeLogic) SendEmailCode(in *auth.EmailCodeReq) (*auth.EmailCodeResp, error) {
	addr := strings.TrimSpace(strings.ToLower(in.Email))
	if _, err := mail.ParseAddress(addr); err != nil {
		return &auth.EmailCodeResp{Code: errcode.ValidationError, Message: "邮箱格式不正确"}, nil
	}

	// 已注册邮箱不再发送注册验证码
	if existed, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, addr); err == nil && existed != nil {
		return &auth.EmailCodeResp{Code: errcode.EmailExisted, Message: "该邮箱已注册，请直接登录"}, nil
	} else if err != nil && err != sqlx.ErrNotFound {
		l.Errorf("check email error: %v", err)
		return &auth.EmailCodeResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	// 60 秒发送频控
	if cd, err := l.svcCtx.Redis.GetCtx(l.ctx, emailCodeCdKey(addr)); err != nil {
		l.Errorf("check email code cooldown error: %v", err)
		return &auth.EmailCodeResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	} else if cd != "" {
		return &auth.EmailCodeResp{Code: errcode.ValidationError, Message: "发送过于频繁，请 60 秒后再试"}, nil
	}

	// 6 位数字验证码
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		code[i] = byte('0' + n.Int64())
	}
	verifyCode := string(code)

	if err := l.svcCtx.Redis.SetexCtx(l.ctx, emailCodeKey(addr), verifyCode, ttlEmailCode); err != nil {
		l.Errorf("store email code error: %v", err)
		return &auth.EmailCodeResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	cfg := l.svcCtx.Config.Email
	sender := &email.Sender{
		Host: cfg.Host, Port: cfg.Port, Account: cfg.Account,
		Password: cfg.Password, FromName: cfg.FromName,
	}
	body := fmt.Sprintf(`<div style="font-family:-apple-system,sans-serif;max-width:480px;margin:0 auto;padding:32px">
  <h2 style="color:#1677ff;margin-bottom:16px">SSO 登录中心 - 注册验证码</h2>
  <p style="color:#333;font-size:14px;line-height:1.8">你正在注册 SSO 统一登录账号，验证码为：</p>
  <div style="font-size:32px;font-weight:700;letter-spacing:8px;color:#1677ff;margin:20px 0">%s</div>
  <p style="color:#999;font-size:12px;line-height:1.8">验证码 5 分钟内有效，请勿泄露给他人。<br>如非本人操作，请忽略此邮件。</p>
</div>`, verifyCode)
	if err := sender.SendHTML(addr, "SSO 登录中心 - 注册验证码", body); err != nil {
		l.Errorf("send email code error: %v", err)
		// 发送失败：删除验证码，不写频控，允许用户立即重试
		_, _ = l.svcCtx.Redis.DelCtx(l.ctx, emailCodeKey(addr))
		return &auth.EmailCodeResp{Code: errcode.SystemError, Message: "验证码邮件发送失败，请稍后重试"}, nil
	}

	// 发送成功后写 60 秒频控
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, emailCodeCdKey(addr), "1", ttlEmailCodeCd); err != nil {
		l.Errorf("store email code cooldown error: %v", err)
	}
	return &auth.EmailCodeResp{
		Code:    errcode.Success,
		Message: "验证码已发送，请查收邮件",
		Ttl:     ttlEmailCode,
	}, nil
}

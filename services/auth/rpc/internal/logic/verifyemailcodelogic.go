package logic

import (
	"context"
	"net/mail"
	"strings"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyEmailCodeLogic {
	return &VerifyEmailCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// VerifyEmailCode 仅校验验证码是否正确，不做删除（注册成功时一次性消费）。
// 供前端在注册按钮点击前实时校验，"校验通过"后才允许提交注册。
func (l *VerifyEmailCodeLogic) VerifyEmailCode(in *auth.EmailCodeReq) (*auth.EmailCodeResp, error) {
	addr := strings.TrimSpace(strings.ToLower(in.Email))
	code := strings.TrimSpace(in.Code)
	if _, err := mail.ParseAddress(addr); err != nil {
		return &auth.EmailCodeResp{Code: errcode.ValidationError, Message: "邮箱格式不正确"}, nil
	}
	if len(code) != 6 {
		return &auth.EmailCodeResp{Code: errcode.EmailVerifyFail, Message: "请输入 6 位验证码"}, nil
	}

	stored, err := l.svcCtx.Redis.GetCtx(l.ctx, emailCodeKey(addr))
	if err != nil {
		l.Errorf("get email code error: %v", err)
		return &auth.EmailCodeResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if stored == "" {
		return &auth.EmailCodeResp{Code: errcode.EmailVerifyFail, Message: "验证码已过期，请重新获取"}, nil
	}
	if stored != code {
		return &auth.EmailCodeResp{Code: errcode.EmailVerifyFail, Message: "验证码不正确"}, nil
	}
	return &auth.EmailCodeResp{Code: errcode.Success, Message: "校验通过"}, nil
}

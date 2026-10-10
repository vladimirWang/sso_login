package logic

import (
	"context"
	"strings"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/mojocn/base64Captcha"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaptchaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaptchaLogic {
	return &GetCaptchaLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 验证码字符集：去除 0/o/1/i 等易混淆字符（对齐 repo_backend ignoreChars: "0o1i"）
const captchaAlphabet = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

// GetCaptcha 生成图形验证码（对齐 repo_backend generateCaptcha）：
// 4 位字符、噪声干扰，答案小写存 Redis 5 分钟（captcha:login:{captchaId}），登录时一次性消费。
// 返回 PNG base64 dataURL，前端可直接用于 <img src>。
func (l *GetCaptchaLogic) GetCaptcha(in *auth.CaptchaReq) (*auth.CaptchaResp, error) {
	driver := base64Captcha.NewDriverString(
		40, 100, // 高、宽
		3, // 干扰点数量 noise
		2, // 干扰线选项
		4, // 验证码长度
		captchaAlphabet,
		nil, nil,
	)
	store := base64Captcha.DefaultMemStore
	c := base64Captcha.NewCaptcha(driver, store)
	captchaId, image, err := c.Generate()
	if err != nil {
		l.Errorf("generate captcha error: %v", err)
		return &auth.CaptchaResp{Code: errcode.SystemError, Message: "验证码生成失败"}, nil
	}
	// 答案从 store 读取（不清除，Redis 侧负责一次性消费）
	answer := store.Get(captchaId, false)

	if err = l.svcCtx.Redis.SetexCtx(l.ctx, captchaKey(captchaId),
		strings.ToLower(answer), ttlCaptcha); err != nil {
		l.Errorf("store captcha error: %v", err)
		return &auth.CaptchaResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	return &auth.CaptchaResp{
		Code:      errcode.Success,
		Message:   "验证码生成成功",
		Image:     image,
		CaptchaId: captchaId,
	}, nil
}

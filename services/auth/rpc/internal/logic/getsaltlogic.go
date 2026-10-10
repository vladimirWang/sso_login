package logic

import (
	"context"
	"strings"

	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type GetSaltLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSaltLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSaltLogic {
	return &GetSaltLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetSalt 按邮箱取 salt（登录第一步，客户端用它推导密码哈希）。
// 对齐 repo_backend：邮箱未注册返回校验失败，不暴露"用户不存在"。
func (l *GetSaltLogic) GetSalt(in *auth.EmailReq) (*auth.SaltResp, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	user, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, email)
	if err == sqlx.ErrNotFound {
		return &auth.SaltResp{Code: errcode.ValidationError, Message: "邮箱未注册"}, nil
	}
	if err != nil {
		l.Errorf("find user error: %v", err)
		return &auth.SaltResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	return &auth.SaltResp{Code: errcode.Success, Message: "获取salt成功", Salt: user.Salt}, nil
}

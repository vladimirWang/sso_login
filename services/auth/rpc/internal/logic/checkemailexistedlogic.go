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

type CheckEmailExistedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckEmailExistedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckEmailExistedLogic {
	return &CheckEmailExistedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CheckEmailExisted 邮箱是否已注册（对齐 repo_backend checkEmailExisted）
func (l *CheckEmailExistedLogic) CheckEmailExisted(in *auth.EmailReq) (*auth.EmailExistedResp, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	user, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, email)
	if err != nil && err != sqlx.ErrNotFound {
		l.Errorf("find user error: %v", err)
		return &auth.EmailExistedResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	existed := err == nil && user != nil
	msg := "邮箱不存在"
	if existed {
		msg = "邮箱已存在"
	}
	return &auth.EmailExistedResp{
		Code:    errcode.Success,
		Message: msg,
		Existed: existed,
	}, nil
}

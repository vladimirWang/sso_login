package logic

import (
	"context"

	"sso_login/common/crypto"
	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type UpdatePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdatePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePasswordLogic {
	return &UpdatePasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UpdatePassword 修改密码（对齐 repo_backend updateUserPassword，需登录）
func (l *UpdatePasswordLogic) UpdatePassword(in *auth.UpdatePasswordReq) (*auth.BaseResp, error) {
	if len(in.Password) < 6 {
		return &auth.BaseResp{Code: errcode.ValidationError, Message: "新密码长度不能少于6位"}, nil
	}

	claims, err := loadSession(l.ctx, l.svcCtx.Redis, sessionKey(in.Token))
	if err != nil {
		l.Errorf("load session error: %v", err)
		return &auth.BaseResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	if claims == nil {
		return &auth.BaseResp{Code: errcode.SessionInvalid, Message: "未登录或登录已过期"}, nil
	}

	user, err := l.svcCtx.UserModel.FindOneById(l.ctx, claims.UserID)
	if err == sqlx.ErrNotFound {
		return &auth.BaseResp{Code: errcode.UserNotFound, Message: "用户不存在"}, nil
	}
	if err != nil {
		l.Errorf("find user error: %v", err)
		return &auth.BaseResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	// 当前密码：sha256(库中password + "_" + nonce) 必须等于提交的 current
	if crypto.Sha256(user.Password+"_"+in.Nonce) != in.Current {
		return &auth.BaseResp{Code: errcode.PasswordIncorrect, Message: "密码不正确"}, nil
	}

	// 新密码不能与旧密码一致
	newHash := crypto.Sha256(in.Password + "_" + user.Salt)
	if newHash == user.Password {
		return &auth.BaseResp{Code: errcode.ValidationError, Message: "新密码不能与旧密码一致"}, nil
	}

	if err = l.svcCtx.UserModel.UpdatePassword(l.ctx, user.Id, newHash); err != nil {
		l.Errorf("update password error: %v", err)
		return &auth.BaseResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}
	return &auth.BaseResp{Code: errcode.Success, Message: "密码修改成功"}, nil
}

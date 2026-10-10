package logic

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/mail"
	"strconv"
	"strings"

	"sso_login/common/crypto"
	"sso_login/common/errcode"
	"sso_login/services/auth/rpc/auth"
	"sso_login/services/auth/rpc/internal/svc"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 租户编码字符集（6 位，去除易混淆字符）
const tenantCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomTenantCode() string {
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(tenantCodeAlphabet))))
		b[i] = tenantCodeAlphabet[n.Int64()]
	}
	return string(b)
}

// Register 底座简化注册：直接注册普通用户并自动创建其所属租户（注册人即超管）。
// 密码存储与 repo_backend 一致：password = sha256(明文 + "_" + salt)。
func (l *RegisterLogic) Register(in *auth.RegisterReq) (*auth.RegisterResp, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	username := strings.TrimSpace(in.Username)
	password := in.Password

	if _, err := mail.ParseAddress(email); err != nil {
		return &auth.RegisterResp{Code: errcode.ValidationError, Message: "邮箱格式不正确"}, nil
	}
	if len(password) < 6 {
		return &auth.RegisterResp{Code: errcode.ValidationError, Message: "密码长度不能少于6位"}, nil
	}
	if len([]rune(username)) > 8 {
		return &auth.RegisterResp{Code: errcode.ValidationError, Message: "用户名最长8位"}, nil
	}

	// 邮箱已存在
	if existed, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, email); err == nil && existed != nil {
		return &auth.RegisterResp{Code: errcode.EmailExisted, Message: "邮箱已存在"}, nil
	} else if err != nil && err != sqlx.ErrNotFound {
		l.Errorf("check email error: %v", err)
		return &auth.RegisterResp{Code: errcode.SystemError, Message: "系统错误"}, nil
	}

	salt := crypto.GenerateSalt()
	passwordHash := crypto.Sha256(password + "_" + salt)

	// 租户名：优先用户名，否则取邮箱前缀；唯一冲突时追加随机数字后缀（最多重试 5 次）
	nameBase := username
	if nameBase == "" {
		nameBase = strings.Split(email, "@")[0]
	}
	if r := []rune(nameBase); len(r) > 20 {
		nameBase = string(r[:20])
	}

	var userId int64
	registered := false
	for attempt := 0; attempt < 5 && !registered; attempt++ {
		tenantName := nameBase
		if attempt > 0 {
			n, _ := rand.Int(rand.Reader, big.NewInt(9000))
			base := nameBase
			if r := []rune(base); len(r) > 16 {
				base = string(r[:16])
			}
			tenantName = base + strconv.Itoa(int(n.Int64())+1000)
		}
		code := randomTenantCode()

		err := l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
			tenantId, err := l.svcCtx.TenantModel.Insert(ctx, session, tenantName, code)
			if err != nil {
				return err
			}
			uid, err := l.svcCtx.UserModel.Insert(ctx, session, tenantId,
				email, nullString(username), passwordHash, salt)
			if err != nil {
				return err
			}
			if err = l.svcCtx.TenantModel.UpdateSuperUser(ctx, session, tenantId, uid); err != nil {
				return err
			}
			userId = uid
			return nil
		})
		if err == nil {
			registered = true
			break
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			// email 冲突直接返回；name/code 冲突换后缀重试
			if strings.Contains(pqErr.Constraint, "email") {
				return &auth.RegisterResp{Code: errcode.EmailExisted, Message: "邮箱已存在"}, nil
			}
			continue
		}
		l.Errorf("register transaction error: %v", err)
		return &auth.RegisterResp{Code: errcode.SystemError, Message: "注册失败"}, nil
	}

	if !registered {
		return &auth.RegisterResp{Code: errcode.SystemError, Message: "注册失败，请重试"}, nil
	}
	return &auth.RegisterResp{Code: errcode.Success, Message: "用户注册成功", UserId: userId}, nil
}

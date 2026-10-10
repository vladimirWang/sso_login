package model

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type TenantModel struct {
	conn sqlx.SqlConn
}

func NewTenantModel(conn sqlx.SqlConn) *TenantModel {
	return &TenantModel{conn: conn}
}

// Insert 创建租户（status 默认 ACTIVE），返回新 ID
func (m *TenantModel) Insert(ctx context.Context, session sqlx.Session, name, code string) (int64, error) {
	query := `INSERT INTO tenants (name, code, status) VALUES ($1, $2, 'ACTIVE') RETURNING id`
	var id int64
	if err := session.QueryRowCtx(ctx, &id, query, name, code); err != nil {
		return 0, err
	}
	return id, nil
}

// UpdateSuperUser 绑定租户超级管理员
func (m *TenantModel) UpdateSuperUser(ctx context.Context, session sqlx.Session, tenantId, superUserId int64) error {
	_, err := session.ExecCtx(ctx,
		`UPDATE tenants SET super_user_id = $1 WHERE id = $2`, superUserId, tenantId)
	return err
}

// SuperUserExists 判断某用户是否为某租户的超级管理员（对齐 repo_backend 登录时查 Tenant.superUserId）
func (m *TenantModel) SuperUserExists(ctx context.Context, userId int64) (bool, error) {
	var exists bool
	err := m.conn.QueryRowCtx(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM tenants WHERE super_user_id = $1)`, userId)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}

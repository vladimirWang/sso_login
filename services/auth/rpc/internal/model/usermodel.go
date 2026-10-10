// Package model users / tenants 表数据访问（PostgreSQL + go-zero sqlx）
package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// User 对应 users 表（对齐 repo_backend Prisma User 的登录相关字段）
type User struct {
	Id        int64          `db:"id"`
	TenantId  int64          `db:"tenant_id"`
	Email     string         `db:"email"`
	Username  sql.NullString `db:"username"`
	Password  string         `db:"password"`
	Salt      string         `db:"salt"`
	Avatar    sql.NullString `db:"avatar"`
	GithubId  sql.NullString `db:"github_id"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
	DeletedAt sql.NullTime   `db:"deleted_at"`
}

type UserModel struct {
	conn sqlx.SqlConn
}

func NewUserModel(conn sqlx.SqlConn) *UserModel {
	return &UserModel{conn: conn}
}

// Insert 创建用户，返回新 ID（PostgreSQL 使用 RETURNING 而非 LastInsertId）。
// 接受 sqlx.Session 以支持外层事务。
func (m *UserModel) Insert(ctx context.Context, session sqlx.Session, tenantId int64,
	email string, username sql.NullString, password, salt string,
) (int64, error) {
	query := `
		INSERT INTO users (tenant_id, email, username, password, salt)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id int64
	if err := session.QueryRowCtx(ctx, &id, query,
		tenantId, email, username, password, salt); err != nil {
		return 0, err
	}
	return id, nil
}

func (m *UserModel) FindOneByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, tenant_id, email, username, password, salt, avatar, github_id,
		created_at, updated_at, deleted_at
		FROM users WHERE email = $1 AND deleted_at IS NULL LIMIT 1`
	var u User
	if err := m.conn.QueryRowCtx(ctx, &u, query, email); err != nil {
		return nil, err
	}
	return &u, nil
}

func (m *UserModel) FindOneById(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, tenant_id, email, username, password, salt, avatar, github_id,
		created_at, updated_at, deleted_at
		FROM users WHERE id = $1 AND deleted_at IS NULL LIMIT 1`
	var u User
	if err := m.conn.QueryRowCtx(ctx, &u, query, id); err != nil {
		return nil, err
	}
	return &u, nil
}

func (m *UserModel) UpdatePassword(ctx context.Context, id int64, password string) error {
	_, err := m.conn.ExecCtx(ctx,
		`UPDATE users SET password = $1 WHERE id = $2`, password, id)
	return err
}

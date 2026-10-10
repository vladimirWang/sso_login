package model

import (
	"context"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// OAuthClient 对应 oauth_clients 表（接入 SSO 的第三方业务系统）
type OAuthClient struct {
	Id           int64     `db:"id"`
	ClientId     string    `db:"client_id"`
	ClientSecret string    `db:"client_secret"` // sha256(明文secret) 十六进制
	Name         string    `db:"name"`
	RedirectUris string    `db:"redirect_uris"` // 逗号分隔
	Status       string    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type OAuthClientModel struct {
	conn sqlx.SqlConn
}

func NewOAuthClientModel(conn sqlx.SqlConn) *OAuthClientModel {
	return &OAuthClientModel{conn: conn}
}

func (m *OAuthClientModel) FindOneByClientId(ctx context.Context, clientId string) (*OAuthClient, error) {
	var c OAuthClient
	query := `SELECT id, client_id, client_secret, name, redirect_uris, status, created_at, updated_at
		FROM oauth_clients WHERE client_id = $1 LIMIT 1`
	if err := m.conn.QueryRowCtx(ctx, &c, query, clientId); err != nil {
		return nil, err
	}
	return &c, nil
}

// ValidateRedirectUri redirect_uris 逗号分隔存储，授权时精确匹配
func (c *OAuthClient) ValidateRedirectUri(uri string) bool {
	for _, u := range strings.Split(c.RedirectUris, ",") {
		if strings.TrimSpace(u) == uri {
			return true
		}
	}
	return false
}

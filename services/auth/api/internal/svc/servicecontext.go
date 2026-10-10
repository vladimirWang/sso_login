package svc

import (
	"sso_login/services/auth/api/internal/config"
	"sso_login/services/auth/api/internal/middleware"
	"sso_login/services/auth/rpc/authclient"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	// AuthRpc gRPC 客户端
	AuthRpc authclient.Auth
	// Auth 登录态中间件
	Auth rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	authClient := authclient.NewAuth(zrpc.MustNewClient(c.AuthRpc))
	return &ServiceContext{
		Config:  c,
		AuthRpc: authClient,
		Auth:    middleware.NewAuthMiddleware(authClient).Handle,
	}
}

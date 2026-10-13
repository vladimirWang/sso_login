// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"flag"
	"fmt"

	"sso_login/services/auth/api/internal/config"
	"sso_login/services/auth/api/internal/handler"
	"sso_login/services/auth/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

// 前端页面（/oauth/login、/demo-app）已改为 Vite+React SPA，由 nginx 托管并反代本服务，
// 不再 go:embed 进二进制。

var configFile = flag.String("f", "etc/auth-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	// UseEnv：yaml 中的 ${ETCD_HOST} 等占位符从环境变量展开
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)
	// 仅注册 OAuth2 协议端点（authorize/token/userinfo/clientinfo/logout）；
	// /oauth/login、/demo-app 等页面路由由前端 SPA 承接（nginx try_files）
	handler.RegisterOAuthRoutes(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

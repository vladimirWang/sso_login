// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"embed"
	"flag"
	"fmt"

	"sso_login/services/auth/api/internal/config"
	"sso_login/services/auth/api/internal/handler"
	"sso_login/services/auth/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

//go:embed web/login.html web/demo-app.html
var webFS embed.FS

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
	// OAuth2 协议端点 + SSO 登录页 / 演示业务系统页（静态资源内嵌二进制）
	loginPage, _ := webFS.ReadFile("web/login.html")
	demoPage, _ := webFS.ReadFile("web/demo-app.html")
	handler.RegisterOAuthRoutes(server, ctx, loginPage, demoPage)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	// 通过 etcd 服务发现调用 auth.rpc
	AuthRpc zrpc.RpcClientConf
}

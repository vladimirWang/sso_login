package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	// PostgreSQL 连接串（lib/pq 格式），如：
	// host=127.0.0.1 port=15432 user=postgres password=postgres dbname=sso_login sslmode=disable
	DataSource string
	// 登录态 / 验证码 / nonce / 失败计数
	// 注意：不能叫 Redis，zrpc.RpcServerConf 已内置 Redis redis.RedisKeyConf，会冲突
	RedisStore redis.RedisConf
	// JWT HS256 共享密钥（与 repo_backend JWT_SECRET 语义一致）
	JwtSecret string
	// RabbitMQ 登录事件发布
	RabbitMQ RabbitMQConf
}

type RabbitMQConf struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string `json:",optional"`
	Exchange string
}

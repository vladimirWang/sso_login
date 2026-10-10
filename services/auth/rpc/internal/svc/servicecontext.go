package svc

import (
	_ "github.com/lib/pq" // 注册 postgres 驱动

	"sso_login/services/auth/rpc/internal/config"
	"sso_login/services/auth/rpc/internal/event"
	"sso_login/services/auth/rpc/internal/model"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config           config.Config
	DB               sqlx.SqlConn
	Redis            *redis.Redis
	UserModel        *model.UserModel
	TenantModel      *model.TenantModel
	OAuthClientModel *model.OAuthClientModel
	Publisher        *event.Publisher // 连不上 MQ 时为 nil，登录主链路不受影响
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := sqlx.NewSqlConn("postgres", c.DataSource)
	rdb := redis.MustNewRedis(c.RedisStore)

	publisher, err := event.NewPublisher(
		c.RabbitMQ.Host, c.RabbitMQ.Port,
		c.RabbitMQ.User, c.RabbitMQ.Password,
		c.RabbitMQ.VHost, c.RabbitMQ.Exchange,
	)
	if err != nil {
		logx.Errorf("rabbitmq publisher unavailable, auth events will be dropped: %v", err)
	}

	return &ServiceContext{
		Config:           c,
		DB:               db,
		Redis:            rdb,
		UserModel:        model.NewUserModel(db),
		TenantModel:      model.NewTenantModel(db),
		OAuthClientModel: model.NewOAuthClientModel(db),
		Publisher:        publisher,
	}
}

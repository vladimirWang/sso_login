// Package event 认证领域事件发布（RabbitMQ topic exchange）
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// RoutingUserLogin 登录成功事件
	RoutingUserLogin = "user.login"
	// RoutingUserLogout 登出事件
	RoutingUserLogout = "user.logout"
)

// Event 认证事件消息体
type Event struct {
	Event    string `json:"event"`
	UserID   int64  `json:"userId"`
	Email    string `json:"email"`
	TenantID *int64 `json:"tenantId,omitempty"`
	Ts       int64  `json:"ts"`
}

type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

// NewPublisher 建立连接并声明 topic exchange（durable）
func NewPublisher(host string, port int, user, password, vhost, exchange string) (*Publisher, error) {
	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		url.QueryEscape(user), url.QueryEscape(password), host, port, vhost)
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err = ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &Publisher{conn: conn, channel: ch, exchange: exchange}, nil
}

// Publish 发布事件；消息持久化
func (p *Publisher) Publish(ctx context.Context, routingKey string, e Event) error {
	if p == nil {
		return nil
	}
	e.Event = routingKey
	if e.Ts == 0 {
		e.Ts = time.Now().Unix()
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, p.exchange, routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})
}

func (p *Publisher) Close() {
	if p == nil {
		return
	}
	_ = p.channel.Close()
	_ = p.conn.Close()
}

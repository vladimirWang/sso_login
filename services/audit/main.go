// audit 服务：订阅 auth.events topic 交换机，消费 user.login / user.logout 事件。
// 底座中仅打印审计日志；接入真实审计库时替换 handleEvent 内的落库逻辑即可。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"

	amqp "github.com/rabbitmq/amqp091-go"
)

var configFile = flag.String("f", "etc/audit.yaml", "the config file")

func main() {
	flag.Parse()

	var c Config
	conf.MustLoad(*configFile, &c)

	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		url.QueryEscape(c.RabbitMQ.User), url.QueryEscape(c.RabbitMQ.Password),
		c.RabbitMQ.Host, c.RabbitMQ.Port, c.RabbitMQ.VHost)

	conn, err := amqp.Dial(amqpURL)
	logx.Must(err)
	defer conn.Close()

	ch, err := conn.Channel()
	logx.Must(err)
	defer ch.Close()
	logx.Must(ch.Qos(16, 0, false))

	// 声明交换机 + 持久化队列并绑定（订阅全部认证事件）
	logx.Must(ch.ExchangeDeclare(c.RabbitMQ.Exchange, "topic", true, false, false, false, nil))
	_, err = ch.QueueDeclare(c.RabbitMQ.Queue, true, false, false, false, nil)
	logx.Must(err)
	logx.Must(ch.QueueBind(c.RabbitMQ.Queue, "#", c.RabbitMQ.Exchange, false, nil))

	msgs, err := ch.Consume(c.RabbitMQ.Queue, c.Name, false, false, false, false, nil)
	logx.Must(err)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logx.Infof("audit service consuming queue=%s exchange=%s", c.RabbitMQ.Queue, c.RabbitMQ.Exchange)

	go func() {
		for d := range msgs {
			handleEvent(d.RoutingKey, d.Body)
			if err := d.Ack(false); err != nil {
				logx.Errorf("ack error: %v", err)
			}
		}
	}()

	<-ctx.Done()
	logx.Info("audit service shutting down")
	time.Sleep(200 * time.Millisecond)
	_ = os.Stdout.Sync()
}

func handleEvent(routingKey string, body []byte) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	logx.Infof("[AUDIT] %s event=%s payload=%s", ts, routingKey, string(body))
}

package main

// Config audit 消费者配置
type Config struct {
	Name     string
	RabbitMQ RabbitMQConf
}

type RabbitMQConf struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string `json:",optional"`
	Exchange string
	Queue    string
}

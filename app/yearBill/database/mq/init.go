package mq

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"platform/config"
	"time"
)

var MqChan *amqp.Channel

func InitMQ() {
	mqConfig := config.Conf.Mq["year_bill"]
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/",
		mqConfig.Username, mqConfig.Password, mqConfig.Address, mqConfig.Port)
	conn, err := amqp.DialConfig(url, amqp.Config{
		Heartbeat: 10 * time.Second,
	})
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
		panic(err)
	}
	ch, err := conn.Channel()
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
		return
	}
	MqChan = ch
}

func NewMQConn() *amqp.Channel {
	return MqChan
}

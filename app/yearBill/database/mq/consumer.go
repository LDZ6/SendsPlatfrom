package mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func Consumer(ch *amqp.Channel, queueName string) (msgs <-chan amqp.Delivery, err error) {
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
		return
	}
	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
		return
	}
	msgs, err = ch.Consume(
		q.Name, // 队列名
		"",     // 消费者标签，可为空
		false,  // 是否自动确认消息
		false,  // 是否排他，如果为true，表示只有这个消费者可以访问队列
		false,  // 是否不接收同一个连接发送的消息
		false,  // 是否阻塞，如果为true，表示服务器将在响应准备好时发送响应
		nil,    // 额外的属性
	)
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
	}
	// 返回消息通道和错误
	return msgs, err
}

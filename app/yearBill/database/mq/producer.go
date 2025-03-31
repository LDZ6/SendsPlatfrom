package mq

import (
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"platform/app/yearBill/database/model"
)

func Producer(ch *amqp.Channel, queueName string, exchange string, task *model.DadaInitTask) (err error) {
	//defer ch.Close()
	q, err := ch.QueueDeclare(
		queueName, // 队列名称
		true,      // 是否持久化
		false,     // 是否自动删除
		false,     // 是否排他
		false,     // 是否等待
		nil,       // 其他参数
	)
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
		return
	}
	body, err := json.Marshal(task)
	if err != nil {
		logrus.Info("[MarshalERROR]:%v", err.Error())
		return
	}
	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         body,
			Expiration:   "300000",
		},
	)
	if err != nil {
		logrus.Info("[MQERROR]:%v", err.Error())
		return
	}
	return
}

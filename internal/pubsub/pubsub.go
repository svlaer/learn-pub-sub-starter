package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

var queueTypeName = map[SimpleQueueType]string{
	Durable:   "durable",
	Transient: "transient",
}

func (sqt SimpleQueueType) String() string {
	return queueTypeName[sqt]
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	valJson, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Error marshalling to JSON: %v", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{ContentType: "application/json", Body: valJson})
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	rabbitChan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to create RabbitMQ channel: %v", err)
	}

	var durable, autoDelete, exclusive bool
	switch queueType {
	case Durable:
		durable = true
		autoDelete = false
		exclusive = false
	case Transient:
		durable = false
		autoDelete = true
		exclusive = true
	default:
		return nil, amqp.Queue{}, fmt.Errorf("Unknown SimpleQueueType: %s", queueType)
	}

	rabbitQueue, err := rabbitChan.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to declare queue: %v", err)
	}

	if err = rabbitChan.QueueBind(rabbitQueue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to bind exchange to queue: %v", err)
	}

	return rabbitChan, rabbitQueue, nil
}

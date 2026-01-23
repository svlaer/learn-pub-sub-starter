package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
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

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(val); err != nil {
		return fmt.Errorf("Error encoding to Gob: %v", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{ContentType: "application/gob", Body: buf.Bytes()})
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	valJson, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Error marshalling to JSON: %v", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{ContentType: "application/json", Body: valJson})
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	unmarshaller := func(data []byte) (T, error) {
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		var target T
		err := dec.Decode(&target)
		return target, err
	}

	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	return subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	rabbitChan, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("Failed to declare and bind queue: %v", err)
	}

	deliveryChan, err := rabbitChan.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Error consuming queue: %v", err)
	}

	go func() {
		defer rabbitChan.Close()
		for delivery := range deliveryChan {
			target, err := unmarshaller(delivery.Body)
			if err != nil {
				fmt.Printf("Could not unmarshall message: %v\n", err)
				continue
			}

			ack := handler(target)
			switch ack {
			case Ack:
				delivery.Ack(false)
			case NackRequeue:
				delivery.Nack(false, true)
			case NackDiscard:
				delivery.Nack(false, false)
			default:
				fmt.Println("Unkown ackType!")
			}
		}
	}()

	return nil
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

	table := amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	}

	rabbitQueue, err := rabbitChan.QueueDeclare(queueName, durable, autoDelete, exclusive, false, table)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to declare queue: %v", err)
	}

	if err = rabbitChan.QueueBind(rabbitQueue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("Failed to bind exchange to queue: %v", err)
	}

	return rabbitChan, rabbitQueue, nil
}

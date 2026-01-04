package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	valJson, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Error marshalling to JSON: %v", err)
	}

	return ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{ContentType: "application/json", Body: valJson})
}

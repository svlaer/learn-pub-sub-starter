package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString string = "amqp://guest:guest@localhost:5672"

	fmt.Println("Starting Peril client...")

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("RabbitMQ failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("Peril client connected to RabbitMQ.")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, rabbitQueue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		routing.PauseKey,
		pubsub.Transient,
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to pause: %v", err)
	}

	fmt.Printf("Queue %v declared and bound!\n", rabbitQueue.Name)

	// Wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("RabbitMQ connection closed.")
}

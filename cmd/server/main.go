package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString string = "amqp://guest:guest@localhost:5672"

	fmt.Println("Starting Peril server...")

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("RabbitMQ failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("Peril game server connected to RabbitMQ.")

	rabbitChan, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}

	if err = pubsub.PublishJSON(rabbitChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true}); err != nil {
		log.Printf("Error while publishing JSON to RabbitMQ: %v", err)
	}

	// Wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("RabbitMQ connection closed.")
}

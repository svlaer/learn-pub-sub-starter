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

	_, topicQueue, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		fmt.Sprintf("%s.*", routing.GameLogSlug),
		pubsub.Durable,
	)
	if err != nil {
		log.Fatalf("Failed to declare and bind game_logs queue: %v", err)
	}

	fmt.Printf("Queue %v declared and bound!\n", topicQueue)

	gamelogic.PrintServerHelp()

	for {
		inputWords := gamelogic.GetInput()
		if len(inputWords) == 0 {
			continue
		}

		switch inputWords[0] {
		case "pause":
			log.Println("Sending pause message...")
			if err = pubsub.PublishJSON(rabbitChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true}); err != nil {
				log.Printf("Error while publishing JSON to RabbitMQ: %v", err)
			}
		case "resume":
			log.Println("Sending resume message...")
			if err = pubsub.PublishJSON(rabbitChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false}); err != nil {
				log.Printf("Error while publishing JSON to RabbitMQ: %v", err)
			}
		case "quit":
			log.Println("Exiting...")
			return
		default:
			log.Printf("Unrecognised command: %s\n", inputWords[0])
		}
	}

	// Wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("RabbitMQ connection closed.")
}

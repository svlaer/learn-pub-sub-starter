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

	gs := gamelogic.NewGameState(username)

	for {
		inputWords := gamelogic.GetInput()
		if len(inputWords) == 0 {
			continue
		}

		switch inputWords[0] {
		case "spawn":
			log.Println("Spawning...")
			if err = gs.CommandSpawn(inputWords); err != nil {
				log.Println(err)
				continue
			}
		case "move":
			fmt.Println("Moving...")
			if _, err = gs.CommandMove(inputWords); err != nil {
				log.Println(err)
				continue
			}
			log.Println("Move successful")
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			log.Printf("Unrecognised command: %s\n", inputWords[0])
			continue
		}
	}

	// Wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("RabbitMQ connection closed.")
}

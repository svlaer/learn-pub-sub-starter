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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

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

	gs := gamelogic.NewGameState(username)

	if err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		fmt.Sprintf("pause.%s", gs.GetUsername()),
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gs),
	); err != nil {
		log.Fatalf("Failed to subscribe to pause: %v", err)
	}

	fmt.Println("Subscribed to pause.")

	for {
		inputWords := gamelogic.GetInput()
		if len(inputWords) == 0 {
			continue
		}

		switch inputWords[0] {
		case "spawn":
			fmt.Println("Spawning...")
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
			fmt.Println("Move successful")
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
			fmt.Printf("Unrecognised command: %s\n", inputWords[0])
			continue
		}
	}

	// Wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("RabbitMQ connection closed.")
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
)

var ctx context.Context

func main() {
	// Handle signal interrupts.
	var cancel func()
	ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		panic("Please set a DISCORD_TOKEN environment variable to your bot token")
	}

	b, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
	defer b.Close()

	// Register handlers
	b.AddHandler(readyHandler)
	b.AddHandler(commandsHandler)

	// Begin listening for events
	err = b.Open()
	if err != nil {
		log.Panic("Could not connect to discord", err)
	}
	log.Print("Bot is now running. Check out Discord!")

	// Wait until the application is shutting down
	<-ctx.Done()
	guiders.Wait()
}

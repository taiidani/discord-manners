package main

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/taiidani/discord-manners/internal"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		panic("Please set a DISCORD_TOKEN environment variable to your bot token")
	}

	b, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
	defer b.Close()

	bot := internal.NewBot(b)
	if err := bot.Start(); err != nil {
		log.Fatal(err)
	}
}

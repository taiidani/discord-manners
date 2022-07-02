package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	ctx     context.Context
	guiders sync.WaitGroup
	session *discordgo.Session
}

func NewBot(s *discordgo.Session) *Bot {
	return &Bot{
		session: s,
	}
}

func (b *Bot) Start() error {
	// Handle signal interrupts.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Register handlers
	b.addHandlers()

	// Prep guiders
	b.guiders = sync.WaitGroup{}

	// Begin listening for events
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("could not connect to discord: %w", err)
	}
	log.Print("Bot is now running. Check out Discord!")

	// Wait until the application is shutting down
	<-ctx.Done()
	b.session.Close()
	b.guiders.Wait()
	return nil
}

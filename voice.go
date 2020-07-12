package main

import (
	"context"
	"log"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var guiders = sync.WaitGroup{}

type voiceGuider struct {
	session   *discordgo.Session
	message   *discordgo.Message
	voice     *discordgo.VoiceConnection
	reactions []string
}

func startGuider(ctx context.Context, g voiceGuider) {
	guiders.Add(1)
	defer guiders.Done()

	// Remove the channel message from both state and Discord at the end of the guidance session
	defer g.session.ChannelMessageDelete(g.message.ChannelID, g.message.ID)

	// Update the message to begin watching
	start := time.Now()
	_, err := g.session.ChannelMessageEdit(
		g.message.ChannelID,
		g.message.ID,
		"Listening for speakers. Watch the reactions on this message for guidance",
	)
	if err != nil {
		log.Println("Could not edit message: ", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Guider %s.%s shutting down", g.message.ChannelID, g.message.ID)
			return
		case <-g.voice.OpusRecv:
			// Someone spoke, reset the counter
			start = time.Now()
		default:
		}

		if err := g.updateMessage(time.Since(start)); err != nil {
			log.Println("Error modifying message: ", err)
		}
	}
}

func (v *voiceGuider) updateMessage(elapsed time.Duration) error {
	const waitTime = 3
	amount := (int)(math.Ceil(waitTime - elapsed.Seconds()))

	// Build the new emoji list
	emoji := []string{"🔈"}
	if amount >= 1 {
		switch amount {
		case 1:
			emoji = []string{"🙊"} // "1️⃣"}
		case 2:
			emoji = []string{"🙊"} // "2️⃣"}
		case 3:
			emoji = []string{"🙊"} // "3️⃣"}
		case 4:
			emoji = []string{"🙊"} // "4️⃣"}
		default:
			emoji = []string{"🙊"} // "5️⃣"}
		}
	}

	// If the state has drifted, update the status
	if !reflect.DeepEqual(v.reactions, emoji) {
		log.Printf("Updating guidance session %s.%s status to %v", v.message.ChannelID, v.message.ID, emoji)
		if err := v.session.MessageReactionsRemoveAll(v.message.ChannelID, v.message.ID); err != nil {
			return err
		}

		for _, reaction := range emoji {
			if err := v.session.MessageReactionAdd(v.message.ChannelID, v.message.ID, reaction); err != nil {
				return err
			}
		}
		v.reactions = emoji
	}

	return nil
}

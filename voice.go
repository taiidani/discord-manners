package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var guiders = sync.WaitGroup{}

type voiceGuider struct {
	cancel        func()                     // A cancellation function to trigger cleanup operations
	trigger       *discordgo.Message         // The original message that opened this session
	session       *discordgo.Session         // The current Discord connection
	message       *discordgo.Message         // The message that is providing guidance to the users
	voice         *discordgo.VoiceConnection // The voice byte stream for listening to the channel
	reactionState string                     // The current reaction set on the message
}

func startGuider(ctx context.Context, g *voiceGuider, channelName string) {
	guiders.Add(1)
	defer guiders.Done()

	// Set up a cancellable context to end this guider
	ctx, g.cancel = context.WithCancel(ctx)
	defer g.cancel()

	// Find the channel that they want us to join
	voiceChannel, err := findVoiceChannel(g.session, g.trigger.GuildID, channelName)
	if err != nil {
		g.session.ChannelMessageSend(g.trigger.ChannelID, err.Error())
		return
	} else if voiceChannel == nil {
		log.Println("Nil voice channel encountered for ", channelName)
		return
	}
	log.Println("Starting guidance session for ", voiceChannel.ID)

	// Join the voice channel
	g.voice, err = g.session.ChannelVoiceJoin(voiceChannel.GuildID, voiceChannel.ID, true, false)
	if err != nil {
		log.Println(err)
		return
	}
	defer g.voice.Disconnect()

	// Add a message to the channel for updating users
	g.message, err = g.session.ChannelMessageSend(
		g.trigger.ChannelID,
		"Listening for speakers. Watch the reactions on this message for guidance.\nYou may add the 🛑 reaction to cancel this session.",
	)
	if err != nil {
		log.Println(err)
		return
	}
	defer g.session.ChannelMessageEdit(g.message.ChannelID, g.message.ID, "Guidance session has ended. Use '!manners guide-voice <channel-name>' again to restart.")
	defer g.session.MessageReactionAdd(g.message.ChannelID, g.message.ID, "☠")

	// Handle reaction adds so that cancels can be triggered
	// NOTE: Possible memory issue here, as a new one is added every session
	g.session.AddHandler(g.messageUpdated)

	start := time.Now()
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

func (v *voiceGuider) messageUpdated(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.ChannelID != v.message.ChannelID || m.MessageID != v.message.ID {
		return
	}

	switch m.Emoji.Name {
	case "🛑", "⏹️":
		log.Printf("Cancelling %s.%s guider", v.message.ChannelID, v.message.ID)
		v.cancel()
	}
}

func (v *voiceGuider) updateMessage(elapsed time.Duration) error {
	const waitTime = 3
	amount := (int)(math.Ceil(waitTime - elapsed.Seconds()))

	// Build the new emoji
	emoji := "🔈"
	if amount >= 1 {
		emoji = "🙊"
	}

	// If the state has drifted, update the status
	if v.reactionState != emoji {
		log.Printf("Updating guidance session %s.%s status to %v", v.message.ChannelID, v.message.ID, emoji)
		if v.reactionState != "" {
			if err := v.session.MessageReactionRemove(v.message.ChannelID, v.message.ID, v.reactionState, "@me"); err != nil {
				return err
			}
		}

		if err := v.session.MessageReactionAdd(v.message.ChannelID, v.message.ID, emoji); err != nil {
			return err
		}
		v.reactionState = emoji
	}

	return nil
}

func findVoiceChannel(s *discordgo.Session, guildID, channelName string) (*discordgo.Channel, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("Could not enumerate guild channels: %w", err)
	}

	for _, channel := range channels {
		if channel.Name == channelName {
			if channel.Bitrate <= 0 {
				return channel, fmt.Errorf("This does not appear to be a voice channel")
			}
			return channel, nil
		}
	}

	return nil, fmt.Errorf("Sorry, I couldn't find a voice channel called '" + channelName + "'")
}

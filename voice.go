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
	waitTime      float64                    // How long to wait in silence before guiding to speak
	cancel        func()                     // A cancellation function to trigger cleanup operations
	trigger       *discordgo.Message         // The original message that opened this session
	session       *discordgo.Session         // The current Discord connection
	message       *discordgo.Message         // The message that is providing guidance to the users
	voiceChannel  *discordgo.Channel         // THe voice channel that is being listened to
	voice         *discordgo.VoiceConnection // The voice byte stream for listening to the channel
	reactionState string                     // The current reaction set on the message
}

func startGuider(ctx context.Context, g *voiceGuider, channelName string) {
	guiders.Add(1)
	defer guiders.Done()

	const defaultWaitTime = 3
	g.waitTime = defaultWaitTime

	// Set up a cancellable context to end this guider
	ctx, g.cancel = context.WithCancel(ctx)
	defer g.cancel()

	// Find the channel that they want us to join
	var err error
	g.voiceChannel, err = findVoiceChannel(g.session, g.trigger.GuildID, channelName)
	if err != nil {
		g.session.ChannelMessageSend(g.trigger.ChannelID, err.Error())
		return
	} else if g.voiceChannel == nil {
		log.Println("Nil voice channel encountered for ", channelName)
		return
	}
	log.Println("Starting guidance session for ", g.voiceChannel.ID)

	// Join the voice channel
	g.voice, err = g.session.ChannelVoiceJoin(g.voiceChannel.GuildID, g.voiceChannel.ID, true, false)
	if err != nil {
		log.Println(err)
		return
	}
	defer g.voice.Disconnect()

	// Add a message to the channel for updating users
	g.message, err = g.session.ChannelMessageSend(g.trigger.ChannelID, g.generateMessageText())
	if err != nil {
		log.Println(err)
		return
	}
	defer g.session.ChannelMessageEdit(g.message.ChannelID, g.message.ID, "Guidance session has ended. Use '!manners guide-voice <channel>' again to restart.")
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

		if err := g.updateMessageReactions(time.Since(start)); err != nil {
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
	case "1️⃣":
		v.waitTime = 1
		v.session.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
		v.session.ChannelMessageEdit(m.ChannelID, m.MessageID, v.generateMessageText())
	case "2️⃣":
		v.waitTime = 2
		v.session.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
		v.session.ChannelMessageEdit(m.ChannelID, m.MessageID, v.generateMessageText())
	case "3️⃣":
		v.waitTime = 3
		v.session.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
		v.session.ChannelMessageEdit(m.ChannelID, m.MessageID, v.generateMessageText())
	case "4️⃣":
		v.waitTime = 4
		v.session.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
		v.session.ChannelMessageEdit(m.ChannelID, m.MessageID, v.generateMessageText())
	case "5️⃣":
		v.waitTime = 5
		v.session.MessageReactionRemove(m.ChannelID, m.MessageID, m.Emoji.Name, m.UserID)
		v.session.ChannelMessageEdit(m.ChannelID, m.MessageID, v.generateMessageText())
	}
}

func (v *voiceGuider) generateMessageText() string {
	return fmt.Sprintf(`Listening in the "%s" channel.
	* A 🙊 reaction indicates someone is speaking or has recently spoken.
	* A 🔊 reaction encourages speaking again after the desired pause has occurred.
	* Add the 🛑 reaction at any time to cancel this session.
	* Add the 1️⃣2️⃣3️⃣4️⃣5️⃣ reactions to configure how many seconds to the hold the silence for. Currently at %0.0f.`,
		v.voiceChannel.Name,
		v.waitTime)
}

func (v *voiceGuider) updateMessageReactions(elapsed time.Duration) error {
	amount := (int)(math.Ceil(v.waitTime - elapsed.Seconds()))

	// Build the new emoji
	emoji := "🔊"
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

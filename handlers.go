package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateListeningStatus("!manners")
}

func commandsHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, "!manners ") {
		return
	}

	args := strings.Split(m.Content, " ")[1:]
	if len(args) == 0 {
		return
	}

	switch args[0] {
	case "guide-voice":
		if len(args) != 2 {
			sendHelpText(s, m.ChannelID)
			return
		}

		// Find the channel that they want us to join
		voiceChannel, err := findVoiceChannel(s, m.GuildID, args[1])
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, err.Error())
			return
		} else if voiceChannel == nil {
			log.Println("Nil voice channel encountered for ", args[1])
			return
		}

		// Join the channel and build a guider
		g := voiceGuider{session: s}
		g.voice, err = s.ChannelVoiceJoin(m.GuildID, voiceChannel.ID, true, false)
		if err != nil {
			log.Println(err)
			return
		}

		g.message, err = s.ChannelMessageSend(m.ChannelID, "Setting up guidance session...")
		if err != nil {
			log.Println(err)
			return
		}

		// Add the guider and start it running
		go startGuider(ctx, g)
	default:
		sendHelpText(s, m.ChannelID)
	}
}

func sendHelpText(s *discordgo.Session, channelID string) {
	const helpText = `Hello there! Here are the commands that I support:
* ` + "`guide-voice <channel>`" + `: Joins the given voice channel and displays a message in this text channel guiding attendees to pause before responding.
`

	s.ChannelMessageSend(channelID, helpText)
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

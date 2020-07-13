package main

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const helpText = `Hello there! Here are the commands that I support:
* ` + "`guide-voice <channel>`" + `: Joins the given voice channel and displays a message in this text channel guiding attendees to pause before responding.
`

func readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateListeningStatus("!manners")
}

func commandsHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, "!manners ") || m.Member.User.Bot {
		return
	}

	args := strings.Split(m.Content, " ")[1:]
	if len(args) == 0 {
		return
	}

	log.Println(args)

	switch args[0] {
	case "guide-voice":
		if len(args) != 2 {
			s.ChannelMessageSend(m.ChannelID, helpText)
			return
		}

		// Build a guidance session and start it
		g := voiceGuider{session: s, trigger: m.Message}
		go startGuider(ctx, &g, args[1])
	default:
		s.ChannelMessageSend(m.ChannelID, helpText)
	}
}

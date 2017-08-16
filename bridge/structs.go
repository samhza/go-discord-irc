package bridge

import (
	"github.com/bwmarrin/discordgo"
)

// DiscordMessage is a chat message sent to IRC (from Discord)
type DiscordMessage struct {
	*discordgo.Message
	Content  string
	IsAction bool
}

// IRCMessage is a chat message sent to Discord (from IRCListener)
type IRCMessage struct {
	IRCChannel string
	Username   string
	Message    string
	IsAction   bool
}

// DiscordUser is information that IRC needs to know about a user
type DiscordUser struct {
	ID            string // globally unique id
	Discriminator string // locally unique ID
	Nick          string // still non-unique
	Bot           bool   // are they a bot?
	Online        bool
}

//

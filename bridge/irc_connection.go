package bridge

import (
	"fmt"
	"time"

	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

// An ircConnection should only ever communicate with its manager
// Refer to `(m *ircManager) CreateConnection` to see how these are spawned
type ircConnection struct {
	innerCon *irc.Connection

	discord DiscordUser
	nick    string

	messages      chan IRCMessage
	cooldownTimer *time.Timer

	manager *IRCManager

	// channel ID for their discord channel for PMs
	pmDiscordChannel string
}

func (i *ircConnection) OnWelcome(e *irc.Event) {
	i.JoinChannels()
	i.innerCon.SendRawf("MODE %s +D", i.innerCon.GetNick())

	go func(i *ircConnection) {
		for m := range i.messages {
			if m.IsAction {
				i.innerCon.Action(m.IRCChannel, m.Message)
			} else {
				i.innerCon.Privmsg(m.IRCChannel, m.Message)
			}
		}
	}(i)
}

func (i *ircConnection) JoinChannels() {
	i.innerCon.SendRaw(i.manager.bridge.GetJoinCommand(i.manager.RequestChannels(i.discord.ID)))
}

func (i *ircConnection) UpdateDetails(discord DiscordUser) {
	if i.discord.Username != discord.Username {
		i.innerCon.QuitMessage = fmt.Sprintf("Changing real name from %s to %s", i.discord.Username, discord.Username)
		i.manager.CloseConnection(i)

		// After one second make the user reconnect.
		// This should be enough time for the nick tracker to update.
		time.AfterFunc(time.Second, func() {
			i.manager.HandleUser(discord)
		})
		return
	}

	// if their details haven't changed, don't do anything
	if (i.discord.Nick == discord.Nick) && (i.discord.Discriminator == discord.Discriminator) {
		return
	}

	i.discord = discord
	i.nick = i.manager.generateNickname(i.discord)
	i.innerCon.RealName = discord.Username

	go i.innerCon.Nick(i.nick)
}

func (i *ircConnection) OnPrivateMessage(e *irc.Event) {
	// Alert private messages
	if string(e.Arguments[0][0]) != "#" {
		if e.Message() == "help" {
			i.innerCon.Privmsg(e.Nick, "Commands: help, who")
		} else if e.Message() == "who" {
			i.innerCon.Privmsgf(e.Nick, "I am: %s#%s with ID %s", i.discord.Nick, i.discord.Discriminator, i.discord.ID)
		} else {
			// i.innerCon.Privmsg(e.Nick, "Private messaging Discord users is not supported, but I support commands! Type 'help'.")
		}

		d := i.manager.bridge.discord

		msg := fmt.Sprintf("%s,%s: %s", e.Connection.Server, e.Source, e.Message())
		_, err := d.ChannelMessageSend(i.pmDiscordChannel, msg)
		if err != nil {
			log.Warnln("Could not send PM", i.discord, err)
			return
		}
		return
	}

	// GTANet does not support deafness so the below logmsg has been disabled
	// log.Println("Non listener IRC connection received PRIVMSG from channel. Something went wrong.")
}

func (i *ircConnection) SetAway(status string) {
	i.innerCon.SendRawf("AWAY :%s", status)
}

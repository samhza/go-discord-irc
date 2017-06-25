package bridge

import (
	"fmt"
)

// The brains of the operation.
// We coordinate concurrency between all connections and data stores.
// I also write a GUI interface in Visual Basic to track your IP address.
// TODO: Rename to something less comfortable
type home struct {
	dib         *Bridge
	discord     *discordBot
	ircListener *ircListener
	ircManager  *ircManager

	done chan interface{}

	discordMessagesChan      chan DiscordNewMessage
	discordMessageEventsChan chan DiscordMessageEvent
}

func prepareHome(dib *Bridge, discord *discordBot, ircListener *ircListener, ircManager *ircManager) {
	dib.h = &home{
		dib:         dib,
		discord:     discord,
		ircListener: ircListener,
		ircManager:  ircManager,

		done: make(chan interface{}),

		discordMessagesChan:      make(chan DiscordNewMessage),
		discordMessageEventsChan: make(chan DiscordMessageEvent),
	}

	go dib.h.loop()
}

func (h *home) GetIRCChannels() []string {
	return h.dib.chanIRC
}

func (h *home) GetDiscordUserInfo(userID string) (discriminator, username string, err error) {
	// TODO: Catch username changes, and cache UserID:Username mappings somewhere
	u, err := h.discord.User(userID)
	if err != nil {
		fmt.Println("Could not find user", err)
		return "", "", err
	}

	discriminator = u.Discriminator
	username = u.Username

	return
}

func (h *home) SendDiscordMessage(msg DiscordNewMessage) {
	h.discordMessagesChan <- msg
}

func (h *home) OnDiscordMessage(authorID, channelID, content string) {
	h.discordMessageEventsChan <- DiscordMessageEvent{
		userID:    authorID,
		channelID: channelID,
		message:   content,
	}
}

func (h *home) loop() {
	for {
		select {
		case msg := <-h.discordMessagesChan:
			fmt.Println("Received, sending to", h.dib.chanMapToDiscord[msg.ircChannel])
			_, err := h.discord.ChannelMessageSend(h.dib.chanMapToDiscord[msg.ircChannel], msg.str)
			if err != nil {
				fmt.Println("Message from IRC to Discord was unsuccessfully sent!", err.Error())
			}
		case msg := <-h.discordMessageEventsChan:
			ircChan := h.dib.chanMapToIRC[msg.channelID]

			h.ircManager.PulseID(msg.userID)
			h.ircManager.SendMessage(msg.userID, ircChan, msg.message)
		case <-h.done:
			fmt.Println("Closing all connections!")
			h.discord.Close()
			h.ircListener.Disconnect()
			h.ircManager.DisconnectAll()
		default:
		}

	}
}

// DiscordMessageEvent is a chat message from Discord to IRCManager
type DiscordMessageEvent struct {
	channelID string
	userID    string
	message   string
}

// DiscordNewMessage is a chat message from IRCListener to Discord
type DiscordNewMessage struct {
	ircChannel string
	str        string
}
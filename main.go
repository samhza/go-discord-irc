package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/qaisjp/go-discord-irc/bridge"
	"github.com/spf13/viper"
)

func main() {
	config := flag.String("config", "", "Config file to read configuration stuff from")
	debugMode := flag.Bool("debug", false, "Debug mode? False will use the value set in the settings")
	insecure := flag.Bool("insecure", false, "Skip TLS verification? (INSECURE MODE)")
	ircNoTLS := flag.Bool("no_irc_tls", false, "Disable TLS for IRC bots?")
	simple := flag.Bool("simple", false, "When in simple mode, the bridge will only spawn one IRC connection for listening and speaking")

	flag.Parse()

	if *config == "" {
		log.Fatalln("--config argument is required!")
		return
	}

	if *simple {
		log.Println("Running in simple mode.")
	}

	viper := viper.New()
	ext := filepath.Ext(*config)
	viper.SetConfigName(strings.TrimSuffix(filepath.Base(*config), ext))
	viper.SetConfigType(ext[1:])
	viper.AddConfigPath(filepath.Dir(*config))

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalln(errors.Wrap(err, "could not read config"))
	}

	discordBotToken := viper.GetString("discord_token")             // Discord Bot User Token
	channelMappings := viper.GetStringMapString("channel_mappings") // Discord:IRC mappings in format '#discord1:#irc1,#discord2:#irc2,...'
	ircServer := viper.GetString("irc_server")                      // Server address to use, example `irc.freenode.net:7000`.
	guildID := viper.GetString("guild_id")                          // Guild to use
	webIRCPass := viper.GetString("webirc_pass")                    // Password for WEBIRC
	//
	if !*debugMode {
		*debugMode = viper.GetBool("debug")
	}
	//
	ircUsername := viper.GetString("irc_listener_name") // Name for IRC-side bot, for listening to messages.
	viper.SetDefault("irc_listener_name", "~d")
	//
	suffix := viper.GetString("suffix") // The suffix to append to IRC connections (not in use when simple mode is on)
	viper.SetDefault("suffix", "~d")

	if webIRCPass == "" {
		log.Println("Warning: webirc_pass is empty")
	}

	// Validate mappings
	if channelMappings == nil || len(channelMappings) == 0 {
		log.Fatalln("Channel mappings are missing!")
		return
	}

	dib, err := bridge.New(&bridge.Config{
		DiscordBotToken:    discordBotToken,
		GuildID:            guildID,
		IRCListenerName:    ircUsername,
		IRCServer:          ircServer,
		IRCUseTLS:          !*ircNoTLS, // exclamation mark is NOT a typo
		WebIRCPass:         webIRCPass,
		Debug:              *debugMode,
		InsecureSkipVerify: *insecure,
		Suffix:             suffix,
		SimpleMode:         *simple,
		ChannelMappings:    channelMappings,
	})

	if err != nil {
		log.Printf("Go-Discord-IRC failed to start because: %s", err.Error())
		return
	}

	fmt.Println("Go-Discord-IRC is now running. Press Ctrl-C to exit.")

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Println("Configuration file has changed!")
		if newUsername := viper.GetString("irc_listener_name"); ircUsername != newUsername {
			log.Printf("Changed irc_listener_name from '%s' to '%s'", ircUsername, newUsername)
			// Listener name has changed
			ircUsername = newUsername
			dib.SetIRCListenerName(ircUsername)
		}

		if debug := viper.GetBool("debug"); *debugMode != debug {
			log.Printf("Debug changed from %+v to %+v", *debugMode, debug)
			*debugMode = debug
			dib.SetDebugMode(*debugMode)
		}

		chans := viper.GetStringMapString("channel_mappings")
		equalChans := reflect.DeepEqual(chans, channelMappings)
		if !equalChans {
			log.Println("Channel mappings updated!")
			if chans == nil || len(chans) == 0 {
				log.Println("Channel mappings are missing!")
			}

			if dib.SetChannelMappings(chans) {
				channelMappings = chans
			}
		}
	})

	// Create new signal receiver
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	err = dib.Open()
	if err != nil {
		panic(err)
	}

	// Watch for a signal
	<-sc

	fmt.Println("Shutting down Go-Discord-IRC...")

	// Cleanly close down the Discord session.
	dib.Close()
}

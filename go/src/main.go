package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// We keep this in memory for now but eventually we have to set up a mariaDB

var commands = getCommands()
var config *Config
var bot *tgbotapi.BotAPI

func init() {
	var configError error
	config, configError = getConfig("config.json")
	checkForFatalError("Error loading config: ", configError)

	var botError error
	bot, botError = tgbotapi.NewBotAPI(config.Token)
	checkForFatalError("Error initializing bot: ", botError)
	bot.Debug = true
	log.Printf("Connected as %s", bot.Self.UserName)
}

func main() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	var channelError error
	updates, channelError := bot.GetUpdatesChan(u)
	checkForFatalError("Error opening bot channel: ", channelError)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			command := update.Message.Command()

			cmd, ok := commands[command]

			if !ok {
				go handleUnknownCommand(bot, update.Message)
			} else {
				go cmd(bot, update.Message)
			}

		}

	}
}

func checkForFatalError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

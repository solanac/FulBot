package main

import (
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Game struct {
	Active  bool
	Players []int
}

var currentGame *Game

func main() {
	bot, err := tgbotapi.NewBotAPI("6063474758:AAGNNDKnO3IZsIdrVxPDOgYKv_lbedCGnew")
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true

	log.Printf("Conectado como %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			command := update.Message.Command()
			switch command {
			case "yojuego":
				handleYoJuegoCommand(bot, update.Message)
			case "verpartido":
				handleVerPartidoCommand(bot, update.Message)
			case "nuevopartido":
				handleNuevoPartidoCommand(bot, update.Message)
			default:
				handleUnknownCommand(bot, update.Message)
			}
		}
	}
}

func handleYoJuegoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame == nil {
		response := "No hay un partido activo en este momento. Puedes iniciar uno nuevo con /nuevopartido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if currentGame.Active {
		playerID := message.From.ID
		if !contains(currentGame.Players, playerID) {
			currentGame.Players = append(currentGame.Players, playerID)
			response := "Te has unido al partido. ¡Buena suerte!"
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			bot.Send(msg)
		} else {
			response := "Ya estás en el partido. ¡A jugar!"
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			bot.Send(msg)
		}
	} else {
		response := "El partido no está activo en este momento. Espera a que se inicie uno nuevo."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
	}
}

func handleVerPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame == nil || !currentGame.Active {
		response := "No hay un partido activo en este momento. Puedes iniciar uno nuevo con /nuevopartido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	playerCount := len(currentGame.Players)
	response := "Partido activo:\n\n"
	response += "Jugadores:\n"

	for i, playerID := range currentGame.Players {
		user := getUserInfo(bot, message, playerID)
		response += strconv.Itoa(i+1) + ". " + user.FirstName + " " + user.LastName + "\n"
	}
	response += "\nTotal de jugadores: " + strconv.Itoa(playerCount)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleNuevoPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame != nil && currentGame.Active {
		response := "Ya hay un partido activo. Finalízalo antes de iniciar uno nuevo."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	currentGame = &Game{
		Active:  true,
		Players: make([]int, 0),
	}

	response := "Se ha iniciado un nuevo partido. Puedes unirte al partido con el comando /yojuego."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	response := "Comando desconocido. Los comandos disponibles son: /yojuego, /verpartido, /nuevopartido"
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getUserInfo(bot *tgbotapi.BotAPI, message *tgbotapi.Message, userID int) *tgbotapi.User {
	userConfig := tgbotapi.ChatConfigWithUser{
		ChatID: message.Chat.ID,
		UserID: userID,
	}

	user, err := bot.GetChatMember(userConfig)
	if err != nil {
		log.Printf("Error obteniendo información del usuario: %v", err)
		return nil
	}

	return user.User
}

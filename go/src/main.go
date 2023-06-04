package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Game struct {
	Active      bool
	Players     []int
	OrganizerID int
	Cancha      string
	Tamano      string
	MaxPlayers  int
}

var currentGame *Game

func main() {

	config, err := readConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(config.Token)
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
			case "cancelarpartido":
				handleCancelarPartidoCommand(bot, update.Message)
			case "help":
				handleHelpCommand(bot, update.Message)
			default:
				handleUnknownCommand(bot, update.Message)
			}
		}
	}
}

func readConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
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
		user := getUserInfo(bot, message.Chat.ID, playerID)
		if user != nil {
			response += strconv.Itoa(i+1) + ". " + user.FirstName + " " + user.LastName + "\n"
		}
	}
	response += "\nTotal de jugadores: " + strconv.Itoa(playerCount) + "/" + strconv.Itoa(currentGame.MaxPlayers)
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

	// Obtener los parámetros de la creación del partido
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 2 {
		response := "Para iniciar un nuevo partido, debes proporcionar la cancha y el tamaño. Ejemplo: /nuevopartido [cancha] [tamaño]"
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	cancha := params[0]
	tamano := params[1]

	maxPlayers, err := getMaxPlayersByTamano(tamano)
	if err != nil {
		response := "Error: " + err.Error()
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	currentGame = &Game{
		Active:      true,
		Players:     make([]int, 0),
		OrganizerID: message.From.ID,
		Cancha:      cancha,
		Tamano:      tamano,
		MaxPlayers:  maxPlayers,
	}

	response := "Se ha iniciado un nuevo partido de " + tamano + ". Puedes unirte al partido con el comando /yojuego."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleCancelarPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame == nil || !currentGame.Active {
		response := "No hay un partido activo en este momento."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if currentGame.OrganizerID != message.From.ID {
		response := "Solo el organizador del partido puede cancelarlo."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	currentGame = nil

	response := "El partido ha sido cancelado por el organizador."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleHelpCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	response := "Los comandos disponibles son:\n\n"
	response += "/yojuego - Únete al partido activo\n"
	response += "/verpartido - Muestra la información del partido activo\n"
	response += "/nuevopartido <tamaño> <cancha> - Inicia un nuevo partido\n"
	response += "/cancelarpartido - Cancela el partido activo\n"
	response += "/help - Muestra la lista de comandos disponibles"
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	response := "Comando desconocido. Usa /help para ver la lista de comandos disponibles."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func getUserInfo(bot *tgbotapi.BotAPI, chatID int64, userID int) *tgbotapi.User {
	userConfig := tgbotapi.ChatConfigWithUser{
		ChatID: chatID,
		UserID: userID,
	}
	user, err := bot.GetChatMember(userConfig)
	if err != nil {
		log.Printf("Error al obtener información del usuario: %v", err)
		return nil
	}
	return user.User
}

func contains(slice []int, item int) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func getMaxPlayersByTamano(tamano string) (int, error) {
	maxPlayers, err := strconv.Atoi(tamano)
	if err != nil || maxPlayers < 1 || maxPlayers > 15 {
		return 0, errors.New("El tamaño especificado no es válido. Debe ser un número entre 1 y 15.")
	}
	return maxPlayers * 2, nil
}

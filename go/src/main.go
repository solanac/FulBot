package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Game struct {
	Active      bool
	Players     []int
	OrganizerID int
	Field       []string
	Size        string
	MaxPlayers  int
}

var currentGame *Game

type CommandHandlerFunc func(bot *tgbotapi.BotAPI, message *tgbotapi.Message)

var commands map[string]CommandHandlerFunc
var config *Config
var bot *tgbotapi.BotAPI

func init() {
	commands = map[string]CommandHandlerFunc{
		"yojuego":         handleYoJuegoCommand,
		"verpartido":      handleVerPartidoCommand,
		"nuevopartido":    handleNuevoPartidoCommand,
		"cancelarpartido": handleCancelarPartidoCommand,
		"darsedebaja":     handleDarseDeBajaCommand,
		"ayuda":           handleayudaCommand,
	}

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

			if ok == false {
				handleUnknownCommand(bot, update.Message)
			} else {
				cmd(bot, update.Message)
			}

		}

	}
}

func checkForFatalError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func handleDarseDeBajaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame == nil {
		response := fmt.Sprintf("No hay un partido activo en este momento, @%s. Puedes iniciar uno nuevo con /nuevopartido.", message.From.UserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}
	if currentGame.Active {
		playerId := message.From.ID
		if !contains(currentGame.Players, playerId) {
			response := fmt.Sprintf("No es posible darse de baja, @%s. No te encontras en el partido.", message.From.UserName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			bot.Send(msg)
		} else {
			currentGame.Players = remove(currentGame.Players, playerId)
			response := fmt.Sprintf("Te has dado de baja, @%s.", message.From.UserName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			msg.ReplyToMessageID = message.MessageID
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}

	}

}
func remove(slice []int, value int) []int {
	index := -1
	for i, v := range slice {
		if v == value {
			index = i
			break
		}
	}
	if index == -1 {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}

func handleYoJuegoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	playerID := message.From.ID
	if currentGame == nil {
		response := fmt.Sprintf("No hay un partido activo en este momento, @%s. Puedes iniciar uno nuevo con /nuevopartido.", message.From.UserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if currentGame.Active {
		if !contains(currentGame.Players, playerID) {
			currentGame.Players = append(currentGame.Players, playerID)
			response := fmt.Sprintf("¡Hola @%s! Te has unido al partido. ¡Buena suerte!", message.From.UserName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			msg.ReplyToMessageID = message.MessageID
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		} else {
			response := "Ya estás en el partido . ¡A jugar!"
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			bot.Send(msg)
		}
	} else {
		response := fmt.Sprintf("El partido no está activo en este momento, @%s. Espera a que se inicie uno nuevo.", message.From.UserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
	}
}

func handleVerPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame == nil || !currentGame.Active {
		response := fmt.Sprintf("No hay un partido activo en este momento, @%s. Puedes iniciar uno nuevo con /nuevopartido.", message.From.UserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	playerCount := len(currentGame.Players)
	response := "Partido activo:\n\n"
	response += "Cancha: " + strings.Join(currentGame.Field, " ") + "\n"
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
		response := fmt.Sprintf("Ya hay un partido activo, @%s .Finalízalo antes de iniciar uno nuevo", message.From.UserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	// Obtener los parámetros de la creación del partido
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 2 {
		response := fmt.Sprintf("Para iniciar un nuevo partido @%s, debes proporcionar el tamaño de equipo y la cancha. Ejemplo: /nuevopartido [tamaño] [cancha]", message.From.UserName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	size := params[0]
	field := params[1:]

	maxPlayers, err := getMaxPlayersByTamano(size)
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
		Field:       field,
		Size:        size,
		MaxPlayers:  maxPlayers,
	}

	response := "Se ha iniciado un nuevo partido de " + size + ". Puedes unirte al partido con el comando /yojuego."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleCancelarPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if currentGame == nil || !currentGame.Active {
		response := fmt.Sprintf("No hay un partido activo en este momento, @%s.", message.From.UserName)
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

	response := fmt.Sprintf("El partido ha sido cancelado por  @%s.", message.From.UserName)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleayudaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

	emojiBall := "\u26BD"
	emojiCross := "\u2718"
	emojiHelp := " \U0001F91A"
	emojiCalendar := "\U0001F4C5"
	emojiThumbsUp := "\U0001F44D"
	emojiThumbsDown := "\U0001F44E"

	response := "Los comandos disponibles son:\n\n"
	response += emojiThumbsUp + " /yojuego - Únete al partido activo\n"
	response += emojiCalendar + " /verpartido - Muestra la información del partido activo\n"
	response += emojiBall + " /nuevopartido - Inicia un nuevo partido\n"
	response += emojiCross + " /cancelarpartido -  Cancela el partido activo\n"
	response += emojiThumbsDown + " /darsedebaja - Para bajarte del partido \n"
	response += emojiHelp + " /ayuda - Muestra la lista de comandos disponibles"
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	response := "Comando desconocido. Usa /ayuda para ver la lista de comandos disponibles."
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
		log.Printf("Error obtaining user info for: %v", err)
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

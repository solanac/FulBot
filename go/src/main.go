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
	Id          int
	Active      bool
	Players     []int
	OrganizerID int
	Size        string
	MaxPlayers  int
	Address     []string
	Schedule    []string
	Date        []string
}

// We keep this in memory for now but eventually we have to set up a mariaDB
var games map[int]Game = make(map[int]Game)
var nextGameId = 1

type CommandHandlerFunc func(bot *tgbotapi.BotAPI, message *tgbotapi.Message)

var commands map[string]CommandHandlerFunc
var config *Config
var bot *tgbotapi.BotAPI

func init() {
	commands = map[string]CommandHandlerFunc{
		"yojuego":          handleYoJuegoCommand,
		"verpartido":       handleVerPartidoCommand,
		"verpartidos":      handleVerPartidosCommand,
		"nuevopartido":     handleNuevoPartidoCommand,
		"agregardireccion": handleAgregarDireccionCommand,
		"agregarfecha":     handleAgregarFechaCommand,
		"agregarhorario":   handleAgregarHorarioCommand,
		"cancelarpartido":  handleCancelarPartidoCommand,
		"darsedebaja":      handleDarseDeBajaCommand,
		"ayuda":            handleayudaCommand,
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

func handleDarseDeBajaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para darse de baja de un partido @%s, debes proporcionar el numero del partido. Ejemplo: /darsedebaja [numero]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s.", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	} else {
		playerId := message.From.ID
		if !contains(game.Players, playerId) {
			response := fmt.Sprintf("No es posible darse de baja, @%s. No te encontras en el partido.", message.From.FirstName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			bot.Send(msg)
		} else {
			game.Players = remove(game.Players, playerId)
			response := fmt.Sprintf("Te has dado de baja, @%s.", message.From.FirstName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			msg.ReplyToMessageID = message.MessageID
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}
	}
}

func handleYoJuegoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

	args := message.CommandArguments()
	params := strings.Split(args, " ")
	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para sumarte a un partido @%s, debes proporcionar el numero del partido. Ejemplo: /yojuego [numero]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	} else {
		playerID := message.From.ID
		if !contains(game.Players, playerID) {
			game.Players = append(game.Players, playerID)
			games[gameId] = game
			response := fmt.Sprintf("¡Hola @%s! Te has unido al partido. ¡Buena suerte!", message.From.FirstName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			msg.ReplyToMessageID = message.MessageID
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		} else {
			response := fmt.Sprintf("Ya estás en el partido @%s. ¡A jugar!", message.From.FirstName)
			msg := tgbotapi.NewMessage(message.Chat.ID, response)
			bot.Send(msg)
		}
	}
}

func handleVerPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para ver un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /verpartido [numero]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	playerCount := len(game.Players)
	response := "Partido pendiente:\n\n"

	address := func() string {
		if game.Address != nil {
			return strings.Join(game.Address, " ")
		}
		return "<direccion>"
	}()

	schedule := func() string {
		if game.Schedule != nil {
			return strings.Join(game.Schedule, " ")
		}
		return "<horario>"
	}()

	date := func() string {
		if game.Date != nil {
			return strings.Join(game.Date, " ")
		}
		return "<fecha>"
	}()

	response += "Cancha: " + address + "\n"
	response += "Horario: " + schedule + "\n"
	response += "Fecha: " + date + "\n"
	response += "\n"

	response += "Jugadores:\n"

	for i, playerID := range game.Players {
		user := getUserInfo(bot, message.Chat.ID, playerID)
		if user != nil {
			response += strconv.Itoa(i+1) + ". " + user.FirstName + " " + user.LastName + "\n"
		}
	}
	response += "\nTotal de jugadores: " + strconv.Itoa(playerCount) + "/" + strconv.Itoa(game.MaxPlayers)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleVerPartidosCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

	if len(games) < 1 {
		response := fmt.Sprintf("No hay partidos pendientes, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	response := "Proximos partidos:" + "\n\n"
	var activeGamesTotal = 0
	for _, game := range games {
		if !game.Active {
			continue
		}
		convertedID := strconv.Itoa(game.Id)
		playerCount := strconv.Itoa(len(game.Players)) + "/" + strconv.Itoa(game.MaxPlayers)
		bulletPoint := "\u2022"

		response += bulletPoint + " Partido " + convertedID + ", Jugadores: " + playerCount
		if game.Date != nil {
			response += "\n    - Fecha: " + strings.Join(game.Date, " ")
		}
		if game.Schedule != nil {
			response += "\n    - Horario: " + strings.Join(game.Schedule, " ")
		}
		if game.Address != nil {
			response += "\n    - Direccion: " + strings.Join(game.Address, " ")
		}
		response += "\n"
		activeGamesTotal++
	}
	response += "Total de partidos: " + strconv.Itoa(activeGamesTotal) + ".\n\n"
	response += "Puedes usar /verpartido [numero de partido] para mas inforamcion."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleNuevoPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	var game Game

	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 {
		response := fmt.Sprintf("Para iniciar un nuevo partido @%s, debes proporcionar el tamaño del partido. Ejemplo: /nuevopartido [tamaño]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	size := params[0]

	maxPlayers, err := getMaxPlayersByTamano(size)
	if err != nil {
		response := "Error: " + err.Error()
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game = Game{
		Id:          nextGameId,
		Active:      true,
		Players:     make([]int, 0),
		OrganizerID: message.From.ID,
		Size:        size,
		MaxPlayers:  maxPlayers,
	}

	games[nextGameId] = game

	response := "Se ha iniciado un nuevo partido de " + size + ". Puedes unirte al partido con el comando /yojuego " + strconv.Itoa(nextGameId)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	nextGameId++
	bot.Send(msg)
}

func handleAgregarDireccionCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para modificar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /agregardireccion [numero] [direccion]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if len(params) < 2 || params[1] == "" {
		response := fmt.Sprintf("@%s debes agregar una direccion!  Ejemplo: /agregardireccion [numero de partido] [direccion]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}
	address := params[1:]
	game.Address = address
	games[gameId] = game

	response := "Se ha agregado la dirección al partido " + strconv.Itoa(gameId) + "."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleAgregarHorarioCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para modificar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /agregarhorario [numero] [horario]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if len(params) < 2 || params[1] == "" {
		response := fmt.Sprintf("@%s debes agregar un horario!  Ejemplo: /agregarhorario [numero de partido] [horario]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	schedule := params[1:]
	game.Schedule = schedule
	games[gameId] = game

	response := "Se ha agregado el horario al partido " + strconv.Itoa(gameId) + "."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleAgregarFechaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para modificar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /agregarfecha [numero] [fecha]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if len(params) < 2 || params[1] == "" {
		response := fmt.Sprintf("@%s debes agregar una fecha!  Ejemplo: /agregarfecha [numero de partido] [fecha]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	date := params[1:]
	game.Date = date
	games[gameId] = game

	response := "Se ha agregado la fecha al partido " + strconv.Itoa(gameId) + "."
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleCancelarPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	args := message.CommandArguments()
	params := strings.Split(args, " ")

	if len(params) < 1 || params[0] == "" {
		response := fmt.Sprintf("Para cancelar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /cancelarpartido [numero] [horario]", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	gameId, err := strconv.Atoi(params[0])
	if err != nil {
		response := params[0] + " no es un numero de partido valido."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game, exists := games[gameId]

	if !exists || !game.Active {
		response := fmt.Sprintf("No hay un partido pendiente con ese numero, @%s.", message.From.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	if game.OrganizerID != message.From.ID {
		response := "Solo el organizador del partido puede cancelarlo."
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		bot.Send(msg)
		return
	}

	game.Active = false
	games[gameId] = game

	response := fmt.Sprintf("El partido ha sido cancelado por  @%s.", message.From.FirstName)
	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	bot.Send(msg)
}

func handleayudaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {

	emojiBall := "\u26BD"
	emojiClock := "\U0001F551"
	emojiAddress := "\U0001F4CD"
	emojiCross := "\u2718"
	emojiHelp := " \U0001F91A"
	emojiCalendar := "\U0001F4C5"
	emojiThumbsUp := "\U0001F44D"
	emojiThumbsDown := "\U0001F44E"

	response := "Los comandos disponibles son:\n\n"
	response += emojiThumbsUp + " /yojuego [numero de partido] - Únete a un partido\n"
	response += emojiCalendar + " /verpartido [numero de partido] - Muestra la información de un partido\n"
	response += emojiCalendar + " /verpartidos - Muestra la información de todos los partidos\n"
	response += emojiBall + " /nuevopartido [tamaño] - Inicia un nuevo partido\n"
	response += emojiCalendar + " /agregarfecha [numero de partido] [fecha] - Agrega la fecha a un partido\n"
	response += emojiClock + " /agregarhorario [numero de partido] [horario] - Agrega un horario a un partido\n"
	response += emojiAddress + " /agregardireccion [numero de partido] [direccion] - Agrega una dirección a un partido\n"
	response += emojiCross + " /cancelarpartido [numero de partido] -  Cancela un partido, solo la persona que lo creo puede cancelarlo\n"
	response += emojiThumbsDown + " /darsedebaja [numero de partido] - Para bajarte de un partido \n"
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

func getMaxPlayersByTamano(tamano string) (int, error) {
	maxPlayers, err := strconv.Atoi(tamano)
	if err != nil || maxPlayers < 1 || maxPlayers > 15 {
		return 0, errors.New("El tamaño especificado no es válido. Debe ser un número entre 1 y 15.")
	}
	return maxPlayers * 2, nil
}

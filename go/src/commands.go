package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var mutex sync.Mutex
var games map[int]Game = make(map[int]Game)
var nextGameId = 1

type CommandHandlerFunc func(bot *tgbotapi.BotAPI, message *tgbotapi.Message)

func getCommands() map[string]CommandHandlerFunc {
	return map[string]CommandHandlerFunc{
		"yojuego":          handleYoJuegoCommand,
		"verpartido":       handleVerPartidoCommand,
		"verpartidos":      handleVerPartidosCommand,
		"nuevopartido":     handleNuevoPartidoCommand,
		"agregardireccion": handleAgregarDireccionCommand,
		"agregarfecha":     handleAgregarFechaCommand,
		"agregarhorario":   handleAgregarHorarioCommand,
		"cancelarpartido":  handleCancelarPartidoCommand,
		"darsedebaja":      handleDarseDeBajaCommand,
		"agregarinvitado":  handleAgregrarInvitadoCommand,
		"bajarinvitado":    handleBajarInvitadoCommand,
		"ayuda":            handleayudaCommand,
	}
}

/*
##############################################################
#                                                            #
#                   Handlers	                             #
#                                                            #
##############################################################
*/
func handleBajarInvitadoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para dar de baja a un invitado @%s, debes proporcionar el numero del partido y el nombre. Ejemplo: /bajarinivitado \\[numero] \\[nombre]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s.", message.From.FirstName)
			} else {
				playerName := strings.Join(params[1:], " ")
				if !containsString(game.Guests, playerName) {
					response = fmt.Sprintf("No es posible dar de baja a %s. No se encuentra en el partido.", playerName)
				} else {
					game.Guests = removeString(game.Guests, playerName)
					updateGame(gameId, game)
					response = fmt.Sprintf("@%s diste de baja a %s.", message.From.FirstName, playerName)
				}
			}
		}

	}

	respondToMessage(message, response)
}
func handleAgregrarInvitadoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para agregar a un tercero @%s, debes proporcionar el numero del partido y el nombre del jugador. Ejemplo: /agregarinvitado [numero] [nombre]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + "no es un numero de partido valido"
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s.", message.From.FirstName)
			} else if len(params) < 2 || params[1] == "" {
				response = fmt.Sprintf("@%s debes agregar el nombre del jugador! Ejemplo: /agregartercero [numero] [nombre]", message.From.FirstName)
			} else {
				playerName := strings.Join(params[1:], " ")
				if game.MaxPlayers > (len(game.Players) + len(game.Guests)) {
					game.Guests = append(game.Guests, playerName)
					updateGame(gameId, game)
					response = fmt.Sprintf("@%s has invitado a %s al partido .", message.From.FirstName, playerName)

				} else {
					response = fmt.Sprintf("El partido esta completo @%s, no podes invitar a %s", message.From.FirstName, playerName)
				}

			}
		}
	}
	respondToMessage(message, response)
}

func handleDarseDeBajaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para darse de baja de un partido @%s, debes proporcionar el numero del partido. Ejemplo: /darsedebaja [numero]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s.", message.From.FirstName)
			} else {
				playerId := message.From.ID
				if !contains(game.Players, playerId) {
					response = fmt.Sprintf("No es posible darse de baja, @%s. No te encontras en el partido.", message.From.FirstName)
				} else {
					game.Players = remove(game.Players, playerId)
					updateGame(gameId, game)
					response = fmt.Sprintf("Te has dado de baja, @%s.", message.From.FirstName)
				}
			}
		}

	}

	respondToMessage(message, response)
}

func handleYoJuegoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para sumarte a un partido @%s, debes proporcionar el numero del partido. Ejemplo: /yojuego [numero]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
			} else {
				if game.MaxPlayers > (len(game.Players) + len(game.Guests)) {
					playerID := message.From.ID
					if !contains(game.Players, playerID) {
						game.Players = append(game.Players, playerID)
						updateGame(gameId, game)
						response = fmt.Sprintf("¡Hola @%s! Te has unido al partido. ¡Buena suerte!", message.From.FirstName)
					} else {
						response = fmt.Sprintf("Ya estás en el partido @%s. ¡A jugar!", message.From.FirstName)
					}
				} else {
					response = fmt.Sprintf("El partido esta completo @%s, no te podes anotar", message.From.FirstName)
				}
			}
		}
	}

	respondToMessage(message, response)
}

func handleVerPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para ver un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /verpartido [numero]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
			} else {

				response = "Partido " + strconv.Itoa(gameId) + ":\n"

				if game.Date != nil {
					response += "\n    - Fecha: " + strings.Join(game.Date, " ")
				}
				if game.Schedule != nil {
					response += "\n    - Horario: " + strings.Join(game.Schedule, " ")
				}
				if game.Address != nil {
					response += "\n    - Direccion: " + strings.Join(game.Address, " ")
				}
				response += "\n" + "Jugadores:" + "\n"
				countPlayers := 0
				for _, playerID := range game.Players {
					user := getUserInfo(bot, message.Chat.ID, playerID)
					if user != nil {
						countPlayers++
						response += strconv.Itoa(countPlayers) + ". " + user.FirstName + " " + user.LastName + "\n"

					}

				}
				for _, playerName := range game.Guests {
					countPlayers++
					response += strconv.Itoa(countPlayers) + ". " + playerName + "\n"

				}
				response += "\nTotal de jugadores: " + strconv.Itoa(countPlayers) + "/" + strconv.Itoa(game.MaxPlayers)
			}
		}
	}
	respondToMessage(message, response)
}

func handleVerPartidosCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	var response string

	if len(games) < 1 {
		response = fmt.Sprintf("No hay partidos pendientes, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
	} else {
		response = "Proximos partidos:" + "\n\n"
		var activeGamesTotal = 0
		for _, game := range games {
			if !game.Active {
				continue
			}
			convertedID := strconv.Itoa(game.Id)
			playerCount := strconv.Itoa(len(game.Players)) + "/" + strconv.Itoa(game.MaxPlayers)

			response += unicodeBulletPoint + " Partido " + convertedID + ", Jugadores: " + playerCount
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
	}

	respondToMessage(message, response)
}

func handleNuevoPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 {
		response = fmt.Sprintf("Para iniciar un nuevo partido @%s, debes proporcionar el tamaño del partido. Ejemplo: /nuevopartido [tamaño]", message.From.FirstName)
	} else {
		size := params[0]
		maxPlayers, err := getMaxPlayersByTamano(size)
		if err != nil {
			response = "Error al crear nuevo partido: " + err.Error()
		} else {
			game := Game{
				Id:          nextGameId,
				Active:      true,
				Players:     make([]int, 0),
				Guests:      make([]string, 0),
				OrganizerID: message.From.ID,
				Size:        size,
				MaxPlayers:  maxPlayers,
			}
			updateGame(nextGameId, game)
			response = "Se ha iniciado un nuevo partido de " + size + ". Puedes unirte al partido con el comando /yojuego " + strconv.Itoa(nextGameId)
		}

	}
	nextGameId++
	respondToMessage(message, response)
}

func handleAgregarDireccionCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para modificar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /agregardireccion [numero] [direccion]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
			} else if len(params) < 2 || params[1] == "" {
				response = fmt.Sprintf("@%s debes agregar una direccion!  Ejemplo: /agregardireccion [numero de partido] [direccion]", message.From.FirstName)
			} else {
				address := params[1:]
				game.Address = address
				updateGame(gameId, game)
				response = "Se ha agregado la dirección al partido " + strconv.Itoa(gameId) + "."
			}
		}
	}
	respondToMessage(message, response)
}

func handleAgregarHorarioCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para modificar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /agregarhorario [numero] [horario]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
			} else if len(params) < 2 || params[1] == "" {
				response = fmt.Sprintf("@%s debes agregar un horario!  Ejemplo: /agregarhorario [numero de partido] [horario]", message.From.FirstName)
			} else {
				schedule := params[1:]
				game.Schedule = schedule
				updateGame(gameId, game)
				response = "Se ha agregado el horario al partido " + strconv.Itoa(gameId) + "."
			}
		}
	}
	respondToMessage(message, response)
}

func handleAgregarFechaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para modificar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /agregarfecha [numero] [fecha]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s. Puedes iniciar uno nuevo con /nuevopartido", message.From.FirstName)
			} else if len(params) < 2 || params[1] == "" {
				response = fmt.Sprintf("@%s debes agregar una fecha!  Ejemplo: /agregarfecha [numero de partido] [fecha]", message.From.FirstName)
			} else {
				date := params[1:]
				game.Date = date
				updateGame(gameId, game)
				response = "Se ha agregado la fecha al partido " + strconv.Itoa(gameId) + "."
			}
		}
	}
	respondToMessage(message, response)
}

func handleCancelarPartidoCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	params := getCommandParams(message)
	var response string

	if len(params) < 1 || params[0] == "" {
		response = fmt.Sprintf("Para cancelar un partido @%s, debes proporcionar el numero del mismo. Ejemplo: /cancelarpartido [numero] [horario]", message.From.FirstName)
	} else {
		gameId, err := strconv.Atoi(params[0])
		if err != nil {
			response = params[0] + " no es un numero de partido valido."
		} else {
			game, exists := games[gameId]
			if !exists || !game.Active {
				response = fmt.Sprintf("No hay un partido pendiente con ese numero, @%s.", message.From.FirstName)
			} else if game.OrganizerID != message.From.ID {
				response = "Solo el organizador del partido puede cancelarlo."
			} else {
				game.Active = false
				updateGame(gameId, game)
				response = fmt.Sprintf("El partido ha sido cancelado por  @%s.", message.From.FirstName)
			}
		}
	}
	respondToMessage(message, response)
}

func handleayudaCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	response := "Los comandos disponibles son:\n\n"
	response += emojiThumbsUp + " /yojuego \\[numero de partido] - Únete a un partido\n"
	response += emojiCalendar + " /verpartido \\[numero de partido] - Muestra la información de un partido\n"
	response += emojiCalendar + " /verpartidos - Muestra la información de todos los partidos\n"
	response += emojiBall + " /nuevopartido \\[tamaño] - Inicia un nuevo partido\n"
	response += emojiCalendar + " /agregarfecha \\[numero de partido] \\[fecha] - Agrega la fecha a un partido\n"
	response += emojiClock + " /agregarhorario \\[numero de partido] \\[horario] - Agrega un horario a un partido\n"
	response += emojiAddress + " /agregardireccion \\[numero de partido] \\[direccion] - Agrega una dirección a un partido\n"
	response += emojiCross + " /cancelarpartido \\[numero de partido] -  Cancela un partido, solo la persona que lo creo puede cancelarlo\n"
	response += emojiThumbsDown + " /darsedebaja \\[numero de partido] - Para bajarte de un partido \n"
	response += emojiGhost + " /agregarinvitado \\[numero de partido] \\[nombre] - Para agregar a un invitado a un partido \n"
	response += emojiCross + " /bajarinvitado \\[numero de partido] \\[nombre] - Para dar de baja a un invitado de un partido \n"
	response += emojiHelp + " /ayuda - Muestra la lista de comandos disponibles"
	respondToMessage(message, response)
}

func handleUnknownCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	response := "Comando desconocido. Usa /ayuda para ver la lista de comandos disponibles."
	respondToMessage(message, response)
}

/*
##############################################################
#                                                            #
#                   Auxiliar functions                       #
#                                                            #
##############################################################
*/

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

func containsString(slice []string, item string) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func removeString(slice []string, value string) []string {
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

func getCommandParams(message *tgbotapi.Message) []string {
	return strings.Split(message.CommandArguments(), " ")
}

func respondToMessage(originalMessage *tgbotapi.Message, messageToSend string) {
	if len(messageToSend) < 1 || messageToSend == "" {
		messageToSend = "Lo siento, ocurrio un error al intentar procesar el comando."
	}

	msg := tgbotapi.NewMessage(originalMessage.Chat.ID, messageToSend)
	msg.ReplyToMessageID = originalMessage.MessageID
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
func updateGame(gameId int, game Game) {
	mutex.Lock()
	defer mutex.Unlock()
	games[gameId] = game

}

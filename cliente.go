package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
	"card_game/protocolo"
)

// Estados do jogo
type GameState int

const (
	MenuState GameState = iota // Soma +1 em todos aq embaixo
	WaitingState
	InGameState
	LoginState
)

func main() {
	var conn net.Conn
	var err error

	for {
		conn, err = net.Dial("tcp", "servidor:8080")
		if err == nil {
			break
		}
		fmt.Println("Aguardando o servidor...")
		time.Sleep(1 * time.Second)
	}
	defer conn.Close()
	fmt.Printf("Conectado ao servidor %s\n", conn.RemoteAddr())

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	gameChannel := make(chan string)
	go interpreter(reader, gameChannel)

	userInputReader := bufio.NewReader(os.Stdin)
	currentState := LoginState

	for {
		select {
		case msg := <-gameChannel:
			if msg == "PAREADO" {
				currentState = InGameState
				fmt.Println("\nPartida encontrada! Você agora pode enviar mensagens de chat.")
			}
		default:
		}

		if currentState == LoginState {
			showLoginMenu(userInputReader, writer)

			switch strings.TrimSpace(readLine(userInputReader)) {
			case "1":
				// ação para opção 1

			case "2":
				fmt.Print("Digite um login: ")
				login := strings.TrimSpace(readLine(userInputReader))

				fmt.Print("Agora digite uma senha: ")
				senha := strings.TrimSpace(readLine(userInputReader))

				fmt.Printf("Login: %s e Senha: %s\n", login,senha)

				req:= protocolo.Message{
					Type: "CADASTRO",
					Data: protocolo.SignInRequest{
						Login: login,
						Senha: senha,
					},
				}

				// Envia
				sendJSON(writer, req)

				time.Sleep(5 * time.Second) // Pausinha 
				showLoginMenu(userInputReader, writer)

			}
		}else if currentState == MenuState {
			showMainMenu()
			input, _ := userInputReader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "1":
				fmt.Println("Buscando sala pública...")
				req := protocolo.Message{
					Type: "FIND_ROOM",
					Data: protocolo.RoomRequest{Mode: "PUBLIC"},
				}
				sendJSON(writer, req)
				currentState = WaitingState
			case "2":
				fmt.Printf("Digite o código da sala:\n> ")
				codigoDaSala, _ := userInputReader.ReadString('\n')
				codigoDaSala = strings.TrimSpace(codigoDaSala)
				req := protocolo.Message{
					Type: "PRIV_ROOM",
					Data: protocolo.RoomRequest{RoomCode: strings.ToUpper(codigoDaSala)},
				}
				sendJSON(writer, req)
				currentState = WaitingState
			case "3":
				req := protocolo.Message{
					Type: "CREATE_ROOM",
					Data: nil,
				}
				sendJSON(writer, req)
				currentState = WaitingState
			case "0":
				req := protocolo.Message{
					Type: "QUIT",
					Data: nil,
				}
				sendJSON(writer, req)
				fmt.Println("Saindo do jogo. Desconectando...")
				time.Sleep(1 * time.Second)
				return
			default:
				fmt.Println("Opção inválida. Tente novamente.")
			}
		} else if currentState == WaitingState {
			fmt.Println("Aguardando um oponente...")
			time.Sleep(5 * time.Second)
		} else if currentState == InGameState {
			showInGameMenu(userInputReader, writer)
		}
	}
}

func showMainMenu() {
	fmt.Println("\nEscolha uma opção:")
	fmt.Println("1. Entrar em Sala Pública.")
	fmt.Println("2. Entrar em Sala Privada.")
	fmt.Println("3. Criar sala Privada.")
	fmt.Println("0. Sair")
	fmt.Printf("> ")
}


func showLoginMenu(reader *bufio.Reader, writer *bufio.Writer) {
	fmt.Println("Bem vindo ao Super Trunfo online!")
	fmt.Println("1. Login")
	fmt.Println("2. Cadastro")
	fmt.Println("> ")
}


func showInGameMenu(reader *bufio.Reader, writer *bufio.Writer) {
	fmt.Printf("\nDigite sua Mensagem:\n> ")
	message, _ := reader.ReadString('\n')
	req := protocolo.Message{
		Type: "CHAT",
		Data: protocolo.ChatMessage{From: "leo", Content: strings.TrimSpace(message)},
	}
	sendJSON(writer, req)
}

// envia qualquer struct em JSON pelo writer
func sendJSON(writer *bufio.Writer, msg protocolo.Message) {
	jsonData, _ := json.Marshal(msg)
	writer.Write(jsonData)
	writer.WriteString("\n")
	writer.Flush()
}

// Lê mensagens JSON do servidor
func interpreter(reader *bufio.Reader, gameChannel chan string) {
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Conexão com o servidor encerrada.")
			} else {
				fmt.Println("Erro ao ler resposta do servidor:", err)
			}
			os.Exit(0)
			return
		}

		var msg protocolo.Message
		if err := json.Unmarshal([]byte(message), &msg); err != nil {
			fmt.Println("Mensagem inválida recebida:", message)
			continue
		}

		switch msg.Type {
		case "PAREADO":
			gameChannel <- "PAREADO"
		case "CHAT":
			var data protocolo.ChatMessage
			_ = mapToStruct(msg.Data, &data)
			fmt.Println(data.From + ": " + data.Content)
		case "SCREEN_MSG":
			var data protocolo.ScreenMessage
			_ = mapToStruct(msg.Data, &data)
			fmt.Println("[INFO] " + data.Content)
		}
	}
}

func mapToStruct(input interface{}, target interface{}) error {
	bytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}

// Funcao pra ajudar na leitura de entradas
func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
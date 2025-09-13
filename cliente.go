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
	StopState
)

var(
	currentUser string
	currentInventario protocolo.Inventario
	currentBalance int
)

func showMainMenu() {
	fmt.Println("\nEscolha uma opção:")
	fmt.Println("1. Entrar em Sala Pública.")
	fmt.Println("2. Entrar em Sala Privada.")
	fmt.Println("3. Criar sala Privada.")
	fmt.Println("4. Consultar Saldo.")
	fmt.Println("5. Abrir pacote de cartas.")
	fmt.Println("0. Sair")
	fmt.Printf("> ")
}					


func showLoginMenu(reader *bufio.Reader, writer *bufio.Writer) {
	fmt.Println("Bem vindo ao Super Trunfo online!")
	fmt.Println("1. Login")
	fmt.Println("2. Cadastro")
	fmt.Println("0. Sair")
	fmt.Printf("> ")
}

func showHelpMenu() {
	fmt.Println("\n=== MENU DE AJUDA ===")
	fmt.Println("/chat <mensagem> - Envia uma mensagem para o chat.")
	fmt.Println("/sair - Sai da partida.")
	fmt.Println("Digite o número da carta (1, 2, 3, 4) para escolher a carta.")
	fmt.Println("Digite o número da característica (1, 2, 3) para escolher a característica.")
	fmt.Println("=====================")
}


func showInGameMenu(reader *bufio.Reader, writer *bufio.Writer) {
	// fmt.Println("Digite /help caso precise de ajuda.")
	// fmt.Printf("\nDigite um comando ou jogada:\n> ")
	message, _ := reader.ReadString('\n')
	message = strings.TrimSpace(message)

	if strings.HasPrefix(message, "/chat") {
		// Remove o prefixo /chat e pega a mensagem
		parts := strings.SplitN(message, " ", 2)
		if len(parts) < 2 {
			fmt.Println("Uso correto: /chat <mensagem>")
			return
		}
		chatMsg := strings.TrimSpace(parts[1])
		req := protocolo.Message{
			Type: "CHAT",
			Data: protocolo.ChatMessage{From: currentUser, Content: chatMsg},
		}
		sendJSON(writer, req)
	} else if strings.HasPrefix(message, "/help") {
		showHelpMenu()
	} else if strings.HasPrefix(message, "/sair"){
		// AQUI MANDA UMA REQUISICAO PRO SERVIDOR PEDINDO PRA DESCONECTAR O PLAYER DA SALA E DAR VITORIA AO OUTRO.
	} else {
		// Aqui você pode expandir depois para jogadas do tipo escolher carta
		fmt.Println("Comando ou jogada inválida. Digite /help para ver os comandos.")
	}
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
		case "LOGIN":
			var data protocolo.LoginResponse
			_ = mapToStruct(msg.Data, &data)
			gameChannel <- data.Status
			if data.Status == "LOGADO"{
				currentBalance = data.Saldo
				currentInventario = data.Inventario
			}
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
		case "COMPRA_RESPONSE":
			var data protocolo.CompraResponse
			_ = mapToStruct(msg.Data, &data)
			gameChannel <- data.Status // Envia FALHA_COMPRA ou COMPRA_APROVADA pro channel.
			if data.Status == "COMPRA_APROVADA"{
				fmt.Printf("Voce ganhou uma carta " +data.CartaNova.Raridade + ": " + data.CartaNova.Nome + "\n") // Atualizar o inventario do player.
				currentInventario = data.Inventario
			}
		case "BALANCE_RESPONSE":
			var data protocolo.BalanceResponse
			_ = mapToStruct(msg.Data, &data)
			fmt.Printf("Seu saldo atual de moedas: %d\n", data.Saldo)
			currentBalance = data.Saldo
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


func main() {
	var conn net.Conn
	var err error

	for {
		conn, err = net.Dial("tcp", "127.0.0.1:8080")
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
				fmt.Println("Digite /help caso precise de ajuda.")
				fmt.Printf("\nDigite um comando ou jogada:\n> ")
			} else if msg == "LOGADO" {
				fmt.Println("Login realizado com sucesso!")
				currentState = MenuState
			} else if msg == "ONLINE_JA" {
				fmt.Println("O player ja esta conectado em outro dispositivo.")
				currentState = LoginState
			} else if msg == "N_EXIST"{
				fmt.Println("O usuario nao existe.")
				currentState = LoginState
			} else if msg == "COMPRA_APROVADA"{
				currentState = MenuState
			} else if msg == "EMPTY_STORAGE"{
				fmt.Println("Erro, armazem geral vazio!")
				currentState = MenuState
			}else if msg == "NO_BALANCE"{
				fmt.Println("Você não tem saldo suficiente.")
				currentState = MenuState
			}
		default:
		}

		if currentState == LoginState {
			showLoginMenu(userInputReader, writer)
			

			switch strings.TrimSpace(readLine(userInputReader)) {
			case "1":
				// Login aqui
				fmt.Print("Digite seu login: ")
				login := strings.TrimSpace(readLine(userInputReader))

				fmt.Print("Agora digite sua senha: ")
				senha := strings.TrimSpace(readLine(userInputReader))

				req:= protocolo.Message{
					Type: "LOGIN",
					Data: protocolo.LoginRequest{
						Login: login,
						Senha: senha,
					},
				}

				currentUser = login

				// Envia
				sendJSON(writer, req)
				currentState = StopState

			case "2":
				fmt.Print("Digite um login: ")
				login := strings.TrimSpace(readLine(userInputReader))

				fmt.Print("Agora digite uma senha: ")
				senha := strings.TrimSpace(readLine(userInputReader))

				req:= protocolo.Message{
					Type: "CADASTRO",
					Data: protocolo.SignInRequest{
						Login: login,
						Senha: senha,
					},
				}

				// Envia
				sendJSON(writer, req)

			case "0":
				req := protocolo.Message{
					Type: "QUIT",
					Data: nil,
				}
				sendJSON(writer, req)
				fmt.Println("Saindo do jogo. Desconectando...")
				time.Sleep(1 * time.Second)
				return
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
			case "4":
				// Consultar o saldo do jogador
				req := protocolo.Message{
					Type: "CHECK_BALANCE",
					Data: protocolo.CheckBalance{},
				}
				sendJSON(writer, req)
				// Pensar se coloco no estado STOP ou deixo assim mesmo.
			case "5":
				// Abrir pacote de cartas.
				req := protocolo.Message{
					Type: "COMPRA",
					Data: protocolo.OpenPackageRequest{},
				}
				sendJSON(writer, req)
				currentState = StopState

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
		} else{
			// Faz nada
		}
	}
}
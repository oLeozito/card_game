package main

// Imports
import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"card_game/protocolo"
)

// Estados do jogo
type GameState int

const ( // iota - Serve pra somar +1 em todos abaixo, tipo 1,2,3,4...
	MenuState GameState = iota  // Estado pra mostrar o Menu pro player.
	WaitingState 				// Esperando jogador.
	InGameState 			    // Estado igual ao StopState mas pra logica diferente.
	LoginState   				// Estado de login. (Estado Inicial)
	StopState	 				// Estado intermediario para responses do servidor.
	TurnState 	 				// Estado para quando é a vez do jogador.
)

var (
	currentUser       string
	currentInventario protocolo.Inventario
	currentBalance    int
	deckDefinido      bool // Flag para verificar se o deck foi montado
	currentHand       []protocolo.Carta // Mão do jogador no round atual
	currentState      GameState
)

// FUNCOES IMPORTANTES PRO FUNCIONAMENTO DO PROGRAMA
// envia qualquer struct em JSON pelo writer
func sendJSON(writer *bufio.Writer, msg protocolo.Message) {
	jsonData, _ := json.Marshal(msg)
	writer.Write(jsonData)
	writer.WriteString("\n")
	writer.Flush()
}

func mapToStruct(input interface{}, target interface{}) error {
	bytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
// ------------------------------------

// Funcao pra ajudar na leitura de entradas
func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// FUNCOES PRA MOSTRAR ALGO NA TELA
func showMainMenu() {
	fmt.Println("\nEscolha uma opção:")
	fmt.Println("1. Entrar em Sala Pública.")
	fmt.Println("2. Entrar em Sala Privada.")
	fmt.Println("3. Criar sala Privada.")
	fmt.Println("4. Consultar Saldo.")
	fmt.Println("5. Abrir pacote de cartas.")
	fmt.Println("6. Meu Inventário.")
	fmt.Println("7. Montar meu deck.")
	fmt.Println("8. Verificar ping.")
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
	fmt.Println("Digite o número da característica (1, 2, 3, 4) para escolher a característica.")
	fmt.Println("=====================")
}

func showInventory() {
	if len(currentInventario.Cartas) == 0 {
		fmt.Println("Seu inventário está vazio.")
		return
	}

	fmt.Println("\n=== Seu Inventário ===")
	for i, carta := range currentInventario.Cartas {
		fmt.Printf("\nCarta %d:\n", i+1)
		fmt.Printf("Nome: %s\n", carta.Nome)
		fmt.Printf("Raridade: %s\n", carta.Raridade)
		fmt.Printf("Envergadura: %d\n", carta.Envergadura)
		fmt.Printf("Velocidade Max.: %d\n", carta.Velocidade)
		fmt.Printf("Altura Max.: %d\n", carta.Altura)
		fmt.Printf("Capac. de Passageiros: %d\n", carta.Passageiros)
	}
	fmt.Println("======================")
}
// ------------------------------------

// FUNCOES PARA FUNCIONAMENTO DE PARTIDA
func montarDeck(writer *bufio.Writer) {
	if len(currentInventario.Cartas) < 4 {
		fmt.Println("Você precisa ter pelo menos 4 cartas no inventário para montar um deck.")
		return
	}

	showInventory()

	var indices [4]int
	for i := 0; i < 4; i++ {
		fmt.Printf("Escolha a carta %d do deck (digite o número correspondente do inventário): ", i+1)
		var escolha int
		fmt.Scanln(&escolha)

		if escolha < 1 || escolha > len(currentInventario.Cartas) {
			fmt.Println("Índice inválido. Tente novamente.")
			i-- // repete a mesma posição
			continue
		}

		indices[i] = escolha - 1 // ajusta para índice base 0
	}

	// monta deck local
	deck := []protocolo.Carta{
		currentInventario.Cartas[indices[0]],
		currentInventario.Cartas[indices[1]],
		currentInventario.Cartas[indices[2]],
		currentInventario.Cartas[indices[3]],
	}

	// envia para o servidor
	req := protocolo.SetDeckRequest{Cartas: deck}
	sendJSON(writer, protocolo.Message{
		Type: "SET_DECK",
		Data: req,
	})

	// mostra deck escolhido
	fmt.Println("\n=== Seu Deck ===")
	for i, c := range deck {
		fmt.Printf("Carta %d: %s\n", i+1, c.Nome)
	}
	fmt.Println("================")
	deckDefinido = true
}

func handleGameTurn(reader *bufio.Reader, writer *bufio.Writer) {
	var cardIndex int
	var attrIndex int

	// Escolher carta
	for {
		fmt.Printf("Escolha a carta para jogar (1-%d): ", len(currentHand))
		input := readLine(reader)
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(currentHand) {
			fmt.Println("Escolha inválida. Tente novamente.")
			continue
		}
		cardIndex = idx - 1
		break
	}

	// Escolher atributo
	selectedCard := currentHand[cardIndex]
	for {
		fmt.Println("\nEscolha a característica para competir:")
		fmt.Printf("1. Envergadura (%d)\n", selectedCard.Envergadura)
		fmt.Printf("2. Velocidade (%d)\n", selectedCard.Velocidade)
		fmt.Printf("3. Altura (%d)\n", selectedCard.Altura)
		fmt.Printf("4. Passageiros (%d)\n", selectedCard.Passageiros)
		fmt.Printf("> ")
		input := readLine(reader)
		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > 4 {
			fmt.Println("Escolha inválida. Tente novamente.")
			continue
		}
		attrIndex = idx
		break
	}

	var attribute string
	switch attrIndex {
	case 1:
		attribute = "Envergadura"
	case 2:
		attribute = "Velocidade"
	case 3:
		attribute = "Altura"
	case 4:
		attribute = "Passageiros"
	}

	req := protocolo.Message{
		Type: "PLAY_MOVE",
		Data: protocolo.PlayMoveRequest{
			CardIndex: cardIndex,
			Attribute: attribute,
		},
	}
	sendJSON(writer, req)
	fmt.Println("\nJogada enviada. Aguardando oponente...")
	currentState = InGameState // Volta para o estado de jogo, aguardando o resultado
}

// Lê mensagens JSON do servidor e decide o que fazer.
func interpreter(reader *bufio.Reader, writer *bufio.Writer, gameChannel chan string) {
	for {

		// Fica lendo o que o servidor envia e caso venha um erro ou EOF sai da funcao.
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

		// DECIDE O QUE FAZER COM BASE NO HEADER DA STRUCT
		switch msg.Type {

		case "LOGIN":
			var data protocolo.LoginResponse
			_ = mapToStruct(msg.Data, &data)
			gameChannel <- data.Status
			if data.Status == "LOGADO" {
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
			if data.Status == "COMPRA_APROVADA" {
				fmt.Printf("Voce ganhou uma carta " + data.CartaNova.Raridade + ": " + data.CartaNova.Nome + "\n") // Atualizar o inventario do player.
				currentInventario = data.Inventario
			}

		case "BALANCE_RESPONSE":
			var data protocolo.BalanceResponse
			_ = mapToStruct(msg.Data, &data)
			fmt.Printf("Seu saldo atual de moedas: %d\n", data.Saldo)
			currentBalance = data.Saldo

		case "PING":
			var ts int64
			_ = mapToStruct(msg.Data, &ts) // recebe timestamp enviado pelo servidor

			pong := protocolo.Message{
				Type: "PONG",
				Data: ts, // devolve o mesmo timestamp
			}
			sendJSON(writer, pong)

		case "LATENCY_RESPONSE":
			var resp protocolo.LatencyResponse
			_ = mapToStruct(msg.Data, &resp)
			fmt.Println("Sua latência é:", resp.Latencia, "ms")

		// Cases do funcionamento da partida.
		case "GAME_START":
			var data protocolo.GameStartMessage
			_ = mapToStruct(msg.Data, &data)
			fmt.Printf("\n--- PARTIDA INICIADA! ---\nVocê está jogando contra: %s\n", data.Opponent)
			currentState = InGameState // Jogo começou, pode usar o chat

		case "ROUND_START":
			var data protocolo.RoundStartMessage
			_ = mapToStruct(msg.Data, &data)
			currentHand = data.Hand
			fmt.Printf("\n--- ROUND %d ---\n", data.Round)
			fmt.Println("Sua mão:")
			for i, carta := range currentHand {
				fmt.Printf("%d. %s\n", i+1, carta.Nome)
			}
			currentState = TurnState // É a sua vez de jogar

		case "ROUND_RESULT":
			var data protocolo.RoundResultMessage
			_ = mapToStruct(msg.Data, &data)
			fmt.Println("\n--- RESULTADO DO ROUND ---")
			fmt.Printf("%s jogou %s (Atributo: %s - Valor: %d)\n", data.Player1Move.PlayerName, data.Player1Move.CardName, data.Player1Move.Attribute, data.Player1Move.AttributeValue)
			fmt.Printf("%s jogou %s (Atributo: %s - Valor: %d)\n", data.Player2Move.PlayerName, data.Player2Move.CardName, data.Player2Move.Attribute, data.Player2Move.AttributeValue)
			fmt.Printf("Pontos de %s no round: %d\n", data.Player1Move.PlayerName, data.RoundPointsP1)
			fmt.Printf("Pontos de %s no round: %d\n", data.Player2Move.PlayerName, data.RoundPointsP2)
			fmt.Printf("\nPlacar Total: %s %d x %d %s\n", data.Player1Move.PlayerName, data.TotalScoreP1, data.TotalScoreP2, data.Player2Move.PlayerName)
			fmt.Println("Iniciando próximo round...")
			currentState = InGameState

		case "GAME_OVER":
            var data protocolo.GameOverMessage
            _ = mapToStruct(msg.Data, &data)
            fmt.Println("\n\n--- FIM DE JOGO ---")
            if data.Winner == "EMPATE" {
                fmt.Println("A partida terminou em EMPATE!")
            } else {
                fmt.Printf("O vencedor é: %s\n", data.Winner)
            }
            
            // --- ALTERAÇÃO: Sempre exibe o ganho de moedas e atualiza o saldo ---
            // O servidor agora envia o valor correto para cada jogador (pode ser 0).
            if data.CoinsEarned > 0 {
                fmt.Printf("Você ganhou %d moedas!\n", data.CoinsEarned)
                currentBalance += data.CoinsEarned
            }

            fmt.Printf("Placar Final: %d x %d\n", data.FinalScoreP1, data.FinalScoreP2)
            fmt.Println("Voltando para o menu principal...")
            time.Sleep(5 * time.Second)
            currentState = MenuState
		}
	}
}
// FUNCAO PRINCIPAL
func main() {
	var conn net.Conn
	var err error

	// Loop pra abrir conexão com o servidor.
	for {
		conn, err = net.Dial("tcp", "127.0.0.1:8080") //ALTERAR O IP DO SERVIDOR PRA TESTAR
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

	// Channel pra compartilhar uma variavel entre duas threads e manter sincronismo.
	gameChannel := make(chan string)
	go interpreter(reader, writer, gameChannel)

	userInputReader := bufio.NewReader(os.Stdin)
	// Estado inicial é sempre no Login.
	currentState = LoginState

	for {
		// Comunicação entre threads ocorre aqui basicamente.

		select {// MUDAR ISSO AQUI PRA CASE !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
		case msg := <-gameChannel:
			if msg == "PAREADO" {
				currentState = InGameState
				fmt.Println("\nPartida encontrada! Aguardando início do jogo...")
				// fmt.Println("Digite /help caso precise de ajuda.")
				// fmt.Printf("\nDigite um comando ou jogada:\n> ")
			} else if msg == "LOGADO" {
				fmt.Println("Login realizado com sucesso!")
				currentState = MenuState
			} else if msg == "ONLINE_JA" {
				fmt.Println("O player ja esta conectado em outro dispositivo.")
				currentState = LoginState
			} else if msg == "N_EXIST" {
				fmt.Println("O usuario nao existe.")
				currentState = LoginState
			} else if msg == "COMPRA_APROVADA" {
				currentState = MenuState
			} else if msg == "EMPTY_STORAGE" {
				fmt.Println("Erro, armazem geral vazio!")
				currentState = MenuState
			} else if msg == "NO_BALANCE" {
				fmt.Println("Você não tem saldo suficiente.")
				currentState = MenuState
			}
		default:
		}

		// Direcionamento de acordo com estado atual.
		if currentState == LoginState {
			showLoginMenu(userInputReader, writer)

			switch strings.TrimSpace(readLine(userInputReader)) {

			case "1": // LOGIN
				fmt.Print("Digite seu login: ")
				login := strings.TrimSpace(readLine(userInputReader))

				fmt.Print("Agora digite sua senha: ")
				senha := strings.TrimSpace(readLine(userInputReader))

				req := protocolo.Message{
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

			case "2": // CADASTRO
				fmt.Print("Digite um login: ")
				login := strings.TrimSpace(readLine(userInputReader))

				fmt.Print("Agora digite uma senha: ")
				senha := strings.TrimSpace(readLine(userInputReader))

				req := protocolo.Message{
					Type: "CADASTRO",
					Data: protocolo.SignInRequest{
						Login: login,
						Senha: senha,
					},
				}

				// Envia
				sendJSON(writer, req)

			case "0": // Manda requisicao e fecha conexao
				req := protocolo.Message{
					Type: "QUIT",
					Data: nil,
				}
				sendJSON(writer, req)
				fmt.Println("Saindo do jogo. Desconectando...")
				time.Sleep(1 * time.Second)
				return
			}
		} else if currentState == MenuState {
			showMainMenu()

			input, _ := userInputReader.ReadString('\n')
			input = strings.TrimSpace(input)
			
			// MENU PRINCIPAL.
			switch input {
			case "1":
				if !deckDefinido {
					fmt.Println("Você precisa montar um deck primeiro! (Opção 7)")
					continue
				}
				fmt.Println("Buscando sala pública...")
				req := protocolo.Message{
					Type: "FIND_ROOM",
					Data: protocolo.RoomRequest{Mode: "PUBLIC"},
				}
				sendJSON(writer, req)
				currentState = WaitingState

			case "2":
				if !deckDefinido {
					fmt.Println("Você precisa montar um deck primeiro! (Opção 7)")
					continue
				}
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
				if !deckDefinido {
					fmt.Println("Você precisa montar um deck primeiro! (Opção 7)")
					continue
				}
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
				
			case "5":
				// Abrir pacote de cartas.
				req := protocolo.Message{
					Type: "COMPRA",
					Data: protocolo.OpenPackageRequest{},
				}
				sendJSON(writer, req)
				currentState = StopState

			case "6":
				// Meu inventario
				showInventory()

			case "7":
				// Montar um deck
				montarDeck(writer)

			case "8":
				// Ver meu ping
				req := protocolo.Message{
					Type: "CHECK_LATENCY",
					Data: protocolo.LatencyRequest{},
				}
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

			default:
				fmt.Println("Opção inválida. Tente novamente.")
			}

		} else if currentState == WaitingState {
			fmt.Println("Aguardando um oponente...")
			time.Sleep(100 * time.Millisecond)

		} else if currentState == InGameState {
			// Neste estado, o jogo está ativo, mas não é a vez do jogador.
    		// O loop principal pausa, aguardando que a goroutine 'interpreter' receba uma mensagem
    		// do servidor e altere o 'currentState' (para TurnState ou MenuState, por exemplo).
			time.Sleep(100 * time.Millisecond) // Sleep pra evitar de o for ficar girando freneticamente enquanto espero response do servidor.

		} else if currentState == TurnState {
			handleGameTurn(userInputReader, writer)
		} else {
			// Faz nada no StopState
			time.Sleep(100 * time.Millisecond)
		}
	}
}
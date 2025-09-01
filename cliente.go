package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// Define o estado do jogo
type GameState int

const (
	MenuState GameState = 0
	WaitingState GameState = 1
	InGameState GameState = 2
	LoginState GameState = 3
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
	
	// Cria um canal de comunicação entre as goroutines
	gameChannel := make(chan string)

	go interpreter(reader, gameChannel)

	userInputReader := bufio.NewReader(os.Stdin)
	currentState := MenuState

	for {
		// Verifica se há alguma mensagem no canal para mudar o estado do jogo
		select {
		case msg := <-gameChannel:
			if msg == "PAREADO" {
				currentState = InGameState
				fmt.Println("\nPartida encontrada! Você agora pode enviar mensagens de chat.")
			}
		default:
			// Continua no loop se o canal estiver vazio
		}

		if currentState == LoginState {
			showLoginMenu(userInputReader,writer)
			option, _ := userInputReader.ReadString('\n')
			input = strings.TrimSpace(option)

			switch option{
			case "1":

			case "2":
				fmt.Print("Digite um login: ")
				fmt.Scanln(&login)

				fmt.Print("Agora digite uma senha: ")
				fmt.Scanln(&senha)
			}
		}else if currentState == MenuState {
			showMainMenu(userInputReader, writer)
			input, _ := userInputReader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "1":
				fmt.Println("Buscando sala publica...")
				writer.WriteString("FIND_ROOM:PUBLIC\n")
				writer.Flush()
				// Talvez tenha que colocar uma confirmacao do servidor aqui antes. 
				currentState = WaitingState // Muda o estado para aguardar
			case "2":
				fmt.Printf("Digite o código da sala:\n> ")
				codigoDaSala, _ := userInputReader.ReadString('\n')
				codigoDaSala = strings.TrimSpace(codigoDaSala)
				codigoDaSala = strings.ToUpper(codigoDaSala)
				pack := fmt.Sprintf("PRIV_ROOM:%s\n",codigoDaSala)
				writer.WriteString(pack)
				writer.Flush()
				currentState = WaitingState // Muda o estado para aguardar
			case "3":
				writer.WriteString("CREATE_ROOM:\n")
				writer.Flush()
				currentState = WaitingState // Muda o estado para aguardar
			case "0":
				writer.WriteString("QUIT:\n")
				writer.Flush()
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

func showLoginMenu(reader *bufio.Reader, writer *bufio.Writer) {
	fmt.Println("Bem vindo ao Super Trunfo online!")
	fmt.Println("1. Login")
	fmt.Println("2. Cadastro")
	fmt.Println("> ")
}

func showMainMenu(reader *bufio.Reader, writer *bufio.Writer) {
	fmt.Println("\nEscolha uma opção:")
	fmt.Println("1. Entrar em Sala Pública.")
	fmt.Println("2. Entrar em Sala Privada.")
	fmt.Println("3. Criar sala Privada.")
	fmt.Println("0. Sair")
	fmt.Printf("> ")
}

func showInGameMenu(reader *bufio.Reader, writer *bufio.Writer) {
    fmt.Printf("\nDigite sua Mensagem:\n> ")
    message, _ := reader.ReadString('\n')
    writer.WriteString("CHAT:" + strings.TrimSuffix(message, "\n") + "\n")
    writer.Flush()
}

// O interpreter agora envia uma mensagem para o canal quando a partida inicia
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
		

		parts := strings.SplitN(message, ":", 2)
    	command := strings.TrimSpace(parts[0])
    	content := ""

		if len(parts) > 1 {
        content = strings.TrimSpace(parts[1])
    	}
		
		switch command{
		case "PAREADO":
			gameChannel <- "PAREADO"
			fmt.Println(command)
			fmt.Printf("> ")
		case "CHAT": // Mensagem PlayerToPlayer
			fmt.Println(content)
			fmt.Printf("> ")
		case "SCREEN_MSG": // Mensagem informativa
			fmt.Println(content)
			//fmt.Printf("> ") // Tirar depois
		default:
			// Por enquanto faz nada.
		}
	}
}
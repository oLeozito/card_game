package main

import (
	"fmt"
	"net"
	"time"
	"strings"
	"bufio"
	"io"
	"os"
)

func main() {
	// Tenta se conectar ao servidor.
	// "servidor" é o nome do serviço no Docker Compose, que atua como um hostname.
	var conn net.Conn // conexao
	var err error // erro se houver

	// Fica tentando conectar
	for {
		conn, err = net.Dial("tcp", "servidor:5000")
		if err == nil {
			break // Conectado, sai do loop
		}
		fmt.Println("Aguardando o servidor...")
		time.Sleep(1 * time.Second) // Espera e tenta novamente
	}
	defer conn.Close()
	fmt.Printf("Conectado ao servidor %s\n", conn.RemoteAddr())

	//AQUI
	// Cria o writer e reader para a conexao
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	
	go interpreter(reader)

	// Loop do Menu pra o usuario ja generalizando
	for {
		fmt.Println("\nEscolha uma opção:")
		fmt.Println("1. Entrar em Sala")
		fmt.Println("2. Criar nova sala privada")
		fmt.Println("3. Mensagem Global")
		fmt.Println("0. Sair")
		fmt.Printf("> ")

		// Lê a entrada do usuário do terminal
		userInputReader := bufio.NewReader(os.Stdin)
		input, _ := userInputReader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		
		case "1":
			local_r := bufio.NewReader(os.Stdin)

			fmt.Printf("Quer entrar numa sala privada ou publica?\n1-Publica\n2-Privada\n>")
			opcao, _ := local_r.ReadString('\n')
			opcao = strings.TrimSpace(opcao)

			switch opcao{
			case "1":
				// CHAMA O BAGULHO QUE ENVIA A MENSAGEM PRO SERVIDOR
			case "2":
				// CHAMA O BAGULHO QUE ENVIA A MENSAGEM PRO SERVIDOR
			default:
				fmt.Println("Opção inválida. Tente novamente.")
			}

		case "2":
			fmt.Println("Criando Sala Privada")
		
		case "3":
			fmt.Printf("Digite sua Mensagem:\n> ")
			// Usa o leitor do terminal para ler a mensagem
			message, _ := userInputReader.ReadString('\n')
			message = strings.TrimSuffix(message, "\n")
			
			// O print agora funcionará, pois você leu a entrada do terminal
			fmt.Println("So de teste aqui: ", message)

		case "0":
			// Envia uma mensagem para o servidor informando que o cliente está saindo
			writer.WriteString("QUIT:\n")
			writer.Flush()
			fmt.Println("Saindo do jogo. Desconectando...")
			time.Sleep(1 * time.Second)
			return // Sai da goroutine principal e fecha a conexao
		default:
			fmt.Println("Opção inválida. Tente novamente.")
		}
	}


}

func interpreter(reader *bufio.Reader) {
	for {
		// Tenta ler uma mensagem do servidor
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
		
		message = strings.TrimSpace(message)

		// Logica de interpretar a mensagem do servidor aqui depois !!!!!!!!!!!!!!!!!!!!!
		fmt.Printf("\n[Mensagem do servidor]: %s\n", message)
		
		// Volta a mostrar o prompt para o usuário
		fmt.Printf("> ")
	}
}
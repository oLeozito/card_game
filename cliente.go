package main

import (
	"fmt"
	"net"
	"time"
	"strings"
	"bufio"
)

func main() {
	// Tenta se conectar ao servidor.
	// "servidor" é o nome do serviço no Docker Compose, que atua como um hostname.
	var conn net.Conn
	var err error
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

	// Envia uma mensagem
	message := "FIND_ROOM:\n"
	conn.Write([]byte(message))
	fmt.Println("Mensagem enviada.")
	
	// Depois que o usuario enviar algum comando, ele tem que esperar SEMPRE a resposta do comando que ele enviou.

	// Lê a resposta do servidor
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Erro ao ler resposta:", err)
		return
	}
	response = strings.TrimSpace(response) // remove \n e espaços

	// separa em comando:conteúdo
	parts := strings.SplitN(response, ":", 2)
	command := parts[0]

	fmt.Println("Recebido do servidor:", response)

	if command == "PAREADO" {
	// manda msg inicial
		conn.Write([]byte("CHAT:Eai jogador2 beleza?\n"))

		// fica sempre esperando msgs do outro jogador
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Conexão encerrada ou erro:", err)
				return
			}

			// limpa e mostra
			msg = strings.TrimSpace(msg)
			if msg != "" {
				// remove o prefixo "CHAT:" se existir
				if strings.HasPrefix(msg, "CHAT:") {
					msg = strings.TrimPrefix(msg, "CHAT:")
				}
				fmt.Println("Mensagem recebida do outro jogador:", msg)
			}
		}
	}
}
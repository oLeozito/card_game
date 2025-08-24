package main

import (
	"fmt"
	"net"
	"time"
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
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Erro ao ler resposta:", err)
		return
	}
	response := string(buffer[:n])
	fmt.Println("Recebido do servidor:", response)

	if response == "O jogo esta inciando"{
		conn.Write([]byte("CHAT:Eai mano!\n"))
	}

}
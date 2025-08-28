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
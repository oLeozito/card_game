package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"card_game/protocolo" // Importa o pacote do seu projeto
)

// ============== PARÂMETROS EDITÁVEIS ==============
const (
	// Altere aqui a quantidade de clientes para simular
	numClients = 200

	serverAddress = "127.0.0.1:8080"
)
// =================================================

// simulateLoginClient simula um único cliente que se cadastra, loga e desconecta.
func simulateLoginClient(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", serverAddress, 5*time.Second)
	if err != nil {
		fmt.Printf("[Cliente %d] Erro ao conectar: %v\n", id, err)
		return
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Inicia uma goroutine para consumir mensagens do servidor e evitar bloqueio
	go func() {
		for {
			_, err := reader.ReadString('\n')
			if err != nil {
				return
			}
		}
	}()

	login := "stress_user_" + strconv.Itoa(id)
	senha := "password"

	// 1. Cadastrar
	cadastroReq := protocolo.Message{
		Type: "CADASTRO",
		Data: protocolo.SignInRequest{Login: login, Senha: senha},
	}
	if err := sendJSON(writer, cadastroReq); err != nil {
		fmt.Printf("[Cliente %d] Erro ao enviar cadastro: %v\n", id, err)
		return
	}

	time.Sleep(50 * time.Millisecond) // Pequena pausa

	// 2. Login
	loginReq := protocolo.Message{
		Type: "LOGIN",
		Data: protocolo.LoginRequest{Login: login, Senha: senha},
	}
	if err := sendJSON(writer, loginReq); err != nil {
		fmt.Printf("[Cliente %d] Erro ao enviar login: %v\n", id, err)
		return
	}

	time.Sleep(200 * time.Millisecond) // Simula um tempo online

	// 3. Sair
	quitReq := protocolo.Message{Type: "QUIT"}
	sendJSON(writer, quitReq)

	fmt.Printf("[Cliente %d] Concluído.\n", id)
}

func main() {
	fmt.Printf("Iniciando teste de estresse com %d clientes...\n", numClients)
	startTime := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go simulateLoginClient(i, &wg)
		time.Sleep(10 * time.Millisecond) // Pequeno intervalo para não sobrecarregar o SO de uma vez
	}

	wg.Wait()

	duration := time.Since(startTime)
	fmt.Printf("\nTeste de estresse concluído em %s.\n", duration)
	fmt.Printf("%.2f clientes por segundo.\n", float64(numClients)/duration.Seconds())
}

// sendJSON é uma função auxiliar para este teste
func sendJSON(writer *bufio.Writer, msg protocolo.Message) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	writer.Write(jsonData)
	writer.WriteString("\n")
	return writer.Flush()
}
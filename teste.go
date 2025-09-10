package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"card_game/protocolo"
)

const (
	serverAddr = "127.0.0.1:8080" // Trocar o IP do servidor aqui
	numPlayers = 5500             // Número de jogadores virtuais para teste
)

func main() {
	var wg sync.WaitGroup
	wg.Add(numPlayers)

	for i := 0; i < numPlayers; i++ {
		go func(id int) {
			defer wg.Done()
			simulatePlayer(id)
		}(i + 1)
	}

	wg.Wait()
	fmt.Println("Teste finalizado!")
}

func simulatePlayer(id int) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("[Player %d] Erro ao conectar: %v\n", id, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	playerName := fmt.Sprintf("player%d", id)
	playerPassword := "1234"

	// Cadastro
	cadastroMsg := protocolo.Message{
		Type: "CADASTRO",
		Data: protocolo.SignInRequest{
			Login: playerName,
			Senha: playerPassword,
		},
	}
	sendJSON(writer, cadastroMsg)
	resp := readServerResponse(reader)
	fmt.Printf("[Player %d] Cadastro: %v\n", id, resp.Type)

	// Login
	loginMsg := protocolo.Message{
		Type: "LOGIN",
		Data: protocolo.LoginRequest{
			Login: playerName,
			Senha: playerPassword,
		},
	}
	sendJSON(writer, loginMsg)
	resp = readServerResponse(reader)
	fmt.Printf("[Player %d] Login: %v\n", id, resp.Type)

	// Procurar sala pública
	findRoom := protocolo.Message{
		Type: "FIND_ROOM",
		Data: protocolo.RoomRequest{Mode: "PUBLIC"},
	}
	sendJSON(writer, findRoom)
	resp = readServerResponse(reader)
	fmt.Printf("[Player %d] Procurando sala: %v\n", id, resp.Type)

	// Esperar pareamento
	for {
		msg := readServerResponse(reader)
		if msg.Type == "PAREADO" {
			fmt.Printf("[Player %d] Pareado!\n", id)
			break
		} else if msg.Type == "ERROR" {
			fmt.Printf("[Player %d] Erro: conexão encerrada pelo servidor\n", id)
			return
		}
		time.Sleep(time.Millisecond * 500)
	}

	// Enviar algumas mensagens de chat
	for i := 0; i < 3; i++ {
		content := fmt.Sprintf("Olá do player %d mensagem %d", id, i+1)
		chatMsg := protocolo.Message{
			Type: "CHAT",
			Data: protocolo.ChatMessage{
				From:    playerName,
				Content: content,
			},
		}
		sendJSON(writer, chatMsg)
		fmt.Printf("[Player %d] Enviou mensagem: %s\n", id, content)
		time.Sleep(time.Millisecond * 300)
	}
}

func sendJSON(writer *bufio.Writer, msg protocolo.Message) {
	jsonData, _ := json.Marshal(msg)
	writer.Write(jsonData)
	writer.WriteString("\n")
	writer.Flush()
}

func readServerResponse(reader *bufio.Reader) protocolo.Message {
	msgStr, err := reader.ReadString('\n')
	if err != nil {
		return protocolo.Message{Type: "ERROR", Data: nil}
	}
	var msg protocolo.Message
	json.Unmarshal([]byte(strings.TrimSpace(msgStr)), &msg)
	return msg
}

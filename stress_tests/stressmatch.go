package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"card_game/protocolo"
)

// ============== PARÂMETROS EDITÁVEIS ==============
const (
	// Altere aqui a quantidade de clientes. Use um número PAR para que todos achem uma partida.
	numClients = 50

	serverAddress = "127.0.0.1:8080"
)
// =================================================

// simulateMatchmakingClient simula um cliente completo: cadastra, loga, monta deck, busca partida e joga automaticamente.
func simulateMatchmakingClient(id int, wg *sync.WaitGroup, successCounter *int, mu *sync.Mutex) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", serverAddress, 5*time.Second)
	if err != nil {
		fmt.Printf("[Cliente %d] Erro ao conectar: %v\n", id, err)
		return
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	
	login := "match_user_" + strconv.Itoa(id)
	senha := "password"
	
	// 1. Cadastrar e Logar
	sendJSON(writer, protocolo.Message{Type: "CADASTRO", Data: protocolo.SignInRequest{Login: login, Senha: senha}})
	time.Sleep(50 * time.Millisecond)
	sendJSON(writer, protocolo.Message{Type: "LOGIN", Data: protocolo.LoginRequest{Login: login, Senha: senha}})
	
	// 2. Montar um deck falso
	dummyDeck := []protocolo.Carta{
		{Nome: "Carta1"}, {Nome: "Carta2"}, {Nome: "Carta3"}, {Nome: "Carta4"},
	}
	sendJSON(writer, protocolo.Message{Type: "SET_DECK", Data: protocolo.SetDeckRequest{Cartas: dummyDeck}})
	time.Sleep(50 * time.Millisecond)

	// 3. Buscar sala pública
	sendJSON(writer, protocolo.Message{Type: "FIND_ROOM", Data: protocolo.RoomRequest{Mode: "PUBLIC"}})

	// Loop principal para ler mensagens e reagir
	hand := []protocolo.Carta{}
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[Cliente %d] Desconectado inesperadamente: %v\n", id, err)
			return
		}

		var msg protocolo.Message
		if err := json.Unmarshal([]byte(message), &msg); err != nil {
			continue // Ignora mensagens malformadas
		}
		
		switch msg.Type {
		case "PAREADO":
			fmt.Printf("[Cliente %d] Pareado! Entrando no jogo...\n", id)
		
		case "ROUND_START":
			var data protocolo.RoundStartMessage
			mapToStruct(msg.Data, &data)
			hand = data.Hand
			// Jogada automática: joga sempre a primeira carta com o primeiro atributo
			if len(hand) > 0 {
				move := protocolo.PlayMoveRequest{CardIndex: 0, Attribute: "Envergadura"}
				sendJSON(writer, protocolo.Message{Type: "PLAY_MOVE", Data: move})
			}

		case "GAME_OVER":
			fmt.Printf("[Cliente %d] Jogo concluído. Desconectando.\n", id)
			mu.Lock()
			*successCounter++
			mu.Unlock()
			sendJSON(writer, protocolo.Message{Type: "QUIT"})
			return // Termina a função
		}
	}
}

func main() {
	fmt.Printf("Iniciando teste de matchmaking com %d clientes...\n", numClients)
	startTime := time.Now()
	
	var wg sync.WaitGroup
	var successCounter int
	var mu sync.Mutex

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go simulateMatchmakingClient(i, &wg, &successCounter, &mu)
		time.Sleep(20 * time.Millisecond)
	}
	
	wg.Wait()
	
	duration := time.Since(startTime)
	fmt.Printf("\nTeste de matchmaking concluído em %s.\n", duration)
	fmt.Printf("%d de %d clientes concluíram uma partida com sucesso.\n", successCounter, numClients)
}

// Funções auxiliares para o teste
func sendJSON(writer *bufio.Writer, msg protocolo.Message) error {
	jsonData, err := json.Marshal(msg)
	if err != nil { return err }
	writer.Write(jsonData)
	writer.WriteString("\n")
	return writer.Flush()
}

func mapToStruct(input interface{}, target interface{}) {
	bytes, _ := json.Marshal(input)
	json.Unmarshal(bytes, target)
}
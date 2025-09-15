package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
	"os/signal"
	"syscall"

	"card_game/protocolo"
)

// Declaracoes

type User struct {
	Login      string
	Senha      string
	Conn       net.Conn
	Online     bool
	Inventario Inventario
	Moedas     int
	Latencia   int64 // em milissegundos
	Deck       []protocolo.Carta
}

type Carta struct {
	Nome        string
	Raridade    string
	Envergadura int
	Velocidade  int
	Altura      int
	Passageiros int
}

type Inventario struct {
	Cartas []Carta
}

// Estrutura para armazenar a jogada de um jogador no round atual
type PlayerMove struct {
	CardIndex int
	Attribute string
	Submitted bool
}

// Estrutura para gerenciar o estado de uma partida
type GameState struct {
	Round         int
	Player1Score  int
	Player2Score  int
	Player1Hand   []protocolo.Carta
	Player2Hand   []protocolo.Carta
	Player1Move   PlayerMove
	Player2Move   PlayerMove
	GameMutex     sync.Mutex
}

type Sala struct {
	ID        string
	Jogador1  net.Conn
	Jogador2  net.Conn
	Status    string
	IsPrivate bool
	Game      *GameState // Adicionado para gerenciar o estado do jogo
}

// Variaveis globais
var (
	salas         map[string]*Sala
	salasEmEspera []*Sala
	playersInRoom map[string]*Sala
	players       map[string]*User // Declarei como map porque posso usar futuramente pra verificar se ja esta online.
	cartas        []Carta          // Lista de cartas EXISTENTES (Se quiser adicionar mais é so mexer no JSON na pasta data)
	storage       []Carta          // Armazem onde ficam as cartas a serem "compradas"
	mu            sync.Mutex
)
const playerDataFile = "data/players.json"

// FUNCOES PARA PERSISTENCIA DE DADOS
// loadPlayerData carrega os dados dos jogadores de um arquivo JSON.
func loadPlayerData() {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(playerDataFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Arquivo de jogadores (%s) não encontrado. Um novo será criado ao fechar o servidor.\n", playerDataFile)
			players = make(map[string]*User)
		} else {
			fmt.Printf("Erro ao ler o arquivo de jogadores: %v\n", err)
		}
		return
	}

	if err := json.Unmarshal(data, &players); err != nil {
		fmt.Printf("Erro ao decodificar o JSON dos jogadores: %v\n", err)
		return
	}

	fmt.Printf("%d jogadores carregados do arquivo %s.\n", len(players), playerDataFile)
}
// savePlayerData salva os dados dos jogadores em um arquivo JSON.
func savePlayerData() {
	fmt.Println("\nSalvando dados dos jogadores...")
	mu.Lock()
	defer mu.Unlock()

	// Garante que ninguém seja salvo como "online"
	for _, player := range players {
		player.Online = false
		player.Conn = nil
	}

	data, err := json.MarshalIndent(players, "", "  ") // Usa MarshalIndent para um JSON formatado
	if err != nil {
		fmt.Printf("Erro ao codificar os dados dos jogadores para JSON: %v\n", err)
		return
	}

	if err := os.WriteFile(playerDataFile, data, 0644); err != nil {
		fmt.Printf("Erro ao salvar os dados dos jogadores no arquivo: %v\n", err)
		return
	}

	fmt.Printf("Dados de %d jogadores salvos com sucesso em %s.\n", len(players), playerDataFile)
}

// FUNCOES PRA GERENCIAR CONEXAO INICIAL
func loginUser(conn net.Conn, data protocolo.LoginRequest) {
	mu.Lock()
	defer mu.Unlock()

	player, exists := players[data.Login]

	if !exists {
		// Usuário não existe
		msg := protocolo.Message{
			Type: "LOGIN",
			Data: protocolo.LoginResponse{Status: "N_EXIST"},
		}
		sendJSON(conn, msg)
		return
	}

	if player.Online {
		// Usuário já está logado em outro lugar
		msg := protocolo.Message{
			Type: "LOGIN",
			Data: protocolo.LoginResponse{Status: "ONLINE_JA"},
		}
		sendJSON(conn, msg)
		return
	}

	// Usuário existe e não está online -> loga
	player.Conn = conn
	player.Online = true

	// Converte inventário do servidor para protocolo
	// #################################################
	invProto := protocolo.Inventario{
		Cartas: make([]protocolo.Carta, len(player.Inventario.Cartas)),
	}
	for i, c := range player.Inventario.Cartas {
		invProto.Cartas[i] = protocolo.Carta{
			Nome:        c.Nome,
			Raridade:    c.Raridade,
			Envergadura: c.Envergadura,
			Velocidade:  c.Velocidade,
			Altura:      c.Altura,
			Passageiros: c.Passageiros,
		}
	}
	// #################################################

	// Resposta completa com status + inventário + moedas
	msg := protocolo.Message{
		Type: "LOGIN",
		Data: protocolo.LoginResponse{
			Status:     "LOGADO",
			Inventario: invProto,
			Saldo:      player.Moedas,
		},
	}
	sendJSON(conn, msg)
}
func cadastrarUser(conn net.Conn, data protocolo.SignInRequest) {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := players[data.Login]; exists {
		sendScreenMsg(conn, "Login já existe.")
		return
	}

	players[data.Login] = &User{
		Login:      data.Login,
		Senha:      data.Senha,
		Online:     false,
		Conn:       nil,
		Inventario: Inventario{},
		Moedas:     50, // Player novo comeca com 50 moedas pra conseguir montar ao menos 1 deck
	}

	sendScreenMsg(conn, "Cadastro realizado com sucesso!")
}

// FUNCOES DE MENSAGENS
func sendScreenMsg(conn net.Conn, text string) {
	msg := protocolo.Message{
		Type: "SCREEN_MSG",
		Data: protocolo.ScreenMessage{Content: text},
	}
	sendJSON(conn, msg)
}
func messageRouter(conn net.Conn, msg protocolo.ChatMessage) {
	mu.Lock()
	defer mu.Unlock()
	room, ok := playersInRoom[conn.RemoteAddr().String()]
	if !ok || room.Jogador2 == nil {
		sendScreenMsg(conn, "Aguardando oponente.")
		return
	}

	jsonMsg := protocolo.Message{
		Type: "CHAT",
		Data: msg,
	}
	if conn == room.Jogador1 {
		sendJSON(room.Jogador2, jsonMsg)
	} else if conn == room.Jogador2 {
		sendJSON(room.Jogador1, jsonMsg)
	}
}

// FUNCOES AUXILIARES
func sendJSON(conn net.Conn, msg protocolo.Message) {
	jsonData, _ := json.Marshal(msg)
	conn.Write(jsonData)
	conn.Write([]byte("\n"))
}
func mapToStruct(input interface{}, target interface{}) error {
	bytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
func randomGenerate() string {
	const charset = "ACDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for {
		for i := range b {
			b[i] = charset[seededRand.Intn(len(charset))]
		}
		codigo := string(b)
		if _, ok := salas[codigo]; !ok {
			return codigo
		}
	}
}
func findPlayerByConn(conn net.Conn) *User {
	for _, player := range players {
		if player.Conn == conn {
			return player
		}
	}
	return nil
}
// Funcao pra medir a latencia periodicamente usando um ping-pong
func measureLatency(player *User) {
	if player == nil || player.Conn == nil {
		return
	}

	// Timestamp em nanossegundos
	ts := time.Now().UnixNano()

	pingMsg := protocolo.Message{
		Type: "PING",
		Data: ts,
	}

	sendJSON(player.Conn, pingMsg)
}

// FUNCOES PRO MENU DO PLAYER

// Funcao pra buscar o json com cartas existentes no jogo
func carregarCartas() error {
	path := filepath.Join("data", "cartas.json")
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cartas); err != nil {
		return err
	}

	fmt.Printf("Foram carregadas %d cartas do arquivo JSON.\n", len(cartas))
	return nil
}
// Funcao pra adicionar cartas aleatórias na fila "storage".
func fillCardStorage() {
	if len(cartas) == 0 {
		fmt.Println("Nenhuma carta cadastrada para preencher o storage.")
		return
	}

	idx := rand.Intn(len(cartas))
	cartaEscolhida := cartas[idx]

	// Adiciona carta normalmente ao final da fila
	storage = append(storage, cartaEscolhida)

	//fmt.Println("Carta do topo é: "+ storage[0].Nome) //Print de debug
}
func buyCard(player *User) *Carta {
	carta := storage[0]   // pega a primeira carta
	storage = storage[1:] // remove da fila

	fillCardStorage() // adiciona uma nova carta no storage

	player.Moedas -= 10 // desconta o valor da compra
	return &carta
}
func findRoom(conn net.Conn, mode string, roomCode string) {
	mu.Lock()
	defer mu.Unlock()

	if mode == "PUBLIC" {
		if len(salasEmEspera) > 0 {
			// Pega a primeira da fila e tira ela.
			sala := salasEmEspera[0]
			salasEmEspera = salasEmEspera[1:]

			sala.Jogador2 = conn
			sala.Status = "Em_Jogo"

			// Caminho duplo para chat (Sem uso no momento)
			playersInRoom[sala.Jogador1.RemoteAddr().String()] = sala
			playersInRoom[sala.Jogador2.RemoteAddr().String()] = sala

			sendPairing(sala.Jogador1)
			sendPairing(sala.Jogador2)
			removeSala(sala.ID)
			// Inicia o Jogo
			go startGame(sala)

		} else {
			codigo := randomGenerate()
			novaSala := &Sala{
				Jogador1:  conn,
				ID:        codigo,
				Status:    "Waiting_Player",
				IsPrivate: false,
			}
			salas[codigo] = novaSala
			salasEmEspera = append(salasEmEspera, novaSala)
			playersInRoom[conn.RemoteAddr().String()] = novaSala
		}
	} else if roomCode != "" {
		sala, ok := salas[roomCode]
		if !ok {
			sendScreenMsg(conn, "Código inválido.")
			return
		}
		sala.Jogador2 = conn
		sala.Status = "Em_Jogo"
		playersInRoom[sala.Jogador1.RemoteAddr().String()] = sala
		playersInRoom[sala.Jogador2.RemoteAddr().String()] = sala

		// Pareado
		sendPairing(sala.Jogador1)
		sendPairing(sala.Jogador2)

		// Inicia o Jogo
		go startGame(sala)
	} else {
		sendScreenMsg(conn, "Opção inválida.")
	}
}
func createRoom(conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	codigo := randomGenerate()
	novaSala := &Sala{
		Jogador1:  conn,
		ID:        codigo,
		Status:    "Waiting_Player",
		IsPrivate: true,
	}
	salas[codigo] = novaSala
	playersInRoom[conn.RemoteAddr().String()] = novaSala
	sendScreenMsg(conn, "Código da sala: "+codigo)
}
func removeSala(salaID string) {
	for i, sala := range salasEmEspera {
		if sala.ID == salaID {
			salasEmEspera = append(salasEmEspera[:i], salasEmEspera[i+1:]...)
			return
		}
	}
}
func sendPairing(conn net.Conn) {
	msg := protocolo.Message{
		Type: "PAREADO",
		Data: protocolo.PairingMessage{Status: "PAREADO"},
	}
	sendJSON(conn, msg)
}

// LÓGICA DO JOGO
//#######################################################
func startGame(sala *Sala) {
	mu.Lock()
	p1 := findPlayerByConn(sala.Jogador1)
	p2 := findPlayerByConn(sala.Jogador2)
	mu.Unlock()

	if p1 == nil || p2 == nil {
		// Lógica de erro, um jogador desconectou antes de começar
		return
	}
	
	// Copia os decks para não modificar o deck original do jogador
	deck1 := make([]protocolo.Carta, len(p1.Deck))
	copy(deck1, p1.Deck)
	deck2 := make([]protocolo.Carta, len(p2.Deck))
	copy(deck2, p2.Deck)

	sala.Game = &GameState{
		Round:        1,
		Player1Score: 0,
		Player2Score: 0,
		Player1Hand:  deck1,
		Player2Hand:  deck2,
	}

	// Envia mensagem de início de jogo
	sendJSON(sala.Jogador1, protocolo.Message{Type: "GAME_START", Data: protocolo.GameStartMessage{Opponent: p2.Login}})
	sendJSON(sala.Jogador2, protocolo.Message{Type: "GAME_START", Data: protocolo.GameStartMessage{Opponent: p1.Login}})

	time.Sleep(1 * time.Second) // Pequena pausa
	startRound(sala)
}
func startRound(sala *Sala) {
	game := sala.Game
	game.Player1Move = PlayerMove{Submitted: false}
	game.Player2Move = PlayerMove{Submitted: false}

	// Envia o estado do round para cada jogador
	sendJSON(sala.Jogador1, protocolo.Message{Type: "ROUND_START", Data: protocolo.RoundStartMessage{Round: game.Round, Hand: game.Player1Hand}})
	sendJSON(sala.Jogador2, protocolo.Message{Type: "ROUND_START", Data: protocolo.RoundStartMessage{Round: game.Round, Hand: game.Player2Hand}})
}
func handlePlayMove(conn net.Conn, data interface{}) {
	var req protocolo.PlayMoveRequest
	_ = mapToStruct(data, &req)

	mu.Lock()
	sala, ok := playersInRoom[conn.RemoteAddr().String()]
	mu.Unlock()

	if !ok || sala.Game == nil {
		sendScreenMsg(conn, "Você não está em um jogo ativo.")
		return
	}
	
	sala.Game.GameMutex.Lock()
	defer sala.Game.GameMutex.Unlock()

	move := PlayerMove{CardIndex: req.CardIndex, Attribute: req.Attribute, Submitted: true}

	if conn == sala.Jogador1 {
		sala.Game.Player1Move = move
	} else {
		sala.Game.Player2Move = move
	}

	// Se ambos os jogadores fizeram suas jogadas, processa o round
	if sala.Game.Player1Move.Submitted && sala.Game.Player2Move.Submitted {
		processRound(sala)
	}
}
func getAttributeValue(card protocolo.Carta, attribute string) int {
	switch attribute {
	case "Envergadura":
		return card.Envergadura
	case "Velocidade":
		return card.Velocidade
	case "Altura":
		return card.Altura
	case "Passageiros":
		return card.Passageiros
	default:
		return 0
	}
}
// Retorna 1 se p1 ganha, 2 se p2 ganha, 0 para empate
func compareAttributes(v1, v2 int) int {
	if v1 > v2 {
		return 1
	}
	if v2 > v1 {
		return 2
	}
	return 0
}
func processRound(sala *Sala) {
	game := sala.Game
	
	p1Move := game.Player1Move
	p2Move := game.Player2Move
	
	p1Card := game.Player1Hand[p1Move.CardIndex]
	p2Card := game.Player2Hand[p2Move.CardIndex]
	
	// Atributo escolhido pelo player 1
	p1AttrValueP1Choice := getAttributeValue(p1Card, p1Move.Attribute)
	p2AttrValueP1Choice := getAttributeValue(p2Card, p1Move.Attribute)

	// Atributo escolhido pelo player 2
	p1AttrValueP2Choice := getAttributeValue(p1Card, p2Move.Attribute)
	p2AttrValueP2Choice := getAttributeValue(p2Card, p2Move.Attribute)
	
	// Compara na característica escolhida por P1
	resultP1Choice := compareAttributes(p1AttrValueP1Choice, p2AttrValueP1Choice)
	// Compara na característica escolhida por P2
	resultP2Choice := compareAttributes(p1AttrValueP2Choice, p2AttrValueP2Choice)

	p1RoundPoints := 0
	p2RoundPoints := 0

	// Lógica de pontuação para o Jogador 1
	if resultP1Choice == 1 && resultP2Choice == 1 { // Ganha nas duas
		p1RoundPoints = 3
	} else if (resultP1Choice == 1 && resultP2Choice == 2) || (resultP1Choice == 2 && resultP2Choice == 1) { // Ganha em uma, perde na outra
		p1RoundPoints = 2
	} else if (resultP1Choice == 1 && resultP2Choice == 0) || (resultP1Choice == 0 && resultP2Choice == 1) { // Ganha em uma, empata na outra
		p1RoundPoints = 2
	} else if resultP1Choice == 0 && resultP2Choice == 0 { // Empata nas duas
		p1RoundPoints = 2
	} else if (resultP1Choice == 2 && resultP2Choice == 0) || (resultP1Choice == 0 && resultP2Choice == 2) { // Perde em uma, empata na outra
		p1RoundPoints = 1
	} // Se perde nas duas (resultP1Choice == 2 && resultP2Choice == 2), p1RoundPoints = 0

	// Lógica de pontuação para o Jogador 2 (é o inverso do jogador 1)
	if resultP1Choice == 2 && resultP2Choice == 2 { // Ganha nas duas
		p2RoundPoints = 3
	} else if (resultP1Choice == 2 && resultP2Choice == 1) || (resultP1Choice == 1 && resultP2Choice == 2) {
		p2RoundPoints = 2
	} else if (resultP1Choice == 2 && resultP2Choice == 0) || (resultP1Choice == 0 && resultP2Choice == 2) {
		p2RoundPoints = 2
	} else if resultP1Choice == 0 && resultP2Choice == 0 {
		p2RoundPoints = 2
	} else if (resultP1Choice == 1 && resultP2Choice == 0) || (resultP1Choice == 0 && resultP2Choice == 1) {
		p2RoundPoints = 1
	}

	// Adiciona os pontos de cada um no round
	game.Player1Score += p1RoundPoints
	game.Player2Score += p2RoundPoints

	mu.Lock()
	p1 := findPlayerByConn(sala.Jogador1)
	p2 := findPlayerByConn(sala.Jogador2)
	mu.Unlock()

	// Envia o resultado do round
	resultMsg := protocolo.RoundResultMessage{
		Round: game.Round,
		Player1Move: protocolo.PlayerMoveInfo{
			PlayerName: p1.Login, CardName: p1Card.Nome, Attribute: p1Move.Attribute, AttributeValue: p1AttrValueP1Choice,
		},
		Player2Move: protocolo.PlayerMoveInfo{
			PlayerName: p2.Login, CardName: p2Card.Nome, Attribute: p2Move.Attribute, AttributeValue: p2AttrValueP2Choice,
		},
		RoundPointsP1: p1RoundPoints,
		RoundPointsP2: p2RoundPoints,
		TotalScoreP1:  game.Player1Score,
		TotalScoreP2:  game.Player2Score,
		ResultText:    fmt.Sprintf("Fim do Round %d!", game.Round),
	}
	
	sendJSON(sala.Jogador1, protocolo.Message{Type: "ROUND_RESULT", Data: resultMsg})
	sendJSON(sala.Jogador2, protocolo.Message{Type: "ROUND_RESULT", Data: resultMsg})

	// Remove as cartas usadas das mãos
	// Player 1
	newHand1 := []protocolo.Carta{}
	for i, card := range game.Player1Hand {
		if i != p1Move.CardIndex {
			newHand1 = append(newHand1, card)
		}
	}
	game.Player1Hand = newHand1
	
	// Player 2
	newHand2 := []protocolo.Carta{}
	for i, card := range game.Player2Hand {
		if i != p2Move.CardIndex {
			newHand2 = append(newHand2, card)
		}
	}
	game.Player2Hand = newHand2
	
	// Proximo Round
	game.Round++
	if game.Round > 3 {
		endGame(sala)
	} else {
		time.Sleep(3 * time.Second) // Tempo para os jogadores verem o resultado
		startRound(sala)
	}
}
func endGame(sala *Sala) {
    game := sala.Game
    mu.Lock()
    p1 := findPlayerByConn(sala.Jogador1)
    p2 := findPlayerByConn(sala.Jogador2)
    mu.Unlock()

    // Atribui moedas relativas aos pontos pra os dois jogadores
    p1.Moedas += game.Player1Score
    p2.Moedas += game.Player2Score

    var winner string
    if game.Player1Score > game.Player2Score {
        winner = p1.Login
    } else if game.Player2Score > game.Player1Score {
        winner = p2.Login
    } else {
        winner = "EMPATE"
    }

    // Cria mensagens personalizadas para cada jogador ---

    // Mensagem para o Jogador 1
    gameOverMsgP1 := protocolo.GameOverMessage{
        Winner:       winner,
        FinalScoreP1: game.Player1Score,
        FinalScoreP2: game.Player2Score,
        CoinsEarned:  game.Player1Score, // Informa o ganho individual do P1
    }
    sendJSON(sala.Jogador1, protocolo.Message{Type: "GAME_OVER", Data: gameOverMsgP1})

    // Mensagem para o Jogador 2
    gameOverMsgP2 := protocolo.GameOverMessage{
        Winner:       winner,
        FinalScoreP1: game.Player1Score,
        FinalScoreP2: game.Player2Score,
        CoinsEarned:  game.Player2Score, // Informa o ganho individual do P2
    }
    sendJSON(sala.Jogador2, protocolo.Message{Type: "GAME_OVER", Data: gameOverMsgP2})


    // Limpa a sala
    mu.Lock()
    delete(playersInRoom, sala.Jogador1.RemoteAddr().String())
    delete(playersInRoom, sala.Jogador2.RemoteAddr().String())
    delete(salas, sala.ID)
    mu.Unlock()
}
//#######################################################
// FIM DA LÓGICA DO JOGO

// Funcao que vai ser aberta pra gerenciar cada conexao em uma thread
func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		// Verificacao se o player se desconectou
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Conexão com %s encerrada pelo cliente.\n", conn.RemoteAddr())
			} else {
				fmt.Println("Erro ao ler dados:", err)
			}

			// Logout automático
			mu.Lock()
			player := findPlayerByConn(conn)
			if player != nil {
				player.Online = false
				player.Conn = nil
				fmt.Printf("Usuário %s deslogou automaticamente\n", player.Login)
			}
			mu.Unlock()

			// Posso verificar se o jogador ta em alguma sala, desconectar ele e fazer o outro ganhar
			return
		}
		interpreter(conn, message)
	}
}

// Funcao que recebe as requests interpreta e devolve uma response.
func interpreter(conn net.Conn, fullMessage string) {
	var msg protocolo.Message
	if err := json.Unmarshal([]byte(fullMessage), &msg); err != nil {
		sendScreenMsg(conn, "Mensagem inválida.")
		return
	}

	switch msg.Type {
	case "CADASTRO":
		// Nao implementei permanencia de dados ainda.
		var data protocolo.SignInRequest
		_ = mapToStruct(msg.Data, &data)
		// Consigo pegar data.Login e data.Senha e criar um usuario novo.

		cadastrarUser(conn, data)

	case "LOGIN":
		var data protocolo.LoginRequest
		_ = mapToStruct(msg.Data, &data)

		loginUser(conn, data)

	case "CREATE_ROOM":
		player := findPlayerByConn(conn)
		if len(player.Deck) < 4 {
			sendScreenMsg(conn, "Você precisa montar um deck de 4 cartas primeiro!")
			return
		}
		createRoom(conn)

	case "FIND_ROOM":
		player := findPlayerByConn(conn)
		if len(player.Deck) < 4 {
			sendScreenMsg(conn, "Você precisa montar um deck de 4 cartas primeiro!")
			return
		}
		var data protocolo.RoomRequest
		_ = mapToStruct(msg.Data, &data)
		findRoom(conn, data.Mode, "")

	case "PRIV_ROOM":
		player := findPlayerByConn(conn)
		if len(player.Deck) < 4 {
			sendScreenMsg(conn, "Você precisa montar um deck de 4 cartas primeiro!")
			return
		}
		var data protocolo.RoomRequest
		_ = mapToStruct(msg.Data, &data)
		findRoom(conn, "", data.RoomCode)

	case "CHAT":
		var data protocolo.ChatMessage
		_ = mapToStruct(msg.Data, &data)
		messageRouter(conn, data)

	case "COMPRA":
		mu.Lock()
		defer mu.Unlock()

		player := findPlayerByConn(conn) // encontra o player

		if player == nil {
			sendScreenMsg(conn, "Usuário não encontrado.")
			return
		}

		if len(storage) == 0 {
			resp := protocolo.CompraResponse{
				Status: "EMPTY_STORAGE", // sem carta no storage (isso nao é pra ocorrer nunca)
			}
			sendJSON(conn, protocolo.Message{
				Type: "COMPRA_RESPONSE",
				Data: resp,
			})
			return
		}

		if player.Moedas < 10 {
			resp := protocolo.CompraResponse{
				Status: "NO_BALANCE", // saldo insuficiente
			}
			sendJSON(conn, protocolo.Message{
				Type: "COMPRA_RESPONSE",
				Data: resp,
			})
			return
		}

		// Compra aprovada
		carta := buyCard(player)
		player.Inventario.Cartas = append(player.Inventario.Cartas, *carta)

		// Converte carta e inventário para o tipo protocolo
		// #################################################
		cartaProto := &protocolo.Carta{
			Nome:        carta.Nome,
			Raridade:    carta.Raridade,
			Envergadura: carta.Envergadura,
			Velocidade:  carta.Velocidade,
			Altura:      carta.Altura,
			Passageiros: carta.Passageiros,
		}

		invProto := protocolo.Inventario{
			Cartas: make([]protocolo.Carta, len(player.Inventario.Cartas)),
		}

		for i, c := range player.Inventario.Cartas {
			invProto.Cartas[i] = protocolo.Carta{
				Nome:        c.Nome,
				Raridade:    c.Raridade,
				Envergadura: c.Envergadura,
				Velocidade:  c.Velocidade,
				Altura:      c.Altura,
				Passageiros: c.Passageiros,
			}
		}
		// #################################################

		resp := protocolo.CompraResponse{
			Status:     "COMPRA_APROVADA",
			CartaNova:  cartaProto,
			Inventario: invProto,
		}

		sendJSON(conn, protocolo.Message{
			Type: "COMPRA_RESPONSE",
			Data: resp,
		})

	case "CHECK_BALANCE":
		player := findPlayerByConn(conn)
		if player == nil {
			sendScreenMsg(conn, "Usuário não encontrado.")
			return
		}

		resp := protocolo.BalanceResponse{
			Saldo: player.Moedas,
		}

		sendJSON(conn, protocolo.Message{
			Type: "BALANCE_RESPONSE",
			Data: resp,
		})

	case "CHECK_LATENCY":
		player := findPlayerByConn(conn)
		if player == nil {
			sendScreenMsg(conn, "Usuário não encontrado.")
			return
		}

		resp := protocolo.LatencyResponse{
			Latencia: player.Latencia,
		}

		sendJSON(conn, protocolo.Message{
			Type: "LATENCY_RESPONSE",
			Data: resp,
		})

	case "PONG":
		player := findPlayerByConn(conn)
		if player == nil {
			return
		}

		var ts int64
		_ = mapToStruct(msg.Data, &ts) // timestamp original do PING

		// Latência em milissegundos
		player.Latencia = (time.Now().UnixNano() - ts) / int64(time.Millisecond)

	case "SET_DECK":
		var req protocolo.SetDeckRequest
		_ = mapToStruct(msg.Data, &req)

		player := findPlayerByConn(conn)
		if player == nil {
			sendScreenMsg(conn, "Usuário não encontrado para montar deck.")
			return
		}

		player.Deck = req.Cartas
		sendScreenMsg(conn, "Deck salvo com sucesso!")

	case "PLAY_MOVE":
		handlePlayMove(conn, msg.Data)

	case "QUIT":
		conn.Close()

	default:
		sendScreenMsg(conn, "Comando inválido.")
	}
}

// Arquivo: servidor.go

func main() {
	rand.Seed(time.Now().UnixNano())

	// Carrega os dados dos jogadores ao iniciar
	loadPlayerData()

	// Iniciando maps e listas
	salas = make(map[string]*Sala)
	salasEmEspera = make([]*Sala, 0)
	playersInRoom = make(map[string]*Sala)

	// Chama a funcao pra carregar o Json de cartas cadastradas.
	if err := carregarCartas(); err != nil {
		fmt.Println("Erro ao carregar cartas:", err)
		return
	}
	// Preenche a fila de pacotes de cartas.
	for i := 0; i < 500; i++ {
		fillCardStorage()
	}
	fmt.Println("Armazenamento preenchido com 500 cartas!")

	// LÓGICA DE DESLIGAMENTO GRACIOSO
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs // Espera por um sinal (Ctrl+C)
		savePlayerData()
		os.Exit(0)
	}()
	// -----------------------------------------

	// Funcao pra ficar monitorando o ping de TODOS os players. (altere o tempo do sleep pra aumentar a frequencia de leitura)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			mu.Lock()
			for _, player := range players {
				if player.Online {
					measureLatency(player)
				}
			}
			mu.Unlock()
		}
	}()
	
	// Escuta na porta 8080
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro ao iniciar o servidor:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Servidor iniciado na porta 8080. Pressione Ctrl+C para salvar e fechar.")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar a conexão:", err)
			continue
		}
		go handleConnection(conn)
	}
}
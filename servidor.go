package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"
	"os"
    "path/filepath"
	"card_game/protocolo"

)

type User struct {
    Login     string
    Senha     string
	Conn      net.Conn
    Online    bool
    Inventario Inventario
	Moedas int
	Latencia  int64 // em milissegundos
	Deck []protocolo.Carta
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


type Sala struct {
	ID        string
	Jogador1  net.Conn
	Jogador2  net.Conn
	Status    string
	IsPrivate bool
}

var (
	salas         map[string]*Sala
	salasEmEspera []*Sala
	playersInRoom map[string]*Sala
    players       map[string]*User  // Declarei como map porque posso usar futuramente pra verificar se ja esta online.
	cartas        []Carta			// Lista de cartas EXISTENTES
	storage       []Carta			// Armazem onde ficam as cartas a serem "compradas"
	mu            sync.Mutex
)


func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Conexão com %s encerrada pelo cliente.\n", conn.RemoteAddr())
			} else {
				fmt.Println("Erro ao ler dados:", err)
			}

			// Logout automático
			mu.Lock()
			for _, player := range players {
				if player.Conn == conn {
					player.Online = false
					player.Conn = nil
					fmt.Printf("Usuário %s deslogou automaticamente\n", player.Login)
					break
				}
			}
			mu.Unlock()

			// Posso verificar se o jogador ta em alguma sala, desconectar ele e fazer o outro ganhar
			return
		}
		interpreter(conn, message)
	}
}


func interpreter(conn net.Conn, fullMessage string) {
	var msg protocolo.Message
	if err := json.Unmarshal([]byte(fullMessage), &msg); err != nil {
		sendScreenMsg(conn, "Mensagem inválida.")
		return
	}

	switch msg.Type {
    case "CADASTRO":
        // Cadastro colocar json.
        var data protocolo.SignInRequest
        _ = mapToStruct(msg.Data, &data)
        // Consigo pegar data.Login e data.Senha e criar um usuario novo.

        cadastrarUser(conn,data)

    case "LOGIN":
		var data protocolo.LoginRequest
        _ = mapToStruct(msg.Data, &data)

		loginUser(conn,data)

	case "CREATE_ROOM":
		createRoom(conn)
	case "FIND_ROOM":
		var data protocolo.RoomRequest
		_ = mapToStruct(msg.Data, &data)
		findRoom(conn, data.Mode, "")
	case "PRIV_ROOM":
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
				Status: "EMPTY_STORAGE", // sem carta no storage
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

	case "QUIT":
		conn.Close()
	default:
		sendScreenMsg(conn, "Comando inválido.")
	}
}

func findRoom(conn net.Conn, mode string, roomCode string) {
	mu.Lock()
	defer mu.Unlock()

	if mode == "PUBLIC" {
		if len(salasEmEspera) > 0 {
			sala := salasEmEspera[0]
			salasEmEspera = salasEmEspera[1:]
			sala.Jogador2 = conn
			sala.Status = "Em_Jogo"
			playersInRoom[sala.Jogador1.RemoteAddr().String()] = sala
			playersInRoom[sala.Jogador2.RemoteAddr().String()] = sala
			sendPairing(sala.Jogador1)
			sendPairing(sala.Jogador2)
			removeSala(sala.ID)
		} else {
			codigo := randomGenerate()
			novaSala := &Sala{
				Jogador1: conn,
				ID:       codigo,
				Status:   "Waiting_Player",
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
		sendPairing(sala.Jogador1)
		sendPairing(sala.Jogador2)
	} else {
		sendScreenMsg(conn, "Opção inválida.")
	}
}

func cadastrarUser(conn net.Conn, data protocolo.SignInRequest){

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
		Moedas:		50,
	}

    sendScreenMsg(conn, "Cadastro realizado com sucesso!")
}
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
	invProto := protocolo.Inventario{
		Cartas: make([]protocolo.Carta, len(player.Inventario.Cartas)),
	}
	for i, c := range player.Inventario.Cartas {
		invProto.Cartas[i] = protocolo.Carta{
			Nome:        c.Nome,
			Raridade:   c.Raridade,
			Envergadura: c.Envergadura,
			Velocidade:  c.Velocidade,
			Altura:      c.Altura,
			Passageiros: c.Passageiros,
		}
	}

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

func createRoom(conn net.Conn) {
	mu.Lock()
	defer mu.Unlock()
	codigo := randomGenerate()
	novaSala := &Sala{
		Jogador1: conn,
		ID:       codigo,
		Status:   "Waiting_Player",
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

func sendPairing(conn net.Conn) {
	msg := protocolo.Message{
		Type: "PAREADO",
		Data: protocolo.PairingMessage{Status: "PAREADO"},
	}
	sendJSON(conn, msg)
}

func sendScreenMsg(conn net.Conn, text string) {
	msg := protocolo.Message{
		Type: "SCREEN_MSG",
		Data: protocolo.ScreenMessage{Content: text},
	}
	sendJSON(conn, msg)
}

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

// Funcao pra adicionar cartas aleatórias na fila "storage".

func fillCardStorage() {

    if len(cartas) == 0 {
        fmt.Println("Nenhuma carta cadastrada para preencher o storage.")
        return
    }

    rand.Seed(time.Now().UnixNano())
    idx := rand.Intn(len(cartas))
    cartaEscolhida := cartas[idx]

    // Adiciona carta normalmente ao final da fila
    storage = append(storage, cartaEscolhida)

	//fmt.Println("Carta do topo é: "+ storage[0].Nome) //Print de debug
}

func buyCard(player *User) *Carta {
	carta := storage[0]       // pega a primeira carta
	storage = storage[1:]     // remove da fila

	fillCardStorage()         // adiciona uma nova carta no storage

	player.Moedas -= 10       // desconta o valor da compra
	return &carta
}

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


func findPlayerByConn(conn net.Conn) *User {
    for _, player := range players {
        if player.Conn == conn {
            return player
        }
    }
    return nil
}



func main() {
    players = make(map[string]*User)
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

	// Funcao pra ficar monitorando o ping de TODOS os players.
	go func() {
        for {
            time.Sleep(10 * time.Second)
            mu.Lock()
            for _, player := range players {
                if player.Online {
                    measureLatency(player)
                }
            }
            mu.Unlock()
        }
    }()

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro ao iniciar o servidor:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Servidor iniciado na porta 8080. Esperando jogadores...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar a conexão:", err)
			continue
		}
		go handleConnection(conn)
	}
}
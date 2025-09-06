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
	"card_game/protocolo"

)

type User struct {
    Login     string
    Senha     string
    Online    bool
    Inventario Inventario
}

type Carta struct {
    Nome        string
    Descricao   string
    Envergadura int
    Velocidade  int
    // outros atributos...
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
    players         map[string]*User  // Declarei como map porque posso usar futuramente pra verificar se ja esta online.
	mu            sync.Mutex
)

func main() {
    players = make(map[string]*User)
	salas = make(map[string]*Sala)
	salasEmEspera = make([]*Sala, 0)
	playersInRoom = make(map[string]*Sala)

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
        //Colocar aq o codigi
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
        Login: data.Login,
        Senha: data.Senha,
        Online: false,
        Inventario: Inventario{},
    }
    sendScreenMsg(conn, "Cadastro realizado com sucesso!")
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
	sendJSON(room.Jogador1, jsonMsg)
	sendJSON(room.Jogador2, jsonMsg)
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

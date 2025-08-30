package main

import (
    "fmt"
    "math/rand"
    "net"
    "strings"
    "sync"
    "time"
    "bufio"
    "io"
)

type Sala struct {
    ID       string
    Jogador1 net.Conn
    Jogador2 net.Conn
    Status   string
    isPrivate bool
}

var (
    salas         map[string]*Sala
    salasEmEspera []*Sala
    playersInRoom map[string]*Sala
    mu            sync.Mutex
)

func main() {
    salas = make(map[string]*Sala)
    salasEmEspera = make([]*Sala, 0)
    playersInRoom = make(map[string]*Sala)

    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        fmt.Println("Erro ao iniciar o servidor:", err)
        return
    }
    defer listener.Close()

    fmt.Println("Servidor de sala de bate-papo iniciado na porta 8080. Esperando por jogadores...")

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
        
        fmt.Printf("Recebido de %s: %s", conn.RemoteAddr(), message)
        interpreter(conn, message)
    }
}

func findRoom(conn net.Conn, command string, content string) {
    mu.Lock()
    defer mu.Unlock()

    if command == "FIND_ROOM" {
        if len(salasEmEspera) > 0 {
            sala_em_espera := salasEmEspera[0]
            salasEmEspera = salasEmEspera[1:]
            
            sala_em_espera.Jogador2 = conn
            playersInRoom[sala_em_espera.Jogador1.RemoteAddr().String()] = sala_em_espera
            playersInRoom[sala_em_espera.Jogador2.RemoteAddr().String()] = sala_em_espera

            sala_em_espera.Status = "Em_Jogo"
            sala_em_espera.Jogador1.Write([]byte("PAREADO\n"))
            sala_em_espera.Jogador2.Write([]byte("PAREADO\n"))
            fmt.Println("Sala iniciada")
        } else{
            codigo := randomGenerate()
            novaSala := &Sala{
                Jogador1: conn,
                ID:       codigo,
                Status:   "Waiting_Player",
                isPrivate: false,
            }
            fmt.Println("Criei uma nova sala pública aqui")
            salas[codigo] = novaSala
            salasEmEspera = append(salasEmEspera, novaSala)
            playersInRoom[conn.RemoteAddr().String()] = novaSala
            conn.Write([]byte("SCREEN_MSG:Esperando por um oponente...\n"))
        }
    }else if command == "PRIV_ROOM" {
        salas[content].Jogador2 = conn
        playersInRoom[conn.RemoteAddr().String()] = salas[content]
        
        salas[content].Status = "Em_Jogo"
        // Print de confirmacao
        fmt.Println("Passei aq e entrou na privada.")
        salas[content].Jogador1.Write([]byte("PAREADO:\n"))
        salas[content].Jogador2.Write([]byte("PAREADO:\n"))

        fmt.Println(salas[content].Jogador1.RemoteAddr().String())
        fmt.Println(salas[content].Jogador2.RemoteAddr().String())

	} else {
        conn.Write([]byte("SCREEN_MSG:Opção de sala inválida ou código não encontrado.\n"))
    }
    return
}

func create_room(conn net.Conn) {
    mu.Lock()
    defer mu.Unlock()
    codigo := randomGenerate()

    novaSala := &Sala{
        Jogador1: conn,
        ID:       codigo,
        Status:   "Waiting_Player",
        isPrivate: true,
    }

    salas[codigo] = novaSala
    playersInRoom[conn.RemoteAddr().String()] = novaSala
    
    formattedMessage := fmt.Sprintf("SCREEN_MSG:Código da sala > %s\n", codigo)
    conn.Write([]byte(formattedMessage))
}

func messageRouter(conn net.Conn, message string) {
    mu.Lock()
    defer mu.Unlock()
    
    room_to_chat, ok := playersInRoom[conn.RemoteAddr().String()]
    if !ok || room_to_chat.Jogador2 == nil {
        conn.Write([]byte("SCREEN_MSG:Aguardando oponente para iniciar o chat.\n"))
        return
    }

    formattedMessage := "CHAT:" + message
    if !strings.HasSuffix(formattedMessage, "\n") {
        formattedMessage += "\n"
    }

    if conn.RemoteAddr().String() == room_to_chat.Jogador1.RemoteAddr().String() {
        room_to_chat.Jogador2.Write([]byte(formattedMessage))
    } else if conn.RemoteAddr().String() == room_to_chat.Jogador2.RemoteAddr().String() {
        room_to_chat.Jogador1.Write([]byte(formattedMessage))
    }
}

func interpreter(conn net.Conn, fullMessage string) {
    parts := strings.SplitN(fullMessage, ":", 2)
    command := strings.TrimSpace(parts[0])
    content := ""

    if len(parts) > 1 {
        content = strings.TrimSpace(parts[1])
    }
    
    switch command {
    case "CREATE_ROOM":
        create_room(conn)
    case "FIND_ROOM":
        findRoom(conn, command, content)
    case "PRIV_ROOM":
        findRoom(conn, command, content)
    case "CHAT":
        messageRouter(conn, content)
    case "QUIT":
        // Adicionar lógica para remover o jogador da sala
        conn.Close()
    default:
        conn.Write([]byte("SCREEN_MSG:Comando inválido.\n"))
    }
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
        _, flag := salas[codigo]
        if !flag {
            return codigo
        }
    }
}

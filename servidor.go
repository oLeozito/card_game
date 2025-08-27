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
	ID       string   // Um ID único para a sala
	Jogador1 net.Conn // A conexão do primeiro jogador
	Jogador2 net.Conn // A conexão do segundo jogador
	Status   string   // "esperando_jogador", "em_jogo", "Finalizada"
	// Colocar os outros atributos aqui dps.
}

// Mapas e Mutex declarados como variáveis globais para que possam ser
// Podem ser acessados por todas as goroutines.
var (
	salas         map[string]*Sala
	salasEmEspera []*Sala
	playersInRoom map[string]*Sala
	mu            sync.Mutex
)

func main() {
	salas = make(map[string]*Sala)
	salasEmEspera = make([]*Sala, 0) // Opcional, mas boa prática
	playersInRoom = make(map[string]*Sala)
	// Cria um "listener" TCP na porta 5000.
	// O '0.0.0.0' em Go é representado por uma string vazia ":5000"
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		fmt.Println("Erro ao iniciar o servidor:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Servidor de sala de bate-papo iniciado na porta 5000. Esperando por jogadores...")

	// Loop infinito para aceitar novas conexões
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar a conexão:", err)
			continue
		}

		// Para cada nova conexão, inicia uma goroutine para lidar com ela.
		// Isso permite que o servidor aceite múltiplas conexões ao mesmo tempo.
		go handleConnection(conn)
	}
}

// Aqui vai abrir uma Thread de comunicacao com o usuario e interpretar os comandos enviados.
func handleConnection(conn net.Conn) {
    defer conn.Close()

    // Cria um leitor que envolve a sua conexão.
    reader := bufio.NewReader(conn) 

    // Loop que lê a mensagem até o delimitador que por enquanto e um \n
    for {
        // Usa ReadString para ler até encontrar o caractere de nova linha ('\n').
        message, err := reader.ReadString('\n') 
        
        if err != nil {
            // Verifica se o cliente desconectou (EOF)
            if err == io.EOF { 
                fmt.Printf("Conexão com %s encerrada pelo cliente.\n", conn.RemoteAddr())
            } else {
                fmt.Println("Erro ao ler dados:", err)
            }
            return // Sai da goroutine
        }
		//Print debug
        fmt.Printf("Recebido de %s: %s\n", conn.RemoteAddr(), message)

		interpreter(conn,message)

        // Resposta simples (sem \n, porque o ReadString do cliente vai esperar o \n)
        //conn.Write([]byte("Olá, jogador! Sua mensagem foi recebida."))
    }
}

func findRoom(conn net.Conn) {

	//Mutex que impede dar conflito de dados.
	mu.Lock()
	defer mu.Unlock()

	if len(salasEmEspera) == 0 {
		// Chama a funcao de gerar codigo aleatorio aqui.
		codigo := randomGenerate()
		// Cria uma nova sala, coloca o usuario nela e coloca a nova sala na lista.
		novaSala := &Sala{
			Jogador1: conn,
			ID:       codigo,
			Status:   "Waiting_Player",
		}

		// Adicionando a nova sala no map e na lista de salas.
		fmt.Println("Criei uma nova sala aqui")
		salas[codigo] = novaSala
		salasEmEspera = append(salasEmEspera, novaSala)

	} else {
		// Entra aqui caso tenha sala disponivel pra pareamento.
		// pega a primeira sala em espera.
		sala_em_espera := salasEmEspera[0]

		// Jogador 2 agora esta conectado na sala tambem.
		sala_em_espera.Jogador2 = conn

		playersInRoom[sala_em_espera.Jogador1.RemoteAddr().String()] = sala_em_espera
		playersInRoom[sala_em_espera.Jogador2.RemoteAddr().String()] = sala_em_espera

		sala_em_espera.Status = "Em_Jogo"
		sala_em_espera.Jogador1.Write([]byte("PAREADO:\n"))
		sala_em_espera.Jogador2.Write([]byte("PAREADO:\n"))

		//Print so de debug
		fmt.Println("Sala iniciada")
		// Isso aqui é tipo um pop
		salasEmEspera = salasEmEspera[1:]
	}
	return
}

// Essa funcao vai servir pra encontrar e enviar uma mensagem de um jogador para o outro na mesma sala.
func messageRouter(conn net.Conn, message string) {
	room_to_chat := playersInRoom[conn.RemoteAddr().String()]

	fmt.Println("Entrei na MessageRouter")

	// Prefixa a mensagem com "CHAT:" e garante o \n no final
	formattedMessage := "CHAT:" + message
	if !strings.HasSuffix(formattedMessage, "\n") {
		formattedMessage += "\n"
	}

	// Envia para o outro jogador da sala
	if conn.RemoteAddr().String() == room_to_chat.Jogador1.RemoteAddr().String() {
		room_to_chat.Jogador2.Write([]byte(formattedMessage))
	} else if conn.RemoteAddr().String() == room_to_chat.Jogador2.RemoteAddr().String() {
		room_to_chat.Jogador1.Write([]byte(formattedMessage))
	}
}


// Essa funcao vai ser chamada pra interpretar as mensagens recebidas pelos clientes e decidir o que fazer.
func interpreter(conn net.Conn, fullMessage string) {

	// O padrao de comandos vai ser -  COMANDO:"Complemento\n"
    parts := strings.SplitN(fullMessage, ":", 2)
    command := strings.TrimSpace(parts[0])
    content := ""

    if len(parts) > 1 {
        // Pega a parte da string após o ':' sem remover espaços
        content = parts[1]
    }

	switch command {
	case "FIND_ROOM":
		findRoom(conn)
	case "CHAT":
		messageRouter(conn, content)
	} 
	return
}

// Funcao pra gerar um codigo aleatorio de ID pra a sala, mistura letras e numeros
func randomGenerate() string {
	const charset = "ACDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	// Tem que verificar aqui ainda se o codigo gerado ja nao existe na lista de salas.!!!!!!
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
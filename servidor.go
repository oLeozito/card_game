package main

import (
	"fmt"
	"net"
)

type Sala struct {
    ID         string        // Um ID único para a sala
    Jogador1   net.Conn      // A conexão do primeiro jogador
    Jogador2   net.Conn      // A conexão do segundo jogador
    Status     string        // "esperando_jogador", "em_jogo", "Finalizada"
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
	fmt.Printf("Conectado com %s\n", conn.RemoteAddr())

	// Implementação simples: lê uma mensagem e fecha a conexão
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Erro ao ler dados:", err)
		return
	}
	message := string(buffer[:n])
	fmt.Printf("Recebido de %s: %s\n", conn.RemoteAddr(), message)

	// Resposta simples
	conn.Write([]byte("Olá, jogador! Sua mensagem foi recebida."))
}

func findRoom(conn net.Conn){
	fmt.Printf("Entrei aqui")
	
	//Mutex que impede dar conflito de dados.
	mu.Lock()
    defer mu.Unlock()

	if(len(salasEmEspera) == 0){
		// Cria uma nova sala, coloca o usuario nela e coloca a nova sala na lista.
		novaSala := &Sala{
			Jogador1: conn
			ID: "Um ID qualquer"
			Status: "Waiting_Player"
		}
	}else{
		// Entra aqui caso tenha sala disponivel pra pareamento.
		// pega a primeira sala em espera.
		sala_em_espera := salasEmEspera[0]

		// Isso aqui é tipo um pop
		salasEmEspera = salasEmEspera[1:]
	}

}
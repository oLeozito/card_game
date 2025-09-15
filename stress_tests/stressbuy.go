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
	// Altere o número de clientes que comprarão simultaneamente
	numClients = 500

	// Quantas cartas cada cliente tentará comprar
	comprasPorCliente = 5

	serverAddress = "127.0.0.1:8080"
)
// =================================================

// simulateBuyingClient simula um cliente que loga e tenta comprar várias cartas.
func simulateBuyingClient(id int, wg *sync.WaitGroup, totalCompras *int, mu *sync.Mutex) {
	defer wg.Done()

	conn, err := net.DialTimeout("tcp", serverAddress, 5*time.Second)
	if err != nil {
		fmt.Printf("[Cliente %d] Erro ao conectar: %v\n", id, err)
		return
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Goroutine para descartar mensagens do servidor e não bloquear o reader
	go func() {
		for {
			_, err := reader.ReadString('\n')
			if err != nil {
				return
			}
		}
	}()

	login := "buyer_user_" + strconv.Itoa(id)
	senha := "password"

	// 1. Cadastrar e Logar
	sendJSON(writer, protocolo.Message{Type: "CADASTRO", Data: protocolo.SignInRequest{Login: login, Senha: senha}})
	time.Sleep(50 * time.Millisecond)
	sendJSON(writer, protocolo.Message{Type: "LOGIN", Data: protocolo.LoginRequest{Login: login, Senha: senha}})
	time.Sleep(50 * time.Millisecond) // Espera o login ser processado

	// 2. Tentar comprar cartas repetidamente
	for i := 0; i < comprasPorCliente; i++ {
		compraReq := protocolo.Message{
			Type: "COMPRA",
			Data: protocolo.OpenPackageRequest{},
		}
		if err := sendJSON(writer, compraReq); err != nil {
			fmt.Printf("[Cliente %d] Erro ao enviar requisição de compra: %v\n", id, err)
			break // Sai do loop se houver erro de escrita
		}
		mu.Lock()
		*totalCompras++
		mu.Unlock()
		time.Sleep(time.Duration(50+i*5) * time.Millisecond) // Pequeno delay entre as compras
	}

	fmt.Printf("[Cliente %d] Concluiu suas %d tentativas de compra.\n", id, comprasPorCliente)
	// O cliente simplesmente desconecta ao final
}

func main() {
	fmt.Printf("Iniciando teste de compra com %d clientes, cada um tentando comprar %d cartas...\n", numClients, comprasPorCliente)
	startTime := time.Now()

	var wg sync.WaitGroup
	var totalCompras int
	var mu sync.Mutex

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go simulateBuyingClient(i, &wg, &totalCompras, &mu)
		time.Sleep(15 * time.Millisecond) // Intervalo entre o início de cada cliente
	}

	wg.Wait()

	duration := time.Since(startTime)
	fmt.Printf("\nTeste de compra concluído em %s.\n", duration)
	fmt.Printf("Total de %d requisições de compra enviadas.\n", totalCompras)
	fmt.Printf("Média de %.2f compras por segundo.\n", float64(totalCompras)/duration.Seconds())
}

// Funções auxiliares para o teste
func sendJSON(writer *bufio.Writer, msg protocolo.Message) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	writer.Write(jsonData)
	writer.WriteString("\n")
	return writer.Flush()
}
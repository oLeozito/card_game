# Super Trunfo Multiplayer - Fly Edition

Bem-vindo ao Super Trunfo - Fly Edition, um projeto de jogo de cartas multiplayer inspirado no clássico Super Trunfo focado em aeronaves, desenvolvido em Go. Este sistema implementa uma arquitetura cliente-servidor robusta, focada em concorrência, persistência de dados e uma experiência de jogo em tempo real através do terminal.

## Visão Geral

O projeto consiste em um servidor central que gerencia toda a lógica do jogo e clientes baseados em console que se conectam para jogar. Os jogadores podem se cadastrar, montar decks, desafiar oponentes em salas públicas ou privadas e competir em partidas estratégicas de 3 rodadas.

## ✨ Features Principais

-   **Sistema de Contas:** Cadastro e login de jogadores com persistência de dados. Um novo jogador começa com um saldo inicial de 50 moedas.
-   **Matchmaking:** Salas públicas com fila de espera e salas privadas com códigos de 6 dígitos.
-   **Jogabilidade Estratégica:** Partidas 1v1 com 3 rodadas, onde os jogadores escolhem cartas e atributos para competir.
-   **Sistema de Recompensas:** Pontos ganhos em partidas são convertidos em moedas.
-   **Loja de Cartas:** Os jogadores podem usar moedas para comprar "pacotes" e adquirir novas cartas. Cada carta custa 10 moedas.
-   **Persistência de Dados:** Contas, inventários e saldos são salvos em JSON quando o servidor é encerrado.
-   **Alta Concorrência:** O servidor utiliza goroutines e mutexes para gerenciar múltiplos jogadores e partidas simultaneamente.
-   **Ambiente Containerizado:** Totalmente configurado para execução com Docker e Docker Compose.

---

## 🏛️ Arquitetura e Design (Barema)

Este projeto foi desenvolvido seguindo uma série de requisitos técnicos para garantir robustez e escalabilidade.

### 1. Arquitetura Cliente-Servidor

A lógica é centralizada no servidor (`servidor.go`), que atua como autoridade máxima sobre o estado do jogo. O cliente (`cliente.go`) é uma aplicação de console responsável por enviar as ações do usuário e renderizar as informações recebidas do servidor.

### 2. Comunicação via Sockets TCP/IP

A comunicação é estabelecida através de sockets TCP, garantindo uma conexão confiável e ordenada para a troca de mensagens. O servidor escuta na porta `8080` e gerencia cada cliente em uma goroutine separada.

### 3. API Remota e Encapsulamento em JSON

A interação é definida por uma API de mensagens estruturadas, localizadas em `protocolo/protocolo.go`. Todas as mensagens são encapsuladas no formato **JSON**, o que garante a interoperabilidade e a fácil depuração dos dados transmitidos. O sistema valida as mensagens recebidas e lida com dados malformados para não interromper a execução.

### 4. Tratamento de Concorrência

A concorrência é um aspecto central, gerenciada com **goroutines** para cada cliente e **mutexes (`sync.Mutex`)** para proteger o acesso a dados compartilhados. Mutexes são aplicados em operações críticas para evitar *race conditions*, como:
-   Cadastro de novos usuários (evitando logins duplicados).
-   Login de usuários (prevenindo login duplo).
-   Acesso à fila de matchmaking.
-   Compra de cartas do estoque global.

### 5. Persistência de Dados e *Graceful Shutdown*

Para garantir que os dados dos jogadores não sejam perdidos, o servidor implementa um sistema de persistência. Ao iniciar, ele carrega os perfis dos jogadores do arquivo `data/players.json`. O servidor também implementa um **desligamento gracioso** (*graceful shutdown*): ao receber um sinal de interrupção (`Ctrl+C`), ele captura o sinal, executa a rotina `savePlayerData()` para salvar o estado atual de todos os jogadores no arquivo JSON e só então encerra a execução.

---

## 🕹️ Como Jogar

### Fluxo de Jogo

1.  **Conexão:** Inicie o cliente, que se conectará ao servidor.
2.  **Login/Cadastro:** Crie uma nova conta ou faça login em uma existente.
3.  **Montagem de Deck:** No menu, após adquirir pelo menos 4 cartas, escolha a opção "Montar meu deck" e selecione 4 cartas do seu inventário.
4.  **Matchmaking:**
    -   **Sala Pública:** Entre na fila para ser pareado com o próximo jogador disponível.
    -   **Sala Privada:** Crie uma sala e compartilhe o código de 6 dígitos com um amigo, ou insira um código para entrar em uma sala existente.
5.  **Partida:** Uma vez pareado, a partida de 3 rodadas começa.

### Regras da Partida

-   A partida tem **3 rodadas**.
-   A cada rodada, você escolhe uma das suas cartas do deck e um de seus atributos (Ex: Velocidade, Altura).
-   Seu oponente faz o mesmo.
-   O servidor compara os atributos escolhidos por ambos e distribui pontos de acordo com um fluxo de resultados (vitórias, derrotas ou empates em cada comparação).
-   Ao final das 3 rodadas, os pontos totais são somados para determinar o vencedor.
-   **Todos os jogadores** recebem moedas em quantidade igual aos pontos que fizeram na partida.

---

## 🔧 Configuração e Execução

### Estrutura de Pastas

Para garantir o funcionamento correto, o projeto deve seguir a seguinte estrutura:
```
card_game/
├── cliente.go
├── servidor.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── data/
│   ├── cartas.json
│   └── players.json (será criado automaticamente)
├── protocolo/
│   └── protocolo.go
└── stress_tests/
├── stresslogin.go
├── stressmatch.go
└── stressbuy.go
```
### ❗ Importante: Configuração de IP

Antes de executar, você **precisa** alterar o endereço de IP do servidor nos seguintes arquivos para que a conexão funcione:
-   `cliente.go`: na função `main`, na linha `conn, err = net.Dial("tcp", "127.0.0.1:8080")`.
-   Em todos os arquivos de teste em `stress_tests/`: na constante `serverAddress`.

Substitua `"127.0.0.1:8080"` pelo IP da máquina onde o servidor está rodando e mantenha a porta `8080`.

# 🐳 Execução com Docker

O projeto é totalmente containerizado. Com o Docker e o Docker Compose instalados, basta executar:

```bash
docker-compose up --build
```
Este comando irá construir as imagens e iniciar o contêiner do servidor. Você pode então executar o cliente localmente ou em outro contêiner.

### Execução Local

### Servidor
```bash
go run servidor.go
```
### Cliente (Em outro terminal)
```bash
go run cliente.go
```
---

## 🧪 Testes de Estresse

Para garantir a estabilidade do servidor, foram desenvolvidos três scripts de teste de estresse automáticos, com o auxílio de IA (Google Gemini). Eles simulam cenários de alta concorrência.

-   **`stresslogin.go`:** Testa a capacidade do servidor de lidar com um grande fluxo de conexões, cadastros e logins simultâneos, focando na proteção do mapa de jogadores.
-   **`stressmatch.go`:** Simula o fluxo completo de múltiplos jogadores buscando partidas ao mesmo tempo. Testa a lógica de matchmaking, a criação de múltiplas salas de jogo e o gerenciamento de partidas concorrentes.
-   **`stressbuy.go`:** Foca na operação de compra de cartas, onde múltiplos clientes tentam acessar e modificar o "estoque" global e seus próprios inventários, validando a robustez do mutex nessa operação crítica.

---

## 🚀 Futuras Atualizações

-   **Chat em Partida:** Uma função de chat (`messageRouter`) foi implementada no servidor, mas a capacidade do cliente de enviar mensagens durante uma partida foi despriorizada em favor de uma lógica de jogo mais estável. Isso pode ser implementado em uma futura atualização.
-   **Interface Gráfica:** Migrar o cliente de console para uma interface gráfica (GUI).
-   **Mais Cartas:** O conjunto de cartas pode ser facilmente expandido editando o arquivo `data/cartas.json`.

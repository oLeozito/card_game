# Super Trunfo Multiplayer - Fly Edition

Bem-vindo ao Super Trunfo - Fly Edition, um projeto de jogo de cartas multiplayer inspirado no clássico Super Trunfo focado em aeronaves, desenvolvido em Go. Este sistema implementa uma arquitetura cliente-servidor robusta, focada em concorrência, persistência de dados e uma experiência de jogo em tempo real através do terminal.

## Visão Geral

O projeto consiste em um servidor central que gerencia toda a lógica do jogo e clientes baseados em console que se conectam para jogar. Os jogadores podem se cadastrar, montar decks, desafiar oponentes em salas públicas ou privadas e competir em partidas estratégicas de 3 rodadas.

## ✨ Features Principais

-   **Sistema de Contas:** Cadastro e login de jogadores com persistência de dados. Um novo jogador começa com um saldo inicial de 50 moedas.
-   **Matchmaking:** Salas públicas com fila de espera e salas privadas com códigos de 6 dígitos.
-   **Jogabilidade Estratégica:** Partidas 1v1 com 3 rodadas, onde os jogadores escolhem cartas e atributos para competir.
-   **Sistema de Recompensas:** Pontos ganhos em partidas são convertidos em moedas.
-   **Loja de Cartas:** Os jogadores podem usar moedas para comprar "pacotes" e adquirir novas cartas.
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

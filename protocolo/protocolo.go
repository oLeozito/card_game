package protocolo

// Declaracoes globais (servidor e cliente)
type Carta struct {
	Nome        string `json:"nome"`
	Raridade    string `json:"raridade"`
	Envergadura int    `json:"envergadura"`
	Velocidade  int    `json:"velocidade"`
	Altura      int    `json:"altura"`
	Passageiros int    `json:"passageiros"`
}

type Inventario struct {
	Cartas []Carta `json:"cartas"`
}

// Mensagem genérica que vai pelo socket
type Message struct {
	Type string      `json:"type"` // Tipo de comando (ex: "LOGIN", "CHAT", "FIND_ROOM")
	Data interface{} `json:"data"` // Dados associados ao comando
}

// Estruturas específicas de cada tipo de mensagem

// Login e Cadastro
type LoginRequest struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
}

type LoginResponse struct {
	Status     string     `json:"status"`     // LOGADO, N_EXIST, ONLINE_JA
	Inventario Inventario `json:"inventario"` // inventário inicial
	Saldo      int        `json:"saldo"`      // moedas atuais
}

type SignInRequest struct {
	Login string `json:"login"`
	Senha string `json:"senha"`
}

// Mensagens
type ChatMessage struct {
	From    string `json:"from"`
	Content string `json:"content"`
}

type ScreenMessage struct {
	Content string `json:"content"`
}

// Pareamento e sala
type RoomRequest struct {
	RoomCode string `json:"room_code,omitempty"`
	Mode     string `json:"mode,omitempty"` // "PUBLIC" ou "PRIVATE"
}

type PairingMessage struct {
	Status string `json:"status"` // "PAREADO"
}

// Compra de cartas e inventario
type OpenPackageRequest struct{}

type CompraResponse struct {
	Status     string     `json:"status"` // "OK" ou "FALHA"
	CartaNova  *Carta     `json:"carta_nova,omitempty"`
	Inventario Inventario `json:"inventario,omitempty"`
}

type InventoryResponse struct {
	Inventario Inventario `json:"inventario"`
}

type SetDeckRequest struct {
	Cartas []Carta `json:"cartas"`
}

// Gerenciamento de moedas
type CheckBalance struct{}

type BalanceResponse struct {
	Saldo int `json:"saldo"`
}

// Check de Latencia
type LatencyRequest struct{}

type LatencyResponse struct {
	Latencia int64 `json:"latencia"`
}

// ESTRUTURAS PARA A PARTIDA

type GameStartMessage struct {
	Opponent string `json:"opponent"`
}

type RoundStartMessage struct {
	Round int     `json:"round"`
	Hand  []Carta `json:"hand"`
}

type PlayMoveRequest struct {
	CardIndex int    `json:"card_index"`
	Attribute string `json:"attribute"` // "Envergadura", "Velocidade", "Altura", "Passageiros"
}

// Estrutura para descrever a jogada de um jogador em um round
type PlayerMoveInfo struct {
	PlayerName    string `json:"player_name"`
	CardName      string `json:"card_name"`
	Attribute     string `json:"attribute"`
	AttributeValue int    `json:"attribute_value"`
}

type RoundResultMessage struct {
	Round         int            `json:"round"`
	Player1Move   PlayerMoveInfo `json:"player1_move"`
	Player2Move   PlayerMoveInfo `json:"player2_move"`
	RoundPointsP1 int            `json:"round_points_p1"`
	RoundPointsP2 int            `json:"round_points_p2"`
	TotalScoreP1  int            `json:"total_score_p1"`
	TotalScoreP2  int            `json:"total_score_p2"`
	ResultText    string         `json:"result_text"`
}

type GameOverMessage struct {
	Winner       string `json:"winner"` // Nome do vencedor ou "EMPATE"
	FinalScoreP1 int    `json:"final_score_p1"`
	FinalScoreP2 int    `json:"final_score_p2"`
	CoinsEarned  int    `json:"coins_earned"`
}
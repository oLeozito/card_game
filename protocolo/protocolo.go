package protocolo

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

type LoginRequest struct {
    Login string `json:"login"`
    Senha string `json:"senha"`
}

type SignInRequest struct {
    Login string `json:"login"`
    Senha string `json:"senha"`
}

type ChatMessage struct {
    From    string `json:"from"`
    Content string `json:"content"`
}

type RoomRequest struct {
    RoomCode string `json:"room_code,omitempty"`
    Mode     string `json:"mode,omitempty"` // "PUBLIC" ou "PRIVATE"
}

type ScreenMessage struct {
    Content string `json:"content"`
}

type PairingMessage struct {
    Status string `json:"status"` // exemplo: "PAREADO"
}

type LoginResponse struct {
	Status     string     `json:"status"`     // LOGADO, N_EXIST, ONLINE_JA
	Inventario Inventario `json:"inventario"` // inventário inicial
	Saldo      int        `json:"saldo"`      // moedas atuais
}


type OpenPackageRequest struct{}

type CompraResponse struct {
    Status    string   `json:"status"` // "OK" ou "FALHA"
    CartaNova *Carta   `json:"carta_nova,omitempty"`
    Inventario Inventario `json:"inventario,omitempty"`
}

type InventoryResponse struct {
    Inventario Inventario `json:"inventario"`
}


type CheckBalance struct{}

type BalanceResponse struct {
	Saldo int `json:"saldo"`
}


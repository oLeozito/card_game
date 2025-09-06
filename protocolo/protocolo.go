package protocolo

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

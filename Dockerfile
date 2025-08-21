FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Comando de build que será executado pelo docker-compose.yml
CMD ["go", "build", "-o", "/app/main", "."]
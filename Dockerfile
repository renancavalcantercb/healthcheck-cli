FROM golang:1.21-alpine AS builder

WORKDIR /app

# Instalar dependências necessárias para o SQLite
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o healthcheck cmd/healthcheck/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /root/

COPY --from=builder /app/healthcheck .
COPY test-config.yaml ./config.yaml

CMD ["./healthcheck", "start", "--config", "config.yaml", "--daemon"]



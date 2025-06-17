FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o healthcheck cmd/healthcheck/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/healthcheck .
COPY configs/example.yaml ./config.yaml

CMD ["./healthcheck", "start", "--config", "config.yaml", "--daemon"]



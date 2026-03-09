# ============================================
# Bot Oscar - Backend Go (Multi-stage build)
# ============================================

# Etapa 1: Compilación
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar código fuente completo
COPY . .

# Resolver dependencias y compilar
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /bot-oscar ./cmd/server

# Etapa 2: Imagen final mínima
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bot-oscar .

EXPOSE 8080

CMD ["./bot-oscar"]

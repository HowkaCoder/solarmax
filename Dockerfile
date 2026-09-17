# ---------- сборка ----------
FROM golang:1.22-alpine AS builder
WORKDIR /app

COPY . .
# Внешних зависимостей нет, поэтому ни go mod download, ни сеть при сборке не нужны.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o solarmax ./cmd/api

# ---------- запуск ----------
FROM alpine:3.19
WORKDIR /app

# Нужны для HTTPS-пингов keepalive на внешний адрес.
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/solarmax .

# Данные и картинки лежат в одной папке - её удобно вынести на постоянный диск.
RUN mkdir -p /app/data/media
ENV DATA_DIR=/app/data
VOLUME ["/app/data"]

EXPOSE 8080
CMD ["./solarmax"]

FROM golang:1.22-alpine AS builder
WORKDIR /app

# Копируем всё сразу: go mod tidy должен видеть исходники, чтобы понять
# какие пакеты реально импортируются, и докачать/проверить go.sum под них.
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o solarmax ./cmd/api

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/solarmax .
COPY --from=builder /app/migrations ./migrations

RUN mkdir -p /app/media

EXPOSE 8080
CMD ["./solarmax"]

FROM golang:1.22-alpine AS builder
WORKDIR /app

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

FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /app/message-rpc rpc/message/message.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/message-rpc /app/message-rpc
COPY --from=builder /app/rpc/message/etc /app/etc

EXPOSE 8084

CMD ["./message-rpc", "-f", "etc/message.yaml"]

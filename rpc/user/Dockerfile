FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /app/user-rpc rpc/user/user.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/user-rpc /app/user-rpc
COPY --from=builder /app/rpc/user/etc /app/etc

EXPOSE 8081

CMD ["./user-rpc", "-f", "etc/user.yaml"]

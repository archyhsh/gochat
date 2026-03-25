FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /app/relation-rpc rpc/relation/relation.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/relation-rpc /app/relation-rpc
COPY --from=builder /app/rpc/relation/etc /app/etc

EXPOSE 8083

CMD ["./relation-rpc", "-f", "etc/relation.yaml"]

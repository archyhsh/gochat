FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o /app/group-rpc rpc/group/group.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/group-rpc /app/group-rpc
COPY --from=builder /app/rpc/group/etc /app/etc

EXPOSE 8082

CMD ["./group-rpc", "-f", "etc/group.yaml"]

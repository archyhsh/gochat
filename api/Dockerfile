FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install necessary build tools
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -ldflags="-s -w" -o /app/gateway api/gateway.go

# Final image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gateway /app/gateway
COPY --from=builder /app/api/etc /app/etc
COPY --from=builder /app/web /app/web

EXPOSE 8888

CMD ["./gateway", "-f", "etc/gateway.yaml"]

FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install CGO dependencies (none needed for server)
RUN apk add --no-cache git ca-certificates curl

# Use Chinese Go module mirror for faster download in China
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/secflow-server ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata curl
WORKDIR /app
COPY --from=builder /app/secflow-server .
COPY config/config.example.yaml config.yaml

ENV GIN_MODE=release

EXPOSE 8080

CMD ["./secflow-server", "--config", "config.yaml"]

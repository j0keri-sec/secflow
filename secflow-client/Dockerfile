FROM golang:1.22-alpine AS builder

WORKDIR /build

# CGO is required for go-sqlite3 and gopsutil
RUN apk add --no-cache git ca-certificates gcc musl-dev

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/secflow-client ./cmd/client

# ── Runtime stage ─────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata libgcc libstdc++

WORKDIR /app

# 创建必要的目录
RUN mkdir -p /app/data /app/logs

COPY --from=builder /app/secflow-client .
COPY client.example.yaml client.yaml

# 健康检查端点
HEALTHCHECK --interval=60s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8081/health || exit 1

# 环境变量配置 (可以在运行时覆盖)
ENV SECFLOW_MODE=server
ENV SECFLOW_LOG_LEVEL=info
ENV SECFLOW_PORT=8081
ENV SECFLOW_DB_PATH=/app/data/secflow.db
ENV SECFLOW_LOG_PATH=/app/logs

EXPOSE 8081

CMD ["./secflow-client"]

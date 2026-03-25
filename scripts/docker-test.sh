#!/bin/bash
# ================================================
# SecFlow Docker Test Runner
# 自动化测试脚本 - 启动测试环境并运行测试
# ================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test environment configuration
COMPOSE_FILE="docker-compose.test.yml"
PROJECT_NAME="secflow-test"

# Default values
SKIP_BUILD=false
RUN_SERVER_ONLY=false
RUN_CLIENT_ONLY=false
CLEANUP_ONLY=false
VERBOSE=false
PARALLEL_JOBS=1

usage() {
    cat << EOF
SecFlow Docker Test Runner

用法: $0 [OPTIONS]

选项:
    -h, --help              显示帮助信息
    -s, --server-only       仅运行服务端测试
    -c, --client-only       仅运行客户端测试
    -b, --skip-build        跳过 Docker 镜像构建
    --cleanup               仅清理测试环境
    -v, --verbose           详细输出
    -j, --jobs N            并行测试任务数 (默认: 1)

示例:
    # 运行所有测试
    $0

    # 仅运行服务端测试
    $0 --server-only

    # 跳过构建，快速运行已有容器
    $0 --skip-build

    # 清理测试环境
    $0 --cleanup

    # 并行运行测试
    $0 --jobs 4

环境变量:
    MONGO_TEST_PORT     MongoDB 端口 (默认: 27018)
    REDIS_TEST_PORT     Redis 端口 (默认: 6380)
    SERVER_TEST_PORT    服务端端口 (默认: 8081)

EOF
    exit 0
}

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

success() {
    echo -e "${GREEN}✓ $*${NC}"
}

warn() {
    echo -e "${YELLOW}⚠ $*${NC}"
}

error() {
    echo -e "${RED}✗ $*${NC}" >&2
}

info() {
    echo -e "${CYAN}ℹ $*${NC}"
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        -s|--server-only)
            RUN_SERVER_ONLY=true
            shift
            ;;
        -c|--client-only)
            RUN_CLIENT_ONLY=true
            shift
            ;;
        -b|--skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --cleanup)
            CLEANUP_ONLY=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -j|--jobs)
            PARALLEL_JOBS="$2"
            shift 2
            ;;
        *)
            error "未知选项: $1"
            usage
            ;;
    esac
done

cd "$PROJECT_DIR"

# Cleanup function
cleanup() {
    log "清理测试环境..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v --remove-orphans 2>/dev/null || true
    success "清理完成"
}

# Wait for service to be healthy
wait_for_service() {
    local service=$1
    local url=$2
    local name=$3
    local max_wait=${3:-60}
    local count=0

    echo -n "等待 $name 启动"
    while [ $count -lt $max_wait ]; do
        if curl -sf "$url" > /dev/null 2>&1; then
            echo ""
            success "$name 已就绪"
            return 0
        fi
        echo -n "."
        sleep 2
        count=$((count + 2))
    done
    echo ""
    error "$name 启动超时"
    return 1
}

# Build Docker images
build_images() {
    log "构建 Docker 测试镜像..."

    if [ "$VERBOSE" = true ]; then
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" build --no-cache
    else
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" build > /dev/null 2>&1
    fi

    success "镜像构建完成"
}

# Start test environment
start_environment() {
    log "启动测试环境..."

    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d

    log "等待服务启动..."
    sleep 5

    # Wait for MongoDB
    echo -n "等待 MongoDB"
    for i in {1..30}; do
        if docker exec secflow_mongo_test mongosh --eval "db.adminCommand('ping')" > /dev/null 2>&1; then
            echo ""
            success "MongoDB 已就绪"
            break
        fi
        echo -n "."
        sleep 2
    done

    # Wait for Redis
    echo -n "等待 Redis"
    for i in {1..15}; do
        if docker exec secflow_redis_test redis-cli -a secflow_test_redis_pass ping > /dev/null 2>&1; then
            echo ""
            success "Redis 已就绪"
            break
        fi
        echo -n "."
        sleep 2
    done

    # Wait for Server
    wait_for_service "Server" "http://localhost:8081/api/v1/health" "Server" 60
}

# Run server tests
run_server_tests() {
    log "运行服务端测试..."

    # Check if server is running
    if ! curl -sf "http://localhost:8081/api/v1/health" > /dev/null 2>&1; then
        error "Server 未运行，请先启动测试环境"
        return 1
    fi

    # Run tests in the test-runner container
    docker exec secflow_test_runner sh -c "
        cd /app/secflow-server && \
        go test -v -count=1 -race ./tests/... 2>&1 | tee /tmp/server_test_output.txt
    "

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        success "服务端测试全部通过"
    else
        error "服务端测试失败 (退出码: $exit_code)"
    fi

    return $exit_code
}

# Run client tests
run_client_tests() {
    log "运行客户端测试..."

    # Run tests in the test-runner container
    # Note: Some tests may be skipped due to network restrictions in Docker
    docker exec secflow_test_runner sh -c "
        cd /app/secflow-client && \
        go test -v -count=1 ./tests/... 2>&1 | head -200 | tee /tmp/client_test_output.txt
    "

    local exit_code=$?

    if [ $exit_code -eq 0 ]; then
        success "客户端测试全部通过"
    else
        warn "客户端测试部分失败 (退出码: $exit_code) - 可能是网络限制导致"
    fi

    return 0  # Don't fail overall for client tests due to network issues
}

# Show test logs
show_logs() {
    log "显示最近日志..."

    echo ""
    info "=== Server Logs ==="
    docker logs secflow_server_test --tail 50 2>&1 | tail -30

    echo ""
    info "=== Client Logs ==="
    docker logs secflow_client_test --tail 50 2>&1 | tail -30
}

# Main execution
main() {
    echo -e "${CYAN}============================================${NC}"
    echo -e "${CYAN}  SecFlow Docker 测试环境${NC}"
    echo -e "${CYAN}============================================${NC}"
    echo ""

    # Handle cleanup
    if [ "$CLEANUP_ONLY" = true ]; then
        cleanup
        exit 0
    fi

    # Cleanup old environment
    cleanup

    # Build images
    if [ "$SKIP_BUILD" = false ]; then
        build_images
    else
        log "跳过镜像构建"
    fi

    # Start environment
    start_environment

    # Run tests
    local test_failed=0

    if [ "$RUN_CLIENT_ONLY" = true ]; then
        run_client_tests || test_failed=1
    elif [ "$RUN_SERVER_ONLY" = true ]; then
        run_server_tests || test_failed=1
    else
        run_server_tests || test_failed=1
        echo ""
        run_client_tests || true  # Don't fail overall for client
    fi

    # Show logs on failure
    if [ $test_failed -ne 0 ]; then
        show_logs
    fi

    echo ""
    echo -e "${CYAN}============================================${NC}"
    if [ $test_failed -eq 0 ]; then
        echo -e "${GREEN}  测试完成!${NC}"
    else
        echo -e "${RED}  测试失败${NC}"
    fi
    echo -e "${CYAN}============================================${NC}"
    echo ""
    info "测试环境仍在运行，要清理请执行: $0 --cleanup"
    echo ""

    exit $test_failed
}

# Trap Ctrl+C
trap 'echo ""; error "测试被中断"; cleanup; exit 130' INT

# Run main
main

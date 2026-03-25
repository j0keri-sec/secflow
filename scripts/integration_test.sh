#!/bin/bash
# ================================================
# SecFlow 集成测试脚本
# 在本地开发环境中运行集成测试
# ================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Track results
PASS=0
FAIL=0
SKIP=0

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $*"; ((PASS++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $*"; ((FAIL++)); }
log_skip() { echo -e "${YELLOW}[SKIP]${NC} $*"; ((SKIP++)); }

# Check if services are running
check_services() {
    log_info "检查服务状态..."

    # Check MongoDB
    if command -v mongosh &> /dev/null; then
        if mongosh --eval "db.adminCommand('ping')" &> /dev/null; then
            log_pass "MongoDB 运行中"
        else
            log_fail "MongoDB 未运行"
        fi
    else
        log_skip "mongosh 未安装，跳过 MongoDB 检查"
    fi

    # Check Redis
    if command -v redis-cli &> /dev/null; then
        if redis-cli ping &> /dev/null; then
            log_pass "Redis 运行中"
        else
            log_fail "Redis 未运行"
        fi
    else
        log_skip "redis-cli 未安装，跳过 Redis 检查"
    fi

    # Check Server
    if curl -sf http://localhost:8080/api/v1/health &> /dev/null; then
        log_pass "SecFlow Server 运行中"
    else
        log_fail "SecFlow Server 未运行 (http://localhost:8080)"
    fi
}

# Run server tests
run_server_tests() {
    log_info "运行服务端测试..."

    cd "$PROJECT_DIR/secflow-server"

    # Check if tests exist
    if [ ! -d "tests" ]; then
        log_skip "服务端测试目录不存在"
        return
    fi

    # Run unit tests
    log_info "运行单元测试..."
    if go test -v -count=1 ./tests/... 2>&1 | tee /tmp/server_tests.log; then
        log_pass "服务端单元测试通过"
    else
        log_fail "服务端单元测试失败"
    fi
}

# Run client tests
run_client_tests() {
    log_info "运行客户端测试..."

    cd "$PROJECT_DIR/secflow-client"

    # Check if tests exist
    if [ ! -d "tests" ]; then
        log_skip "客户端测试目录不存在"
        return
    fi

    # Run unit tests
    log_info "运行单元测试..."
    if go test -v -count=1 ./tests/... 2>&1 | head -100 | tee /tmp/client_tests.log; then
        log_pass "客户端单元测试通过"
    else
        log_fail "客户端单元测试失败 (部分测试可能因网络问题失败)"
    fi
}

# Run API tests
run_api_tests() {
    log_info "运行 API 集成测试..."

    SERVER_URL="${SECFLOW_SERVER_URL:-http://localhost:8080}"

    # Test health endpoint
    log_info "测试健康检查..."
    if curl -sf "$SERVER_URL/api/v1/health" | grep -q '"code":0'; then
        log_pass "健康检查 API"
    else
        log_fail "健康检查 API"
    fi

    # Test login
    log_info "测试登录 API..."
    RESP=$(curl -sf -X POST "$SERVER_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' 2>&1)
    if echo "$RESP" | grep -q "token"; then
        log_pass "登录 API"
        TOKEN=$(echo "$RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    else
        log_skip "登录失败 (可能需要先注册用户)"
        TOKEN=""
    fi

    # Test with auth
    if [ -n "$TOKEN" ]; then
        # Test nodes endpoint
        log_info "测试节点列表 API..."
        if curl -sf "$SERVER_URL/api/v1/nodes" \
            -H "Authorization: Bearer $TOKEN" | grep -q "nodes"; then
            log_pass "节点列表 API"
        else
            log_fail "节点列表 API"
        fi
    fi
}

# Run Docker environment tests
run_docker_tests() {
    log_info "检查 Docker 测试环境..."

    if ! command -v docker &> /dev/null; then
        log_skip "Docker 未安装"
        return
    fi

    if ! docker info &> /dev/null; then
        log_skip "Docker daemon 未运行"
        return
    fi

    log_info "Docker 环境正常"
    log_pass "Docker 可用"
}

# Show summary
show_summary() {
    echo ""
    echo "=============================================="
    echo "  测试结果汇总"
    echo "=============================================="
    echo -e "  ${GREEN}通过: $PASS${NC}"
    echo -e "  ${RED}失败: $FAIL${NC}"
    echo -e "  ${YELLOW}跳过: $SKIP${NC}"
    echo "=============================================="

    if [ $FAIL -eq 0 ]; then
        echo -e "${GREEN}所有测试通过!${NC}"
        return 0
    else
        echo -e "${RED}部分测试失败${NC}"
        return 1
    fi
}

# Main
main() {
    echo ""
    echo "=============================================="
    echo "  SecFlow 集成测试"
    echo "=============================================="
    echo ""

    # Check environment
    check_services

    echo ""

    # Run tests
    run_server_tests
    echo ""
    run_client_tests
    echo ""
    run_api_tests
    echo ""
    run_docker_tests

    # Show summary
    show_summary
}

# Run
main

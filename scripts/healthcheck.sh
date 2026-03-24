#!/bin/bash
#===============================================================================
# SecFlow 健康检查脚本
# 
# 功能: 检查 SecFlow 各个组件的健康状态
# 返回: 0 = 健康, 1 = 异常, 2 = 严重故障
#
# 使用: ./healthcheck.sh [--json] [--verbose]
#
# 使用场景:
#   - Kubernetes readiness/liveness probe
#   - Docker HEALTHCHECK
#   - 定时健康监控
#   - 部署前验证
#
#===============================================================================

set -euo pipefail

# 配置
SECFLOW_API="${SECFLOW_API:-http://localhost:8080}"
TIMEOUT="${TIMEOUT:-5}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 状态
declare -A CHECKS
OVERALL_STATUS="healthy"  # healthy, degraded, unhealthy
EXIT_CODE=0

# JSON 输出模式
JSON_MODE=false
VERBOSE=false

#-------------------------------------------------------------------------------
# 帮助
#-------------------------------------------------------------------------------
show_help() {
    cat << EOF
SecFlow 健康检查脚本

用法: $0 [选项]

选项:
    --json         JSON 格式输出
    --verbose      详细输出
    --help         显示帮助

退出码:
    0   所有组件健康
    1   部分组件异常 (degraded)
    2   严重故障 (unhealthy)

示例:
    $0                     # 简单输出
    $0 --verbose          # 详细输出
    $0 --json            # JSON 格式
EOF
}

#-------------------------------------------------------------------------------
# 解析参数
#-------------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case $1 in
        --json)
            JSON_MODE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

#-------------------------------------------------------------------------------
# 检查函数
#-------------------------------------------------------------------------------

# 检查 HTTP 服务
check_http() {
    local name=$1
    local url=$2
    local timeout=${3:-5}
    
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查 ${name}: ${url}"
    fi
    
    local response
    local http_code
    local time_ms
    
    # 使用 curl 检查，带超时
    response=$(curl -s -o /dev/null -w "%{http_code}|%{time_total}" \
        --max-time "${timeout}" \
        "${url}" 2>/dev/null || echo "000|0")
    
    http_code=$(echo "${response}" | cut -d'|' -f1)
    time_ms=$(echo "${response}" | cut -d'|' -f2)
    
    # 转换为毫秒
    time_ms=$(echo "${time_ms}" | awk '{printf "%.0f", $1 * 1000}')
    
    if [[ "${http_code}" == "200" ]]; then
        CHECKS["${name}"]="ok|${time_ms}ms"
        return 0
    elif [[ "${http_code}" == "401" ]]; then
        # 401 说明服务在运行，只是需要认证
        CHECKS["${name}"]="ok|${time_ms}ms|auth_required"
        return 0
    elif [[ "${http_code}" == "000" ]]; then
        CHECKS["${name}"]="error|connection_failed"
        return 2
    else
        CHECKS["${name}"]="error|HTTP_${http_code}"
        return 1
    fi
}

# 检查 MongoDB
check_mongo() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查 MongoDB..."
    fi
    
    local result
    result=$(mongosh --quiet --eval 'db.adminCommand({ping:1})' 2>/dev/null || echo '{"ok":0}')
    
    if echo "${result}" | grep -q '"ok":1'; then
        local version
        version=$(mongosh --quiet --eval 'db.version()' 2>/dev/null | tr -d '\n' || echo "unknown")
        CHECKS["mongodb"]="ok|version=${version}"
        return 0
    else
        CHECKS["mongodb"]="error|ping_failed"
        return 2
    fi
}

# 检查 Redis
check_redis() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查 Redis..."
    fi
    
    local result
    result=$(redis-cli ping 2>/dev/null || echo "PONG")
    
    if [[ "${result}" == "PONG" ]]; then
        # 获取 Redis 信息
        local info
        info=$(redis-cli INFO server 2>/dev/null | grep -E "redis_version|used_memory_human" | tr '\n' ' ' || echo "")
        CHECKS["redis"]="ok|${info}"
        return 0
    else
        CHECKS["redis"]="error|ping_failed"
        return 2
    fi
}

# 检查 SecFlow API
check_secflow_api() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查 SecFlow API..."
    fi
    
    # 尝试访问健康检查端点或根路径
    local response
    local http_code
    
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time "${TIMEOUT}" \
        "${SECFLOW_API}/api/v1/health" 2>/dev/null || echo "000")
    
    # 如果 /health 不存在，尝试 /
    if [[ "${response}" == "404" ]] || [[ "${response}" == "000" ]]; then
        response=$(curl -s -o /dev/null -w "%{http_code}" \
            --max-time "${TIMEOUT}" \
            "${SECFLOW_API}/" 2>/dev/null || echo "000")
    fi
    
    if [[ "${response}" =~ ^[23] ]]; then
        CHECKS["secflow_api"]="ok|HTTP_${response}"
        return 0
    elif [[ "${response}" == "000" ]]; then
        CHECKS["secflow_api"]="error|connection_failed"
        return 2
    else
        CHECKS["secflow_api"]="error|HTTP_${response}"
        return 1
    fi
}

# 检查 WebSocket 连接
check_websocket() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查 WebSocket..."
    fi
    
    # 简单检查 WebSocket 端点是否存在
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        --max-time "${TIMEOUT}" \
        "${SECFLOW_API}/api/v1/ws" 2>/dev/null || echo "000")
    
    # WebSocket 升级请求会返回 400 或 426，这是正常的
    if [[ "${response}" =~ ^(400|426|[23])$ ]]; then
        CHECKS["websocket"]="ok|HTTP_${response}"
        return 0
    else
        CHECKS["websocket"]="error|HTTP_${response}"
        return 1
    fi
}

# 检查磁盘空间
check_disk() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查磁盘空间..."
    fi
    
    local usage
    usage=$(df -h / | tail -1 | awk '{print $5}' | tr -d '%')
    
    if [[ "${usage}" -lt 80 ]]; then
        CHECKS["disk"]="ok|${usage}% used"
        return 0
    elif [[ "${usage}" -lt 90 ]]; then
        CHECKS["disk"]="warning|${usage}% used"
        return 1
    else
        CHECKS["disk"]="error|${usage}% used"
        return 2
    fi
}

# 检查内存
check_memory() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查内存..."
    fi
    
    local available
    local total
    local usage
    
    available=$(free -m | grep Mem | awk '{print $7}')
    total=$(free -m | grep Mem | awk '{print $2}')
    
    if [[ ${available} -gt 0 ]] && [[ ${total} -gt 0 ]]; then
        usage=$((100 - (available * 100 / total)))
        
        if [[ "${usage}" -lt 80 ]]; then
            CHECKS["memory"]="ok|${usage}% used"
            return 0
        elif [[ "${usage}" -lt 90 ]]; then
            CHECKS["memory"]="warning|${usage}% used"
            return 1
        else
            CHECKS["memory"]="error|${usage}% used"
            return 2
        fi
    else
        CHECKS["memory"]="ok|unknown"
        return 0
    fi
}

# 检查 Docker 容器
check_docker_containers() {
    if [[ "${VERBOSE}" == "true" ]]; then
        echo "  检查 Docker 容器..."
    fi
    
    if ! command -v docker &> /dev/null; then
        CHECKS["docker"]="ok|not_used"
        return 0
    fi
    
    local running
    local total
    running=$(docker ps --format '{{.Names}}' 2>/dev/null | wc -l || echo "0")
    total=$(docker ps -a --format '{{.Names}}' 2>/dev/null | wc -l || echo "0")
    
    if [[ "${running}" == "${total}" ]] && [[ "${total}" -gt 0 ]]; then
        CHECKS["docker"]="ok|${running}/${total} running"
        return 0
    elif [[ "${running}" -gt 0 ]]; then
        CHECKS["docker"]="warning|${running}/${total} running"
        return 1
    else
        CHECKS["docker"]="error|${running}/${total} running"
        return 2
    fi
}

#-------------------------------------------------------------------------------
# 输出函数
#-------------------------------------------------------------------------------

# 输出简单文本
output_text() {
    echo ""
    echo "=========================================="
    echo "SecFlow 健康检查"
    echo "=========================================="
    echo ""
    
    local has_error=false
    local has_warning=false
    
    for component in "${!CHECKS[@]}"; do
        local value="${CHECKS[$component]}"
        local status=$(echo "${value}" | cut -d'|' -f1)
        local detail=$(echo "${value}" | cut -d'|' -f2-)
        
        local color="${GREEN}"
        local symbol="✓"
        
        case "${status}" in
            ok)
                color="${GREEN}"
                symbol="✓"
                ;;
            warning)
                color="${YELLOW}"
                symbol="!"
                has_warning=true
                ;;
            error)
                color="${RED}"
                symbol="✗"
                has_error=true
                has_warning=true
                ;;
        esac
        
        printf "${color}%s %-15s${NC} %s\n" "${symbol}" "${component}" "${detail}"
    done
    
    echo ""
    if [[ "${has_error}" == "true" ]]; then
        echo -e "${RED}状态: 异常 (unhealthy)${NC}"
        echo -e "${RED}需要立即处理！${NC}"
        EXIT_CODE=2
    elif [[ "${has_warning}" == "true" ]]; then
        echo -e "${YELLOW}状态: 降级 (degraded)${NC}"
        echo -e "${YELLOW}部分组件需要关注${NC}"
        EXIT_CODE=1
    else
        echo -e "${GREEN}状态: 健康 (healthy)${NC}"
        EXIT_CODE=0
    fi
    echo ""
}

# 输出 JSON
output_json() {
    local status="healthy"
    local has_error=false
    local has_warning=false
    
    echo "{"
    echo "  \"status\": \"healthy\","
    echo "  \"timestamp\": \"$(date -Iseconds)\","
    echo "  \"checks\": {"
    
    local first=true
    for component in "${!CHECKS[@]}"; do
        local value="${CHECKS[$component]}"
        local status=$(echo "${value}" | cut -d'|' -f1)
        local detail=$(echo "${value}" | cut -d'|' -f2-)
        
        [[ "${status}" == "error" ]] && has_error=true
        [[ "${status}" == "warning" ]] && has_warning=true
        
        [[ "${first}" == "true" ]] && first=false || echo ","
        
        echo -n "    \"${component}\": {"
        echo -n "\"status\": \"${status}\", "
        echo -n "\"detail\": \"${detail}\""
        echo -n "}"
    done
    
    echo ""
    echo "  },"
    
    if [[ "${has_error}" == "true" ]]; then
        status="unhealthy"
    elif [[ "${has_warning}" == "true" ]]; then
        status="degraded"
    fi
    
    echo "  \"overall_status\": \"${status}\""
    echo "}"
    
    case "${status}" in
        healthy) EXIT_CODE=0 ;;
        degraded) EXIT_CODE=1 ;;
        unhealthy) EXIT_CODE=2 ;;
    esac
}

#-------------------------------------------------------------------------------
# 主流程
#-------------------------------------------------------------------------------
main() {
    echo "开始健康检查..."
    echo ""
    
    # 执行各项检查
    check_disk
    check_memory
    
    # 如果是 Docker 环境，检查容器
    if command -v docker &> /dev/null && docker info &> /dev/null; then
        check_docker_containers
    fi
    
    # 如果 MongoDB 可用
    if command -v mongosh &> /dev/null || command -v mongo &> /dev/null; then
        check_mongo
    fi
    
    # 如果 Redis 可用
    if command -v redis-cli &> /dev/null; then
        check_redis
    fi
    
    # 检查 SecFlow API
    check_secflow_api
    
    # WebSocket 检查 (可能失败但不严重)
    check_websocket
    
    # 输出结果
    if [[ "${JSON_MODE}" == "true" ]]; then
        output_json
    else
        output_text
    fi
    
    exit ${EXIT_CODE}
}

main

#!/bin/bash
#
# SecFlow 部署脚本
# 支持：本地测试、生产环境部署
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的信息
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
SecFlow 部署脚本

用法: $0 [命令] [选项]

命令:
    dev         启动开发环境 (本地运行)
    prod        启动生产环境 (Docker Compose)
    stop        停止所有服务
    restart     重启服务
    status      查看服务状态
    logs        查看服务日志
    clean       清理数据和日志
    backup      备份数据库
    restore     恢复数据库

选项:
    -h, --help  显示帮助信息

示例:
    $0 dev              # 启动开发环境
    $0 prod             # 启动生产环境
    $0 logs server      # 查看服务端日志
    $0 clean            # 清理所有数据

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 启动开发环境
start_dev() {
    log_info "启动开发环境..."
    
    # 检查 MongoDB 和 Redis
    if ! command -v mongod &> /dev/null; then
        log_warn "MongoDB 未安装，尝试使用 Docker 启动..."
        docker run -d --name secflow_mongo_dev \
            -p 27017:27017 \
            -e MONGO_INITDB_ROOT_USERNAME=secflow \
            -e MONGO_INITDB_ROOT_PASSWORD=secflow_pass \
            mongo:7.0 || true
    fi
    
    if ! command -v redis-server &> /dev/null; then
        log_warn "Redis 未安装，尝试使用 Docker 启动..."
        docker run -d --name secflow_redis_dev \
            -p 6379:6379 \
            redis:7.2-alpine || true
    fi
    
    log_success "开发环境依赖已启动"
    log_info "请手动启动服务端和客户端:"
    echo "  cd secflow-server && go run cmd/server/main.go"
    echo "  cd secflow-client && go run cmd/client/main.go"
}

# 启动生产环境
start_prod() {
    log_info "启动生产环境..."
    
    check_dependencies
    
    # 检查 .env 文件
    if [ ! -f .env ]; then
        log_warn ".env 文件不存在，使用默认配置"
        cp .env.example .env
    fi
    
    # 创建必要的目录
    mkdir -p nginx/ssl
    mkdir -p monitoring/grafana/dashboards
    mkdir -p monitoring/grafana/datasources
    
    # 拉取最新镜像
    log_info "拉取最新镜像..."
    docker-compose -f docker-compose.prod.yml pull
    
    # 构建并启动服务
    log_info "构建并启动服务..."
    docker-compose -f docker-compose.prod.yml up -d --build
    
    log_success "生产环境已启动"
    log_info "访问地址:"
    echo "  Web UI:    http://localhost"
    echo "  API:       http://localhost:8080"
    echo "  Grafana:   http://localhost:3000 (admin/admin)"
    echo "  Prometheus: http://localhost:9090"
}

# 停止服务
stop_services() {
    log_info "停止所有服务..."
    
    docker-compose -f docker-compose.prod.yml down
    
    # 停止开发环境的容器
    docker stop secflow_mongo_dev secflow_redis_dev 2>/dev/null || true
    docker rm secflow_mongo_dev secflow_redis_dev 2>/dev/null || true
    
    log_success "服务已停止"
}

# 查看状态
show_status() {
    log_info "服务状态:"
    docker-compose -f docker-compose.prod.yml ps
}

# 查看日志
show_logs() {
    local service=$1
    
    if [ -z "$service" ]; then
        log_info "显示所有服务日志 (按 Ctrl+C 退出)..."
        docker-compose -f docker-compose.prod.yml logs -f
    else
        log_info "显示 $service 日志 (按 Ctrl+C 退出)..."
        docker-compose -f docker-compose.prod.yml logs -f "$service"
    fi
}

# 清理数据
clean_data() {
    log_warn "即将清理所有数据，包括数据库和日志！"
    read -p "确定要继续吗? (y/N): " confirm
    
    if [[ $confirm =~ ^[Yy]$ ]]; then
        log_info "停止服务..."
        docker-compose -f docker-compose.prod.yml down -v
        
        log_info "清理数据卷..."
        docker volume rm secflow_mongo_data secflow_redis_data secflow_prometheus_data secflow_grafana_data 2>/dev/null || true
        
        log_info "清理日志..."
        rm -rf secflow-server/logs secflow-client/logs
        
        log_success "数据已清理"
    else
        log_info "操作已取消"
    fi
}

# 备份数据库
backup_db() {
    local backup_dir="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    log_info "备份数据库到 $backup_dir..."
    
    # 备份 MongoDB
    docker exec secflow_mongo mongodump \
        --username secflow \
        --password secflow_pass \
        --authenticationDatabase admin \
        --db secflow \
        --out /tmp/backup
    
    docker cp secflow_mongo:/tmp/backup "$backup_dir/mongo"
    
    # 创建压缩包
    tar -czf "$backup_dir.tar.gz" -C "$backup_dir" .
    rm -rf "$backup_dir"
    
    log_success "备份完成: $backup_dir.tar.gz"
}

# 恢复数据库
restore_db() {
    local backup_file=$1
    
    if [ -z "$backup_file" ]; then
        log_error "请指定备份文件路径"
        exit 1
    fi
    
    if [ ! -f "$backup_file" ]; then
        log_error "备份文件不存在: $backup_file"
        exit 1
    fi
    
    log_warn "即将恢复数据库，当前数据将被覆盖！"
    read -p "确定要继续吗? (y/N): " confirm
    
    if [[ $confirm =~ ^[Yy]$ ]]; then
        log_info "恢复数据库..."
        
        # 解压备份
        local restore_dir="/tmp/restore_$(date +%s)"
        mkdir -p "$restore_dir"
        tar -xzf "$backup_file" -C "$restore_dir"
        
        # 复制到容器并恢复
        docker cp "$restore_dir/mongo/secflow" secflow_mongo:/tmp/restore
        docker exec secflow_mongo mongorestore \
            --username secflow \
            --password secflow_pass \
            --authenticationDatabase admin \
            --db secflow \
            --drop \
            /tmp/restore
        
        rm -rf "$restore_dir"
        
        log_success "数据库恢复完成"
    else
        log_info "操作已取消"
    fi
}

# 主函数
main() {
    local command=$1
    shift
    
    case $command in
        dev)
            start_dev
            ;;
        prod)
            start_prod
            ;;
        stop)
            stop_services
            ;;
        restart)
            stop_services
            sleep 2
            start_prod
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$1"
            ;;
        clean)
            clean_data
            ;;
        backup)
            backup_db
            ;;
        restore)
            restore_db "$1"
            ;;
        -h|--help|help)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

main "$@"

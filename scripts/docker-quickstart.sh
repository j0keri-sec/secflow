#!/bin/bash
# ================================================
# SecFlow Docker 快速启动脚本
# ================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "  SecFlow Docker 快速启动"
echo "=========================================="

# 检查 Docker 和 Docker Compose
print_info "检查 Docker 环境..."

if ! command -v docker &> /dev/null; then
    print_error "Docker 未安装，请先安装 Docker"
    exit 1
fi

if ! docker --version &> /dev/null; then
    print_error "Docker 不可用"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    print_error "Docker Compose 未安装"
    exit 1
fi

print_success "Docker 环境检查通过"

# 检查 docker compose 是 v1 还是 v2
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
else
    DOCKER_COMPOSE="docker compose"
fi

# 创建必要的目录
print_info "创建必要目录..."
mkdir -p nginx/ssl
mkdir -p monitoring/grafana/dashboards
mkdir -p monitoring/grafana/datasources

# 检查 .env 文件
if [ ! -f ".env" ]; then
    if [ -f ".env.example" ]; then
        print_warning ".env 文件不存在，从 .env.example 创建..."
        cp .env.example .env
        print_info "请编辑 .env 文件配置密码"
    else
        print_warning ".env 文件不存在，将使用默认配置"
    fi
fi

# 解析命令行参数
ACTION=${1:-start}
ENV=${2:-dev}

case "$ACTION" in
    start|up)
        print_info "启动 SecFlow 服务..."

        if [ "$ENV" = "prod" ]; then
            print_info "使用生产环境配置"
            $DOCKER_COMPOSE -f docker-compose.prod.yml up -d
        else
            print_info "使用开发环境配置"
            $DOCKER_COMPOSE -f docker-compose.yml up -d
        fi

        print_success "服务启动成功!"
        echo ""
        echo "访问地址:"
        echo "  前端界面: http://localhost:3000"
        echo "  API 接口: http://localhost:8080"
        echo "  API 文档: http://localhost:8080/api/v1/health"
        echo ""
        echo "默认账号: admin / admin123"
        echo ""
        print_info "查看日志: $DOCKER_COMPOSE logs -f"
        ;;
    stop|down)
        print_info "停止 SecFlow 服务..."
        $DOCKER_COMPOSE -f docker-compose.yml down
        $DOCKER_COMPOSE -f docker-compose.prod.yml down
        print_success "服务已停止"
        ;;
    restart)
        print_info "重启 SecFlow 服务..."
        $DOCKER_COMPOSE -f docker-compose.yml restart
        print_success "服务已重启"
        ;;
    logs)
        SERVICE=${2:-}
        if [ -n "$SERVICE" ]; then
            $DOCKER_COMPOSE -f docker-compose.yml logs -f "$SERVICE"
        else
            $DOCKER_COMPOSE -f docker-compose.yml logs -f
        fi
        ;;
    status)
        print_info "服务状态:"
        $DOCKER_COMPOSE -f docker-compose.yml ps
        ;;
    clean)
        print_warning "清理所有数据 (包括数据库)..."
        read -p "确定要继续吗? (y/N): " confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            $DOCKER_COMPOSE -f docker-compose.yml down -v
            $DOCKER_COMPOSE -f docker-compose.prod.yml down -v
            print_success "所有数据已清理"
        else
            print_info "取消清理"
        fi
        ;;
    build)
        print_info "重新构建镜像..."
        $DOCKER_COMPOSE -f docker-compose.yml build --no-cache
        print_success "镜像构建完成"
        ;;
    *)
        echo "用法: $0 {start|stop|restart|logs|status|clean|build} [dev|prod]"
        echo ""
        echo "示例:"
        echo "  $0 start          # 启动开发环境"
        echo "  $0 start prod     # 启动生产环境"
        echo "  $0 stop           # 停止所有服务"
        echo "  $0 logs server    # 查看服务端日志"
        echo "  $0 status         # 查看服务状态"
        echo "  $0 clean          # 清理所有数据"
        exit 1
        ;;
esac

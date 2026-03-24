#!/bin/bash
#===============================================================================
# SecFlow 数据恢复脚本
# 
# 功能: 从备份恢复 MongoDB 和 Redis 数据
# 使用: ./restore.sh <backup_file> [options]
#
# 重要: 恢复前请确保已停止相关服务！
#
# 使用示例:
#   ./restore.sh /opt/secflow/backups/mongo/secflow_mongo_20240101_120000.gz
#   ./restore.sh /opt/secflow/backups/redis/secflow_redis_20240101_120000.rdb.gz --redis
#
#===============================================================================

set -euo pipefail

# 配置
BACKUP_DIR="${BACKUP_DIR:-/opt/secflow/backups}"
LOG_FILE="${LOG_FILE:-/var/log/secflow/restore.log}"

# MongoDB 配置
MONGO_HOST="${MONGO_HOST:-localhost}"
MONGO_PORT="${MONGO_PORT:-27017}"
MONGO_DB="${MONGO_DB:-secflow}"
MONGO_USER="${MONGO_USER:-}"
MONGO_PASS="${MONGO_PASS:-}"

# Redis 配置
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASS="${REDIS_PASS:-}"
REDIS_CONF_DIR="${REDIS_CONF_DIR:-/etc/redis}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 日志函数
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "${LOG_FILE}"
}

info()    { log "INFO"    "$*"; }
warn()    { log "WARN"    "$*"; }
error()   { log "ERROR"   "$*"; }
success(){ log "SUCCESS"  "${GREEN}$*${NC}"; }

# 显示帮助
show_help() {
    cat << EOF
SecFlow 数据恢复脚本

用法: $0 <backup_file> [选项]

参数:
    backup_file        备份文件路径 (.gz 文件)

选项:
    --mongo           恢复 MongoDB (默认)
    --redis           恢复 Redis
    --mongo-host      MongoDB 主机 (默认: localhost)
    --mongo-port      MongoDB 端口 (默认: 27017)
    --mongo-db        MongoDB 数据库 (默认: secflow)
    --redis-host      Redis 主机 (默认: localhost)
    --redis-port      Redis 端口 (默认: 6379)
    --dry-run         模拟恢复 (不实际执行)
    --help            显示帮助

示例:
    $0 /opt/secflow/backups/mongo/secflow_mongo_20240101_120000.gz
    $0 /opt/secflow/backups/redis/secflow_redis_20240101_120000.rdb.gz --redis

警告: 恢复操作会覆盖现有数据，请谨慎操作！
EOF
}

# 解析参数
RESTORE_TYPE="mongo"
DRY_RUN=false

if [[ $# -lt 1 ]]; then
    error "缺少备份文件参数"
    show_help
    exit 1
fi

BACKUP_FILE="$1"
shift

while [[ $# -gt 0 ]]; do
    case $1 in
        --mongo)
            RESTORE_TYPE="mongo"
            shift
            ;;
        --redis)
            RESTORE_TYPE="redis"
            shift
            ;;
        --mongo-host)
            MONGO_HOST="$2"
            shift 2
            ;;
        --mongo-port)
            MONGO_PORT="$2"
            shift 2
            ;;
        --mongo-db)
            MONGO_DB="$2"
            shift 2
            ;;
        --redis-host)
            REDIS_HOST="$2"
            shift 2
            ;;
        --redis-port)
            REDIS_PORT="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

#-------------------------------------------------------------------------------
# 恢复 MongoDB
#-------------------------------------------------------------------------------
restore_mongo() {
    local backup_file=$1
    
    info "=========================================="
    warn "开始恢复 MongoDB..."
    info "=========================================="
    warn "备份文件: ${backup_file}"
    warn "目标数据库: ${MONGO_DB}@${MONGO_HOST}:${MONGO_PORT}"
    
    # 确认操作
    if [[ "${DRY_RUN}" == "false" ]]; then
        echo ""
        read -p "此操作将覆盖现有数据，是否继续? (yes/no): " -r
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            info "取消恢复操作"
            exit 0
        fi
    else
        info "[DRY RUN] 模拟恢复..."
    fi
    
    # 检查备份文件
    if [[ ! -f "${backup_file}" ]]; then
        error "备份文件不存在: ${backup_file}"
        return 1
    fi
    
    # 构建 MongoDB URI
    local mongo_uri=""
    if [[ -n "${MONGO_USER}" && -n "${MONGO_PASS}" ]]; then
        mongo_uri="mongodb://${MONGO_USER}:${MONGO_PASS}@${MONGO_HOST}:${MONGO_PORT}/${MONGO_DB}?authSource=admin"
    else
        mongo_uri="mongodb://${MONGO_HOST}:${MONGO_PORT}/${MONGO_DB}"
    fi
    
    # 执行恢复
    if [[ "${DRY_RUN}" == "true" ]]; then
        info "[DRY RUN] mongorestore --uri=\"${mongo_uri}\" --drop --archive=\"${backup_file}\" --gzip"
        return 0
    fi
    
    if command -v mongorestore &> /dev/null; then
        info "使用 mongorestore 恢复..."
        mongorestore \
            --uri="${mongo_uri}" \
            --drop \
            --archive="${backup_file}" \
            --gzip \
            --oplogReplay \
            2>&1 | tee -a "${LOG_FILE}" || {
                error "MongoDB 恢复失败"
                return 1
            }
        success "MongoDB 恢复完成"
    else
        warn "mongorestore 未安装，尝试使用 Docker..."
        if command -v docker &> /dev/null; then
            info "使用 Docker 恢复 MongoDB..."
            docker run --rm \
                --network=host \
                -v "${backup_file}:/backup.gz" \
                mongo:7 \
                mongorestore \
                --uri="${mongo_uri}" \
                --drop \
                --archive=/backup.gz \
                --gzip \
                2>&1 | tee -a "${LOG_FILE}" || {
                    error "MongoDB Docker 恢复失败"
                    return 1
                }
            success "MongoDB 恢复完成 (Docker)"
        else
            error "无法恢复 MongoDB: mongorestore 和 Docker 都不可用"
            return 1
        fi
    fi
    
    return 0
}

#-------------------------------------------------------------------------------
# 恢复 Redis
#-------------------------------------------------------------------------------
restore_redis() {
    local backup_file=$1
    
    info "=========================================="
    warn "开始恢复 Redis..."
    info "=========================================="
    warn "备份文件: ${backup_file}"
    warn "目标 Redis: ${REDIS_HOST}:${REDIS_PORT}"
    
    # 确认操作
    if [[ "${DRY_RUN}" == "false" ]]; then
        echo ""
        read -p "此操作将覆盖现有数据，是否继续? (yes/no): " -r
        if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
            info "取消恢复操作"
            exit 0
        fi
    else
        info "[DRY RUN] 模拟恢复..."
    fi
    
    # 检查备份文件
    if [[ ! -f "${backup_file}" ]]; then
        error "备份文件不存在: ${backup_file}"
        return 1
    fi
    
    # 解压
    local temp_file=""
    if [[ "${backup_file}" == *.gz ]]; then
        temp_file=$(mktemp)
        info "解压文件..."
        if ! gzip -dc "${backup_file}" > "${temp_file}"; then
            error "解压失败"
            rm -f "${temp_file}"
            return 1
        fi
    else
        temp_file="${backup_file}"
    fi
    
    # 获取 Redis 配置文件路径
    local redis_conf="${REDIS_CONF_DIR}/redis.conf"
    local redis_dir="/var/lib/redis"
    
    # 尝试获取 Redis 数据目录
    if command -v redis-cli &> /dev/null; then
        redis_dir=$(redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ${REDIS_PASS:+-a "${REDIS_PASS}"} CONFIG GET dir 2>/dev/null | tail -1 || echo "/var/lib/redis")
    fi
    
    # 停止 Redis (可选)
    info "准备恢复文件..."
    
    # 复制 RDB 文件
    if [[ "${DRY_RUN}" == "true" ]]; then
        info "[DRY RUN] cp ${temp_file} ${redis_dir}/dump.rdb"
        return 0
    fi
    
    if command -v redis-cli &> /dev/null; then
        # 通知 Redis 保存当前数据并停止写入
        info "通知 Redis 保存..."
        redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" ${REDIS_PASS:+-a "${REDIS_PASS}"} SAVE 2>/dev/null || true
        
        # 复制备份文件
        info "复制备份文件到 ${redis_dir}/dump.rdb..."
        if ! cp "${temp_file}" "${redis_dir}/dump.rdb"; then
            error "复制文件失败，请检查权限"
            [[ "${temp_file}" != "${backup_file}" ]] && rm -f "${temp_file}"
            return 1
        fi
        
        # 设置正确的权限
        chmod 644 "${redis_dir}/dump.rdb"
        
        # 重启 Redis (如果使用 systemd)
        if command -v systemctl &> /dev/null; then
            info "重启 Redis 服务..."
            systemctl restart redis 2>/dev/null || systemctl restart redis-server 2>/dev/null || true
        fi
        
        success "Redis 恢复完成"
    else
        warn "redis-cli 未安装，尝试使用 Docker..."
        if command -v docker &> /dev/null; then
            docker run --rm \
                -v "${redis_dir}:/data" \
                redis:latest \
                sh -c "cp /backup/dump.rdb /data/dump.rdb && chmod 644 /data/dump.rdb" \
                2>&1 | tee -a "${LOG_FILE}" || {
                    error "Redis Docker 恢复失败"
                    [[ "${temp_file}" != "${backup_file}" ]] && rm -f "${temp_file}"
                    return 1
                }
            
            # 重启
            if command -v systemctl &> /dev/null; then
                systemctl restart redis 2>/dev/null || true
            fi
            
            success "Redis 恢复完成 (Docker)"
        else
            error "无法恢复 Redis: redis-cli 和 Docker 都不可用"
            [[ "${temp_file}" != "${backup_file}" ]] && rm -f "${temp_file}"
            return 1
        fi
    fi
    
    # 清理临时文件
    [[ "${temp_file}" != "${backup_file}" ]] && rm -f "${temp_file}"
    
    return 0
}

#-------------------------------------------------------------------------------
# 主流程
#-------------------------------------------------------------------------------
main() {
    info "=========================================="
    info "SecFlow 数据恢复"
    info "=========================================="
    
    case "${RESTORE_TYPE}" in
        mongo)
            restore_mongo "${BACKUP_FILE}"
            ;;
        redis)
            restore_redis "${BACKUP_FILE}"
            ;;
    esac
}

main

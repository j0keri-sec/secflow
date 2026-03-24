#!/bin/bash
#===============================================================================
# SecFlow 数据库备份脚本
# 
# 功能: 备份 MongoDB 和 Redis 数据
# 使用: ./backup.sh [options]
#
# 选项:
#   --mongo-only    只备份 MongoDB
#   --redis-only    只备份 Redis
#   --keep N        保留最近 N 份备份 (默认: 7)
#   --help         显示帮助
#
# 使用示例:
#   ./backup.sh                    # 备份所有
#   ./backup.sh --mongo-only      # 只备份 MongoDB
#   ./backup.sh --keep 30        # 备份并保留30天
#
# 定时任务示例 (crontab):
#   0 2 * * * /opt/secflow/scripts/backup.sh --keep 7 >> /var/log/secflow/backup.log 2>&1
#===============================================================================

set -euo pipefail

# 配置
BACKUP_DIR="${BACKUP_DIR:-/opt/secflow/backups}"
DATE=$(date +%Y%m%d_%H%M%S)
KEEP_DAYS="${KEEP_DAYS:-7}"
LOG_FILE="${LOG_FILE:-/var/log/secflow/backup.log}"

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

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log() {
    local level=$1
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "${LOG_FILE}"
}

info()  { log "INFO"  "$*"; }
warn()  { log "WARN"  "$*"; }
error() { log "ERROR" "$*"; }
success(){ log "INFO"  "${GREEN}$*${NC}"; }

# 显示帮助
show_help() {
    cat << EOF
SecFlow 数据库备份脚本

用法: $0 [选项]

选项:
    --mongo-only    只备份 MongoDB
    --redis-only    只备份 Redis
    --keep N        保留最近 N 份备份 (默认: 7)
    --help          显示此帮助

环境变量:
    BACKUP_DIR      备份存储目录 (默认: /opt/secflow/backups)
    MONGO_HOST      MongoDB 主机 (默认: localhost)
    MONGO_PORT      MongoDB 端口 (默认: 27017)
    MONGO_DB        MongoDB 数据库名 (默认: secflow)
    MONGO_USER      MongoDB 用户名 (可选)
    MONGO_PASS      MongoDB 密码 (可选)
    REDIS_HOST      Redis 主机 (默认: localhost)
    REDIS_PORT      Redis 端口 (默认: 6379)
    REDIS_PASS      Redis 密码 (可选)
    LOG_FILE        日志文件路径

示例:
    $0                      # 备份所有数据库
    $0 --mongo-only         # 只备份 MongoDB
    $0 --keep 30           # 保留30天备份
EOF
}

# 解析参数
BACKUP_MONGO=true
BACKUP_REDIS=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --mongo-only)
            BACKUP_MONGO=true
            BACKUP_REDIS=false
            shift
            ;;
        --redis-only)
            BACKUP_MONGO=false
            BACKUP_REDIS=true
            shift
            ;;
        --keep)
            KEEP_DAYS="$2"
            shift 2
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

# 创建备份目录
mkdir -p "${BACKUP_DIR}"/{mongo,redis}

#-------------------------------------------------------------------------------
# 备份 MongoDB
#-------------------------------------------------------------------------------
backup_mongo() {
    info "开始备份 MongoDB..."
    
    local backup_file="${BACKUP_DIR}/mongo/secflow_mongo_${DATE}.gz"
    local mongo_uri=""
    
    # 构建 MongoDB URI
    if [[ -n "${MONGO_USER}" && -n "${MONGO_PASS}" ]]; then
        mongo_uri="mongodb://${MONGO_USER}:${MONGO_PASS}@${MONGO_HOST}:${MONGO_PORT}/${MONGO_DB}?authSource=admin"
    else
        mongo_uri="mongodb://${MONGO_HOST}:${MONGO_PORT}/${MONGO_DB}"
    fi
    
    # 执行 mongodump
    if command -v mongodump &> /dev/null; then
        mongodump \
            --uri="${mongo_uri}" \
            --archive="${backup_file}" \
            --gzip \
            --oplog \
            2>&1 | tee -a "${LOG_FILE}" || {
                error "MongoDB 备份失败"
                return 1
            }
        
        # 验证备份文件
        if [[ -f "${backup_file}" && -s "${backup_file}" ]]; then
            local size=$(du -h "${backup_file}" | cut -f1)
            success "MongoDB 备份完成: ${backup_file} (${size})"
        else
            error "MongoDB 备份文件无效"
            return 1
        fi
    else
        warn "mongodump 未安装，跳过 MongoDB 备份"
        # 备选方案: 使用 docker
        if command -v docker &> /dev/null; then
            info "使用 Docker 备份 MongoDB..."
            docker run --rm \
                -v "${BACKUP_DIR}/mongo:/backup" \
                mongo:7 \
                mongodump \
                --uri="${mongo_uri}" \
                --archive=/backup/secflow_mongo_${DATE} \
                --gzip \
                2>&1 | tee -a "${LOG_FILE}" || {
                    error "MongoDB Docker 备份失败"
                    return 1
                }
            success "MongoDB 备份完成 (Docker)"
        else
            error "无法备份 MongoDB: mongodump 和 Docker 都不可用"
            return 1
        fi
    fi
    
    return 0
}

#-------------------------------------------------------------------------------
# 备份 Redis
#-------------------------------------------------------------------------------
backup_redis() {
    info "开始备份 Redis..."
    
    local backup_file="${BACKUP_DIR}/redis/secflow_redis_${DATE}.rdb"
    
    # Redis 配置获取
    local redis_cli_opts=""
    if [[ -n "${REDIS_PASS}" ]]; then
        redis_cli_opts="-a ${REDIS_PASS}"
    fi
    
    # 使用 BGSAVE 触发后台备份
    if command -v redis-cli &> /dev/null; then
        # 检查 Redis 是否可用
        if redis-cli ${redis_cli_opts} -h "${REDIS_HOST}" -p "${REDIS_PORT}" ping 2>/dev/null | grep -q PONG; then
            # 触发 BGSAVE
            info "触发 Redis BGSAVE..."
            redis-cli ${redis_cli_opts} -h "${REDIS_HOST}" -p "${REDIS_PORT}" BGSAVE 2>/dev/null || true
            
            # 等待备份完成
            local max_wait=60
            local waited=0
            while [[ $waited -lt $max_wait ]]; do
                local saving=$(redis-cli ${redis_cli_opts} -h "${REDIS_HOST}" -p "${REDIS_PORT}" LASTSAVE 2>/dev/null)
                sleep 1
                ((waited++))
                
                # 检查是否还在保存
                local bgsave_status=$(redis-cli ${redis_cli_opts} -h "${REDIS_HOST}" -p "${REDIS_PORT}" INFO replication 2>/dev/null | grep "rdb_bgsave_in_progress" | cut -d: -f2 | tr -d '\r')
                if [[ "${bgsave_status}" == "0" ]]; then
                    break
                fi
            done
            
            # 获取 RDB 文件位置
            local redis_conf=$(redis-cli ${redis_cli_opts} -h "${REDIS_HOST}" -p "${REDIS_PORT}" CONFIG GET dir 2>/dev/null | tail -1)
            local redis_rdb="${redis_conf}/dump.rdb"
            
            if [[ -f "${redis_rdb}" ]]; then
                # 复制到备份目录
                cp "${redis_rdb}" "${backup_file}"
                
                # 压缩
                gzip "${backup_file}"
                backup_file="${backup_file}.gz"
                
                local size=$(du -h "${backup_file}" | cut -f1)
                success "Redis 备份完成: ${backup_file} (${size})"
            else
                error "Redis RDB 文件未找到: ${redis_rdb}"
                return 1
            fi
        else
            warn "Redis 不可用，跳过备份"
            return 1
        fi
    else
        warn "redis-cli 未安装，尝试使用 Docker..."
        if command -v docker &> /dev/null; then
            docker run --rm \
                -v "${BACKUP_DIR}/redis:/backup" \
                redis:latest \
                redis-cli \
                -h "${REDIS_HOST}" \
                -p "${REDIS_PORT}" \
                ${redis_cli_opts} \
                --rdb /backup/secflow_redis_${DATE}.rdb 2>&1 | tee -a "${LOG_FILE}" || {
                    error "Redis Docker 备份失败"
                    return 1
                }
            
            # 压缩
            gzip "${BACKUP_DIR}/redis/secflow_redis_${DATE}.rdb"
            backup_file="${BACKUP_DIR}/redis/secflow_redis_${DATE}.rdb.gz"
            success "Redis 备份完成 (Docker): ${backup_file}"
        else
            error "无法备份 Redis: redis-cli 和 Docker 都不可用"
            return 1
        fi
    fi
    
    return 0
}

#-------------------------------------------------------------------------------
# 清理旧备份
#-------------------------------------------------------------------------------
cleanup_old_backups() {
    info "清理 ${KEEP_DAYS} 天前的旧备份..."
    
    local deleted_mongo=0
    local deleted_redis=0
    
    # 清理 MongoDB 备份
    if [[ -d "${BACKUP_DIR}/mongo" ]]; then
        while IFS= read -r -d '' file; do
            rm -f "$file"
            ((deleted_mongo++))
        done < <(find "${BACKUP_DIR}/mongo" -name "*.gz" -mtime +${KEEP_DAYS} -print0 2>/dev/null)
    fi
    
    # 清理 Redis 备份
    if [[ -d "${BACKUP_DIR}/redis" ]]; then
        while IFS= read -r -d '' file; do
            rm -f "$file"
            ((deleted_redis++))
        done < <(find "${BACKUP_DIR}/redis" -name "*.gz" -name "*.rdb*" -mtime +${KEEP_DAYS} -print0 2>/dev/null)
    fi
    
    if [[ $deleted_mongo -gt 0 ]] || [[ $deleted_redis -gt 0 ]]; then
        success "清理完成: MongoDB ${deleted_mongo} 个, Redis ${deleted_redis} 个"
    else
        info "没有需要清理的旧备份"
    fi
}

#-------------------------------------------------------------------------------
# 验证备份
#-------------------------------------------------------------------------------
verify_backup() {
    local backup_file=$1
    local type=$2
    
    if [[ ! -f "${backup_file}" ]]; then
        error "${type} 备份文件不存在: ${backup_file}"
        return 1
    fi
    
    if [[ ! -s "${backup_file}" ]]; then
        error "${type} 备份文件为空: ${backup_file}"
        return 1
    fi
    
    # 对于 gzip 文件，验证格式
    if [[ "${backup_file}" == *.gz ]]; then
        if ! gzip -t "${backup_file}" 2>/dev/null; then
            error "${type} 备份文件损坏: ${backup_file}"
            return 1
        fi
    fi
    
    info "${type} 备份验证通过: ${backup_file}"
    return 0
}

#-------------------------------------------------------------------------------
# 列出备份
#-------------------------------------------------------------------------------
list_backups() {
    info "当前备份列表:"
    
    echo ""
    echo "MongoDB 备份:"
    if [[ -d "${BACKUP_DIR}/mongo" ]] && [[ -n "$(ls -A ${BACKUP_DIR}/mongo 2>/dev/null)" ]]; then
        ls -lh "${BACKUP_DIR}/mongo/" | tail -n +2 | awk '{print "  " $9 " (" $5 ")"}'
    else
        echo "  (无)"
    fi
    
    echo ""
    echo "Redis 备份:"
    if [[ -d "${BACKUP_DIR}/redis" ]] && [[ -n "$(ls -A ${BACKUP_DIR}/redis 2>/dev/null)" ]]; then
        ls -lh "${BACKUP_DIR}/redis/" | tail -n +2 | awk '{print "  " $9 " (" $5 ")"}'
    else
        echo "  (无)"
    fi
}

#-------------------------------------------------------------------------------
# 主流程
#-------------------------------------------------------------------------------
main() {
    info "=========================================="
    info "SecFlow 数据库备份开始"
    info "=========================================="
    info "备份目录: ${BACKUP_DIR}"
    info "保留天数: ${KEEP_DAYS}"
    
    # 显示开始时间
    local start_time=$(date +%s)
    
    # 执行备份
    local mongo_success=false
    local redis_success=false
    
    if $BACKUP_MONGO; then
        if backup_mongo; then
            mongo_success=true
        fi
    fi
    
    if $BACKUP_REDIS; then
        if backup_redis; then
            redis_success=true
        fi
    fi
    
    # 清理旧备份
    cleanup_old_backups
    
    # 计算耗时
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # 总结
    echo ""
    info "=========================================="
    info "备份完成!"
    info "耗时: ${duration} 秒"
    if $BACKUP_MONGO; then
        if $mongo_success; then
            success "MongoDB: 成功"
        else
            error "MongoDB: 失败"
        fi
    fi
    if $BACKUP_REDIS; then
        if $redis_success; then
            success "Redis: 成功"
        else
            error "Redis: 失败"
        fi
    fi
    info "=========================================="
    
    # 显示备份列表
    list_backups
    
    # 返回状态
    if $BACKUP_MONGO && ! $mongo_success; then
        exit 1
    fi
    if $BACKUP_REDIS && ! $redis_success; then
        exit 1
    fi
    
    exit 0
}

# 运行
main

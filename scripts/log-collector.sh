#!/bin/bash
# Log Collection Script for SecFlow
# Collects and analyzes logs from all components

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="${LOG_DIR:-$PROJECT_DIR/logs}"
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_DIR/logs-collected-$(date +%Y%m%d-%H%M%S)}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  SecFlow Log Collection Tool${NC}"
echo -e "${BLUE}=========================================${NC}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to collect logs from a container
collect_container_logs() {
    local container="$1"
    local output_file="$2"
    
    echo -n "  Collecting from $container ... "
    
    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        docker logs "$container" > "$output_file" 2>&1
        local size=$(du -h "$output_file" | cut -f1)
        echo -e "${GREEN}OK${NC} ($size)"
        return 0
    else
        echo -e "${YELLOW}SKIP${NC} (not running)"
        return 1
    fi
}

# Function to collect logs from a directory
collect_log_dir() {
    local log_src="$1"
    local output_subdir="$2"
    
    echo -n "  Collecting from $log_src ... "
    
    if [ -d "$log_src" ]; then
        mkdir -p "$OUTPUT_DIR/$output_subdir"
        cp -r "$log_src"/* "$OUTPUT_DIR/$output_subdir/" 2>/dev/null || true
        local count=$(find "$OUTPUT_DIR/$output_subdir" -type f | wc -l)
        echo -e "${GREEN}OK${NC} ($count files)"
        return 0
    else
        echo -e "${YELLOW}SKIP${NC} (not found)"
        return 1
    fi
}

echo ""
echo -e "${BLUE}[1/5] Creating output directory${NC}"
echo "  Output: $OUTPUT_DIR"

echo ""
echo -e "${BLUE}[2/5] Collecting Docker container logs${NC}"
collect_container_logs "secflow-server" "$OUTPUT_DIR/server-container.log"
collect_container_logs "secflow-client" "$OUTPUT_DIR/client-container.log"
collect_container_logs "secflow-mongodb" "$OUTPUT_DIR/mongodb-container.log"
collect_container_logs "secflow-redis" "$OUTPUT_DIR/redis-container.log"
collect_container_logs "secflow-nginx" "$OUTPUT_DIR/nginx-container.log"

echo ""
echo -e "${BLUE}[3/5] Collecting file-based logs${NC}"
collect_log_dir "$LOG_DIR/server" "server-files"
collect_log_dir "$LOG_DIR/client" "client-files"
collect_log_dir "/var/log/secflow" "var-log-secflow"

echo ""
echo -e "${BLUE}[4/5] Collecting system info${NC}"
{
    echo "=== System Information ==="
    uname -a
    echo ""
    echo "=== Docker Version ==="
    docker --version 2>/dev/null || echo "Docker not available"
    echo ""
    echo "=== Docker Containers ==="
    docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Image}}" 2>/dev/null || echo "Docker not available"
    echo ""
    echo "=== Disk Usage ==="
    df -h
    echo ""
    echo "=== Memory Usage ==="
    free -h
} > "$OUTPUT_DIR/system-info.txt"
echo -e "  ${GREEN}OK${NC}"

echo ""
echo -e "${BLUE}[5/5] Creating summary${NC}"
{
    echo "=== Log Collection Summary ==="
    echo "Collected: $(find "$OUTPUT_DIR" -type f | wc -l) files"
    echo ""
    echo "=== Error Summary (last 24h equivalent) ==="
    for log in "$OUTPUT_DIR"/*.log; do
        if [ -f "$log" ]; then
            echo ""
            echo "--- $(basename "$log") ---"
            grep -i "error\|fatal\|panic" "$log" 2>/dev/null | tail -20 || echo "No errors found"
        fi
    done
} > "$OUTPUT_DIR/summary.txt"
echo -e "  ${GREEN}OK${NC}"

# Create archive
ARCHIVE="${OUTPUT_DIR}.tar.gz"
tar -czf "$ARCHIVE" -C "$(dirname "$OUTPUT_DIR")" "$(basename "$OUTPUT_DIR")"

echo ""
echo -e "${BLUE}=========================================${NC}"
echo -e "${GREEN}Collection complete!${NC}"
echo ""
echo "Output directory: $OUTPUT_DIR"
echo "Archive: $ARCHIVE"
echo ""
echo "Next steps:"
echo "  1. Review summary.txt for error overview"
echo "  2. Analyze logs in $OUTPUT_DIR/"
echo "  3. Ship logs to Loki/Prometheus for advanced analysis"
echo -e "${BLUE}=========================================${NC}"

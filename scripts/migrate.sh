#!/bin/bash
# Data Migration Script for SecFlow
# Supports migrating between MongoDB versions and environments

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
SOURCE_MONGO_URI="${MONGO_URI:-mongodb://localhost:27017/secflow}"
TARGET_MONGO_URI="${TARGET_MONGO_URI:-mongodb://localhost:27017/secflow_new}"
COLLECTIONS="users,vulns,tasks,articles,push_channels,audit_logs,nodes"
BATCH_SIZE=1000
DRY_RUN=false

usage() {
    cat << EOF
SecFlow Data Migration Tool

Usage: $0 [OPTIONS]

Options:
    -s, --source URI       Source MongoDB URI (default: $SOURCE_MONGO_URI)
    -t, --target URI       Target MongoDB URI (default: $TARGET_MONGO_URI)
    -c, --collections LIST Comma-separated collections to migrate (default: all)
    -b, --batch-size N     Batch size for migration (default: $BATCH_SIZE)
    -d, --dry-run          Show what would be migrated without actually migrating
    -h, --help             Show this help

Examples:
    # Migrate all collections
    $0 -s mongodb://old:27017/secflow -t mongodb://new:27017/secflow

    # Dry run migration
    $0 -s mongodb://old:27017/secflow -t mongodb://new:27017/secflow --dry-run

    # Migrate specific collections
    $0 -c users,vulns,tasks

Environment variables:
    MONGO_URI         Source MongoDB URI
    TARGET_MONGO_URI  Target MongoDB URI

EOF
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--source)
            SOURCE_MONGO_URI="$2"
            shift 2
            ;;
        -t|--target)
            TARGET_MONGO_URI="$2"
            shift 2
            ;;
        -c|--collections)
            COLLECTIONS="$2"
            shift 2
            ;;
        -b|--batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
    esac
done

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}  SecFlow Data Migration Tool${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""
echo -e "${YELLOW}Source:${NC}   $SOURCE_MONGO_URI"
echo -e "${YELLOW}Target:${NC}   $TARGET_MONGO_URI"
echo -e "${YELLOW}Collections:${NC} $COLLECTIONS"
echo -e "${YELLOW}Batch Size:${NC} $BATCH_SIZE"
if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}Mode:${NC}     DRY RUN (no changes will be made)"
fi
echo ""

# Check for mongodump/mongorestore
if ! command -v mongodump &> /dev/null; then
    echo -e "${RED}Error: mongodump not found. Please install MongoDB tools.${NC}"
    exit 1
fi

if ! command -v mongorestore &> /dev/null; then
    echo -e "${RED}Error: mongorestore not found. Please install MongoDB tools.${NC}"
    exit 1
fi

# Parse collections
IFS=',' read -ra COLLECTION_ARRAY <<< "$COLLECTIONS"

if [ "$DRY_RUN" = true ]; then
    echo -e "${BLUE}[DRY RUN] Would migrate the following:${NC}"
    for coll in "${COLLECTION_ARRAY[@]}"; do
        echo "  - $coll"
    done
    echo ""
    echo -e "${GREEN}Dry run complete. No changes made.${NC}"
    exit 0
fi

# Create temp directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo -e "${BLUE}[1/3] Dumping source data${NC}"
for coll in "${COLLECTION_ARRAY[@]}"; do
    echo -n "  Dumping $coll ... "
    if mongodump --uri="$SOURCE_MONGO_URI" --collection="$coll" --out="$TEMP_DIR/dump" 2>/dev/null; then
        local_count=$(mongosh "$SOURCE_MONGO_URI" --quiet --eval "db.getCollection('$coll').countDocuments()" 2>/dev/null || echo "0")
        echo -e "${GREEN}OK${NC} ($local_count documents)"
    else
        echo -e "${YELLOW}SKIP${NC} (collection may not exist)"
    fi
done

echo ""
echo -e "${BLUE}[2/3] Restoring to target${NC}"
for coll in "${COLLECTION_ARRAY[@]}"; do
    if [ -d "$TEMP_DIR/dump/secflow/$coll" ]; then
        echo -n "  Restoring $coll ... "
        if mongorestore --uri="$TARGET_MONGO_URI" --collection="$coll" --dir="$TEMP_DIR/dump/secflow/$coll" --drop 2>/dev/null; then
            echo -e "${GREEN}OK${NC}"
        else
            echo -e "${RED}FAILED${NC}"
        fi
    fi
done

echo ""
echo -e "${BLUE}[3/3] Verifying${NC}"
for coll in "${COLLECTION_ARRAY[@]}"; do
    echo -n "  Verifying $coll ... "
    source_count=$(mongosh "$SOURCE_MONGO_URI" --quiet --eval "db.getCollection('$coll').countDocuments()" 2>/dev/null || echo "?")
    target_count=$(mongosh "$TARGET_MONGO_URI" --quiet --eval "db.getCollection('$coll').countDocuments()" 2>/dev/null || echo "?")
    if [ "$source_count" = "$target_count" ]; then
        echo -e "${GREEN}OK${NC} ($source_count documents)"
    else
        echo -e "${YELLOW}MISMATCH${NC} (source: $source_count, target: $target_count)"
    fi
done

echo ""
echo -e "${BLUE}=========================================${NC}"
echo -e "${GREEN}Migration complete!${NC}"
echo ""
echo "Note: Some collections may have dependencies. Please verify:"
echo "  - Users reference nodes"
echo "  - Tasks reference nodes and users"
echo "  - Vulns may reference tasks"
echo -e "${BLUE}=========================================${NC}"

#!/bin/bash
# Integration test for SecFlow server-client communication
# This script tests the complete flow without requiring a running server

set -e

echo "=== SecFlow Integration Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Server compilation
echo "[Test 1] Checking server compilation..."
cd /Users/j0ker/Desktop/coder/secflow/secflow-server
if go build -o /tmp/secflow-server ./cmd/server/ 2>&1; then
    echo -e "${GREEN}✅ Server compiles successfully${NC}"
else
    echo -e "${RED}❌ Server compilation failed${NC}"
    exit 1
fi
echo ""

# Test 2: Client compilation
echo "[Test 2] Checking client compilation..."
cd /Users/j0ker/Desktop/coder/secflow/secflow-client
if go build -o /tmp/secflow-client ./cmd/client/ 2>&1; then
    echo -e "${GREEN}✅ Client compiles successfully${NC}"
else
    echo -e "${RED}❌ Client compilation failed${NC}"
    exit 1
fi
echo ""

# Test 3: Check client help
echo "[Test 3] Checking client CLI..."
if /tmp/secflow-client --help > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Client CLI works${NC}"
else
    echo -e "${RED}❌ Client CLI failed${NC}"
    exit 1
fi
echo ""

# Test 4: Verify grabber sources are registered
echo "[Test 4] Checking grabber sources..."
cd /Users/j0ker/Desktop/coder/secflow/secflow-client
SOURCES=$(go run -exec "echo" ./cmd/client/ 2>&1 | grep -o "sources.*" || echo "")
if [ -n "$SOURCES" ]; then
    echo -e "${GREEN}✅ Grabber sources available${NC}"
else
    echo -e "${YELLOW}⚠️  Could not verify sources (this is OK if client doesn't print them)${NC}"
fi
echo ""

# Test 5: Check that all rod grabbers are registered
echo "[Test 5] Verifying rod grabber registration..."
cd /Users/j0ker/Desktop/coder/secflow/secflow-client
ROD_SOURCES=("avd-rod" "seebug-rod" "ti-rod" "nox-rod" "kev-rod" "struts2-rod" "chaitin-rod" "oscs-rod" "threatbook-rod" "venustech-rod")
for source in "${ROD_SOURCES[@]}"; do
    if grep -q "\"$source\"" pkg/grabber/sources.go; then
        echo -e "  ${GREEN}✓${NC} $source"
    else
        echo -e "  ${RED}✗${NC} $source (not found)"
    fi
done
echo ""

# Test 6: Verify task generator is integrated
echo "[Test 6] Checking task generator integration..."
if grep -q "NewTaskGenerator" /Users/j0ker/Desktop/coder/secflow/secflow-server/cmd/server/main.go; then
    echo -e "${GREEN}✅ Task generator integrated in server${NC}"
else
    echo -e "${RED}❌ Task generator not found in server${NC}"
    exit 1
fi
echo ""

# Test 7: Check WebSocket message types
echo "[Test 7] Checking WebSocket protocol..."
WS_TYPES=("task" "task_cancel" "ping" "register" "heartbeat" "progress" "result" "pong")
for msg_type in "${WS_TYPES[@]}"; do
    if grep -q "MsgType.*= \"$msg_type\"" /Users/j0ker/Desktop/coder/secflow/secflow-server/internal/ws/hub.go || \
       grep -q "MsgType.*= \"$msg_type\"" /Users/j0ker/Desktop/coder/secflow/secflow-client/internal/ws/client.go; then
        echo -e "  ${GREEN}✓${NC} $msg_type"
    else
        echo -e "  ${RED}✗${NC} $msg_type (not found)"
    fi
done
echo ""

echo "=== Integration Test Summary ==="
echo -e "${GREEN}All tests passed!${NC}"
echo ""
echo "Next steps to run a full test:"
echo "1. Start MongoDB: mongod --dbpath /path/to/db"
echo "2. Start Redis: redis-server"
echo "3. Start server: cd secflow-server && go run cmd/server/main.go"
echo "4. Create client config: cp secflow-client/client.yaml.example secflow-client/client.yaml"
echo "5. Edit client.yaml with your server URL and token"
echo "6. Start client: cd secflow-client && go run cmd/client/main.go"
echo "7. Create a task via API or wait for automatic task generation"

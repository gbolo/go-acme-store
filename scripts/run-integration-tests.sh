#!/bin/bash
# Integration test runner script
# This script starts the acme-store daemon in the background and runs integration tests

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
CONFIG_FILE="./testdata/config/test.yml"
DAEMON_PID=""
LOG_FILE="/tmp/acme-store-test.log"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up...${NC}"
    
    if [ -n "$DAEMON_PID" ]; then
        echo "Stopping acme-store daemon (PID: $DAEMON_PID)..."
        kill $DAEMON_PID 2>/dev/null || true
        wait $DAEMON_PID 2>/dev/null || true
    fi
    
    # Remove log file
    rm -f "$LOG_FILE"
    
    echo -e "${GREEN}Cleanup complete${NC}"
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

# Check if test environment is running
check_environment() {
    echo -e "${YELLOW}Checking test environment...${NC}"
    
    # Check Vault
    if ! curl -s http://localhost:8200/v1/sys/health > /dev/null 2>&1; then
        echo -e "${RED}✗ Vault is not running${NC}"
        echo "  Run: make test-setup"
        exit 1
    fi
    echo -e "${GREEN}✓ Vault is running${NC}"
    
    # Check Pebble
    if ! curl -sk https://localhost:14000/dir > /dev/null 2>&1; then
        echo -e "${RED}✗ Pebble is not running${NC}"
        echo "  Run: make test-setup"
        exit 1
    fi
    echo -e "${GREEN}✓ Pebble is running${NC}"
}

# Start acme-store daemon
start_daemon() {
    echo -e "${YELLOW}Starting acme-store daemon...${NC}"
    
    if [ ! -f "./acme-store" ]; then
        echo "Building acme-store..."
        make build-daemon
    fi
    
    # Start daemon in background
    CONFIG_FILE="$CONFIG_FILE" VAULT_TOKEN="root" ./acme-store > "$LOG_FILE" 2>&1 &
    DAEMON_PID=$!
    
    echo "Daemon started with PID: $DAEMON_PID"
    echo "Logs: $LOG_FILE"
    
    # Wait for daemon to be ready
    echo "Waiting for daemon to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:15872/api/healthz > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Daemon is ready${NC}"
            return 0
        fi
        sleep 1
    done
    
    echo -e "${RED}✗ Daemon failed to start${NC}"
    echo "Last 20 lines of log:"
    tail -20 "$LOG_FILE"
    exit 1
}

# Run tests
run_tests() {
    echo -e "${YELLOW}Running integration tests...${NC}"
    
    export CONFIG_FILE="$CONFIG_FILE"
    export VAULT_TOKEN="root"
    
    if go test -v -tags=integration ./tests/integration/... -timeout 5m; then
        echo -e "${GREEN}✓ All tests passed${NC}"
        return 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        echo ""
        echo "Daemon logs:"
        tail -50 "$LOG_FILE"
        return 1
    fi
}

# Main execution
main() {
    echo "========================================="
    echo "  ACME Store Integration Tests"
    echo "========================================="
    echo ""
    
    # Check environment
    check_environment
    echo ""
    
    # Start daemon
    start_daemon
    echo ""
    
    # Run tests
    run_tests
    TEST_RESULT=$?
    echo ""
    
    # Show summary
    echo "========================================="
    if [ $TEST_RESULT -eq 0 ]; then
        echo -e "${GREEN}  ✓ Integration tests PASSED${NC}"
    else
        echo -e "${RED}  ✗ Integration tests FAILED${NC}"
    fi
    echo "========================================="
    
    exit $TEST_RESULT
}

# Run main
main


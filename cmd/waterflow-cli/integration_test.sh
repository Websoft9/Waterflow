#!/bin/bash
# Waterflow CLI Integration Test Script

set -e

CLI="./bin/waterflow"
FAILED=0
PASSED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Test function
test_cmd() {
    local desc="$1"
    local cmd="$2"
    local expected="$3"
    
    echo -n "Testing: $desc... "
    
    if eval "$cmd" | grep -q "$expected"; then
        echo -e "${GREEN}PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}FAILED${NC}"
        echo "  Command: $cmd"
        echo "  Expected: $expected"
        FAILED=$((FAILED + 1))
    fi
}

# Test 1: Help command
test_cmd "Help command" \
    "$CLI --help" \
    "Waterflow CLI provides"

# Test 2: Version command (long form)
test_cmd "Version command (long)" \
    "$CLI version" \
    "Version:"

# Test 3: Version flag (short form)
test_cmd "Version flag (short)" \
    "$CLI --version" \
    "Waterflow CLI"

# Test 4: Invalid command
if $CLI invalidcmd 2>&1 | grep -q "unknown command"; then
    echo -e "Testing: Invalid command... ${GREEN}PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "Testing: Invalid command... ${RED}FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 5: Debug mode
test_cmd "Debug mode" \
    "$CLI --debug version 2>&1" \
    "DEBUG: Server URL"

# Test 6: Environment variable (cleanup first)
rm -f ~/.waterflow/config.yaml
if WATERFLOW_SERVER=http://test.example.com $CLI --debug version 2>&1 | grep -q "http://test.example.com"; then
    echo -e "Testing: Environment variable... ${GREEN}PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "Testing: Environment variable... ${RED}FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 7: Config file
mkdir -p ~/.waterflow
cat > ~/.waterflow/config.yaml << CFGEOF
server: http://config.example.com
api_key: test-api-key
debug: false
CFGEOF

test_cmd "Config file loading" \
    "$CLI --debug version 2>&1" \
    "http://config.example.com"

# Test 8: Command line parameter priority
test_cmd "Command line parameter priority" \
    "$CLI --server http://cli.example.com --debug version 2>&1" \
    "http://cli.example.com"

# Test 9: Missing required argument (AC4 scenario 2)
if $CLI submit 2>&1 | grep -qE "required argument|arg\(s\)"; then
    echo -e "Testing: Missing required argument... ${GREEN}PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "Testing: Missing required argument... ${RED}FAILED${NC}"
    FAILED=$((FAILED + 1))
fi

# Test 10: Invalid config file format (AC4 scenario 3)
echo "invalid: yaml: content:" > ~/.waterflow/config.yaml
if $CLI version 2>&1 | grep -q "failed to read config file"; then
    echo -e "Testing: Invalid config file format... ${GREEN}PASSED${NC}"
    PASSED=$((PASSED + 1))
else
    echo -e "Testing: Invalid config file format... ${RED}FAILED${NC}"
    FAILED=$((FAILED + 1))
fi
rm -f ~/.waterflow/config.yaml

# Summary
echo ""
echo "============================="
echo "Integration Test Summary"
echo "============================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo "============================="

if [ $FAILED -gt 0 ]; then
    exit 1
fi

echo "All tests passed!"

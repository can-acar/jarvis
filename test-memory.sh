#!/bin/bash

echo "=== Testing Jarvis Conversation Memory ==="
cd /Users/r00t/github/jarvis

echo -e "test folder oluştur\ndevam edelim\nexit" | timeout 30 ./jarvis --config action-oriented-config.json > test_output.log 2>&1 &
JARVIS_PID=$!

sleep 15
kill $JARVIS_PID 2>/dev/null || true

echo "Jarvis test output:"
cat test_output.log

echo "=== Test completed ==="

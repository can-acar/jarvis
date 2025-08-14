#!/bin/bash

# Test conversation flow for Jarvis
echo "=== Testing Jarvis Conversation History ==="

cd /Users/r00t/github/jarvis

# Create a temporary test file to interact with Jarvis
cat > /tmp/jarvis_test_input.txt << 'EOF'
test klasörü oluştur
devam edelim
exit
EOF

echo "Starting Jarvis with test input..."
./jarvis < /tmp/jarvis_test_input.txt

echo "=== Test completed ==="

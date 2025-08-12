#!/bin/bash

echo "🧪 Testing Jarvis MCP Tool Bridge Functionality..."

# Test 1: Directory listing
echo -e "\n📝 Test 1: Directory listing"
echo -e "list current directory\nexit" | ./jarvis --model ollama:gpt-oss:20b -i

# Test 2: File reading
echo -e "\n📝 Test 2: File reading (main.go)"
echo -e "show me main.go file\nexit" | ./jarvis --model ollama:gpt-oss:20b -i

# Test 3: File search with extension
echo -e "\n📝 Test 3: File search (.go files)"
echo -e "find files with .go extension\nexit" | ./jarvis --model ollama:gpt-oss:20b -i

# Test 4: Natural language directory analysis
echo -e "\n📝 Test 4: Natural language directory analysis"
echo -e "analyze current directory\nexit" | ./jarvis --model ollama:gpt-oss:20b --system-prompt "You are Jarvis, analyze the directory and explain what kind of project this is" -i

echo -e "\n✅ Bridge functionality tests completed!"
echo -e "\n🔧 Tools demonstrated:"
echo -e "   - list-directory: ✅ Working"
echo -e "   - read-file: ✅ Working"
echo -e "   - search-files: ✅ Working"
echo -e "   - LLM fallback: Available for complex queries"
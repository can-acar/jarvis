#!/bin/bash

# Test script for Jarvis Agent functionality
echo "🧪 Testing Jarvis Agent functionality..."

# Test 1: Help command
echo -e "\n📝 Test 1: Help functionality"
echo -e "help\nexit" | ./jarvis --model ollama:llama3.2 --interactive

# Test 2: Built-in commands
echo -e "\n📝 Test 2: Built-in commands"
echo -e "pwd\ncontext\nexit" | ./jarvis --model ollama:llama3.2 --interactive

# Test 3: Directory navigation
echo -e "\n📝 Test 3: Directory commands"
echo -e "cd ..\npwd\ncd jarvis\nexit" | ./jarvis --model ollama:llama3.2 --interactive

# Test 4: LLM Query (will fail if model not available)
echo -e "\n📝 Test 4: LLM Query (expected to fail if model not available)"
echo -e "Analyze current directory\nexit" | ./jarvis --model ollama:llama3.2 --system-prompt "You are Jarvis, a helpful AI assistant" --interactive

echo -e "\n✅ Tests completed!"
echo -e "\n💡 To use with a real model:"
echo -e "   1. Install Ollama: https://ollama.ai"
echo -e "   2. Pull a model: ollama pull llama3.2"
echo -e "   3. Run: ./jarvis --model ollama:llama3.2 --interactive"